package cmd

import (
	"fmt"
	"github.com/ville6000/toggl-cli/internal/utils"
	"log"
	"time"

	"github.com/ville6000/toggl-cli/internal/api"

	"github.com/spf13/cobra"
)

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

		var projectId int
		if projectName != "" {
			projectId, err = client.GetProjectIdByName(workspaceId, projectName)
			if err != nil {
				log.Fatal("Failed to get project ID:", err)
			}
		}

		timeEntry := api.TimeEntry{
			CreatedWith: "toggl-cli",
			Description: description,
			Tags:        []string{},
			Billable:    false,
			WorkspaceID: workspaceId,
			Duration:    -1,
			Start:       time.Now().Format(time.RFC3339),
			Stop:        nil,
			ProjectID:   projectId,
		}

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
	startCmd.MarkFlagRequired("project")
}
