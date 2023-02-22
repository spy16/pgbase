package app

import (
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/spy16/pgbase/httpx"
	"github.com/spy16/pgbase/log"
)

func (app *App) cmdServe() *cobra.Command {
	var graceDur time.Duration
	var addr string

	cmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start HTTP server",
		Aliases: []string{"server", "start"},
		Run: func(cmd *cobra.Command, args []string) {
			if app.Router == nil {
				log.Info(cmd.Context(), "no router set, using default")
				app.Router = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					httpx.WriteJSON(w, r, http.StatusOK, nil)
				})
			}

			log.Info(cmd.Context(), "starting http server", log.Fields{"addr": addr})
			if err := httpx.GracefulServe(cmd.Context(), addr, app.Router, graceDur); err != nil {
				log.Fatal(cmd.Context(), "server exited with error", err)
			}
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&addr, "addr", ":8080", "HTTP bind addr for server")
	flags.DurationVarP(&graceDur, "grace", "g", 5*time.Second, "Grace period for graceful shutdown")
	return cmd
}
