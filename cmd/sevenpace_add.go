package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/data"
	"github.com/ville6000/toggl-cli/internal/utils"
)

var sevenpaceAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Post a single worklog to 7pace",
	Long: "Post a single worklog to 7pace Timetracker. A worklog must have either a work item id\n" +
		"or a comment, and a duration greater than zero.",
	RunE: func(cmd *cobra.Command, args []string) error {
		spCfg, err := utils.GetSevenPaceConfig()
		if err != nil {
			return err
		}

		location, err := utils.GetTimezone()
		if err != nil {
			return err
		}

		workItem, err := cmd.Flags().GetInt("work-item")
		if err != nil {
			return fmt.Errorf("failed to get work-item flag: %w", err)
		}

		comment, err := cmd.Flags().GetString("comment")
		if err != nil {
			return fmt.Errorf("failed to get comment flag: %w", err)
		}

		durationStr, err := cmd.Flags().GetString("duration")
		if err != nil {
			return fmt.Errorf("failed to get duration flag: %w", err)
		}

		dateStr, err := cmd.Flags().GetString("date")
		if err != nil {
			return fmt.Errorf("failed to get date flag: %w", err)
		}

		activityType, err := cmd.Flags().GetString("activity-type")
		if err != nil {
			return fmt.Errorf("failed to get activity-type flag: %w", err)
		}

		length, err := parseDurationSeconds(durationStr)
		if err != nil {
			return err
		}
		if length <= 0 {
			return fmt.Errorf("--duration must be greater than zero")
		}

		if workItem == 0 && comment == "" {
			return fmt.Errorf("a worklog must have either --work-item or --comment")
		}

		timestamp, err := parseWorkLogDate(dateStr, location)
		if err != nil {
			return err
		}

		if activityType == "" {
			activityType = spCfg.ActivityTypeID
		}

		workLog := data.SevenPaceWorkLog{
			Timestamp: timestamp.Format(time.RFC3339),
			Length:    length,
			Comment:   comment,
		}
		if workItem != 0 {
			workLog.WorkItemID = &workItem
		}
		if activityType != "" {
			workLog.ActivityType = &data.SevenPaceActivityRef{ID: activityType}
		}

		spClient := api.NewSevenPaceClient(spCfg)
		if _, err := spClient.CreateWorkLog(workLog); err != nil {
			return fmt.Errorf("failed to create worklog: %w", err)
		}

		fmt.Printf("Posted worklog: %s for %s\n", api.FormatDuration(float64(length)), timestamp.Format("2006-01-02 15:04"))
		return nil
	},
}

// parseDurationSeconds accepts a Go duration string (e.g. "1h30m") or a plain
// number of seconds and returns the duration in seconds.
func parseDurationSeconds(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("--duration is required")
	}

	if d, err := time.ParseDuration(value); err == nil {
		return int(d.Seconds()), nil
	}

	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid --duration %q: use e.g. 1h30m or a number of seconds", value)
	}

	return seconds, nil
}

// parseWorkLogDate parses the --date flag in the given location, accepting
// either "YYYY-MM-DD HH:MM" or "YYYY-MM-DD". An empty value defaults to now.
func parseWorkLogDate(value string, location *time.Location) (time.Time, error) {
	if value == "" {
		return time.Now().In(location), nil
	}

	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, value, location); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid --date %q: use YYYY-MM-DD or \"YYYY-MM-DD HH:MM\"", value)
}

func init() {
	sevenpaceCmd.AddCommand(sevenpaceAddCmd)

	sevenpaceAddCmd.Flags().Int("work-item", 0, "Azure DevOps work item id")
	sevenpaceAddCmd.Flags().String("comment", "", "Worklog comment")
	sevenpaceAddCmd.Flags().String("duration", "", "Duration, e.g. 1h30m or a number of seconds")
	sevenpaceAddCmd.Flags().String("date", "", "Date/time of the worklog: YYYY-MM-DD or \"YYYY-MM-DD HH:MM\" (default now)")
	sevenpaceAddCmd.Flags().String("activity-type", "", "Activity type UUID (overrides config)")
}
