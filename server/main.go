package main

import (
	"app/internal/controllers"
	"app/internal/database"
	"app/internal/repositories"
	"app/internal/services"
	"app/internal/storage"

	"go.uber.org/fx"
)

func main() {
	// A non-serving mode, matching the -open-api flag the template already has.
	// This is the recovery path for a locked-out sole admin, so it must work
	// without the server running.
	if handleCLI() {
		return
	}

	app := fx.New(
		fx.NopLogger,
		controllers.Module,
		services.Module,
		repositories.Module,
		database.Module,
		storage.Module,

		fx.Invoke(NewServer),
	)

	app.Run()
}
