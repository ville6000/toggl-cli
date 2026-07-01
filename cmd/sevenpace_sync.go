package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/data"
	"github.com/ville6000/toggl-cli/internal/utils"
)

// plannedWorkLog pairs a built 7pace payload with the display columns used in
// the preview / result tables.
type plannedWorkLog struct {
	workItem string
	started  string
	duration string
	comment  string
	payload  data.SevenPaceWorkLog
}

var sevenpaceSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Toggl time entries to 7pace as worklogs",
	Long: "Fetch Toggl time entries for a date range and post each one to 7pace as a worklog.\n" +
		"The work item id is parsed from the entry description (e.g. \"#1234\" or a leading number);\n" +
		"entries without a work item id are skipped. There is no de-duplication, so re-running the\n" +
		"same range creates duplicate worklogs — use --dry-run first to preview.",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, _, err := utils.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		spCfg, err := utils.GetSevenPaceConfig()
		if err != nil {
			return err
		}

		location, err := utils.GetTimezone()
		if err != nil {
			return err
		}

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return fmt.Errorf("failed to get dry-run flag: %w", err)
		}

		assumeYes, err := cmd.Flags().GetBool("yes")
		if err != nil {
			return fmt.Errorf("failed to get yes flag: %w", err)
		}

		startTime, endTime, err := getDateParams(cmd)
		if err != nil {
			return err
		}

		client := api.NewAPIClient(token)
		timeEntries, err := client.GetHistory(&startTime, &endTime)
		if err != nil {
			return fmt.Errorf("failed to get history: %w", err)
		}

		var planned []plannedWorkLog
		var skipped [][]interface{}
		for _, entry := range timeEntries {
			if entry.Duration <= 0 {
				continue // skip running or zero-length entries
			}

			workLog, ok := toWorkLog(entry, spCfg.ActivityTypeID, location)
			started := entry.Start.In(location).Format("2006-01-02 15:04")
			duration := api.FormatDuration(float64(entry.Duration))

			if !ok {
				skipped = append(skipped, []interface{}{"—", started, duration, entry.Description})
				continue
			}

			planned = append(planned, plannedWorkLog{
				workItem: fmt.Sprintf("%d", *workLog.WorkItemID),
				started:  started,
				duration: duration,
				comment:  entry.Description,
				payload:  workLog,
			})
		}

		if len(planned) == 0 && len(skipped) == 0 {
			return fmt.Errorf("no time entries found for the specified date range")
		}

		headers := []interface{}{"Work Item", "Started At", "Duration", "Comment"}
		if len(planned) > 0 {
			rows := make([][]interface{}, 0, len(planned))
			for _, p := range planned {
				rows = append(rows, []interface{}{p.workItem, p.started, p.duration, p.comment})
			}
			utils.RenderTable("Worklogs to post", headers, rows, nil)
			fmt.Println()
		}
		if len(skipped) > 0 {
			utils.RenderTable("Skipped (no work item id)", headers, skipped, nil)
			fmt.Println()
		}

		if dryRun {
			fmt.Printf("Dry run: %d worklog(s) would be posted, %d skipped.\n", len(planned), len(skipped))
			return nil
		}

		if len(planned) == 0 {
			fmt.Println("Nothing to post.")
			return nil
		}

		if !assumeYes && !confirm(fmt.Sprintf("Post %d worklog(s) to 7pace?", len(planned))) {
			fmt.Println("Aborted.")
			return nil
		}

		spClient := api.NewSevenPaceClient(spCfg)
		posted := 0
		var failures [][]interface{}
		for _, p := range planned {
			if _, postErr := spClient.CreateWorkLog(p.payload); postErr != nil {
				failures = append(failures, []interface{}{p.workItem, p.started, p.duration, postErr.Error()})
				continue
			}
			posted++
		}

		fmt.Printf("Posted %d worklog(s), %d skipped, %d failed.\n", posted, len(skipped), len(failures))
		if len(failures) > 0 {
			utils.RenderTable("Failed", []interface{}{"Work Item", "Started At", "Duration", "Error"}, failures, nil)
			return fmt.Errorf("%d worklog(s) failed to post", len(failures))
		}

		return nil
	},
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func init() {
	sevenpaceCmd.AddCommand(sevenpaceSyncCmd)

	sevenpaceSyncCmd.Flags().BoolP("today", "t", false, "Sync only today's entries (default when no date flags are given)")
	sevenpaceSyncCmd.Flags().BoolP("week", "w", false, "Sync entries for the current week")
	sevenpaceSyncCmd.Flags().BoolP("month", "m", false, "Sync entries for the current month")
	sevenpaceSyncCmd.Flags().StringP("start", "s", "", "Start date, format: YYYY-MM-DD")
	sevenpaceSyncCmd.Flags().StringP("end", "e", "", "End date, format: YYYY-MM-DD")
	sevenpaceSyncCmd.Flags().Bool("dry-run", false, "Preview the worklogs without posting")
	sevenpaceSyncCmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt")
}
