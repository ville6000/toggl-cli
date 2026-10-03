package list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"
)

var ProjectsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List projects",
	Long:    "List all projects associated with the default workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, workspaceID, err := config.TokenAndWorkspace()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		client := api.NewClientFromConfig(token)

		return ProjectListOutput(cmd.OutOrStdout(), client, workspaceID)
	},
}
