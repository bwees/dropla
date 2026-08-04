package controllers

import (
	"app/internal/dtos"
	"app/internal/services"
	"app/internal/storage"

	"github.com/go-fuego/fuego"
)

type UsersController struct {
	svc     *services.AuthService
	storage *storage.Storage
}

func NewUsersController(svc *services.AuthService, store *storage.Storage) *UsersController {
	return &UsersController{svc: svc, storage: store}
}

func (c *UsersController) Register(api *fuego.Server) {
	route := fuego.Group(api, "/users")

	fuego.Get(route, "/", func(ctx fuego.ContextNoBody) ([]dtos.UserDto, error) {
		if _, err := requireAdmin(ctx.Request().Context()); err != nil {
			return nil, err
		}

		users, err := c.svc.ListUsers()
		if err != nil {
			return nil, err
		}

		out := make([]dtos.UserDto, len(users))
		for i := range users {
			out[i] = userDto(&users[i])
		}
		return out, nil
	}, fuego.OptionOperationID("getUsers"))

	fuego.Post(route, "/", func(ctx fuego.ContextWithBody[dtos.CreateUserDto]) (dtos.UserDto, error) {
		forceJSON(ctx.Response())

		if _, err := requireAdmin(ctx.Request().Context()); err != nil {
			return dtos.UserDto{}, err
		}
		body, err := ctx.Body()
		if err != nil {
			return dtos.UserDto{}, err
		}
		created, err := c.svc.CreateUser(body)
		if err != nil {
			return dtos.UserDto{}, err
		}
		return userDto(created), nil
	}, fuego.OptionOperationID("createUser"), fuego.OptionDefaultStatusCode(201))

	// Deleting an account cascades to its Shares, files and blobs. Links that
	// account handed out stop working, and none of it is recoverable.
	fuego.Delete(route, "/{id}", func(ctx fuego.ContextNoBody) (any, error) {
		stripTrailer(ctx.Response())

		current, err := requireAdmin(ctx.Request().Context())
		if err != nil {
			return nil, err
		}

		id := ctx.PathParam("id")
		if id == current.ID {
			return nil, fuego.BadRequestError{Title: "Bad Request", Detail: "cannot delete your own account"}
		}

		shareIDs, err := c.svc.DeleteUser(id)
		if err != nil {
			return nil, err
		}
		for _, shareID := range shareIDs {
			_ = c.storage.RemoveShare(shareID)
		}
		return nil, nil
	}, fuego.OptionOperationID("deleteUser"), fuego.OptionDefaultStatusCode(204))

	// Admins may reset anyone's password, including another admin's: they can
	// already delete any account, so withholding this would guard a lesser power.
	fuego.Post(route, "/{id}/password", func(ctx fuego.ContextWithBody[dtos.SetPasswordDto]) (any, error) {
		stripTrailer(ctx.Response())

		if _, err := requireAdmin(ctx.Request().Context()); err != nil {
			return nil, err
		}
		body, err := ctx.Body()
		if err != nil {
			return nil, err
		}
		return nil, c.svc.ResetPassword(ctx.PathParam("id"), body.NewPassword)
	}, fuego.OptionOperationID("resetUserPassword"), fuego.OptionDefaultStatusCode(204))

	fuego.Patch(route, "/{id}/quota", func(ctx fuego.ContextWithBody[dtos.SetQuotaDto]) (any, error) {
		stripTrailer(ctx.Response())

		if _, err := requireAdmin(ctx.Request().Context()); err != nil {
			return nil, err
		}
		body, err := ctx.Body()
		if err != nil {
			return nil, err
		}
		return nil, c.svc.SetQuota(ctx.PathParam("id"), body.QuotaBytes)
	}, fuego.OptionOperationID("setUserQuota"), fuego.OptionDefaultStatusCode(204))
}

type TokensController struct {
	svc *services.AuthService
}

func NewTokensController(svc *services.AuthService) *TokensController {
	return &TokensController{svc: svc}
}

func (c *TokensController) Register(api *fuego.Server) {
	route := fuego.Group(api, "/tokens")

	fuego.Get(route, "/", func(ctx fuego.ContextNoBody) ([]dtos.AccessTokenDto, error) {
		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return nil, err
		}
		list, err := c.svc.ListAccessTokens(user.ID)
		if err != nil {
			return nil, err
		}

		out := make([]dtos.AccessTokenDto, len(list))
		for i, t := range list {
			out[i] = dtos.AccessTokenDto{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt}
		}
		return out, nil
	}, fuego.OptionOperationID("getAccessTokens"))

	// The response carries the only readable copy of the secret, inlined into a
	// runnable command. Nothing re-displays it afterwards.
	fuego.Post(route, "/", func(ctx fuego.ContextWithBody[dtos.CreateAccessTokenDto]) (dtos.AccessTokenCreatedDto, error) {
		forceJSON(ctx.Response())

		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return dtos.AccessTokenCreatedDto{}, err
		}
		body, err := ctx.Body()
		if err != nil {
			return dtos.AccessTokenCreatedDto{}, err
		}
		created, err := c.svc.CreateAccessToken(user.ID, body.Name)
		if err != nil {
			return dtos.AccessTokenCreatedDto{}, err
		}
		return *created, nil
	}, fuego.OptionOperationID("createAccessToken"), fuego.OptionDefaultStatusCode(201))

	fuego.Delete(route, "/{id}", func(ctx fuego.ContextNoBody) (any, error) {
		stripTrailer(ctx.Response())

		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return nil, err
		}
		return nil, c.svc.DeleteAccessToken(user.ID, ctx.PathParam("id"))
	}, fuego.OptionOperationID("deleteAccessToken"), fuego.OptionDefaultStatusCode(204))
}
