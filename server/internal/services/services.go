package services

import (
	"github.com/google/uuid"
	"go.uber.org/fx"
)

func uuidString() string { return uuid.NewString() }

var Module = fx.Provide(
	NewAuthService,
	NewSharesService,
	NewNotifyService,
	NewSweepService,
)
