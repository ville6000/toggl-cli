package cmd

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/data"
)

// newDateFlagCmd builds a command carrying the same date flags as history /
// 7pace sync. withToday mirrors sync, which also has --today.
func newDateFlagCmd(withToday bool, args ...string) (*cobra.Command, error) {
	cmd := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.Flags().Bool("week", false, "")
	cmd.Flags().Bool("month", false, "")
	cmd.Flags().String("start", "", "")
	cmd.Flags().String("end", "", "")
	if withToday {
		cmd.Flags().Bool("today", false, "")
	}

	return cmd, cmd.Flags().Parse(args)
}

func localDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return parsed
}

func TestGetDateParams_EndIsInclusive(t *testing.T) {
	cmd, err := newDateFlagCmd(false, "--start", "2024-03-04", "--end", "2024-03-04")
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	start, end, err := getDateParams(cmd, false)
	if err != nil {
		t.Fatalf("getDateParams: %v", err)
	}

	// A single day is [midnight, next midnight), not an empty range.
	if want := localDate(t, "2024-03-04"); !start.Equal(want) {
		t.Errorf("start: got %s, want %s", start, want)
	}
	if want := localDate(t, "2024-03-05"); !end.Equal(want) {
		t.Errorf("end: got %s, want %s", end, want)
	}
}

func TestGetDateParams_StartOnlyRunsThroughToday(t *testing.T) {
	cmd, err := newDateFlagCmd(false, "--start", "2024-03-04")
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	_, end, err := getDateParams(cmd, false)
	if err != nil {
		t.Fatalf("getDateParams: %v", err)
	}

	want := startOfDay(time.Now(), time.Local).AddDate(0, 0, 1)
	if !end.Equal(want) {
		t.Errorf("end: got %s, want %s", end, want)
	}
}

func TestGetDateParams_StartOnlyIsSingleDayWhenEndDefaultsToStart(t *testing.T) {
	cmd, err := newDateFlagCmd(true, "--start", "2024-03-04")
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	start, end, err := getDateParams(cmd, true)
	if err != nil {
		t.Fatalf("getDateParams: %v", err)
	}

	if want := localDate(t, "2024-03-04"); !start.Equal(want) {
		t.Errorf("start: got %s, want %s", start, want)
	}
	if want := localDate(t, "2024-03-05"); !end.Equal(want) {
		t.Errorf("end: got %s, want %s", end, want)
	}
}

func TestGetDateParams_NoFlagsIsToday(t *testing.T) {
	for _, withToday := range []bool{false, true} {
		cmd, err := newDateFlagCmd(withToday)
		if err != nil {
			t.Fatalf("parse flags: %v", err)
		}

		start, end, err := getDateParams(cmd, withToday)
		if err != nil {
			t.Fatalf("getDateParams: %v", err)
		}

		today := startOfDay(time.Now(), time.Local)
		if !start.Equal(today) {
			t.Errorf("withToday=%v start: got %s, want %s", withToday, start, today)
		}
		if want := today.AddDate(0, 0, 1); !end.Equal(want) {
			t.Errorf("withToday=%v end: got %s, want %s", withToday, end, want)
		}
	}
}

func TestGetDateParams_WeekIncludesSunday(t *testing.T) {
	cmd, err := newDateFlagCmd(false, "--week")
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	start, end, err := getDateParams(cmd, false)
	if err != nil {
		t.Fatalf("getDateParams: %v", err)
	}

	if start.Weekday() != time.Monday {
		t.Errorf("start weekday: got %s, want Monday", start.Weekday())
	}
	if want := start.AddDate(0, 0, 7); !end.Equal(want) {
		t.Errorf("end: got %s, want %s (Sunday must be inside the range)", end, want)
	}
}

func TestGetDateParams_MonthCoversWholeMonth(t *testing.T) {
	cmd, err := newDateFlagCmd(false, "--month")
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	start, end, err := getDateParams(cmd, false)
	if err != nil {
		t.Fatalf("getDateParams: %v", err)
	}

	if start.Day() != 1 {
		t.Errorf("start: got %s, want the first of the month", start)
	}
	if want := start.AddDate(0, 1, 0); !end.Equal(want) {
		t.Errorf("end: got %s, want %s", end, want)
	}
}

func TestGetDateParams_RangeStartsAtMidnight(t *testing.T) {
	cmd, err := newDateFlagCmd(true, "--today")
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	start, _, err := getDateParams(cmd, true)
	if err != nil {
		t.Fatalf("getDateParams: %v", err)
	}

	if h, m, s := start.Clock(); h != 0 || m != 0 || s != 0 {
		t.Errorf("start clock: got %02d:%02d:%02d, want midnight", h, m, s)
	}
}

func TestGetDateParams_EndBeforeStartErrors(t *testing.T) {
	cmd, err := newDateFlagCmd(false, "--start", "2024-03-04", "--end", "2024-03-01")
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	if _, _, err := getDateParams(cmd, false); err == nil {
		t.Error("expected an error when --end precedes --start")
	}
}

func TestGetDateParams_InvalidDateErrors(t *testing.T) {
	cmd, err := newDateFlagCmd(false, "--start", "04.03.2024")
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	if _, _, err := getDateParams(cmd, false); err == nil {
		t.Error("expected an error for a malformed --start")
	}
}

// ---------- entryDuration ----------

func TestEntryDuration(t *testing.T) {
	now := time.Date(2024, 3, 4, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		entry data.TimeEntryItem
		want  int
	}{
		{
			name:  "stopped entry keeps its duration",
			entry: data.TimeEntryItem{Duration: 3600, Start: now.Add(-2 * time.Hour)},
			want:  3600,
		},
		{
			name: "running entry counts the elapsed time",
			// Toggl encodes a running entry as the negated start timestamp.
			entry: data.TimeEntryItem{Duration: int(-now.Add(-90 * time.Minute).Unix()), Start: now.Add(-90 * time.Minute)},
			want:  5400,
		},
		{
			name:  "running entry starting in the future is not negative",
			entry: data.TimeEntryItem{Duration: -1, Start: now.Add(time.Hour)},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := entryDuration(tt.entry, now); got != tt.want {
				t.Errorf("entryDuration() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ---------- groupEntriesByDate ----------

func TestGroupEntriesByDate_UsesTheConfiguredTimezone(t *testing.T) {
	// +09:00, so 22:30 UTC belongs to the next local day.
	location := time.FixedZone("TEST", 9*60*60)
	entries := []data.TimeEntryItem{
		{ID: 1, Start: time.Date(2024, 3, 4, 22, 30, 0, 0, time.UTC)},
		{ID: 2, Start: time.Date(2024, 3, 5, 1, 0, 0, 0, time.UTC)},
		{ID: 3, Start: time.Date(2024, 3, 4, 10, 0, 0, 0, time.UTC)},
	}

	grouped := groupEntriesByDate(entries, location)

	if got := len(grouped["2024-03-05"]); got != 2 {
		t.Errorf("2024-03-05: got %d entries, want 2", got)
	}
	if got := len(grouped["2024-03-04"]); got != 1 {
		t.Errorf("2024-03-04: got %d entries, want 1", got)
	}
}

// ---------- sumEntriesByDescriptionAndProject ----------

func TestSumEntriesByDescriptionAndProject(t *testing.T) {
	now := time.Date(2024, 3, 4, 12, 0, 0, 0, time.UTC)
	projects := map[int]string{7: "Alpha"}
	entries := []data.TimeEntryItem{
		{Description: "review", ProjectID: 7, Duration: 3600, Start: now.Add(-4 * time.Hour)},
		{Description: "review", ProjectID: 7, Duration: 1800, Start: now.Add(-2 * time.Hour)},
		{Description: "standup", ProjectID: 7, Duration: 900, Start: now.Add(-time.Hour)},
	}

	summary := sumEntriesByDescriptionAndProject(entries, projects, now)

	if got := len(summary); got != 2 {
		t.Fatalf("got %d summary rows, want 2", got)
	}
	if got := summary["review - Alpha"].Duration; got != 5400 {
		t.Errorf("review duration: got %d, want 5400", got)
	}
	if got := summary["review - Alpha"].Project; got != "Alpha" {
		t.Errorf("review project: got %q, want %q", got, "Alpha")
	}
	if got := summary["standup - Alpha"].Duration; got != 900 {
		t.Errorf("standup duration: got %d, want 900", got)
	}
}

func TestSumEntriesByDescriptionAndProject_RunningEntryDoesNotCorruptTheTotal(t *testing.T) {
	now := time.Date(2024, 3, 4, 12, 0, 0, 0, time.UTC)
	start := now.Add(-30 * time.Minute)
	entries := []data.TimeEntryItem{
		{Description: "review", ProjectID: 7, Duration: 3600, Start: now.Add(-4 * time.Hour)},
		{Description: "review", ProjectID: 7, Duration: int(-start.Unix()), Start: start},
	}

	summary := sumEntriesByDescriptionAndProject(entries, map[int]string{7: "Alpha"}, now)

	// An hour done plus half an hour still running, not the raw sentinel.
	if got := summary["review - Alpha"].Duration; got != 5400 {
		t.Errorf("duration: got %d, want 5400", got)
	}
}

// ---------- getSortedTimeEntryDates ----------

func TestGetSortedTimeEntryDates_NewestFirst(t *testing.T) {
	grouped := map[string][]data.TimeEntryItem{
		"2024-03-04": nil,
		"2024-03-06": nil,
		"2024-03-05": nil,
	}

	got := getSortedTimeEntryDates(grouped)
	want := []string{"2024-03-06", "2024-03-05", "2024-03-04"}

	if len(got) != len(want) {
		t.Fatalf("got %d dates, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("date %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
