package repositories

import "go.uber.org/fx"

var Module = fx.Provide(
	NewUsersRepository,
	NewSessionsRepository,
	NewAccessTokensRepository,
	NewSharesRepository,
)
