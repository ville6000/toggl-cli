package cmd

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
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
