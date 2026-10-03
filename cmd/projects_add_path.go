package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/config"
)

func newProjectsAddPathCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-path [project_name]",
		Short: "Save project path to be used with start command",
		Long:  "",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			token, workspaceID, err := config.TokenAndWorkspace(v)
			if err != nil {
				return fmt.Errorf("failed to get configuration: %w", err)
			}

			projectName := args[0]
			currentPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current path: %w", err)
			}

			client := newTogglClient(v, token)

			var projectID int
			if projectName != "" {
				projectID, err = client.ProjectIDByName(ctx, workspaceID, projectName)
				if err != nil {
					return fmt.Errorf("failed to get project ID: %w", err)
				}
			}

			v.Set(fmt.Sprintf("projects.%s.id", projectName), projectID)

			key := fmt.Sprintf("projects.%s.paths", projectName)
			existingPaths := v.GetStringSlice(key)

			if slices.Contains(existingPaths, currentPath) {
				fmt.Fprintln(cmd.OutOrStdout(), "Path already exists for this project.")
				return nil
			}
			existingPaths = append(existingPaths, currentPath)

			v.Set(key, existingPaths)

			if err := v.WriteConfig(); err != nil {
				return fmt.Errorf("error saving configuration: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Configuration saved successfully!")
			return nil
		},
	}

	return cmd
}
