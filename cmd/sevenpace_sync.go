package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"
	"github.com/ville6000/toggl-cli/internal/output"
)

// plannedWorkLog pairs a built 7pace payload with the display columns used in
// the preview / result tables.
type plannedWorkLog struct {
	workItem string
	started  string
	duration string
	comment  string
	payload  api.SevenPaceWorkLog
}

func newSevenPaceSyncCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync Toggl time entries to 7pace as worklogs",
		Long: "Fetch Toggl time entries for a date range and post them to 7pace as worklogs.\n" +
			"Entries sharing the same description are combined into a single worklog, with their\n" +
			"durations summed and rounded up to the nearest minute. The work item id is parsed from\n" +
			"the description (e.g. \"#1234\" or a leading number); entries without a work item id are\n" +
			"skipped. There is no de-duplication, so re-running the same range creates duplicate\n" +
			"worklogs — use --dry-run first to preview.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			token, _, err := config.TokenAndWorkspace(v)
			if err != nil {
				return fmt.Errorf("failed to get configuration: %w", err)
			}

			spCfg, err := config.LoadSevenPace(v)
			if err != nil {
				return err
			}

			location, err := config.Timezone(v)
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

			startTime, endTime, err := getDateParams(cmd, location, true)
			if err != nil {
				return err
			}

			client := newTogglClient(v, token)
			timeEntries, err := client.TimeEntries(ctx, &startTime, &endTime)
			if err != nil {
				return fmt.Errorf("failed to get history: %w", err)
			}

			// Combine entries sharing a description into a single worklog, summing
			// their durations and rounding up to the nearest minute. Running or
			// zero-length entries are dropped here.
			entries := aggregateEntries(timeEntries)

			var planned []plannedWorkLog
			var skipped [][]any
			plannedSeconds := 0
			skippedSeconds := 0
			for _, entry := range entries {
				workLog, ok := toWorkLog(entry, spCfg.ActivityTypeID, location)
				started := entry.Start.In(location).Format("2006-01-02 15:04")
				duration := output.FormatDuration(entry.Duration)

				if !ok {
					skipped = append(skipped, []any{"—", started, duration, entry.Description})
					skippedSeconds += entry.Duration
					continue
				}

				planned = append(planned, plannedWorkLog{
					workItem: strconv.Itoa(*workLog.WorkItemID),
					started:  started,
					duration: duration,
					comment:  entry.Description,
					payload:  workLog,
				})
				plannedSeconds += workLog.Length
			}

			if len(planned) == 0 && len(skipped) == 0 {
				return errors.New("no time entries found for the specified date range")
			}

			out := cmd.OutOrStdout()
			headers := []any{"Work Item", "Started At", "Duration", "Comment"}
			if len(planned) > 0 {
				rows := make([][]any, 0, len(planned))
				for _, p := range planned {
					rows = append(rows, []any{p.workItem, p.started, p.duration, p.comment})
				}
				output.RenderTable(out, "Worklogs to post", headers, rows, totalFooter(plannedSeconds))
				fmt.Fprintln(out)
			}
			if len(skipped) > 0 {
				output.RenderTable(out, "Skipped (no work item id)", headers, skipped, totalFooter(skippedSeconds))
				fmt.Fprintln(out)
			}

			if dryRun {
				fmt.Fprintf(out, "Dry run: %d worklog(s) (%s) would be posted, %d skipped.\n",
					len(planned), output.FormatDuration(plannedSeconds), len(skipped))
				return nil
			}

			if len(planned) == 0 {
				fmt.Fprintln(out, "Nothing to post.")
				return nil
			}

			prompt := fmt.Sprintf("Post %d worklog(s) (%s) to 7pace?", len(planned), output.FormatDuration(plannedSeconds))
			if !assumeYes && !confirm(out, cmd.InOrStdin(), prompt) {
				fmt.Fprintln(out, "Aborted.")
				return nil
			}

			spClient := api.NewSevenPaceClient(spCfg)
			posted := 0
			postedSeconds := 0
			var failures [][]any
			for _, p := range planned {
				if _, postErr := spClient.CreateWorkLog(ctx, p.payload); postErr != nil {
					failures = append(failures, []any{p.workItem, p.started, p.duration, postErr.Error()})
					continue
				}
				posted++
				postedSeconds += p.payload.Length
			}

			fmt.Fprintf(out, "Posted %d worklog(s) (%s), %d skipped, %d failed.\n",
				posted, output.FormatDuration(postedSeconds), len(skipped), len(failures))
			if len(failures) > 0 {
				output.RenderTable(out, "Failed", []any{"Work Item", "Started At", "Duration", "Error"}, failures, nil)
				return fmt.Errorf("%d worklog(s) failed to post", len(failures))
			}

			return nil
		},
	}

	cmd.Flags().BoolP("today", "t", false, "Sync only today's entries (default when no date flags are given)")
	cmd.Flags().BoolP("week", "w", false, "Sync entries for the current week")
	cmd.Flags().BoolP("month", "m", false, "Sync entries for the current month")
	cmd.Flags().StringP("start", "s", "", "Start date, format: YYYY-MM-DD")
	cmd.Flags().StringP("end", "e", "", "End date (inclusive), format: YYYY-MM-DD (defaults to --start)")
	cmd.Flags().Bool("dry-run", false, "Preview the worklogs without posting")
	cmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt")

	return cmd
}

// totalFooter builds the footer row summing the Duration column of the sync
// tables, so the time about to be logged to 7pace is visible at a glance.
func totalFooter(seconds int) table.Row {
	return table.Row{"", "Total", output.FormatDuration(seconds), ""}
}

func confirm(out io.Writer, in io.Reader, prompt string) bool {
	fmt.Fprintf(out, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
