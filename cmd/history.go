package cmd

import (
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
	"log"
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
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
			log.Println("Failed to get projects:", err)

			return
		}

		projectsLookup := toProjectsLookup(projects)

		startTime, endTime := getDateParams(cmd)
		timeEntries, err := client.GetHistory(&startTime, &endTime)
		if err != nil {
			log.Println("Failed to get history:", err)

			return
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"#", "Started At", "Duration", "Description", "Project"})

		for _, entry := range timeEntries {
			formattedDuration := api.FormatDuration(float64(entry.Duration))
			projectName := projectsLookup[entry.ProjectID]

			t.AppendRow([]interface{}{
				entry.ID,
				entry.Start.Format("02.01.2006 15:04"),
				formattedDuration,
				entry.Description,
				projectName,
			})
		}

		t.Render()
	},
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
