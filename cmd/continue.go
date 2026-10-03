package cmd

import (
	"context"
	"fmt"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newContinueCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "continue",
		Short: "Continue latest timer entry",
		Long:  "",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			token, workspaceID, err := config.TokenAndWorkspace(v)
			if err != nil {
				return fmt.Errorf("failed to get configuration: %w", err)
			}

			sel, err := readEntrySelector(cmd)
			if err != nil {
				return err
			}

			client := newTogglClient(v, token)
			entry, err := selectEntry(ctx, client, sel)
			if err != nil {
				return err
			}

			timeEntryDescription, err := createTimeEntryFrom(ctx, entry, client, workspaceID)
			if err != nil {
				return fmt.Errorf("failed to create time entry: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Continuing timer for:", timeEntryDescription)

			return nil
		},
	}

	addEntrySelectorFlags(cmd, "continue")

	return cmd
}

// ContinueService is the subset of api.Client used by the continue command.
type ContinueService interface {
	CreateTimeEntry(ctx context.Context, workspaceID int, entry api.TimeEntry) (*api.TimeEntry, error)
}

// createTimeEntryFrom restarts e. The new entry is created in the workspace of
// the entry being continued — its project id only exists there — falling back
// to the configured workspace when the entry has none.
func createTimeEntryFrom(ctx context.Context, e api.TimeEntryItem, client ContinueService, workspaceID int) (string, error) {
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
