package controllers

import (
	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/services"

	"github.com/go-fuego/fuego"
)

const unlockCookieMaxAge = 24 * 60 * 60

type PublicController struct {
	svc *services.SharesService
}

func NewPublicController(svc *services.SharesService) *PublicController {
	return &PublicController{svc: svc}
}

func (c *PublicController) Register(api *fuego.Server) {
	route := fuego.Group(api, "/s")

	fuego.Get(route, "/{token}", func(ctx fuego.ContextNoBody) (dtos.PublicShareDto, error) {
		share, err := c.svc.ResolvePublic(ctx.PathParam("token"))
		if err != nil {
			return dtos.PublicShareDto{}, err
		}

		unlocked := c.svc.Unlocked(share, unlockCookieValue(ctx.Request(), share.ID))
		return c.publicDto(share, unlocked)
	}, fuego.OptionOperationID("getPublicShare"))

	fuego.Post(route, "/{token}/unlock", func(ctx fuego.ContextWithBody[dtos.UnlockDto]) (dtos.PublicShareDto, error) {
		share, err := c.svc.ResolvePublic(ctx.PathParam("token"))
		if err != nil {
			return dtos.PublicShareDto{}, err
		}
		body, err := ctx.Body()
		if err != nil {
			return dtos.PublicShareDto{}, err
		}

		// The expiry check runs first: making someone produce a password only to
		// then be told the Share lapsed is gratuitous.
		if share.PublicAccessClosed() {
			return dtos.PublicShareDto{}, services.ErrShareExpired
		}

		unlockToken, err := c.svc.Unlock(share, body.Password)
		if err != nil {
			return dtos.PublicShareDto{}, err
		}

		ctx.SetCookie(shareUnlockCookie(share.ID, unlockToken, isSecure(ctx.Request()), unlockCookieMaxAge))
		return c.publicDto(share, true)
	}, fuego.OptionOperationID("unlockShare"))

	// Reports each file's fate for one upload session, so partial acceptance is
	// legible to the uploader. It also fires exactly one notification per batch.
	fuego.Post(route, "/{token}/uploads/finalise", func(ctx fuego.ContextWithBody[dtos.FinalizeUploadDto]) (dtos.FinalizeResultDto, error) {
		share, err := c.svc.ResolvePublic(ctx.PathParam("token"))
		if err != nil {
			return dtos.FinalizeResultDto{}, err
		}
		body, err := ctx.Body()
		if err != nil {
			return dtos.FinalizeResultDto{}, err
		}

		files, err := c.svc.FilesForSession(body.UploadSessionID)
		if err != nil {
			return dtos.FinalizeResultDto{}, err
		}

		outcomes := make([]dtos.UploadOutcomeDto, 0, len(files))
		label := ""
		accepted := 0
		for _, f := range files {
			if f.ShareID != share.ID {
				continue
			}
			if f.UploaderLabel != "" {
				label = f.UploaderLabel
			}
			accepted++
			outcomes = append(outcomes, dtos.UploadOutcomeDto{
				Name: f.Name, Size: f.Size, Accepted: true,
			})
		}

		if accepted > 0 && share.Mode == models.ShareModeUploadBox {
			c.svc.NotifyArrival(share, accepted, label)
		}

		return dtos.FinalizeResultDto{Outcomes: outcomes}, nil
	}, fuego.OptionOperationID("finaliseUpload"))
}

// publicDto is the only place a stranger's view is assembled. For an Upload Box
// it never carries files, counts or remaining capacity: "17 of 20 remaining"
// would reveal that three other people have already uploaded, which for sealed
// submissions is real signal leaking without anyone intending it.
func (c *PublicController) publicDto(share *models.Share, unlocked bool) (dtos.PublicShareDto, error) {
	state, err := c.svc.PublicState(share, unlocked)
	if err != nil {
		return dtos.PublicShareDto{}, err
	}

	out := dtos.PublicShareDto{
		Title:            share.Title,
		Mode:             string(share.Mode),
		State:            state,
		RequiresPassword: share.HasPassword(),
		Unlocked:         unlocked,
		Files:            []dtos.PublicFileDto{},
	}

	if share.Mode == models.ShareModeSend && state == "open" {
		for _, f := range share.Files {
			out.TotalBytes += f.Size
			out.Files = append(out.Files, dtos.PublicFileDto{
				Name:         f.Name,
				Size:         f.Size,
				DownloadPath: "/s/" + share.AccessToken + "/f/" + f.ID,
			})
		}
		out.ZipPath = "/s/" + share.AccessToken + "/all.zip"
	}

	return out, nil
}
