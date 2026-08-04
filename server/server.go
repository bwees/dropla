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
	"app/internal/services"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"go.uber.org/fx"
)

type Controllers struct {
	fx.In
	All  []controllers.Controller     `group:"controllers"`
	Root []controllers.RootController `group:"rootControllers"`
}

func NewServer(
	lc fx.Lifecycle,
	ctrls Controllers,
	authMiddleware *controllers.AuthMiddleware,
	sweeper *services.SweepService,
) *fuego.Server {
	server := fuego.NewServer(
		// DisableRequest/DisableResponse are load-bearing, not a preference.
		// fuego's logging middleware wraps the ResponseWriter without an Unwrap
		// method, which defeats http.ResponseController and io.ReaderFrom.
		// Re-enabling either silently kills resumable uploads at 30s and loses
		// sendfile on downloads, with only a NetworkControlError warning.
		fuego.WithLoggingMiddleware(fuego.LoggingConfig{
			DisableRequest:  true,
			DisableResponse: true,
		}),
		fuego.WithSerializer(fuego.SendJSON),
		fuego.WithEngineOptions(
			fuego.WithRequestContentType("application/json"),
			fuego.WithResponseContentType("application/json"),
		),
		fuego.WithAddr(":"+config.Port()),
	)

	// fuego hardcodes all four timeouts to 30s and exposes no option to change
	// them, which caps uploads at roughly 75MB and severs any download running
	// longer than 30s. *http.Server is embedded, so these are promoted fields.
	//
	// ReadHeaderTimeout is deliberately left at 30s: it bounds only how long
	// headers may take, so it costs transfers nothing while retaining the
	// Slowloris header-drip mitigation. IdleTimeout governs waiting for the next
	// keep-alive request and never interrupts a body in flight.
	server.ReadTimeout = 0
	server.WriteTimeout = 0

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

	fuego.Use(server, authMiddleware.Handler)

	api := fuego.Group(server, "/api")
	for _, controller := range ctrls.All {
		controller.Register(api)
		slog.Info(fmt.Sprintf("Registered controller: %T", controller))
	}

	// Binary endpoints register on the bare server: the public download paths live
	// under /s/ so browser navigations carry the unlock cookie, and tus needs a
	// method-agnostic mount.
	for _, controller := range ctrls.Root {
		controller.RegisterRoot(server)
		slog.Info(fmt.Sprintf("Registered root controller: %T", controller))
	}

	sweeper.Start(lc)

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
