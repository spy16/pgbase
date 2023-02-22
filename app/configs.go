package app

import (
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/spy16/pgbase/log"
)

func (app *App) cmdShowConfigs() *cobra.Command {
	return &cobra.Command{
		Use:         "configs",
		Short:       "Show currently loaded configurations",
		Annotations: map[string]string{"load_configs": "yes"},
		Run: func(cmd *cobra.Command, args []string) {
			m := map[string]interface{}{}
			if err := mapstructure.Decode(app.CfgPtr, &m); err != nil {
				log.Fatal(cmd.Context(), "failed to unmarshal configs", err)
			}

			err := yaml.NewEncoder(os.Stdout).Encode(m)
			if err != nil {
				log.Fatal(cmd.Context(), "failed to display configs", err)
			}
		},
	}
}
