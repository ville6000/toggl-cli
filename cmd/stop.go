package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"
	"github.com/ville6000/toggl-cli/internal/output"
)

func newStopCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the current timer entry",
		Long:  "",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			token, workspaceID, err := config.TokenAndWorkspace(v)
			if err != nil {
				return fmt.Errorf("failed to get configuration: %w", err)
			}
			client := newTogglClient(v, token)
			currentEntry, err := client.CurrentTimeEntry(ctx)
			if err != nil {
				return fmt.Errorf("failed to get current timer entry: %w", err)
			}

			if currentEntry == nil || currentEntry.ID == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No current timer entry.")
				return nil
			}

			stoppedEntry, err := client.StopTimeEntry(ctx, workspaceID, currentEntry.ID)
			if err != nil {
				return fmt.Errorf("failed to stop time entry: %w", err)
			}

			projectsMap, err := client.ProjectNames(ctx, workspaceID)
			if err != nil {
				return fmt.Errorf("failed to get projects lookup map: %w", err)
			}

			return outputStoppedTimeEntry(cmd.OutOrStdout(), stoppedEntry, projectsMap)
		},
	}

	return cmd
}

func outputStoppedTimeEntry(out io.Writer, entry *api.TimeEntryItem, projectsMap map[int]string) error {
	headers := []any{"ID", "Started At", "Duration", "Description", "Project"}
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
