// Package cmd implements the toggl-cli commands.
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "toggl-cli",
	Short: "Toggl CLI is a command line interface for Toggl",
	Long:  "",
}

// Execute runs the command line with ctx, which every command passes on to its
// API requests. Cobra has already printed the returned error.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"",
		"config file (default is $XDG_CONFIG_HOME/toggl-cli/config.yaml or $HOME/.toggl-cli.yaml)",
	)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		configPath, err := ConfigPath()
		cobra.CheckErr(err)
		viper.SetConfigFile(configPath)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(rootCmd.ErrOrStderr(), "Using config file:", viper.ConfigFileUsed())
	}
}
