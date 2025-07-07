package projects

import (
	"github.com/ville6000/toggl-cli/cmd/projects/config"
	"github.com/ville6000/toggl-cli/cmd/projects/list"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects",
	Long:  "The 'projects' command allows you to manage your projects within Toggl.",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	Cmd.AddCommand(list.ProjectsListCmd)
	Cmd.AddCommand(config.AddProjectPathCmd)
}
