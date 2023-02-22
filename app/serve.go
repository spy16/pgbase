package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/cobra"

	"github.com/spy16/pgbase/httpx"
	"github.com/spy16/pgbase/log"
)

func (app *App) cmdServe() *cobra.Command {
	var graceDur time.Duration
	var addr, staticDir string

	cmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start HTTP server",
		Aliases: []string{"server", "start"},
		Run: func(cmd *cobra.Command, args []string) {
			router := chi.NewRouter()
			router.Use(
				middleware.Recoverer,
				middleware.RealIP,
				middleware.RequestID,
				middleware.Compress(5),
			)

			if staticDir != "" {
				app.Static = http.FileServer(http.Dir(staticDir))
			}

			if app.Static != nil {
				router.Mount("/", app.Static)
			}

			if app.Routes != nil {
				router.Group(func(r chi.Router) {
					if err := app.Routes(r); err != nil {
						log.Fatal(cmd.Context(), "router setup failed", err)
					}
				})
			}

			log.Info(cmd.Context(), "starting http server", log.Fields{"addr": addr})
			if err := httpx.GracefulServe(cmd.Context(), addr, router, graceDur); err != nil {
				log.Fatal(cmd.Context(), "server exited with error", err)
			}
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&addr, "addr", ":8080", "HTTP bind addr for server")
	flags.StringVar(&staticDir, "static", "", "Static directory to serve (override embedded static handler)")
	flags.DurationVarP(&graceDur, "grace", "g", 5*time.Second, "Grace period for graceful shutdown")
	return cmd
}
