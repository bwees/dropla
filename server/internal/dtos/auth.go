package dtos

import "time"

type LoginDto struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type CreateUserDto struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
	// IsAdmin must not be `required`: the validator treats false as absent, which
	// would make it impossible to create a non-admin account.
	IsAdmin bool `json:"isAdmin"`
}

type SetPasswordDto struct {
	// CurrentPassword is required when changing your own password and ignored
	// when an admin resets someone else's.
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"`
}

type SetQuotaDto struct {
	// QuotaBytes nil inherits the server default; 0 means unlimited.
	QuotaBytes *int64 `json:"quotaBytes"`
}

// Response DTOs mark their always-present fields required so the generated
// client types are not needlessly optional. These are never validated as input,
// so the validator's treatment of false and 0 as absent does not apply.
type UserDto struct {
	ID         string `json:"id" validate:"required"`
	Username   string `json:"username" validate:"required"`
	IsAdmin    bool   `json:"isAdmin" validate:"required"`
	QuotaBytes *int64 `json:"quotaBytes,omitempty"`
}

type SetupStatusDto struct {
	SetupRequired bool `json:"setupRequired" validate:"required"`
}

type CreateAccessTokenDto struct {
	Name string `json:"name" validate:"required"`
}

// AccessTokenCreatedDto carries the only moment the secret is ever readable,
// together with a ready-to-run command so the token never has to be re-displayed.
type AccessTokenCreatedDto struct {
	ID      string `json:"id" validate:"required"`
	Name    string `json:"name" validate:"required"`
	Token   string `json:"token" validate:"required"`
	Command string `json:"command" validate:"required"`
}

type AccessTokenDto struct {
	ID         string     `json:"id" validate:"required"`
	Name       string     `json:"name" validate:"required"`
	CreatedAt  time.Time  `json:"createdAt" validate:"required"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}
