package cmd

import (
	"encoding/json"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ville6000/toggl-cli/internal/api"
)

// utcEntry builds a stopped entry as the API returns it: a UTC timestamp and a
// positive duration in seconds.
func utcEntry(id int, start time.Time, duration int, description string) api.TimeEntryItem {
	return api.TimeEntryItem{
		ID:          id,
		Description: description,
		Duration:    duration,
		ProjectID:   7,
		WorkspaceID: testWorkspaceID,
		Start:       start.UTC(),
	}
}

func TestHistoryCommand_SumsEntriesPerDayAndAsksForTheRequestedRange(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	stub.stubProjects(api.Project{ID: 7, Name: "Alpha"})
	stub.stubHistory(
		// 10:00 and 14:00 Tokyo time on 2024-03-04.
		utcEntry(1, time.Date(2024, 3, 4, 1, 0, 0, 0, time.UTC), 3600, "review"),
		utcEntry(2, time.Date(2024, 3, 4, 5, 0, 0, 0, time.UTC), 1800, "review"),
		utcEntry(3, time.Date(2024, 3, 4, 6, 0, 0, 0, time.UTC), 900, "standup"),
	)

	out, _, err := executeCommand(t, v, "history", "--start", "2024-03-04", "--end", "2024-03-04")
	if err != nil {
		t.Fatalf("history: %v", err)
	}

	for _, want := range []string{
		"# 2024-03-04",
		"Summary for: 2024-03-04",
		"Alpha",
		"01:30:00", // review: 3600 + 1800
		"00:15:00", // standup
		"01:45:00", // total
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	// The range is sent as instants in the configured timezone, and --end is
	// inclusive, so the last day is covered up to the next midnight.
	query := stub.onlyRequestFor(http.MethodGet, "/me/time_entries").Query
	if got, want := query.Get("start_date"), "2024-03-04T00:00:00+09:00"; got != want {
		t.Errorf("start_date: got %q, want %q", got, want)
	}
	if got, want := query.Get("end_date"), "2024-03-05T00:00:00+09:00"; got != want {
		t.Errorf("end_date: got %q, want %q", got, want)
	}
}

func TestHistoryCommand_GroupsByLocalDateAcrossTimezoneBoundary(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	stub.stubProjects(api.Project{ID: 7, Name: "Alpha"})
	// 22:30 UTC on the 4th is 07:30 on the 5th in Tokyo, so the entry belongs
	// to the 5th as far as the user is concerned.
	stub.stubHistory(utcEntry(1, time.Date(2024, 3, 4, 22, 30, 0, 0, time.UTC), 3600, "review"))

	out, _, err := executeCommand(t, v, "history", "--start", "2024-03-05", "--end", "2024-03-05", "--verbose")
	if err != nil {
		t.Fatalf("history: %v", err)
	}

	if !strings.Contains(out, "# 2024-03-05") {
		t.Errorf("entry not grouped under its local date:\n%s", out)
	}
	if strings.Contains(out, "# 2024-03-04") {
		t.Errorf("entry grouped under its UTC date:\n%s", out)
	}
	if !strings.Contains(out, "Entries for: 05.03.2024") {
		t.Errorf("verbose table titled with the wrong date:\n%s", out)
	}
	if !strings.Contains(out, "07:30") {
		t.Errorf("start time not shown in the configured timezone:\n%s", out)
	}
}

func TestHistoryCommand_RunningEntryCountsAsElapsedTime(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	start := time.Now().Add(-30 * time.Minute)
	stub.stubProjects(api.Project{ID: 7, Name: "Alpha"})
	stub.stubHistory(api.TimeEntryItem{
		ID:          1,
		Description: "running",
		// Toggl reports a running entry as the negated start timestamp.
		Duration:    int(-start.Unix()),
		ProjectID:   7,
		WorkspaceID: testWorkspaceID,
		Start:       start.UTC(),
	})

	out, _, err := executeCommand(t, v, "history")
	if err != nil {
		t.Fatalf("history: %v", err)
	}

	// Half an hour so far, give or take the time the test itself takes.
	elapsed := regexp.MustCompile(`00:(29:5\d|30:0\d)`)
	if !elapsed.MatchString(out) {
		t.Errorf("running entry not reported as elapsed time:\n%s", out)
	}
	if strings.Contains(out, "-4") {
		t.Errorf("negative duration sentinel leaked into the output:\n%s", out)
	}
}

func TestHistoryCommand_VerboseListsIndividualEntries(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	stub.stubProjects(api.Project{ID: 7, Name: "Alpha"})
	stub.stubHistory(
		utcEntry(1, time.Date(2024, 3, 4, 1, 0, 0, 0, time.UTC), 3600, "review"),
		utcEntry(2, time.Date(2024, 3, 4, 5, 0, 0, 0, time.UTC), 1800, "review"),
	)

	verbose, _, err := executeCommand(t, v, "history", "--start", "2024-03-04", "--end", "2024-03-04", "--verbose")
	if err != nil {
		t.Fatalf("history --verbose: %v", err)
	}
	if !strings.Contains(verbose, "Entries for: 04.03.2024") {
		t.Errorf("verbose output missing the per-entry table:\n%s", verbose)
	}
	for _, want := range []string{"10:00", "14:00"} {
		if !strings.Contains(verbose, want) {
			t.Errorf("verbose output missing start time %q:\n%s", want, verbose)
		}
	}
	// The IDs that edit and continue accept with --id.
	if !strings.Contains(verbose, "| ID ") {
		t.Errorf("verbose output missing the ID column:\n%s", verbose)
	}

	plain, _, err := executeCommand(t, v, "history", "--start", "2024-03-04", "--end", "2024-03-04")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if strings.Contains(plain, "Entries for:") {
		t.Errorf("per-entry table shown without --verbose:\n%s", plain)
	}
}

// A project created after the cache was saved still shows its name: the
// missing ID triggers a fresh project list.
func TestHistoryCommand_NamesProjectsMissingFromAStaleCache(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	projectCache, err := api.NewProjectCache()
	if err != nil {
		t.Fatalf("project cache: %v", err)
	}
	if err := projectCache.SaveProjects(testWorkspaceID, []api.Project{{ID: 7, Name: "Alpha"}}); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	stub.stubProjects(api.Project{ID: 7, Name: "Alpha"}, api.Project{ID: 8, Name: "Brand New"})
	entry := utcEntry(1, time.Date(2024, 3, 4, 1, 0, 0, 0, time.UTC), 3600, "kickoff")
	entry.ProjectID = 8
	stub.stubHistory(entry)

	out, _, err := executeCommand(t, v, "history", "--start", "2024-03-04", "--end", "2024-03-04")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if !strings.Contains(out, "Brand New") {
		t.Errorf("output missing the new project's name:\n%s", out)
	}
}

func TestHistoryCommand_NoEntriesIsAnError(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	stub.stubProjects()
	stub.stubHistory()

	_, _, err := executeCommand(t, v, "history", "--start", "2024-03-04")
	if err == nil {
		t.Fatal("expected an error when the range holds no entries")
	}
	if !strings.Contains(err.Error(), "no time entries found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHistoryCommand_ReportsAPIFailure(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	stub.stubProjects()
	stub.respond(http.MethodGet, "/me/time_entries", http.StatusInternalServerError, nil)

	_, _, err := executeCommand(t, v, "history")
	if err == nil {
		t.Fatal("expected an error when the API fails")
	}
	if !strings.Contains(err.Error(), "failed to get history") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHistoryCommand_RejectsEndBeforeStart(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	stub.stubProjects()

	_, _, err := executeCommand(t, v, "history", "--start", "2024-03-04", "--end", "2024-03-01")
	if err == nil {
		t.Fatal("expected an error when --end precedes --start")
	}
	if !strings.Contains(err.Error(), "is before --start") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHistoryCommand_JSONListsEntriesOldestFirstInTheConfiguredTimezone(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	stub.stubProjects(api.Project{ID: 7, Name: "Alpha"})
	tagged := utcEntry(2, time.Date(2024, 3, 4, 1, 0, 0, 0, time.UTC), 1800, "#1234 review")
	tagged.Tags = []string{"billable"}
	// The API lists the newest entry first.
	stub.stubHistory(
		utcEntry(1, time.Date(2024, 3, 4, 5, 0, 0, 0, time.UTC), 900, "standup"),
		tagged,
	)

	out, _, err := executeCommand(t, v, "history", "--start", "2024-03-04", "--end", "2024-03-04", "--json")
	if err != nil {
		t.Fatalf("history --json: %v", err)
	}

	var got []historyJSONEntry
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not a JSON entry list: %v\n%s", err, out)
	}

	want := []historyJSONEntry{
		{ID: 2, Start: "2024-03-04T10:00:00+09:00", Duration: 1800, Description: "#1234 review", Project: "Alpha", Tags: []string{"billable"}},
		{ID: 1, Start: "2024-03-04T14:00:00+09:00", Duration: 900, Description: "standup", Project: "Alpha", Tags: []string{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestHistoryCommand_JSONMarksRunningEntries(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	start := time.Now().Add(-30 * time.Minute)
	stub.stubProjects(api.Project{ID: 7, Name: "Alpha"})
	stub.stubHistory(api.TimeEntryItem{
		ID:          1,
		Description: "running",
		Duration:    int(-start.Unix()),
		ProjectID:   7,
		WorkspaceID: testWorkspaceID,
		Start:       start.UTC(),
	})

	out, _, err := executeCommand(t, v, "history", "--json")
	if err != nil {
		t.Fatalf("history --json: %v", err)
	}

	var got []historyJSONEntry
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not a JSON entry list: %v\n%s", err, out)
	}
	if len(got) != 1 || !got[0].Running {
		t.Fatalf("want one running entry, got %+v", got)
	}
	if got[0].Duration < 29*60 || got[0].Duration > 31*60 {
		t.Errorf("duration: got %d, want the ~1800 seconds elapsed so far", got[0].Duration)
	}
}

// A pipeline gets valid input for an empty range instead of an error.
func TestHistoryCommand_JSONWithNoEntriesIsAnEmptyList(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	stub.stubProjects()
	stub.stubHistory()

	out, _, err := executeCommand(t, v, "history", "--start", "2024-03-04", "--json")
	if err != nil {
		t.Fatalf("history --json: %v", err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("output: got %q, want []", out)
	}
}

func TestHistoryCommand_DayAsksForThatDayOnly(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	stub.stubProjects(api.Project{ID: 7, Name: "Alpha"})
	stub.stubHistory(utcEntry(1, time.Date(2024, 3, 4, 1, 0, 0, 0, time.UTC), 3600, "review"))

	if _, _, err := executeCommand(t, v, "history", "--day", "2024-03-04"); err != nil {
		t.Fatalf("history --day: %v", err)
	}

	query := stub.onlyRequestFor(http.MethodGet, "/me/time_entries").Query
	if got, want := query.Get("start_date"), "2024-03-04T00:00:00+09:00"; got != want {
		t.Errorf("start_date: got %q, want %q", got, want)
	}
	if got, want := query.Get("end_date"), "2024-03-05T00:00:00+09:00"; got != want {
		t.Errorf("end_date: got %q, want %q", got, want)
	}
}

func TestHistoryCommand_DayCannotBeCombinedWithARange(t *testing.T) {
	v := setupCLITest(t, newAPIStub(t))

	for _, other := range [][]string{{"--start", "2024-03-01"}, {"--end", "2024-03-05"}, {"--week"}, {"--month"}} {
		args := append([]string{"history", "--day", "2024-03-04"}, other...)
		if _, _, err := executeCommand(t, v, args...); err == nil {
			t.Errorf("history --day %v: expected an error", other)
		}
	}
}
