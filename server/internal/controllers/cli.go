package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"app/internal/config"
	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/services"
	"app/internal/storage"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"
)

// CliController is the transfer.sh-shaped path: a single PUT with the file as the
// raw request body.
//
//	curl -T ./report.pdf  -H "Authorization: Bearer dpla_…"  https://d/api/cli/upload
//	curl -T ./bid.pdf                                        https://d/s/<token>
//
// The first creates a new share holding that one file. The second drops a file
// into someone's upload box and needs no credential, because the link itself
// already grants that.
//
// The third form needs no credential: an Upload Box link already grants upload
// rights to whoever holds it, which is the same permission the web drop zone uses.
//
// There is no resume here — a single PUT cannot. curl prints its own progress, so
// a failure is at least visible; anyone moving very large files from a script
// should use a tus client instead.
type CliController struct {
	shares  *services.SharesService
	auth    *services.AuthService
	storage *storage.Storage
}

func NewCliController(
	shares *services.SharesService,
	auth *services.AuthService,
	store *storage.Storage,
) *CliController {
	return &CliController{shares: shares, auth: auth, storage: store}
}

func (c *CliController) RegisterRoot(root *fuego.Server) {
	fuego.PutStd(root, "/api/cli/upload", c.uploadNew, fuego.OptionHide())
	fuego.PutStd(root, "/s/{token}", c.uploadToBox, fuego.OptionHide())
}

// fail writes a bare line for a shell and problem JSON for everyone else, so
// scripts do not have to pipe errors through jq to read them.
func (c *CliController) fail(w http.ResponseWriter, r *http.Request, status int, msg string) {
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"title":  http.StatusText(status),
			"detail": msg,
		})
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, msg+"\n")
}

func (c *CliController) authenticate(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	header := r.Header.Get("Authorization")
	secret, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		c.fail(w, r, http.StatusUnauthorized,
			"missing bearer token — create one under Account in the web UI")
		return nil, false
	}

	user, err := c.auth.AuthenticateAccessToken(strings.TrimSpace(secret))
	if err != nil {
		c.fail(w, r, http.StatusUnauthorized, "invalid or revoked token")
		return nil, false
	}
	return user, true
}

func (c *CliController) uploadNew(w http.ResponseWriter, r *http.Request) {
	user, ok := c.authenticate(w, r)
	if !ok {
		return
	}

	name, ok := c.filename(w, r)
	if !ok {
		return
	}

	dto := dtos.CreateShareDto{Title: name, Mode: string(models.ShareModeSend)}
	if days := headerInt(r, "Dropla-Expires-In"); days != nil {
		dto.ExpiresInDays = days
	}
	if max := headerInt(r, "Dropla-Max-Downloads"); max != nil {
		dto.MaxDownloads = max
	}
	if pw := r.Header.Get("Dropla-Password"); pw != "" {
		dto.Password = &pw
	}

	share, err := c.shares.Create(user.ID, dto)
	if err != nil {
		c.fail(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	c.store(w, r, share, user, name, true)
}

func (c *CliController) uploadToBox(w http.ResponseWriter, r *http.Request) {
	share, err := c.shares.ResolvePublic(r.PathValue("token"))
	if err != nil {
		c.fail(w, r, http.StatusNotFound, "not found")
		return
	}

	if share.Mode != models.ShareModeUploadBox {
		c.fail(w, r, http.StatusForbidden, "this share does not accept uploads")
		return
	}
	if share.PublicAccessClosed() {
		c.fail(w, r, http.StatusGone, "this share has expired")
		return
	}

	// curl carries no cookie jar by default, so a password arrives in a header.
	if share.HasPassword() {
		supplied := r.Header.Get("Dropla-Password")
		if _, err := c.shares.Unlock(share, supplied); err != nil {
			c.fail(w, r, http.StatusUnauthorized, "this box is password protected")
			return
		}
	}

	name, ok := c.filename(w, r)
	if !ok {
		return
	}
	c.store(w, r, share, nil, name, false)
}

// filename rejects a hostile name loudly rather than silently normalising it, so
// a script doing something wrong finds out.
func (c *CliController) filename(w http.ResponseWriter, r *http.Request) (string, bool) {
	name := r.Header.Get("Dropla-Filename")
	if name == "" {
		name = r.URL.Query().Get("name")
	}
	if name == "" {
		name = path.Base(r.URL.Path)
		// The trailing path segment is the endpoint itself, not a filename.
		if name == "upload" || name == "" || strings.HasPrefix(name, "s") && r.PathValue("token") == name {
			name = ""
		}
	}
	if name == "" {
		c.fail(w, r, http.StatusBadRequest,
			"a filename is required — pass ?name=report.pdf or a Dropla-Filename header")
		return "", false
	}

	if filenameIsSuspicious(name) {
		c.fail(w, r, http.StatusBadRequest, "invalid filename: "+name)
		return "", false
	}
	return sanitiseFilename(name), true
}

func (c *CliController) store(w http.ResponseWriter, r *http.Request, share *models.Share, owner *models.User, name string, byOwner bool) {
	declared := r.ContentLength
	if declared < 0 {
		declared = 0
	}

	if err := c.shares.CheckUploadAllowed(share, declared); err != nil {
		c.fail(w, r, statusFor(err), err.Error())
		return
	}

	// An upload of unknown length (curl -T - from a pipe) sends no Content-Length,
	// so there is nothing to check up front. Allow it and enforce the quota against
	// the running total instead, which keeps `tar … | curl -T -` working.
	remaining, err := c.shares.RemainingQuota(share)
	if err != nil {
		c.fail(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	body := io.Reader(r.Body)
	limit := int64(-1)
	if remaining >= 0 {
		limit = remaining
	}
	if max := config.MaxUploadBytes(); max > 0 && (limit < 0 || max < limit) {
		limit = max
	}
	if share.MaxBytes != nil {
		used, _, err := c.shares.ShareUsage(share.ID)
		if err == nil {
			boxRoom := *share.MaxBytes - used
			if boxRoom < 0 {
				boxRoom = 0
			}
			if limit < 0 || boxRoom < limit {
				limit = boxRoom
			}
		}
	}
	if limit >= 0 {
		// One byte past the limit is enough to detect a breach.
		body = io.LimitReader(r.Body, limit+1)
	}

	pending := &models.PendingUpload{
		ID:              uuid.NewString(),
		OwnerID:         share.OwnerID,
		ShareID:         share.ID,
		DeclaredSize:    declared,
		UploadSessionID: uuid.NewString(),
		ByOwner:         byOwner,
		UploaderLabel:   truncate(r.Header.Get("Dropla-Uploader"), 120),
		Name:            name,
	}
	if err := c.shares.ReserveWithID(pending); err != nil {
		c.fail(w, r, http.StatusInternalServerError, "could not reserve upload")
		return
	}

	fileID := uuid.NewString()
	written, err := c.storage.Write(share.ID, fileID, body)
	if err != nil {
		_ = c.shares.ReleasePending(pending.ID)
		c.fail(w, r, http.StatusInternalServerError, "upload failed: "+err.Error())
		return
	}

	if limit >= 0 && written > limit {
		_ = c.storage.Remove(share.ID, fileID)
		_ = c.shares.ReleasePending(pending.ID)
		c.fail(w, r, http.StatusRequestEntityTooLarge,
			"upload refused: it would exceed the available space")
		return
	}

	file, err := c.shares.Commit(pending, name, written, r.Header.Get("Content-Type"))
	if err != nil {
		_ = c.storage.Remove(share.ID, fileID)
		c.fail(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Commit generated its own id; move the blob to match it.
	if file.ID != fileID {
		_ = c.storage.Commit(c.storage.Path(share.ID, fileID), share.ID, file.ID)
	}

	if !byOwner && share.Mode == models.ShareModeUploadBox {
		c.shares.NotifyArrival(share, 1, pending.UploaderLabel)
	}

	c.respond(w, r, share, file, written)
}

func (c *CliController) respond(w http.ResponseWriter, r *http.Request, share *models.Share, file *models.ShareFile, size int64) {
	base := config.PublicBaseURL()
	if base == "" {
		// A shell has no window.location for the server to defer to, so fall back
		// to the origin the request actually arrived on.
		scheme := "http"
		if isSecure(r) {
			scheme = "https"
		}
		base = scheme + "://" + r.Host
	}

	out := dtos.CliUploadResultDto{
		ShareID:   share.ID,
		FileID:    file.ID,
		URL:       base + "/s/" + share.AccessToken,
		ManageURL: base + "/shares/" + share.ID,
		Size:      size,
		ExpiresAt: share.ExpiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func statusFor(err error) int {
	if coder, ok := err.(interface{ StatusCode() int }); ok {
		return coder.StatusCode()
	}
	return http.StatusInternalServerError
}

func headerInt(r *http.Request, name string) *int {
	raw := r.Header.Get(name)
	if raw == "" {
		return nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &n
}
