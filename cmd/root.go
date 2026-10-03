// Package cmd implements the toggl-cli commands.
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/config"
)

// Execute runs the command line with ctx, which every command passes on to its
// API requests. Cobra has already printed the returned error.
func Execute(ctx context.Context) error {
	return NewRootCmd().ExecuteContext(ctx)
}

// NewRootCmd builds the toggl-cli command tree. Each call has its own
// configuration and flags, so a tree can be built and run independently of
// any other.
func NewRootCmd() *cobra.Command {
	return newRootCmd(viper.New())
}

// newRootCmd builds the command tree around v, which holds the configuration
// every command reads and writes. Tests pass their own v to preset values.
func newRootCmd(v *viper.Viper) *cobra.Command {
	var cfgFile string

	cmd := &cobra.Command{
		Use:     "toggl-cli",
		Short:   "Toggl CLI is a command line interface for Toggl",
		Long:    "",
		Version: buildVersion(),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return loadConfig(cmd, v, cfgFile)
		},
	}

	cmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"",
		"config file (default is $XDG_CONFIG_HOME/toggl-cli/config.yaml or $HOME/.toggl-cli.yaml)",
	)

	cmd.AddCommand(
		newConfigCmd(v),
		newContinueCmd(v),
		newCurrentCmd(v),
		newEditCmd(v),
		newHistoryCmd(v),
		newProjectsCmd(v),
		newStartCmd(v),
		newStopCmd(v),
		newWorkspacesCmd(v),
		newWwwCmd(),
	)

	return cmd
}

// loadConfig reads the config file into v: cfgFile when given, otherwise the
// default location. A missing file is not an error; commands report the
// settings they need.
func loadConfig(cmd *cobra.Command, v *viper.Viper, cfgFile string) error {
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		configPath, err := ConfigPath()
		if err != nil {
			return err
		}
		v.SetConfigFile(configPath)
	}

	config.UseEnv(v)

	if err := v.ReadInConfig(); err == nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "Using config file:", v.ConfigFileUsed())
	}

	return nil
}
