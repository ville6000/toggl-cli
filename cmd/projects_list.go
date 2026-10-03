package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"
	"github.com/ville6000/toggl-cli/internal/output"
)

var projectsListCmd = &cobra.Command{
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

		return projectListOutput(cmd.OutOrStdout(), client, workspaceID)
	},
}

func init() {
	projectsCmd.AddCommand(projectsListCmd)
}

func projectListOutput(out io.Writer, client api.ProjectService, workspaceID int) error {
	projects, err := client.GetProjects(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get projects: %w", err)
	}

	var rows [][]interface{}
	for _, project := range projects {
		rows = append(rows, []interface{}{
			project.ID,
			project.Name,
		})
	}

	headers := []interface{}{"ID", "Project Name"}
	output.RenderTable(out, "Project list", headers, rows, nil)

	return nil
}
