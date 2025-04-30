package cmd

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
	"log"
	"time"

	"github.com/spf13/cobra"
)

var continueCmd = &cobra.Command{
	Use:   "continue",
	Short: "Continue latest timer entry",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("toggl.token")
		if token == "" {
			log.Fatal("Missing toggl.token in config file")
		}

		workspaceId := viper.GetInt("toggl.workspace_id")
		if workspaceId == 0 {
			log.Fatal("Missing toggl.workspace_id in config file")
		}

		client := api.NewAPIClient(token)
		timeEntries, err := client.GetHistory(nil, nil)
		if err != nil {
			log.Fatal("Failed to retrieve latest time entries:", err)
		}

		if len(timeEntries) == 0 {
			fmt.Println("No time entries found.")
			return
		}

		latestEntry := timeEntries[0]
		timeEntry := api.TimeEntry{
			CreatedWith: "toggl-cli",
			Description: latestEntry.Description,
			Tags:        latestEntry.Tags,
			Billable:    latestEntry.Billable,
			WorkspaceID: workspaceId,
			Duration:    -1,
			Start:       time.Now().Format(time.RFC3339),
			ProjectID:   latestEntry.ProjectID,
		}

		_, err = client.CreateTimeEntry(workspaceId, timeEntry)
		if err != nil {
			log.Fatal("Failed to create time entry:", err)
		}

		fmt.Println("Continuing timer for:", latestEntry.Description)
	},
}

func init() {
	rootCmd.AddCommand(continueCmd)
}
