package controllers

import (
	"github.com/go-fuego/fuego"
	"go.uber.org/fx"
)

type Controller interface {
	Register(*fuego.Server)
}

var Module = fx.Provide(
	fx.Annotate(NewNotesController, fx.As(new(Controller)), fx.ResultTags(`group:"controllers"`)),
)
