package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Get the current timer entry",
	Long:  "Get the current timer entry from Toggl.",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, workspaceId, err := utils.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		client := api.NewAPIClient(token)
		currentEntry, err := client.GetCurrentTimerEntry()
		if err != nil {
			return fmt.Errorf("failed to get current timer entry: %w", err)
		}

		projectsMap, err := client.GetProjectsLookupMap(workspaceId)
		if err != nil {
			return fmt.Errorf("failed to get projects: %w", err)
		}

		return outputCurrentEntry(currentEntry, projectsMap)
	},
}

func outputCurrentEntry(entry *api.TimeEntryItem, projectsMap map[int]string) error {
	if entry == nil || entry.ID == 0 {
		fmt.Println("No current timer entry.")
		return nil
	}

	duration := time.Since(entry.Start).Seconds()
	projectName := projectsMap[entry.ProjectID]

	rows := [][]interface{}{
		{
			entry.ID,
			entry.Start.Format("02.01.2006 15:04"),
			api.FormatDuration(duration),
			entry.Description,
			projectName,
		},
	}

	headers := []interface{}{"#", "Started At", "Duration", "Description", "Project"}
	utils.RenderTable("Current timer entry", headers, rows, nil)
	return nil
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
