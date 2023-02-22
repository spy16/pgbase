package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/spy16/pgbase/config"
	"github.com/spy16/pgbase/log"
)

type App struct {
	Name    string
	Short   string
	Version string
	CfgPtr  any
	Router  http.Handler
}

func (app *App) Run(ctx context.Context) int {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	root := &cobra.Command{
		Use:     fmt.Sprintf("%s <command> [flags] <args>", app.Name),
		Short:   app.Short,
		Version: app.Version,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	root.SetContext(ctx)

	var logLevel, logFormat string
	flags := root.PersistentFlags()
	flags.StringP("config", "c", "", "Config file path override")
	flags.StringVar(&logLevel, "log-level", "info", "Log level")
	flags.StringVar(&logFormat, "log-format", "text", "Log format (json/text)")

	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		log.Setup(logLevel, logFormat)

		if _, ok := cmd.Annotations["load_config"]; ok && app.CfgPtr != nil {
			opts := []config.Option{
				config.WithName(app.Name),
				config.WithCobra(cmd, "config"),
				config.WithEnv(),
			}
			if err := config.Load(app.CfgPtr, opts...); err != nil {
				log.Fatal(cmd.Context(), "failed to load configs", err)
			}
		}
	}

	root.AddCommand(
		app.cmdServe(),
		app.cmdShowConfigs(),
	)

	if err := root.Execute(); err != nil {
		return 1
	}
	return 0
}
