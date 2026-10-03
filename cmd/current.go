package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"
	"github.com/ville6000/toggl-cli/internal/output"
)

func newCurrentCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current",
		Short: "Get the current timer entry",
		Long:  "Get the current timer entry from Toggl.",
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

			projectsMap, err := client.ProjectNames(ctx, workspaceID, currentEntry.ProjectID)
			if err != nil {
				return fmt.Errorf("failed to get projects: %w", err)
			}

			return outputCurrentEntry(cmd.OutOrStdout(), currentEntry, projectsMap)
		},
	}

	return cmd
}

func outputCurrentEntry(out io.Writer, entry *api.TimeEntryItem, projectsMap map[int]string) error {
	if entry == nil || entry.ID == 0 {
		fmt.Fprintln(out, "No current timer entry.")
		return nil
	}

	duration := int(time.Since(entry.Start).Seconds())
	projectName := projectsMap[entry.ProjectID]

	rows := [][]any{
		{
			entry.ID,
			entry.Start.Format("02.01.2006 15:04"),
			output.FormatDuration(duration),
			entry.Description,
			projectName,
		},
	}

	headers := []any{"ID", "Started At", "Duration", "Description", "Project"}
	output.RenderTable(out, "Current timer entry", headers, rows, nil)
	return nil
}
