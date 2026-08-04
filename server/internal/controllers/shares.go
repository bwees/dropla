package controllers

import (
	"app/internal/config"
	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/services"
	"app/internal/storage"

	"github.com/go-fuego/fuego"
)

type SharesController struct {
	svc     *services.SharesService
	auth    *services.AuthService
	storage *storage.Storage
}

func NewSharesController(
	svc *services.SharesService,
	auth *services.AuthService,
	store *storage.Storage,
) *SharesController {
	return &SharesController{svc: svc, auth: auth, storage: store}
}

func sharePath(share *models.Share) string {
	return "/s/" + share.AccessToken
}

// publicURL is absolute only when PUBLIC_BASE_URL is configured. The private URL
// is deliberately not built here: the browser composes it from its own origin,
// which it knows more reliably than the server does behind a proxy.
func publicURL(path string) string {
	base := config.PublicBaseURL()
	if base == "" {
		return ""
	}
	return base + path
}

func shareDto(share *models.Share) dtos.ShareDto {
	files := make([]dtos.ShareFileDto, len(share.Files))
	var total int64

	for i, f := range share.Files {
		total += f.Size
		files[i] = dtos.ShareFileDto{
			ID:            f.ID,
			Name:          f.Name,
			Size:          f.Size,
			ContentType:   f.ContentType,
			ByOwner:       f.ByOwner,
			UploaderLabel: f.UploaderLabel,
			UploadedAt:    f.UploadedAt,
			DownloadPath:  "/api/shares/" + share.ID + "/f/" + f.ID,
		}
	}

	return dtos.ShareDto{
		ID:                share.ID,
		Title:             share.Title,
		Mode:              string(share.Mode),
		Path:              sharePath(share),
		PublicURL:         publicURL(sharePath(share)),
		HasPassword:       share.HasPassword(),
		Password:          share.Password,
		ExpiresAt:         share.ExpiresAt,
		Expired:           share.PublicAccessClosed(),
		MaxDownloads:      share.MaxDownloads,
		Downloads:         share.Downloads,
		MaxFiles:          share.MaxFiles,
		MaxBytes:          share.MaxBytes,
		NotifyURL:         share.NotifyURL,
		NotifyEnabled:     share.NotifyEnabled,
		NotifyLastError:   share.NotifyLastError,
		NotifyLastErrorAt: share.NotifyLastErrorAt,
		TotalBytes:        total,
		Files:             files,
		CreatedAt:         share.CreatedAt,
	}
}

func (c *SharesController) Register(api *fuego.Server) {
	route := fuego.Group(api, "/shares")

	fuego.Get(route, "/", func(ctx fuego.ContextNoBody) ([]dtos.ShareDto, error) {
		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return nil, err
		}

		shares, err := c.svc.List(user.ID)
		if err != nil {
			return nil, err
		}

		out := make([]dtos.ShareDto, len(shares))
		for i := range shares {
			out[i] = shareDto(&shares[i])
		}
		return out, nil
	}, fuego.OptionOperationID("getShares"))

	fuego.Post(route, "/", func(ctx fuego.ContextWithBody[dtos.CreateShareDto]) (dtos.ShareDto, error) {
		forceJSON(ctx.Response())

		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return dtos.ShareDto{}, err
		}
		body, err := ctx.Body()
		if err != nil {
			return dtos.ShareDto{}, err
		}

		share, err := c.svc.Create(user.ID, body)
		if err != nil {
			return dtos.ShareDto{}, err
		}
		return shareDto(share), nil
	}, fuego.OptionOperationID("createShare"), fuego.OptionDefaultStatusCode(201))

	fuego.Get(route, "/{id}", func(ctx fuego.ContextNoBody) (dtos.ShareDto, error) {
		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return dtos.ShareDto{}, err
		}
		share, err := c.svc.Owned(user.ID, ctx.PathParam("id"))
		if err != nil {
			return dtos.ShareDto{}, err
		}
		return shareDto(share), nil
	}, fuego.OptionOperationID("getShare"))

	fuego.Patch(route, "/{id}", func(ctx fuego.ContextWithBody[dtos.UpdateShareDto]) (dtos.ShareDto, error) {
		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return dtos.ShareDto{}, err
		}
		body, err := ctx.Body()
		if err != nil {
			return dtos.ShareDto{}, err
		}

		share, err := c.svc.Update(user.ID, ctx.PathParam("id"), body)
		if err != nil {
			return dtos.ShareDto{}, err
		}
		return shareDto(share), nil
	}, fuego.OptionOperationID("updateShare"))

	fuego.Delete(route, "/{id}", func(ctx fuego.ContextNoBody) (any, error) {
		stripTrailer(ctx.Response())

		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return nil, err
		}
		return nil, c.svc.Delete(user.ID, ctx.PathParam("id"))
	}, fuego.OptionOperationID("deleteShare"), fuego.OptionDefaultStatusCode(204))

	fuego.Post(route, "/{id}/rotate-link", func(ctx fuego.ContextNoBody) (dtos.ShareDto, error) {
		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return dtos.ShareDto{}, err
		}
		share, err := c.svc.RotateLink(user.ID, ctx.PathParam("id"))
		if err != nil {
			return dtos.ShareDto{}, err
		}
		return shareDto(share), nil
	}, fuego.OptionOperationID("rotateShareLink"))

	fuego.Delete(route, "/{id}/files/{fileId}", func(ctx fuego.ContextNoBody) (any, error) {
		stripTrailer(ctx.Response())

		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return nil, err
		}
		return nil, c.svc.DeleteFile(user.ID, ctx.PathParam("id"), ctx.PathParam("fileId"))
	}, fuego.OptionOperationID("deleteShareFile"), fuego.OptionDefaultStatusCode(204))

	usage := fuego.Group(api, "/usage")
	fuego.Get(usage, "/", func(ctx fuego.ContextNoBody) (dtos.UsageDto, error) {
		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return dtos.UsageDto{}, err
		}

		used, err := c.svc.Usage(user.ID)
		if err != nil {
			return dtos.UsageDto{}, err
		}
		shares, err := c.svc.List(user.ID)
		if err != nil {
			return dtos.UsageDto{}, err
		}

		return dtos.UsageDto{
			UsedBytes:  used,
			QuotaBytes: c.auth.QuotaFor(user),
			ShareCount: len(shares),
		}, nil
	}, fuego.OptionOperationID("getUsage"))

	// The admin storage view reports bytes and counts only — never titles or
	// filenames — so Upload Box privacy holds against admins. Free disk is
	// included because quotas have no global ceiling and can over-commit.
	storageRoute := fuego.Group(api, "/storage")
	fuego.Get(storageRoute, "/", func(ctx fuego.ContextNoBody) (dtos.StorageOverviewDto, error) {
		if _, err := requireAdmin(ctx.Request().Context()); err != nil {
			return dtos.StorageOverviewDto{}, err
		}

		users, err := c.auth.ListUsers()
		if err != nil {
			return dtos.StorageOverviewDto{}, err
		}

		rows := make([]dtos.StorageRowDto, len(users))
		for i := range users {
			used, err := c.svc.Usage(users[i].ID)
			if err != nil {
				return dtos.StorageOverviewDto{}, err
			}
			shares, err := c.svc.List(users[i].ID)
			if err != nil {
				return dtos.StorageOverviewDto{}, err
			}
			rows[i] = dtos.StorageRowDto{
				UserID:     users[i].ID,
				Username:   users[i].Username,
				UsedBytes:  used,
				QuotaBytes: c.auth.QuotaFor(&users[i]),
				ShareCount: len(shares),
			}
		}

		free, _ := c.storage.FreeBytes()
		return dtos.StorageOverviewDto{Users: rows, FreeBytes: free}, nil
	}, fuego.OptionOperationID("getStorageOverview"))
}
