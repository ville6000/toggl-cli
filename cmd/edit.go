package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/data"
	"github.com/ville6000/toggl-cli/internal/utils"
)

// EditService is the subset of api.Client used by the edit command.
type EditService interface {
	GetHistory(from, to *time.Time) ([]data.TimeEntryItem, error)
	GetProjectIdByName(workspaceId int, projectName string) (int, error)
	UpdateTimeEntry(workspaceId int, entryId int, entry data.TimeEntry) (*data.TimeEntryItem, error)
	GetProjectsLookupMap(workspaceId int) (map[int]string, error)
}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a recent or running time entry",
	Long:  "Edit the description or project of a recent or currently running time entry.",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, _, err := utils.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		index, err := cmd.Flags().GetInt("index")
		if err != nil {
			return fmt.Errorf("failed to get index flag: %w", err)
		}

		description, err := cmd.Flags().GetString("description")
		if err != nil {
			return fmt.Errorf("failed to get description flag: %w", err)
		}

		project, err := cmd.Flags().GetString("project")
		if err != nil {
			return fmt.Errorf("failed to get project flag: %w", err)
		}

		if description == "" && project == "" {
			return fmt.Errorf("at least one of --description or --project must be provided")
		}

		client := api.NewAPIClient(token)

		return runEdit(client, index, description, project)
	},
}

func runEdit(client EditService, index int, newDescription, newProject string) error {
	entries, err := client.GetHistory(nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no time entries found")
	}

	if index < 0 || index >= len(entries) {
		return fmt.Errorf("index %d out of range (0-%d)", index, len(entries)-1)
	}

	entry := entries[index]

	// Use the workspace from the selected entry for all subsequent operations
	// so multi-workspace accounts target the correct workspace.
	wsID := entry.WorkspaceID

	description := entry.Description
	if newDescription != "" {
		description = newDescription
	}

	projectId := entry.ProjectID
	if newProject != "" {
		projectId, err = client.GetProjectIdByName(wsID, newProject)
		if err != nil {
			return fmt.Errorf("failed to find project '%s': %w", newProject, err)
		}
	}

	updated := data.TimeEntry{
		CreatedWith: "toggl-cli",
		Description: description,
		Tags:        entry.Tags,
		Billable:    entry.Billable,
		WorkspaceID: wsID,
		Duration:    entry.Duration,
		Start:       entry.Start.Format(time.RFC3339),
		ProjectID:   projectId,
	}

	// Preserve stop time for stopped entries to avoid converting them back to running.
	if entry.Duration >= 0 {
		stopTime := entry.Start.Add(time.Duration(entry.Duration) * time.Second).Format(time.RFC3339)
		updated.Stop = &stopTime
	}

	updatedEntry, err := client.UpdateTimeEntry(wsID, entry.ID, updated)
	if err != nil {
		return fmt.Errorf("failed to update time entry: %w", err)
	}

	projectsMap, err := client.GetProjectsLookupMap(wsID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: failed to get projects, showing entry without project name:", err)
		projectsMap = nil
	}

	if updatedEntry.Duration >= 0 {
		return outputStoppedTimeEntry(updatedEntry, projectsMap)
	}

	return outputCurrentEntry(updatedEntry, projectsMap)
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().IntP("index", "i", 0, "Index of the time entry to edit (0 = most recent)")
	editCmd.Flags().StringP("description", "d", "", "New description for the time entry")
	editCmd.Flags().StringP("project", "p", "", "New project for the time entry")
}
