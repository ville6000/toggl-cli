package list

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
)

var ProjectsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List projects",
	Long:    "List all projects associated with the default workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, workspaceId, err := utils.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		client := api.NewAPIClient(token)

		return ProjectListOutput(client, workspaceId, func(s string) {
			fmt.Print(s)
		})
	},
}
