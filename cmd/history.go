package cmd

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
	"log"
	"time"

	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Fetch the history of time entries",
	Long:  "Fetch the history of time entries from Toggl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("history called")
		token := viper.GetString("toggl.token")
		if token == "" {
			log.Fatal("Missing toggl.token in config file")
		}

		client := api.NewAPIClient(token)
		startTime, endTime := getDateParams(cmd)
		timeEntries, err := client.GetHistory(startTime, endTime)
		if err != nil {
			log.Println("Failed to get history:", err)
			return
		}

		for _, entry := range timeEntries {
			duration := time.Since(entry.Start).Seconds()
			formattedDuration := api.FormatDuration(duration)
			fmt.Printf("%d - %s - %s\n", entry.ID, entry.Description, formattedDuration)
		}
	},
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
	startTime := getTimeWithDefault(start)

	end, err := cmd.Flags().GetString("end")
	if err != nil {
		log.Fatal("Error retrieving end flag:", err)
	}
	endTime := getTimeWithDefault(end)

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
	start = start.AddDate(0, 0, -int(start.Day()-1))
	end := start.AddDate(0, 1, 0)

	return start, end
}

func getTimeWithDefault(date string) time.Time {
	if date == "" {
		return time.Now()
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
