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
// It handles three callers: an authenticated Owner adding to their own Share, an
// anonymous visitor uploading to an Upload Box whose link they hold, and an Owner
// sending files that have no Share yet. The distinction is drawn from the request,
// not from metadata.
func (c *UploadsController) authoriseCreate(hook tusd.HookEvent) (tusd.HTTPResponse, tusd.FileInfoChanges, error) {
	meta := hook.Upload.MetaData

	// tus's hook carries only headers, not the request context, so the session is
	// resolved from the cookie here rather than read from middleware state.
	var user *models.User
	if token := cookieFromHeader(hook.HTTPRequest.Header.Get("Cookie"), sessionCookieName); token != "" {
		user, _ = c.auth.Authenticate(token)
	}

	// hook.Upload.ID is empty here: this callback runs before tus assigns one. So
	// the id is chosen here and handed back via FileInfoChanges, which lets the
	// reservation row and the upload share a primary key. Without this the
	// reservation is written under an empty id and the completion hook cannot
	// find it.
	pending := &models.PendingUpload{
		ID:              uuid.NewString(),
		DeclaredSize:    hook.Upload.Size,
		UploadSessionID: meta["uploadSessionId"],
		UploaderLabel:   truncate(meta["uploaderLabel"], 120),
		Name:            sanitiseFilename(meta["filename"]),
	}

	var err error
	if token := meta["shareToken"]; token != "" {
		err = c.authoriseIntoShare(hook.HTTPRequest, user, token, pending)
	} else {
		err = c.authoriseNewShare(user, meta["newShareTitle"], pending)
	}
	if err != nil {
		return tusd.HTTPResponse{}, tusd.FileInfoChanges{}, err
	}

	if err := c.svc.ReserveWithID(pending); err != nil {
		return tusd.HTTPResponse{}, tusd.FileInfoChanges{}, refuse(http.StatusInternalServerError, "could not reserve upload")
	}
	return tusd.HTTPResponse{}, tusd.FileInfoChanges{ID: pending.ID}, nil
}

func (c *UploadsController) authoriseIntoShare(r tusd.HTTPRequest, user *models.User, token string, pending *models.PendingUpload) error {
	share, err := c.svc.ResolvePublic(token)
	if err != nil {
		return refuse(http.StatusNotFound, "share not found")
	}

	byOwner := user != nil && user.ID == share.OwnerID
	if !byOwner {
		// A visitor may only ever add to an Upload Box, never to a Send Share.
		if share.Mode != models.ShareModeUploadBox {
			return refuse(http.StatusForbidden, "this share does not accept uploads")
		}
		if share.PublicAccessClosed() {
			return refuse(http.StatusGone, "this share has expired")
		}
		if share.HasPassword() {
			cookie := cookieFromHeader(r.Header.Get("Cookie"), "dropla_s_"+share.ID)
			if !c.svc.Unlocked(share, cookie) {
				return refuse(http.StatusUnauthorized, "this share is password protected")
			}
		}
	}

	if err := c.svc.CheckUploadAllowed(share, pending.DeclaredSize); err != nil {
		return refuseUpload(err)
	}

	pending.OwnerID = share.OwnerID
	pending.ShareID = share.ID
	pending.ByOwner = byOwner
	return nil
}

// authoriseNewShare admits an upload that has no Share to go into. The Share is
// created when the first file lands, so a drop where everything fails leaves no
// empty Share behind — and a Share is never made on the strength of an upload
// that might never arrive.
func (c *UploadsController) authoriseNewShare(user *models.User, title string, pending *models.PendingUpload) error {
	if user == nil {
		return refuse(http.StatusUnauthorized, "sign in to send files")
	}
	if err := c.svc.CheckOwnerUploadAllowed(user.ID, pending.DeclaredSize); err != nil {
		return refuseUpload(err)
	}

	pending.OwnerID = user.ID
	pending.ByOwner = true
	pending.NewShareTitle = truncate(title, 200)
	return nil
}

func refuse(status int, msg string) error {
	return tusd.NewError("ERR_FORBIDDEN", msg, status)
}

func refuseUpload(err error) error {
	status := http.StatusRequestEntityTooLarge
	if err == services.ErrUploadBoxFull {
		status = http.StatusForbidden
	}
	return refuse(status, err.Error())
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

	share, err := c.svc.ShareForPending(pending)
	if err != nil || share == nil {
		slog.Error("could not resolve the share for a completed upload", "upload", info.ID, "error", err)
		return
	}

	// info.Size is the real byte count, not the client's declared claim.
	file, err := c.svc.Commit(pending, share.ID, name, info.Size, info.MetaData["filetype"])
	if err != nil {
		slog.Error("could not record uploaded file", "upload", info.ID, "error", err)
		return
	}

	src := c.storage.TusDir() + "/" + info.ID
	if err := c.storage.Commit(src, share.ID, file.ID); err != nil {
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
