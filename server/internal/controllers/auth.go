package controllers

import (
	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/services"

	"github.com/go-fuego/fuego"
)

type AuthController struct {
	svc *services.AuthService
}

func NewAuthController(svc *services.AuthService) *AuthController {
	return &AuthController{svc: svc}
}

func userDto(u *models.User) dtos.UserDto {
	return dtos.UserDto{ID: u.ID, Username: u.Username, IsAdmin: u.IsAdmin, QuotaBytes: u.QuotaBytes}
}

func (c *AuthController) Register(api *fuego.Server) {
	route := fuego.Group(api, "/auth")

	fuego.Get(route, "/setup", func(ctx fuego.ContextNoBody) (dtos.SetupStatusDto, error) {
		required, err := c.svc.SetupRequired()
		return dtos.SetupStatusDto{SetupRequired: required}, err
	}, fuego.OptionOperationID("getSetupStatus"))

	fuego.Post(route, "/setup", func(ctx fuego.ContextWithBody[dtos.CreateUserDto]) (dtos.UserDto, error) {
		body, err := ctx.Body()
		if err != nil {
			return dtos.UserDto{}, err
		}

		user, err := c.svc.Setup(body)
		if err != nil {
			return dtos.UserDto{}, err
		}

		_, token, err := c.svc.Login(dtos.LoginDto{Username: body.Username, Password: body.Password})
		if err != nil {
			return dtos.UserDto{}, err
		}

		ctx.SetCookie(sessionCookie(token, isSecure(ctx.Request()), int(services.SessionDuration.Seconds())))
		return userDto(user), nil
	}, fuego.OptionOperationID("setup"))

	fuego.Post(route, "/login", func(ctx fuego.ContextWithBody[dtos.LoginDto]) (dtos.UserDto, error) {
		body, err := ctx.Body()
		if err != nil {
			return dtos.UserDto{}, err
		}

		user, token, err := c.svc.Login(body)
		if err != nil {
			return dtos.UserDto{}, err
		}

		ctx.SetCookie(sessionCookie(token, isSecure(ctx.Request()), int(services.SessionDuration.Seconds())))
		return userDto(user), nil
	}, fuego.OptionOperationID("login"))

	fuego.Post(route, "/logout", func(ctx fuego.ContextNoBody) (any, error) {
		stripTrailer(ctx.Response())

		if cookie, err := ctx.Cookie(sessionCookieName); err == nil {
			if err := c.svc.Logout(cookie.Value); err != nil {
				return nil, err
			}
		}
		ctx.SetCookie(sessionCookie("", isSecure(ctx.Request()), -1))
		return nil, nil
	}, fuego.OptionOperationID("logout"), fuego.OptionDefaultStatusCode(204))

	fuego.Get(route, "/me", func(ctx fuego.ContextNoBody) (dtos.UserDto, error) {
		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return dtos.UserDto{}, err
		}
		return userDto(user), nil
	}, fuego.OptionOperationID("getCurrentUser"))

	// Changing your own password drops every session you have, including this
	// one, so the client must expect to log in again.
	fuego.Post(route, "/password", func(ctx fuego.ContextWithBody[dtos.SetPasswordDto]) (any, error) {
		stripTrailer(ctx.Response())

		user, err := requireUser(ctx.Request().Context())
		if err != nil {
			return nil, err
		}
		body, err := ctx.Body()
		if err != nil {
			return nil, err
		}
		if err := c.svc.ChangeOwnPassword(user.ID, body); err != nil {
			return nil, err
		}
		ctx.SetCookie(sessionCookie("", isSecure(ctx.Request()), -1))
		return nil, nil
	}, fuego.OptionOperationID("changePassword"), fuego.OptionDefaultStatusCode(204))
}
