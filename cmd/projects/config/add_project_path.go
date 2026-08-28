package config

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
)

var AddProjectPathCmd = &cobra.Command{
	Use:   "add-path [project_name]",
	Short: "Save project path to be used with start command",
	Long:  "",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, workspaceId, err := utils.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		projectName := args[0]
		currentPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current path: %w", err)
		}

		client := api.NewAPIClient(token)

		var projectId int
		if projectName != "" {
			projectId, err = client.GetProjectIdByName(workspaceId, projectName)
			if err != nil {
				return fmt.Errorf("failed to get project ID: %w", err)
			}
		}

		viper.Set(fmt.Sprintf("projects.%s.id", projectName), projectId)

		key := fmt.Sprintf("projects.%s.paths", projectName)
		existingPaths := viper.GetStringSlice(key)

		for _, p := range existingPaths {
			if p == currentPath {
				fmt.Println("Path already exists for this project.")
				return nil
			}
		}
		existingPaths = append(existingPaths, currentPath)

		viper.Set(key, existingPaths)

		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("error saving configuration: %w", err)
		}

		fmt.Println("Configuration saved successfully!")
		return nil
	},
}
