package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newProjectsCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Manage projects",
		Long:  "The 'projects' command allows you to manage your projects within Toggl.",
	}

	cmd.AddCommand(
		newProjectsAddPathCmd(v),
		newProjectsListCmd(v),
	)

	return cmd
}
