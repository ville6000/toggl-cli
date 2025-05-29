package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ProjectConfig struct {
	Path string `mapstructure:"path"`
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new time entry",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		token, workspaceId := utils.GetTogglConfig()
		description := args[0]
		projectName, err := cmd.Flags().GetString("project")
		if err != nil {
			log.Fatal("Error retrieving project flag:", err)
		}

		client := api.NewAPIClient(token)
		projectId, err := findProjectIdForEntry(projectName, client, workspaceId)
		if err != nil {
			log.Fatal("Failed to find project ID:", err)
		}

		timeEntry := client.NewTimeEntry(description, workspaceId, projectId, false)
		_, err = client.CreateTimeEntry(workspaceId, timeEntry)
		if err != nil {
			log.Fatal("Failed to create time entry:", err)
		}

		fmt.Println("Timer started...")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringP("project", "p", "", "Project for the time entry")
}

func findProjectIdForEntry(projectName string, client *api.Client, workspaceID int) (int, error) {
	var projectId int
	var err error

	if projectName == "" {
		currentPath, err := os.Getwd()
		if err != nil {
			return 0, fmt.Errorf("failed to get current working directory: %w", err)
		}

		projectName, err = findProjectNameFromConfig(currentPath)
		if err != nil {
			return 0, fmt.Errorf("failed to find project name from config: %w", err)
		}
	}

	if projectName == "" {
		return 0, fmt.Errorf("no project name provided and no matching project found in config for current path")
	}

	projectId, err = client.GetProjectIdByName(workspaceID, projectName)
	if err != nil || projectId == 0 {
		return 0, fmt.Errorf("failed to get project ID for '%s': %w", projectName, err)
	}

	fmt.Printf("Using project '%s' with ID %d for time entry\n", projectName, projectId)
	return projectId, nil
}

func findProjectNameFromConfig(currentPath string) (string, error) {
	var projects map[string]ProjectConfig
	err := viper.UnmarshalKey("projects", &projects)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal projects from config: %w", err)
	}

	for name, p := range projects {
		if p.Path == currentPath {
			return name, nil
		}
	}

	return "", fmt.Errorf("no matching project found for current path '%s'", currentPath)
}
