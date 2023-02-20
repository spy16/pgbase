package config

import (
	"strings"

	"github.com/spf13/cobra"
)

// Option values can be provided to Load() for customisation.
type Option func(l *viperLoader) error

// WithEnv enables loading configurations from environment variables. An
// optional environment prefix can be provided.
func WithEnv(prefix ...string) Option {
	return func(l *viperLoader) error {
		l.useEnv = true
		if len(prefix) > 0 {
			l.envPrefix = strings.TrimSpace(prefix[0])
		}
		return nil
	}
}

// WithName sets the configuration file title to be used for discovery
// of config files.
func WithName(name string) Option {
	return func(l *viperLoader) error {
		l.confName = strings.TrimSpace(name)
		return nil
	}
}

// WithCobra enables reading config file overrides from a flag.
func WithCobra(cmd *cobra.Command, flagName string) Option {
	return func(l *viperLoader) error {
		cfgFile, _ := cmd.Flags().GetString(flagName)
		if cfgFile != "" {
			l.confFile = cfgFile
		}
		return nil
	}
}

// WithFile sets a config file to use explicitly.
func WithFile(filePath string) Option {
	return func(l *viperLoader) error {
		l.confFile = filePath
		return nil
	}
}
