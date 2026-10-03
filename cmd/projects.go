package cmd

import (
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects",
	Long:  "The 'projects' command allows you to manage your projects within Toggl.",
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}
