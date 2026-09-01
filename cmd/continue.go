package cmd

import (
	"fmt"

	"github.com/ville6000/toggl-cli/internal/data"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"

	"github.com/spf13/cobra"
)

var continueCmd = &cobra.Command{
	Use:   "continue",
	Short: "Continue latest timer entry",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, workspaceId, err := utils.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		client := api.NewAPIClientFromConfig(token)
		timeEntries, err := client.GetHistory(nil, nil)
		if err != nil {
			return fmt.Errorf("failed to retrieve latest time entries: %w", err)
		}

		if len(timeEntries) == 0 {
			return fmt.Errorf("no time entries found")
		}

		index, err := cmd.Flags().GetInt("index")
		if err != nil {
			return fmt.Errorf("failed to get index flag: %w", err)
		}

		timeEntryDescription, err := createTimeEntryFrom(index, timeEntries, client, workspaceId)
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
	NewTimeEntry(description string, workspaceID, projectID int, billable bool) data.TimeEntry
	CreateTimeEntry(workspaceId int, entry data.TimeEntry) (*data.TimeEntry, error)
}

// createTimeEntryFrom restarts the entry at index. The new entry is created in
// the workspace of the entry being continued — its project id only exists
// there — falling back to the configured workspace when the entry has none.
func createTimeEntryFrom(index int, timeEntries []data.TimeEntryItem, client ContinueService, workspaceId int) (string, error) {
	if index < 0 || index >= len(timeEntries) {
		return "", fmt.Errorf("index out of range")
	}

	e := timeEntries[index]
	if e.WorkspaceID != 0 {
		workspaceId = e.WorkspaceID
	}

	timeEntry := client.NewTimeEntry(e.Description, workspaceId, e.ProjectID, e.Billable)
	_, err := client.CreateTimeEntry(workspaceId, timeEntry)
	if err != nil {
		return "", err
	}

	return e.Description, nil
}
