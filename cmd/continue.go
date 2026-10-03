package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"

	"github.com/spf13/cobra"
)

var continueCmd = &cobra.Command{
	Use:   "continue",
	Short: "Continue latest timer entry",
	Long:  "",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		token, workspaceID, err := config.TokenAndWorkspace()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		client := api.NewClientFromConfig(token)
		timeEntries, err := client.TimeEntries(ctx, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to retrieve latest time entries: %w", err)
		}

		if len(timeEntries) == 0 {
			return errors.New("no time entries found")
		}

		index, err := cmd.Flags().GetInt("index")
		if err != nil {
			return fmt.Errorf("failed to get index flag: %w", err)
		}

		timeEntryDescription, err := createTimeEntryFrom(ctx, index, timeEntries, client, workspaceID)
		if err != nil {
			return fmt.Errorf("failed to create time entry: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Continuing timer for:", timeEntryDescription)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(continueCmd)
	continueCmd.Flags().IntP("index", "i", 0, "Index of the time entry to continue")
}

// ContinueService is the subset of api.Client used by the continue command.
type ContinueService interface {
	CreateTimeEntry(ctx context.Context, workspaceID int, entry api.TimeEntry) (*api.TimeEntry, error)
}

// createTimeEntryFrom restarts the entry at index. The new entry is created in
// the workspace of the entry being continued — its project id only exists
// there — falling back to the configured workspace when the entry has none.
func createTimeEntryFrom(ctx context.Context, index int, timeEntries []api.TimeEntryItem, client ContinueService, workspaceID int) (string, error) {
	if index < 0 || index >= len(timeEntries) {
		return "", errors.New("index out of range")
	}

	e := timeEntries[index]
	if e.WorkspaceID != 0 {
		workspaceID = e.WorkspaceID
	}

	timeEntry := api.NewTimeEntry(e.Description, workspaceID, e.ProjectID, e.Billable)
	_, err := client.CreateTimeEntry(ctx, workspaceID, timeEntry)
	if err != nil {
		return "", err
	}

	return e.Description, nil
}
