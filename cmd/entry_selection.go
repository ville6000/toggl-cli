package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/ville6000/toggl-cli/internal/api"
)

// EntryLookup is the subset of api.Client used to pick the entry a command
// acts on.
type EntryLookup interface {
	TimeEntries(ctx context.Context, from, to *time.Time) ([]api.TimeEntryItem, error)
	TimeEntry(ctx context.Context, id int) (*api.TimeEntryItem, error)
}

// entrySelector is how the user picked an entry: by Toggl entry ID (the ID
// column of current, stop and history --verbose), or by position in the
// recent entries, 0 being the most recent.
type entrySelector struct {
	id    int
	index int
}

// addEntrySelectorFlags adds the --index and --id flags; verb describes what
// the command does with the entry ("edit", "continue").
func addEntrySelectorFlags(cmd *cobra.Command, verb string) {
	cmd.Flags().IntP("index", "i", 0, fmt.Sprintf("Position of the time entry to %s among your recent entries (0 = most recent)", verb))
	cmd.Flags().Int("id", 0, fmt.Sprintf("ID of the time entry to %s, as shown in the ID column of current, stop and history --verbose", verb))
	cmd.MarkFlagsMutuallyExclusive("index", "id")
}

// readEntrySelector reads the flags added by addEntrySelectorFlags.
func readEntrySelector(cmd *cobra.Command) (entrySelector, error) {
	index, err := cmd.Flags().GetInt("index")
	if err != nil {
		return entrySelector{}, fmt.Errorf("failed to get index flag: %w", err)
	}

	id, err := cmd.Flags().GetInt("id")
	if err != nil {
		return entrySelector{}, fmt.Errorf("failed to get id flag: %w", err)
	}

	return entrySelector{id: id, index: index}, nil
}

// selectEntry returns the entry sel refers to. An ID is looked up directly, so
// it also finds entries older than the recent list an index counts through.
func selectEntry(ctx context.Context, client EntryLookup, sel entrySelector) (api.TimeEntryItem, error) {
	if sel.id != 0 {
		entry, err := client.TimeEntry(ctx, sel.id)
		if err != nil {
			return api.TimeEntryItem{}, fmt.Errorf("failed to get time entry %d: %w", sel.id, err)
		}
		return *entry, nil
	}

	entries, err := client.TimeEntries(ctx, nil, nil)
	if err != nil {
		return api.TimeEntryItem{}, fmt.Errorf("failed to get history: %w", err)
	}

	if len(entries) == 0 {
		return api.TimeEntryItem{}, errors.New("no time entries found")
	}

	if sel.index < 0 || sel.index >= len(entries) {
		return api.TimeEntryItem{}, fmt.Errorf("index %d out of range (0-%d)", sel.index, len(entries)-1)
	}

	return entries[sel.index], nil
}
