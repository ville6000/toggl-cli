package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"
	"github.com/ville6000/toggl-cli/internal/output"
)

// HistoryEntry is one row of the history summary: the total duration logged
// for a description and project.
type HistoryEntry struct {
	Description string
	Duration    int
	Project     string
}

func newHistoryCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Fetch the history of time entries",
		Long:  "Fetch the history of time entries from Toggl",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			token, workspaceID, err := config.TokenAndWorkspace(v)
			if err != nil {
				return fmt.Errorf("failed to get configuration: %w", err)
			}

			displayVerboseOutput, err := cmd.Flags().GetBool("verbose")
			if err != nil {
				return fmt.Errorf("failed to get verbose flag: %w", err)
			}

			jsonOutput, err := cmd.Flags().GetBool("json")
			if err != nil {
				return fmt.Errorf("failed to get json flag: %w", err)
			}

			client := newTogglClient(v, token)

			location, err := config.Timezone(v)
			if err != nil {
				return err
			}

			startTime, endTime, err := getDateParams(cmd, location)
			if err != nil {
				return err
			}

			timeEntries, err := client.TimeEntries(ctx, &startTime, &endTime)
			if err != nil {
				return fmt.Errorf("failed to get history: %w", err)
			}

			projectIDs := make([]int, len(timeEntries))
			for i, entry := range timeEntries {
				projectIDs[i] = entry.ProjectID
			}

			projectsLookup, err := client.ProjectNames(ctx, workspaceID, projectIDs...)
			if err != nil {
				return fmt.Errorf("failed to get projects: %w", err)
			}

			out := cmd.OutOrStdout()
			now := time.Now()

			if jsonOutput {
				return writeHistoryJSON(out, timeEntries, projectsLookup, location, now)
			}

			groupedEntries := groupEntriesByDate(timeEntries, location)
			if len(groupedEntries) == 0 {
				return errors.New("no time entries found for the specified date range")
			}

			sortedKeys := getSortedTimeEntryDates(groupedEntries)
			headers := []any{"ID", "Started At", "Duration", "Description", "Project"}
			summaryHeaders := []any{"Description", "Project", "Duration"}
			for _, key := range sortedKeys {
				fmt.Fprintf(out, "# %s\n", key)
				fmt.Fprintln(out)

				if displayVerboseOutput {
					if err := outputDateEntries(out, key, headers, groupedEntries, projectsLookup, location, now); err != nil {
						return err
					}
				}

				summaryEntries := sumEntriesByDescriptionAndProject(
					groupedEntries[key],
					projectsLookup,
					now,
				)

				if len(summaryEntries) > 0 {
					outputSummaryEntries(out, key, summaryHeaders, summaryEntries)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolP("week", "w", false, "History for the current week")
	cmd.Flags().BoolP("month", "m", false, "History for the current month")
	cmd.Flags().StringP("start", "s", "", "Start date for the history, format: YYYY-MM-DD")
	cmd.Flags().StringP("end", "e", "", "End date for the history (inclusive), format: YYYY-MM-DD")
	cmd.Flags().StringP("day", "d", "", "History for a single day, format: YYYY-MM-DD")
	cmd.Flags().BoolP("verbose", "v", false, "Display separate timer entries for each day")
	cmd.MarkFlagsMutuallyExclusive("day", "week", "month", "start")
	cmd.MarkFlagsMutuallyExclusive("day", "end")
	cmd.Flags().Bool("json", false, "Print the entries as JSON, e.g. to pipe into another tool")

	return cmd
}

// historyJSONEntry is one time entry in the `history --json` output. Other
// tools read this format, so changes to it must stay backwards compatible.
type historyJSONEntry struct {
	ID int `json:"id"`
	// Start is RFC 3339 in the configured timezone, so its date is the local
	// day the entry belongs to.
	Start string `json:"start"`
	// Duration is in seconds; for a running entry, the time elapsed so far.
	Duration    int      `json:"duration"`
	Running     bool     `json:"running"`
	Description string   `json:"description"`
	Project     string   `json:"project"`
	Tags        []string `json:"tags"`
}

// writeHistoryJSON writes entries to out as a JSON array, oldest first. An
// empty range is an empty array rather than an error, so a pipeline sees
// valid input.
func writeHistoryJSON(
	out io.Writer,
	entries []api.TimeEntryItem,
	projectsLookup map[int]string,
	location *time.Location,
	now time.Time,
) error {
	sorted := slices.Clone(entries)
	slices.SortStableFunc(sorted, func(a, b api.TimeEntryItem) int {
		return a.Start.Compare(b.Start)
	})

	result := make([]historyJSONEntry, 0, len(sorted))
	for _, entry := range sorted {
		tags := entry.Tags
		if tags == nil {
			tags = []string{}
		}

		result = append(result, historyJSONEntry{
			ID:          entry.ID,
			Start:       entry.Start.In(location).Format(time.RFC3339),
			Duration:    entryDuration(entry, now),
			Running:     entry.Duration < 0,
			Description: entry.Description,
			Project:     projectsLookup[entry.ProjectID],
			Tags:        tags,
		})
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")

	return encoder.Encode(result)
}

func outputSummaryEntries(out io.Writer, key string, headers []any, entries map[string]HistoryEntry) {
	totalDuration := 0
	var rows [][]any
	for _, entry := range entries {
		formattedDuration := output.FormatDuration(entry.Duration)
		rows = append(rows, []any{
			entry.Description,
			entry.Project,
			formattedDuration,
		})

		totalDuration += entry.Duration
	}

	footer := table.Row{"", "Total", output.FormatDuration(totalDuration)}
	title := "Summary for: " + key

	output.RenderTable(out, title, headers, rows, footer)
	fmt.Fprintln(out)
}

func sumEntriesByDescriptionAndProject(
	entries []api.TimeEntryItem,
	projectsLookup map[int]string,
	now time.Time,
) map[string]HistoryEntry {
	summary := make(map[string]HistoryEntry)

	for _, entry := range entries {
		projectName := projectsLookup[entry.ProjectID]
		key := fmt.Sprintf("%s - %s", entry.Description, projectName)
		duration := entryDuration(entry, now)

		if existingEntry, exists := summary[key]; exists {
			existingEntry.Duration += duration
			summary[key] = existingEntry
		} else {
			summary[key] = HistoryEntry{
				Description: entry.Description,
				Duration:    duration,
				Project:     projectName,
			}
		}
	}

	return summary
}

// entryDuration returns how long an entry has run, in seconds. Toggl encodes a
// running entry as a negative duration (the negated start timestamp), so those
// are reported as the time elapsed so far instead of the raw sentinel, which
// would otherwise wreck the daily totals.
func entryDuration(entry api.TimeEntryItem, now time.Time) int {
	if entry.Duration < 0 {
		elapsed := int(now.Sub(entry.Start).Seconds())
		if elapsed < 0 {
			return 0
		}
		return elapsed
	}

	return entry.Duration
}

func outputDateEntries(
	out io.Writer,
	key string,
	headers []any,
	groupedEntries map[string][]api.TimeEntryItem,
	projectsLookup map[int]string,
	location *time.Location,
	now time.Time,
) error {
	parsedDate, err := time.ParseInLocation("2006-01-02", key, location)
	if err != nil {
		return fmt.Errorf("error parsing date: %w", err)
	}

	title := "Entries for: " + parsedDate.Format("02.01.2006")

	entries := groupedEntries[key]
	var rows [][]any
	for _, entry := range entries {
		formattedDuration := output.FormatDuration(entryDuration(entry, now))
		projectName := projectsLookup[entry.ProjectID]
		localStart := entry.Start.In(location)

		rows = append(rows, []any{
			entry.ID,
			localStart.Format("15:04"),
			formattedDuration,
			entry.Description,
			projectName,
		})
	}

	output.RenderTable(out, title, headers, rows, nil)
	fmt.Fprintln(out)
	return nil
}

// groupEntriesByDate buckets entries by their calendar date in the configured
// timezone. The API hands back UTC timestamps, so grouping on those directly
// would file an entry started at 01:00 in a UTC+2 zone under the previous day.
func groupEntriesByDate(entries []api.TimeEntryItem, location *time.Location) map[string][]api.TimeEntryItem {
	groupedEntries := make(map[string][]api.TimeEntryItem)

	for _, entry := range entries {
		date := entry.Start.In(location).Format("2006-01-02")
		groupedEntries[date] = append(groupedEntries[date], entry)
	}

	return groupedEntries
}

func getSortedTimeEntryDates(groupedEntries map[string][]api.TimeEntryItem) []string {
	sortedKeys := make([]string, 0, len(groupedEntries))
	for key := range groupedEntries {
		sortedKeys = append(sortedKeys, key)
	}

	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i] > sortedKeys[j]
	})

	return sortedKeys
}

// getDateParams resolves the date flags into a half-open [start, end) range of
// instants in location: start is midnight of the first day,
// end is midnight of the day *after* the last one, so --end is inclusive.
// With no flags the range is today.
func getDateParams(cmd *cobra.Command, location *time.Location) (time.Time, time.Time, error) {
	today := startOfDay(time.Now(), location)

	day, err := cmd.Flags().GetString("day")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get day flag: %w", err)
	}

	if day != "" {
		dayTime, err := parseDate(day, location, today)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --day value %q: %w", day, err)
		}
		return dayTime, dayTime.AddDate(0, 0, 1), nil
	}

	week, err := cmd.Flags().GetBool("week")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get week flag: %w", err)
	}

	if week {
		start := today
		for start.Weekday() != time.Monday {
			start = start.AddDate(0, 0, -1)
		}
		return start, start.AddDate(0, 0, 7), nil
	}

	month, err := cmd.Flags().GetBool("month")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get month flag: %w", err)
	}

	if month {
		start := today.AddDate(0, 0, -(today.Day() - 1))
		return start, start.AddDate(0, 1, 0), nil
	}

	start, err := cmd.Flags().GetString("start")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get start flag: %w", err)
	}

	startTime, err := parseDate(start, location, today)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --start value %q: %w", start, err)
	}

	end, err := cmd.Flags().GetString("end")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get end flag: %w", err)
	}

	endTime, err := parseDate(end, location, today)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --end value %q: %w", end, err)
	}

	// --end names the last day to include, so extend it to the next midnight.
	endTime = endTime.AddDate(0, 0, 1)

	if endTime.Before(startTime) {
		return time.Time{}, time.Time{}, fmt.Errorf("--end %q is before --start %q", end, start)
	}

	return startTime, endTime, nil
}

func startOfDay(t time.Time, location *time.Location) time.Time {
	t = t.In(location)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, location)
}

func parseDate(date string, location *time.Location, fallback time.Time) (time.Time, error) {
	if date == "" {
		return fallback, nil
	}
	parsedTime, err := time.ParseInLocation("2006-01-02", date, location)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing date: %w", err)
	}
	return parsedTime, nil
}
