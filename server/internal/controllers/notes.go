package controllers

import (
	"errors"

	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/services"

	"github.com/go-fuego/fuego"
)

type NotesController struct {
	svc *services.NotesService
}

func NewNotesController(svc *services.NotesService) *NotesController {
	return &NotesController{svc: svc}
}

func (c *NotesController) Register(api *fuego.Server) {
	route := fuego.Group(api, "/notes")

	fuego.Get(route, "/", func(ctx fuego.ContextNoBody) ([]models.Note, error) {
		return c.svc.List()
	},
		fuego.OptionOperationID("getNotes"),
	)

	fuego.Post(route, "/", func(ctx fuego.ContextWithBody[dtos.CreateNoteDto]) (models.Note, error) {
		body, err := ctx.Body()
		if err != nil {
			return models.Note{}, err
		}
		return c.svc.Create(body)
	},
		fuego.OptionOperationID("createNote"),
	)

	fuego.Get(route, "/{id}", func(ctx fuego.ContextNoBody) (models.Note, error) {
		note, err := c.svc.Get(ctx.PathParam("id"))
		if errors.Is(err, services.ErrNoteNotFound) {
			return models.Note{}, fuego.NotFoundError{Title: "Not Found", Detail: "note not found"}
		}
		return note, err
	},
		fuego.OptionOperationID("getNote"),
	)

	fuego.Delete(route, "/{id}", func(ctx fuego.ContextNoBody) (any, error) {
		err := c.svc.Delete(ctx.PathParam("id"))
		if errors.Is(err, services.ErrNoteNotFound) {
			return nil, fuego.NotFoundError{Title: "Not Found", Detail: "note not found"}
		}
		return nil, err
	},
		fuego.OptionOperationID("deleteNote"),
		fuego.OptionDefaultStatusCode(204),
	)
}
