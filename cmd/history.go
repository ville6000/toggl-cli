package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/ville6000/toggl-cli/internal/data"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
)

type HistoryEntry struct {
	Description string
	Duration    int
	Project     string
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Fetch the history of time entries",
	Long:  "Fetch the history of time entries from Toggl",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, workspaceId, err := utils.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		displayVerboseOutput, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			return fmt.Errorf("failed to get verbose flag: %w", err)
		}

		client := api.NewAPIClient(token)
		projectsLookup, err := client.GetProjectsLookupMap(workspaceId)
		if err != nil {
			return fmt.Errorf("failed to get projects: %w", err)
		}

		startTime, endTime, err := getDateParams(cmd)
		if err != nil {
			return err
		}

		timeEntries, err := client.GetHistory(&startTime, &endTime)
		if err != nil {
			return fmt.Errorf("failed to get history: %w", err)
		}

		groupedEntries := groupEntriesByDate(timeEntries)
		if len(groupedEntries) == 0 {
			return fmt.Errorf("no time entries found for the specified date range")
		}

		location, err := time.LoadLocation("Europe/Helsinki")
		if err != nil {
			return fmt.Errorf("failed to load location: %w", err)
		}

		sortedKeys := getSortedTimeEntryDates(groupedEntries)
		headers := []interface{}{"Started At", "Duration", "Description", "Project"}
		summaryHeaders := []interface{}{"Description", "Project", "Duration"}
		for _, key := range sortedKeys {
			fmt.Printf("# %s\n", key)
			fmt.Println()

			if displayVerboseOutput {
				if err := outputDateEntries(key, headers, groupedEntries, projectsLookup, location); err != nil {
					return err
				}
			}

			summaryEntries := sumEntriesByDescriptionAndProject(
				groupedEntries[key],
				projectsLookup,
			)

			if len(summaryEntries) > 0 {
				outputSummaryEntries(key, summaryHeaders, summaryEntries)
			}
		}

		return nil
	},
}

func outputSummaryEntries(key string, headers []interface{}, entries map[string]HistoryEntry) {
	totalDuration := 0
	var rows [][]interface{}
	for _, entry := range entries {
		formattedDuration := api.FormatDuration(float64(entry.Duration))
		rows = append(rows, []interface{}{
			entry.Description,
			entry.Project,
			formattedDuration,
		})

		totalDuration += entry.Duration
	}

	footer := table.Row{"", "Total", api.FormatDuration(float64(totalDuration))}
	title := fmt.Sprintf("Summary for: %s", key)

	utils.RenderTable(title, headers, rows, footer)
	fmt.Println()
}

func sumEntriesByDescriptionAndProject(
	entries []data.TimeEntryItem,
	projectsLookup map[int]string,
) map[string]HistoryEntry {
	summary := make(map[string]HistoryEntry)

	for _, entry := range entries {
		projectName := projectsLookup[entry.ProjectID]
		key := fmt.Sprintf("%s - %s", entry.Description, projectName)

		if existingEntry, exists := summary[key]; exists {
			existingEntry.Duration += entry.Duration
			summary[key] = existingEntry
		} else {
			summary[key] = HistoryEntry{
				Description: entry.Description,
				Duration:    entry.Duration,
				Project:     projectName,
			}
		}
	}

	return summary
}

func outputDateEntries(
	key string,
	headers []interface{},
	groupedEntries map[string][]data.TimeEntryItem,
	projectsLookup map[int]string,
	location *time.Location,
) error {
	parsedDate, err := time.Parse("2006-01-02", key)
	if err != nil {
		return fmt.Errorf("error parsing date: %w", err)
	}

	title := fmt.Sprintf("Entries for: %s", parsedDate.In(location).Format("02.01.2006"))

	entries := groupedEntries[key]
	var rows [][]interface{}
	for _, entry := range entries {
		formattedDuration := api.FormatDuration(float64(entry.Duration))
		projectName := projectsLookup[entry.ProjectID]
		startTimeInFinnish := entry.Start.In(location)

		rows = append(rows, []interface{}{
			startTimeInFinnish.Format("15:04"),
			formattedDuration,
			entry.Description,
			projectName,
		})
	}

	utils.RenderTable(title, headers, rows, nil)
	fmt.Println()
	return nil
}

func groupEntriesByDate(entries []data.TimeEntryItem) map[string][]data.TimeEntryItem {
	groupedEntries := make(map[string][]data.TimeEntryItem)

	for _, entry := range entries {
		date := entry.Start.Format("2006-01-02")
		groupedEntries[date] = append(groupedEntries[date], entry)
	}

	return groupedEntries
}

func getSortedTimeEntryDates(groupedEntries map[string][]data.TimeEntryItem) []string {
	sortedKeys := make([]string, 0, len(groupedEntries))
	for key := range groupedEntries {
		sortedKeys = append(sortedKeys, key)
	}

	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i] > sortedKeys[j]
	})

	return sortedKeys
}

func getDateParams(cmd *cobra.Command) (time.Time, time.Time, error) {
	week, err := cmd.Flags().GetBool("week")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get week flag: %w", err)
	}

	if week {
		start, end := getCurrentWeekTimeInterval()
		return start, end, nil
	}

	month, err := cmd.Flags().GetBool("month")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get month flag: %w", err)
	}

	if month {
		start, end := getCurrentMonthTimeInterval()
		return start, end, nil
	}

	start, err := cmd.Flags().GetString("start")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get start flag: %w", err)
	}

	startTime, err := getTimeWithDefault(start, time.Now())
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --start value %q: %w", start, err)
	}

	end, err := cmd.Flags().GetString("end")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get end flag: %w", err)
	}

	endTime, err := getTimeWithDefault(end, time.Now().AddDate(0, 0, 1))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --end value %q: %w", end, err)
	}

	return startTime, endTime, nil
}

func getCurrentWeekTimeInterval() (time.Time, time.Time) {
	start := time.Now()
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	end := start.AddDate(0, 0, 6)

	return start, end
}

func getCurrentMonthTimeInterval() (time.Time, time.Time) {
	start := time.Now()
	start = start.AddDate(0, 0, -(start.Day() - 1))
	end := start.AddDate(0, 1, 0)

	return start, end
}

func getTimeWithDefault(date string, fallback time.Time) (time.Time, error) {
	if date == "" {
		return fallback, nil
	}
	parsedTime, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing date: %w", err)
	}
	return parsedTime, nil
}

func init() {
	rootCmd.AddCommand(historyCmd)

	historyCmd.Flags().BoolP("week", "w", false, "History for the current week")
	historyCmd.Flags().BoolP("month", "m", false, "History for the current month")
	historyCmd.Flags().StringP("start", "s", "", "Start date for the history, format: YYYY-MM-DD")
	historyCmd.Flags().StringP("end", "e", "", "End date for the history, format: YYYY-MM-DD")
	historyCmd.Flags().BoolP("verbose", "v", false, "Display separate timer entries for each day")
}
