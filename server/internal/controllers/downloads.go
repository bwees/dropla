package controllers

import (
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"app/internal/database/models"
	"app/internal/services"
	"app/internal/storage"
	"app/internal/zipstream"

	"github.com/go-fuego/fuego"
)

type DownloadsController struct {
	svc     *services.SharesService
	storage *storage.Storage
}

func NewDownloadsController(svc *services.SharesService, store *storage.Storage) *DownloadsController {
	return &DownloadsController{svc: svc, storage: store}
}

// RegisterRoot mounts the binary endpoints.
//
// The public paths deliberately live under /s/ rather than /api/s/, because they
// are plain browser navigations that must carry the unlock cookie. That puts them
// outside the auth middleware's /api prefix, which is correct — they authorise on
// the Share token themselves — but it has to be deliberate.
//
// Only the sub-paths are registered. /s/{token} itself is an SPA route and must
// fall through to index.html; ServeMux prefers the more specific pattern, so
// registering these does not shadow it.
func (c *DownloadsController) RegisterRoot(root *fuego.Server) {
	fuego.GetStd(root, "/s/{token}/all.zip", c.publicZip, fuego.OptionHide())
	fuego.GetStd(root, "/s/{token}/f/{fileId}", c.publicFile, fuego.OptionHide())

	// The Owner's own downloads, session-authorised because they sit under /api.
	// An Upload Box's contents are reachable only here, never publicly.
	fuego.GetStd(root, "/api/shares/{id}/all.zip", c.ownerZip, fuego.OptionHide())
	fuego.GetStd(root, "/api/shares/{id}/f/{fileId}", c.ownerFile, fuego.OptionHide())
}

func (c *DownloadsController) publicShare(w http.ResponseWriter, r *http.Request) (*models.Share, bool) {
	share, err := c.svc.ResolvePublic(r.PathValue("token"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return nil, false
	}

	if share.PublicAccessClosed() {
		http.Error(w, "this share has expired", http.StatusGone)
		return nil, false
	}

	if !c.svc.Unlocked(share, unlockCookieValue(r, share.ID)) {
		http.Error(w, "this share is password protected", http.StatusUnauthorized)
		return nil, false
	}

	// An Upload Box is write-only to the public: its contents are visible to the
	// Owner and to nobody else.
	if share.Mode != models.ShareModeSend {
		http.Error(w, "not found", http.StatusNotFound)
		return nil, false
	}

	return share, true
}

func (c *DownloadsController) ownerShare(w http.ResponseWriter, r *http.Request) (*models.Share, bool) {
	user := UserFrom(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, false
	}

	share, err := c.svc.Owned(user.ID, r.PathValue("id"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return nil, false
	}
	return share, true
}

func (c *DownloadsController) publicZip(w http.ResponseWriter, r *http.Request) {
	share, ok := c.publicShare(w, r)
	if !ok {
		return
	}
	c.serveZip(w, r, share, true)
}

func (c *DownloadsController) ownerZip(w http.ResponseWriter, r *http.Request) {
	share, ok := c.ownerShare(w, r)
	if !ok {
		return
	}
	c.serveZip(w, r, share, false)
}

func (c *DownloadsController) publicFile(w http.ResponseWriter, r *http.Request) {
	share, ok := c.publicShare(w, r)
	if !ok {
		return
	}
	c.serveFile(w, r, share, true)
}

func (c *DownloadsController) ownerFile(w http.ResponseWriter, r *http.Request) {
	share, ok := c.ownerShare(w, r)
	if !ok {
		return
	}
	c.serveFile(w, r, share, false)
}

func (c *DownloadsController) serveZip(w http.ResponseWriter, r *http.Request, share *models.Share, count bool) {
	if len(share.Files) == 0 {
		http.Error(w, "this share has no files", http.StatusNotFound)
		return
	}

	names := newNameDeduper()
	entries := make([]zipstream.Entry, 0, len(share.Files))
	for _, f := range share.Files {
		shareID, fileID := share.ID, f.ID
		entries = append(entries, zipstream.Entry{
			Name:     names.unique(sanitiseFilename(f.Name)),
			Size:     f.Size,
			Modified: f.UploadedAt,
			Open: func() (io.ReadCloser, error) {
				return c.storage.Open(shareID, fileID)
			},
		})
	}

	// A byte-exact Content-Length is what gives the recipient a real progress bar
	// rather than an indeterminate spinner — and it turns a truncated transfer
	// into "unexpected EOF" instead of an apparently successful download.
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", strconv.FormatInt(zipstream.Size(entries), 10))
	w.Header().Set("Content-Disposition", contentDisposition(share.Title+".zip"))
	// nginx buffers proxied responses by default, which would spool the whole
	// archive to a temp file and undo the point of streaming.
	w.Header().Set("X-Accel-Buffering", "no")

	if err := zipstream.Write(w, entries); err != nil {
		// Returning normally here would end the response cleanly and the browser
		// would report a successful download of a truncated archive. Aborting the
		// handler breaks the connection instead, which the client sees as an error.
		slog.Error("zip stream failed mid-response", "share", share.ID, "error", err)
		panic(http.ErrAbortHandler)
	}

	if count {
		_ = c.svc.CountDownload(share.ID)
	}
}

func (c *DownloadsController) serveFile(w http.ResponseWriter, r *http.Request, share *models.Share, count bool) {
	file, err := c.svc.File(share.ID, r.PathValue("fileId"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	f, err := c.storage.Open(share.ID, file.ID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	// Set Content-Type explicitly: letting ServeContent sniff it would classify
	// visitor-supplied HTML as text/html and serve it from our own origin, which
	// is a stored-XSS vector.
	w.Header().Set("Content-Type", contentTypeFor(file.ContentType, file.Name))
	w.Header().Set("Content-Disposition", contentDisposition(file.Name))
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("ETag", strconv.Quote(file.ID))

	// A download cap counts completed transfers, so range requests have to go:
	// two partial requests would otherwise reassemble the file without ever
	// incrementing the counter. The cost is that a capped Share cannot resume.
	if !share.RangeRequestsAllowed() {
		r.Header.Del("Range")
		w.Header().Set("Accept-Ranges", "none")
		w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
		if _, err := io.Copy(w, f); err != nil {
			slog.Error("file stream failed mid-response", "file", file.ID, "error", err)
			panic(http.ErrAbortHandler)
		}
		if count {
			_ = c.svc.CountDownload(share.ID)
		}
		return
	}

	http.ServeContent(w, r, file.Name, file.UploadedAt, f)

	// Only a complete response counts. A 206 is a slice of a transfer, and
	// counting it would let a resumed download spend several of the cap.
	if count && r.Header.Get("Range") == "" {
		_ = c.svc.CountDownload(share.ID)
	}
}

func contentTypeFor(declared, name string) string {
	if declared != "" && declared != "application/octet-stream" {
		// Never serve visitor-supplied markup as markup from our own origin.
		if isRenderable(declared) {
			return "application/octet-stream"
		}
		return declared
	}

	if byExt := mime.TypeByExtension(filepath.Ext(name)); byExt != "" && !isRenderable(byExt) {
		return byExt
	}
	return "application/octet-stream"
}

func isRenderable(contentType string) bool {
	lowered := strings.ToLower(contentType)
	for _, risky := range []string{"text/html", "application/xhtml", "image/svg", "text/xml", "application/xml"} {
		if strings.HasPrefix(lowered, risky) {
			return true
		}
	}
	return false
}

// contentDisposition always forces a download and never renders inline.
func contentDisposition(name string) string {
	safe := sanitiseFilename(name)
	return fmt.Sprintf("attachment; filename=%s; filename*=UTF-8''%s",
		strconv.Quote(asciiFallback(safe)), percentEncode(safe))
}
