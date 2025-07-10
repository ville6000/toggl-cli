package cmd

import (
	"fmt"
	"github.com/ville6000/toggl-cli/internal/data"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the current timer entry",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, workspaceId := utils.GetTogglConfig()
		client := api.NewAPIClient(token)
		currentEntry, err := client.GetCurrentTimerEntry()
		if err != nil {
			return fmt.Errorf("failed to get current timer entry: %w", err)
		}

		if currentEntry == nil || currentEntry.ID == 0 {
			fmt.Println("No current timer entry.")
			return nil
		}

		stoppedEntry, err := client.StopTimeEntry(workspaceId, currentEntry.ID)
		if err != nil {
			return fmt.Errorf("failed to stop time entry: %w", err)
		}

		projectsMap, err := client.GetProjectsLookupMap(workspaceId)
		if err != nil {
			return fmt.Errorf("failed to get projects lookup map: %w", err)
		}

		return outputStoppedTimeEntry(stoppedEntry, projectsMap)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func outputStoppedTimeEntry(entry *data.TimeEntryItem, projectsMap map[int]string) error {
	headers := []interface{}{"#", "Started At", "Duration", "Description", "Project"}
	duration := float64(entry.Duration)
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

	utils.RenderTable("Stopped timer entry", headers, rows, nil)
	return nil
}
