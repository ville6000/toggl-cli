package cmd

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"

	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Fetch the history of time entries",
	Long:  "Fetch the history of time entries from Toggl",
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("toggl.token")
		if token == "" {
			log.Fatal("Missing toggl.token in config file")
		}

		workspaceId := viper.GetInt("toggl.workspace_id")
		if workspaceId == 0 {
			log.Fatal("Missing toggl.workspace_id in config file")
		}

		client := api.NewAPIClient(token)
		projects, err := client.GetProjects(workspaceId)
		if err != nil {
			log.Fatal("Failed to get projects:", err)
		}

		projectsLookup := toProjectsLookup(projects)
		startTime, endTime := getDateParams(cmd)
		timeEntries, err := client.GetHistory(&startTime, &endTime)
		if err != nil {
			log.Fatal("Failed to get history:", err)
		}

		groupedEntries := groupEntriesByDate(timeEntries)
		if len(groupedEntries) == 0 {
			log.Fatal("No time entries found for the specified date range.")
		}

		location, err := time.LoadLocation("Europe/Helsinki")
		if err != nil {
			log.Fatal("Failed to load location:", err)
		}

		sortedKeys := getSortedTimeEntryDates(groupedEntries)
		headers := []interface{}{"Started At", "Duration", "Description", "Project"}
		for _, key := range sortedKeys {
			outputDateEntries(key, headers, groupedEntries, projectsLookup, location)
		}
	},
}

func outputDateEntries(
	key string,
	headers []interface{},
	groupedEntries map[string][]api.TimeEntryItem,
	projectsLookup map[int]string,
	location *time.Location,
) {
	parsedDate, err := time.Parse("2006-01-02", key)
	if err != nil {
		log.Fatal("Error parsing date:", err)
	}

	fmt.Printf("# %s\n", parsedDate.In(location).Format("02.01.2006"))

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

	utils.RenderTable(headers, rows)
	fmt.Println()
}

func groupEntriesByDate(entries []api.TimeEntryItem) map[string][]api.TimeEntryItem {
	groupedEntries := make(map[string][]api.TimeEntryItem)

	for _, entry := range entries {
		date := entry.Start.Format("2006-01-02")
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

func toProjectsLookup(projects []api.Project) map[int]string {
	lookup := make(map[int]string)
	for _, project := range projects {
		lookup[project.ID] = project.Name
	}

	return lookup
}

func getDateParams(cmd *cobra.Command) (time.Time, time.Time) {
	week, err := cmd.Flags().GetBool("week")
	if err != nil {
		log.Fatal("Error retrieving week flag:", err)
	}

	if week {
		return getCurrentWeekTimeInterval()
	}

	month, err := cmd.Flags().GetBool("month")
	if err != nil {
		log.Fatal("Error retrieving month flag:", err)
	}

	if month {
		return getCurrentMonthTimeInterval()
	}

	start, err := cmd.Flags().GetString("start")
	if err != nil {
		log.Fatal("Error retrieving start flag:", err)
	}

	startTime := getTimeWithDefault(start, time.Now().AddDate(0, 0, -7))

	end, err := cmd.Flags().GetString("end")
	if err != nil {
		log.Fatal("Error retrieving end flag:", err)
	}

	endTime := getTimeWithDefault(end, time.Now())

	return startTime, endTime
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

func getTimeWithDefault(date string, fallback time.Time) time.Time {
	if date == "" {
		return fallback
	}
	parsedTime, err := time.Parse("2006-01-02", date)
	if err != nil {
		log.Fatal("Error parsing date:", err)
	}
	return parsedTime
}

func init() {
	rootCmd.AddCommand(historyCmd)

	historyCmd.Flags().BoolP("week", "w", false, "History for the current week")
	historyCmd.Flags().BoolP("month", "m", false, "History for the current month")
	historyCmd.Flags().StringP("start", "s", "", "Start date for the history, format: YYYY-MM-DD")
	historyCmd.Flags().StringP("end", "e", "", "End date for the history, format: YYYY-MM-DD")
}
