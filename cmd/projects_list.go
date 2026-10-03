package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"
	"github.com/ville6000/toggl-cli/internal/output"
)

func newProjectsListCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List projects",
		Long:    "List all projects associated with the default workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			token, workspaceID, err := config.TokenAndWorkspace(v)
			if err != nil {
				return fmt.Errorf("failed to get configuration: %w", err)
			}

			client := newTogglClient(v, token)

			return projectListOutput(cmd.Context(), cmd.OutOrStdout(), client, workspaceID)
		},
	}

	return cmd
}

// ProjectsListService is the subset of api.Client used by the projects list
// command.
type ProjectsListService interface {
	Projects(ctx context.Context, workspaceID int) ([]api.Project, error)
}

func projectListOutput(ctx context.Context, out io.Writer, client ProjectsListService, workspaceID int) error {
	projects, err := client.Projects(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get projects: %w", err)
	}

	var rows [][]any
	for _, project := range projects {
		rows = append(rows, []any{
			project.ID,
			project.Name,
		})
	}

	headers := []any{"ID", "Project Name"}
	output.RenderTable(out, "Project list", headers, rows, nil)

	return nil
}
