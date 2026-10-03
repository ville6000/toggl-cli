package cmd

import (
	"fmt"
	"io"

	"github.com/ville6000/toggl-cli/internal/data"

	"github.com/spf13/cobra"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"
	"github.com/ville6000/toggl-cli/internal/output"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the current timer entry",
	Long:  "",
	RunE: func(cmd *cobra.Command, _ []string) error {
		token, workspaceID, err := config.TokenAndWorkspace()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		client := api.NewClientFromConfig(token)
		currentEntry, err := client.CurrentTimeEntry()
		if err != nil {
			return fmt.Errorf("failed to get current timer entry: %w", err)
		}

		if currentEntry == nil || currentEntry.ID == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No current timer entry.")
			return nil
		}

		stoppedEntry, err := client.StopTimeEntry(workspaceID, currentEntry.ID)
		if err != nil {
			return fmt.Errorf("failed to stop time entry: %w", err)
		}

		projectsMap, err := client.ProjectNames(workspaceID)
		if err != nil {
			return fmt.Errorf("failed to get projects lookup map: %w", err)
		}

		return outputStoppedTimeEntry(cmd.OutOrStdout(), stoppedEntry, projectsMap)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func outputStoppedTimeEntry(out io.Writer, entry *data.TimeEntryItem, projectsMap map[int]string) error {
	headers := []any{"#", "Started At", "Duration", "Description", "Project"}
	projectName := projectsMap[entry.ProjectID]
	rows := [][]any{
		{
			entry.ID,
			entry.Start.Format("02.01.2006 15:04"),
			output.FormatDuration(entry.Duration),
			entry.Description,
			projectName,
		},
	}

	output.RenderTable(out, "Stopped timer entry", headers, rows, nil)
	return nil
}
