package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"

	"app/internal/config"
	"app/internal/controllers"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"go.uber.org/fx"
)

type Controllers struct {
	fx.In
	All []controllers.Controller `group:"controllers"`
}

func NewServer(lc fx.Lifecycle, controllers Controllers) *fuego.Server {
	addr := ":" + config.Port()

	server := fuego.NewServer(
		fuego.WithLoggingMiddleware(fuego.LoggingConfig{
			DisableRequest:  true,
			DisableResponse: true,
		}),
		fuego.WithSerializer(fuego.SendJSON),
		fuego.WithEngineOptions(
			fuego.WithRequestContentType("application/json"),
			fuego.WithResponseContentType("application/json"),
		),
		fuego.WithAddr(addr),
	)

	server.OpenAPI.Config.JSONFilePath = "openapi.json"
	server.OpenAPI.Config.SpecURL = "/openapi.json"
	server.OpenAPI.Config.DisableSwaggerUI = true
	server.OpenAPI.Config.PrettyFormatJSON = true

	server.OpenAPI.SetGeneratorSchemaCustomizer(func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		if strings.Contains(tag.Get("json"), "omitempty") {
			schema.Nullable = false
		}
		return nil
	})

	api := fuego.Group(server, "/api")
	for _, controller := range controllers.All {
		controller.Register(api)
		slog.Info(fmt.Sprintf("Registered controller: %T", controller))
	}

	registerStaticWeb(server)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			openapi := flag.Bool("open-api", false, "Generate the OpenAPI spec and exit")
			flag.Parse()

			if *openapi {
				server.OutputOpenAPISpec()
				slog.Info("OpenAPI spec generated successfully.")
				os.Exit(0)
			}

			go func() {
				if err := server.Run(); err != nil {
					slog.Error("Failed to start server", "error", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})

	return server
}
