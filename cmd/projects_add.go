package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/config"
)

func newProjectsAddCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [project_name]",
		Short: "Create a project",
		Long:  "Create a new project in the default workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return errors.New("project name must not be empty")
			}

			token, workspaceID, err := config.TokenAndWorkspace(v)
			if err != nil {
				return fmt.Errorf("failed to get configuration: %w", err)
			}

			client := newTogglClient(v, token)

			project, err := client.CreateProject(cmd.Context(), workspaceID, name)
			if err != nil {
				return fmt.Errorf("failed to create project: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created project %s (ID %d)\n", project.Name, project.ID)
			return nil
		},
	}

	return cmd
}
