package controllers

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"app/internal/database/models"
	"app/internal/services"
	"app/internal/storage"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"
	"github.com/tus/tusd/v2/pkg/filelocker"
	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

type UploadsController struct {
	svc     *services.SharesService
	auth    *services.AuthService
	storage *storage.Storage
	handler *tusd.UnroutedHandler
}

func NewUploadsController(
	svc *services.SharesService,
	auth *services.AuthService,
	store *storage.Storage,
) *UploadsController {
	c := &UploadsController{svc: svc, auth: auth, storage: store}

	if err := os.MkdirAll(store.TusDir(), 0o755); err != nil {
		panic("failed to create upload staging dir: " + err.Error())
	}

	composer := tusd.NewStoreComposer()
	fileStore := filestore.New(store.TusDir())
	fileStore.UseIn(composer)
	// A lock store is required for correctness, not optional: without it two
	// concurrent PATCHes to the same upload can interleave.
	filelocker.New(store.TusDir()).UseIn(composer)

	handler, err := tusd.NewUnroutedHandler(tusd.Config{
		BasePath:      "/api/uploads/",
		StoreComposer: composer,
		// Uploads are read back through our own authorised download endpoints, never
		// through tus.
		DisableDownload: true,
		// Behind a reverse proxy the Location header must reflect the public origin,
		// not the internal one.
		RespectForwardedHeaders: true,
		// Opt in to completion notifications, which is what drives committing the
		// blob into its Share. The channel is unbuffered and tus sends on it from
		// inside the request goroutine, so consumeCompletions must be running before
		// any upload finishes or PATCH would block forever.
		NotifyCompleteUploads: true,
		// Terminations must be observed too: without this a cancelled or failed
		// upload keeps its quota reservation until the abandoned-upload sweep runs.
		NotifyTerminatedUploads: true,
		// PreUploadCreateCallback is the only place real authorisation happens.
		// tus runs it on POST alone; the PATCH and HEAD that follow carry nothing
		// but the upload id, which is why that id is treated as a bearer secret
		// and why the share binding is recorded server-side here rather than read
		// back from client-supplied metadata later.
		PreUploadCreateCallback: c.authoriseCreate,
	})
	if err != nil {
		panic("failed to create tus handler: " + err.Error())
	}

	c.handler = handler
	go c.consumeCompletions()
	go c.consumeTerminations()
	return c
}

// RegisterRoot mounts tus method-agnostically. fuego v0.19.0 has no HeadStd, and
// tus needs HEAD, so this cannot be registered per method.
func (c *UploadsController) RegisterRoot(root *fuego.Server) {
	fuego.Handle(root, "/api/uploads/", http.StripPrefix("/api/uploads/", c.tusRouter()))
}

func (c *UploadsController) tusRouter() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			c.handler.PostFile(w, r)
		case http.MethodHead:
			c.handler.HeadFile(w, r)
		case http.MethodPatch:
			c.handler.PatchFile(w, r)
		case http.MethodDelete:
			c.handler.DelFile(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// authoriseCreate decides whether an upload may start, and reserves its quota.
//
// It handles both callers with one code path: an authenticated Owner uploading to
// their own Share, and an anonymous visitor uploading to an Upload Box whose link
// they hold. The distinction is drawn from the request, not from metadata.
func (c *UploadsController) authoriseCreate(hook tusd.HookEvent) (tusd.HTTPResponse, tusd.FileInfoChanges, error) {
	reject := func(status int, msg string) (tusd.HTTPResponse, tusd.FileInfoChanges, error) {
		return tusd.HTTPResponse{}, tusd.FileInfoChanges{}, tusd.NewError("ERR_FORBIDDEN", msg, status)
	}

	meta := hook.Upload.MetaData
	token := meta["shareToken"]
	if token == "" {
		return reject(http.StatusBadRequest, "shareToken metadata is required")
	}

	share, err := c.svc.ResolvePublic(token)
	if err != nil {
		return reject(http.StatusNotFound, "share not found")
	}

	// tus's hook carries only headers, not the request context, so the session is
	// resolved from the cookie here rather than read from middleware state.
	r := hook.HTTPRequest
	var user *models.User
	if token := cookieFromHeader(r.Header.Get("Cookie"), sessionCookieName); token != "" {
		user, _ = c.auth.Authenticate(token)
	}
	byOwner := user != nil && user.ID == share.OwnerID

	if !byOwner {
		// A visitor may only ever add to an Upload Box, never to a Send Share.
		if share.Mode != models.ShareModeUploadBox {
			return reject(http.StatusForbidden, "this share does not accept uploads")
		}
		if share.PublicAccessClosed() {
			return reject(http.StatusGone, "this share has expired")
		}
		if share.HasPassword() {
			cookie := cookieFromHeader(r.Header.Get("Cookie"), "dropla_s_"+share.ID)
			if !c.svc.Unlocked(share, cookie) {
				return reject(http.StatusUnauthorized, "this share is password protected")
			}
		}
	}

	declared := hook.Upload.Size
	if err := c.svc.CheckUploadAllowed(share, declared); err != nil {
		status := http.StatusRequestEntityTooLarge
		if err == services.ErrUploadBoxFull {
			status = http.StatusForbidden
		}
		return reject(status, err.Error())
	}

	// hook.Upload.ID is empty here: this callback runs before tus assigns one. So
	// the id is chosen here and handed back via FileInfoChanges, which lets the
	// reservation row and the upload share a primary key. Without this the
	// reservation is written under an empty id and the completion hook cannot
	// find it.
	uploadID := uuid.NewString()

	pending := &models.PendingUpload{
		ID:              uploadID,
		OwnerID:         share.OwnerID,
		ShareID:         share.ID,
		DeclaredSize:    declared,
		UploadSessionID: meta["uploadSessionId"],
		ByOwner:         byOwner,
		UploaderLabel:   truncate(meta["uploaderLabel"], 120),
		Name:            sanitiseFilename(meta["filename"]),
	}
	if err := c.svc.ReserveWithID(pending); err != nil {
		return reject(http.StatusInternalServerError, "could not reserve upload")
	}

	return tusd.HTTPResponse{}, tusd.FileInfoChanges{ID: uploadID}, nil
}

// consumeCompletions must exist for every notification channel that is enabled:
// tus sends on them from inside the request goroutine and they are unbuffered, so
// an unconsumed channel deadlocks PATCH.
func (c *UploadsController) consumeCompletions() {
	for event := range c.handler.CompleteUploads {
		c.commit(event)
	}
}

// Both notification channels need a consumer for the same reason: tus sends on
// them from inside the request goroutine and they are unbuffered.
func (c *UploadsController) consumeTerminations() {
	for event := range c.handler.TerminatedUploads {
		id := event.Upload.ID
		if err := c.svc.ReleasePending(id); err != nil {
			slog.Warn("could not release reservation for terminated upload", "upload", id, "error", err)
			continue
		}
		slog.Info("released reservation for terminated upload", "upload", id)
	}
}

func (c *UploadsController) commit(event tusd.HookEvent) {
	info := event.Upload

	pending, err := c.svc.Pending(info.ID)
	if err != nil || pending == nil {
		slog.Warn("completed upload has no reservation", "upload", info.ID)
		return
	}

	name := pending.Name
	if name == "" {
		name = sanitiseFilename(info.MetaData["filename"])
	}
	if name == "" {
		name = "file"
	}

	// info.Size is the real byte count, not the client's declared claim.
	file, err := c.svc.Commit(pending, name, info.Size, info.MetaData["filetype"])
	if err != nil {
		slog.Error("could not record uploaded file", "upload", info.ID, "error", err)
		return
	}

	src := c.storage.TusDir() + "/" + info.ID
	if err := c.storage.Commit(src, pending.ShareID, file.ID); err != nil {
		slog.Error("could not move uploaded blob into share", "upload", info.ID, "error", err)
		return
	}
	_ = os.Remove(src + ".info")
}

func cookieFromHeader(header, name string) string {
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if value, ok := strings.CutPrefix(part, name+"="); ok {
			return value
		}
	}
	return ""
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max]
}
