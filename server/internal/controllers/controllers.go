package controllers

import (
	"github.com/go-fuego/fuego"
	"go.uber.org/fx"
)

// Controller registers routes under the /api group.
type Controller interface {
	Register(*fuego.Server)
}

// RootController registers routes on the bare server rather than the /api group.
//
// Needed because the public download paths live under /s/ so that plain browser
// navigations carry the unlock cookie, and because tus needs a method-agnostic
// mount. Implementations write full paths, since the root server has no prefix.
type RootController interface {
	RegisterRoot(*fuego.Server)
}

var Module = fx.Provide(
	fx.Annotate(NewAuthController, fx.As(new(Controller)), fx.ResultTags(`group:"controllers"`)),
	fx.Annotate(NewUsersController, fx.As(new(Controller)), fx.ResultTags(`group:"controllers"`)),
	fx.Annotate(NewTokensController, fx.As(new(Controller)), fx.ResultTags(`group:"controllers"`)),
	fx.Annotate(NewSharesController, fx.As(new(Controller)), fx.ResultTags(`group:"controllers"`)),
	fx.Annotate(NewPublicController, fx.As(new(Controller)), fx.ResultTags(`group:"controllers"`)),
	fx.Annotate(NewConfigController, fx.As(new(Controller)), fx.ResultTags(`group:"controllers"`)),

	fx.Annotate(NewDownloadsController, fx.As(new(RootController)), fx.ResultTags(`group:"rootControllers"`)),
	fx.Annotate(NewUploadsController, fx.As(new(RootController)), fx.ResultTags(`group:"rootControllers"`)),
	fx.Annotate(NewCliController, fx.As(new(RootController)), fx.ResultTags(`group:"rootControllers"`)),

	NewAuthMiddleware,
)
