package main

import (
	"app/internal/controllers"
	"app/internal/database"
	"app/internal/repositories"
	"app/internal/services"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.NopLogger,
		controllers.Module,
		services.Module,
		repositories.Module,
		database.Module,

		fx.Invoke(NewServer),
	)

	app.Run()
}
