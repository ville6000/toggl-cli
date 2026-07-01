package cmd

import (
	"regexp"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/data"
)

// sevenpaceCmd is the parent command for posting worklogs to an on-prem 7pace
// Timetracker instance.
//
// Note: 7pace credentials (domain, username, password) are stored in plaintext
// in the config file. The `sync` command performs no de-duplication, so
// re-running the same date range will create duplicate worklogs — use
// `--dry-run` first to preview.
var sevenpaceCmd = &cobra.Command{
	Use:   "7pace",
	Short: "Post worklogs to 7pace Timetracker",
	Long:  "Post worklogs to an on-prem 7pace Timetracker instance from your Toggl time entries.",
}

var (
	workItemHashRe    = regexp.MustCompile(`#(\d+)`)
	workItemLeadingRe = regexp.MustCompile(`^\s*(\d+)`)
)

// parseWorkItemID extracts an Azure DevOps work item id from a Toggl entry
// description. It prefers a `#1234` style reference (e.g. `AB#1234`) and falls
// back to a leading number (e.g. `1234 - do stuff`). Returns false when no id
// can be found.
func parseWorkItemID(description string) (int, bool) {
	if m := workItemHashRe.FindStringSubmatch(description); m != nil {
		id, err := strconv.Atoi(m[1])
		if err == nil {
			return id, true
		}
	}
	if m := workItemLeadingRe.FindStringSubmatch(description); m != nil {
		id, err := strconv.Atoi(m[1])
		if err == nil {
			return id, true
		}
	}
	return 0, false
}

// toWorkLog maps a Toggl time entry to a 7pace worklog. The bool return
// reports whether a work item id was found in the description.
func toWorkLog(entry data.TimeEntryItem, activityTypeID string, location *time.Location) (data.SevenPaceWorkLog, bool) {
	id, ok := parseWorkItemID(entry.Description)

	workLog := data.SevenPaceWorkLog{
		Timestamp: entry.Start.In(location).Format(time.RFC3339),
		Length:    entry.Duration,
		Comment:   entry.Description,
	}

	if ok {
		workLog.WorkItemID = &id
	}

	if activityTypeID != "" {
		workLog.ActivityType = &data.SevenPaceActivityRef{ID: activityTypeID}
	}

	return workLog, ok
}

func init() {
	rootCmd.AddCommand(sevenpaceCmd)
}
