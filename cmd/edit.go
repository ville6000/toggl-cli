package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"
)

// EditService is the subset of api.Client used by the edit command.
type EditService interface {
	TimeEntries(ctx context.Context, from, to *time.Time) ([]api.TimeEntryItem, error)
	ProjectIDByName(ctx context.Context, workspaceID int, projectName string) (int, error)
	UpdateTimeEntry(ctx context.Context, workspaceID int, entryID int, entry api.TimeEntry) (*api.TimeEntryItem, error)
	ProjectNames(ctx context.Context, workspaceID int) (map[int]string, error)
}

func newEditCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a recent or running time entry",
		Long:  "Edit the description, project or start time of a recent or currently running time entry.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			token, _, err := config.TokenAndWorkspace(v)
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

			start, err := cmd.Flags().GetString("start")
			if err != nil {
				return fmt.Errorf("failed to get start flag: %w", err)
			}

			if description == "" && project == "" && start == "" {
				return errors.New("at least one of --description, --project or --start must be provided")
			}

			location, err := config.Timezone(v)
			if err != nil {
				return err
			}

			client := newTogglClient(v, token)

			return runEdit(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), client, index, description, project, start, location)
		},
	}

	cmd.Flags().IntP("index", "i", 0, "Index of the time entry to edit (0 = most recent)")
	cmd.Flags().StringP("description", "d", "", "New description for the time entry")
	cmd.Flags().StringP("project", "p", "", "New project for the time entry")
	cmd.Flags().StringP("start", "s", "", "New start time in your timezone: \"YYYY-MM-DD HH:MM\", HH:MM (keeps entry's date) or YYYY-MM-DD")

	return cmd
}

func runEdit(
	ctx context.Context,
	out, errOut io.Writer,
	client EditService,
	index int,
	newDescription, newProject, newStart string,
	location *time.Location,
) error {
	entries, err := client.TimeEntries(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	if len(entries) == 0 {
		return errors.New("no time entries found")
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

	projectID := entry.ProjectID
	if newProject != "" {
		projectID, err = client.ProjectIDByName(ctx, wsID, newProject)
		if err != nil {
			return fmt.Errorf("failed to find project '%s': %w", newProject, err)
		}
	}

	updated := api.TimeEntry{
		CreatedWith: "toggl-cli",
		Description: description,
		Tags:        entry.Tags,
		Billable:    entry.Billable,
		WorkspaceID: wsID,
		Duration:    entry.Duration,
		Start:       entry.Start.Format(time.RFC3339),
		ProjectID:   projectID,
	}

	if newStart != "" {
		newStartT, err := parseStartTime(newStart, location, entry.Start)
		if err != nil {
			return err
		}

		updated.Start = newStartT.Format(time.RFC3339)

		// For stopped entries, keep the recorded end time fixed and recompute
		// the duration so the entry grows or shrinks around the new start.
		if entry.Duration >= 0 {
			stopT := entry.Start.Add(time.Duration(entry.Duration) * time.Second)
			if !newStartT.Before(stopT) {
				return fmt.Errorf("start time must be before the end time (%s)", stopT.In(location).Format("2006-01-02 15:04"))
			}

			stopTime := stopT.Format(time.RFC3339)
			updated.Stop = &stopTime
			updated.Duration = int(stopT.Sub(newStartT).Seconds())
		}
	} else if entry.Duration >= 0 {
		// Preserve stop time for stopped entries to avoid converting them back to running.
		stopTime := entry.Start.Add(time.Duration(entry.Duration) * time.Second).Format(time.RFC3339)
		updated.Stop = &stopTime
	}

	updatedEntry, err := client.UpdateTimeEntry(ctx, wsID, entry.ID, updated)
	if err != nil {
		return fmt.Errorf("failed to update time entry: %w", err)
	}

	projectsMap, err := client.ProjectNames(ctx, wsID)
	if err != nil {
		fmt.Fprintln(errOut, "warning: failed to get projects, showing entry without project name:", err)
		projectsMap = nil
	}

	if updatedEntry.Duration >= 0 {
		return outputStoppedTimeEntry(out, updatedEntry, projectsMap)
	}

	return outputCurrentEntry(out, updatedEntry, projectsMap)
}

// parseStartTime parses the --start flag in the given location. It accepts
// "YYYY-MM-DD HH:MM", "YYYY-MM-DD" (midnight) or "HH:MM". The time-only form
// keeps the year, month and day of the entry's existing start (entryDate).
func parseStartTime(value string, location *time.Location, entryDate time.Time) (time.Time, error) {
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, value, location); err == nil {
			return t, nil
		}
	}

	if t, err := time.ParseInLocation("15:04", value, location); err == nil {
		day := entryDate.In(location)
		return time.Date(day.Year(), day.Month(), day.Day(), t.Hour(), t.Minute(), 0, 0, location), nil
	}

	return time.Time{}, fmt.Errorf("invalid --start %q: use \"YYYY-MM-DD HH:MM\", HH:MM or YYYY-MM-DD", value)
}
