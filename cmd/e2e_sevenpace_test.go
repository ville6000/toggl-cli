package cmd

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/data"
)

const sevenPaceWorkLogPath = "/workLogs"

// setupSevenPaceTest wires both the Toggl and the 7pace stubs into the config
// and returns the 7pace stub so tests can inspect what was posted.
func setupSevenPaceTest(t *testing.T, toggl *apiStub) *apiStub {
	t.Helper()

	sevenPace := newAPIStub(t)
	setupCLITest(t, toggl)

	viper.Set("sevenpace.base_url", sevenPace.server.URL)
	viper.Set("sevenpace.username", "user")
	viper.Set("sevenpace.password", "secret")
	viper.Set("sevenpace.activity_type_id", "activity-uuid")

	return sevenPace
}

// syncEntries are the entries the sync tests work from: two that share a
// description and carry a work item id, and one without an id.
func syncEntries() []data.TimeEntryItem {
	return []data.TimeEntryItem{
		utcEntry(1, time.Date(2024, 3, 4, 1, 0, 0, 0, time.UTC), 1500, "#1234 review", 7),
		utcEntry(2, time.Date(2024, 3, 4, 5, 0, 0, 0, time.UTC), 100, "#1234 review", 7),
		utcEntry(3, time.Date(2024, 3, 4, 6, 0, 0, 0, time.UTC), 900, "no work item", 7),
	}
}

func TestSevenPaceSync_DryRunPostsNothing(t *testing.T) {
	toggl := newAPIStub(t)
	sevenPace := setupSevenPaceTest(t, toggl)

	toggl.stubHistory(syncEntries()...)

	out, _, err := executeCommand(t, "7pace", "sync", "--start", "2024-03-04", "--dry-run")
	if err != nil {
		t.Fatalf("7pace sync --dry-run: %v", err)
	}

	for _, want := range []string{
		"Worklogs to post",
		"1234",
		"Skipped (no work item id)",
		"no work item",
		"Dry run: 1 worklog(s) would be posted, 1 skipped.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	if posted := sevenPace.requestsFor(http.MethodPost, sevenPaceWorkLogPath); len(posted) != 0 {
		t.Errorf("--dry-run posted %d worklog(s), want 0", len(posted))
	}
}

func TestSevenPaceSync_AggregatesEntriesSharingADescription(t *testing.T) {
	toggl := newAPIStub(t)
	sevenPace := setupSevenPaceTest(t, toggl)

	toggl.stubHistory(syncEntries()...)
	sevenPace.respond(http.MethodPost, sevenPaceWorkLogPath, http.StatusOK, data.SevenPaceWorkLog{})

	out, _, err := executeCommand(t, "7pace", "sync", "--start", "2024-03-04", "--yes")
	if err != nil {
		t.Fatalf("7pace sync: %v", err)
	}

	var posted data.SevenPaceWorkLog
	sevenPace.onlyRequestFor(http.MethodPost, sevenPaceWorkLogPath).decodeBody(t, &posted)

	if posted.WorkItemID == nil || *posted.WorkItemID != 1234 {
		t.Errorf("work item id: got %v, want 1234", posted.WorkItemID)
	}
	// 1500 + 100 seconds, rounded up to the next whole minute.
	if want := 1620; posted.Length != want {
		t.Errorf("length: got %d, want %d", posted.Length, want)
	}
	// The earliest start of the group, in the configured timezone.
	if want := "2024-03-04T10:00:00+09:00"; posted.Timestamp != want {
		t.Errorf("timestamp: got %q, want %q", posted.Timestamp, want)
	}
	if posted.Comment != "#1234 review" {
		t.Errorf("comment: got %q, want %q", posted.Comment, "#1234 review")
	}
	if posted.ActivityType == nil || posted.ActivityType.ID != "activity-uuid" {
		t.Errorf("activity type: got %v, want the configured uuid", posted.ActivityType)
	}

	if want := "Posted 1 worklog(s), 1 skipped, 0 failed."; !strings.Contains(out, want) {
		t.Errorf("output missing %q:\n%s", want, out)
	}
}

func TestSevenPaceSync_EndDefaultsToStart(t *testing.T) {
	toggl := newAPIStub(t)
	setupSevenPaceTest(t, toggl)

	toggl.stubHistory(syncEntries()...)

	if _, _, err := executeCommand(t, "7pace", "sync", "--start", "2024-03-04", "--dry-run"); err != nil {
		t.Fatalf("7pace sync --dry-run: %v", err)
	}

	// Only the named day: syncing through today would post worklogs the user
	// never asked for.
	query := toggl.onlyRequestFor(http.MethodGet, "/me/time_entries").Query
	if got, want := query.Get("end_date"), "2024-03-05T00:00:00+09:00"; got != want {
		t.Errorf("end_date: got %q, want %q", got, want)
	}
}

func TestSevenPaceSync_AbortsWhenConfirmationIsDeclined(t *testing.T) {
	toggl := newAPIStub(t)
	sevenPace := setupSevenPaceTest(t, toggl)

	toggl.stubHistory(syncEntries()...)

	out, _, err := executeCommandWithInput(t, "n\n", "7pace", "sync", "--start", "2024-03-04")
	if err != nil {
		t.Fatalf("7pace sync: %v", err)
	}

	if !strings.Contains(out, "Post 1 worklog(s) to 7pace? [y/N]: ") {
		t.Errorf("output missing the confirmation prompt:\n%s", out)
	}
	if !strings.Contains(out, "Aborted.") {
		t.Errorf("output missing the abort notice:\n%s", out)
	}
	if posted := sevenPace.requestsFor(http.MethodPost, sevenPaceWorkLogPath); len(posted) != 0 {
		t.Errorf("declined sync posted %d worklog(s), want 0", len(posted))
	}
}

func TestSevenPaceSync_PostsWhenConfirmationIsAccepted(t *testing.T) {
	toggl := newAPIStub(t)
	sevenPace := setupSevenPaceTest(t, toggl)

	toggl.stubHistory(syncEntries()...)
	sevenPace.respond(http.MethodPost, sevenPaceWorkLogPath, http.StatusOK, data.SevenPaceWorkLog{})

	if _, _, err := executeCommandWithInput(t, "y\n", "7pace", "sync", "--start", "2024-03-04"); err != nil {
		t.Fatalf("7pace sync: %v", err)
	}

	if posted := sevenPace.requestsFor(http.MethodPost, sevenPaceWorkLogPath); len(posted) != 1 {
		t.Errorf("confirmed sync posted %d worklog(s), want 1", len(posted))
	}
}

func TestSevenPaceSync_ReportsFailedWorklogs(t *testing.T) {
	toggl := newAPIStub(t)
	sevenPace := setupSevenPaceTest(t, toggl)

	toggl.stubHistory(syncEntries()...)
	sevenPace.respond(http.MethodPost, sevenPaceWorkLogPath, http.StatusInternalServerError, nil)

	out, _, err := executeCommand(t, "7pace", "sync", "--start", "2024-03-04", "--yes")
	if err == nil {
		t.Fatal("expected an error when a worklog fails to post")
	}
	if !strings.Contains(err.Error(), "1 worklog(s) failed to post") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Posted 0 worklog(s), 1 skipped, 1 failed.") {
		t.Errorf("output missing the failure summary:\n%s", out)
	}
	if !strings.Contains(out, "Failed") {
		t.Errorf("output missing the failure table:\n%s", out)
	}
}

func TestSevenPaceSync_SkipsRunningEntries(t *testing.T) {
	toggl := newAPIStub(t)
	sevenPace := setupSevenPaceTest(t, toggl)

	start := time.Now().Add(-30 * time.Minute)
	toggl.stubHistory(data.TimeEntryItem{
		ID:          1,
		Description: "#1234 still going",
		Duration:    int(-start.Unix()),
		ProjectID:   7,
		WorkspaceID: testWorkspaceID,
		Start:       start.UTC(),
	})

	_, _, err := executeCommand(t, "7pace", "sync", "--dry-run")
	if err == nil {
		t.Fatal("expected an error when every entry is still running")
	}
	if !strings.Contains(err.Error(), "no time entries found") {
		t.Errorf("unexpected error: %v", err)
	}
	if posted := sevenPace.requestsFor(http.MethodPost, sevenPaceWorkLogPath); len(posted) != 0 {
		t.Errorf("running entry posted %d worklog(s), want 0", len(posted))
	}
}

func TestSevenPaceAdd_PostsASingleWorklog(t *testing.T) {
	sevenPace := setupSevenPaceTest(t, nil)
	sevenPace.respond(http.MethodPost, sevenPaceWorkLogPath, http.StatusOK, data.SevenPaceWorkLog{})

	out, _, err := executeCommand(t,
		"7pace", "add",
		"--work-item", "99",
		"--comment", "manual entry",
		"--duration", "1h30m",
		"--date", "2024-03-04 13:15",
	)
	if err != nil {
		t.Fatalf("7pace add: %v", err)
	}

	var posted data.SevenPaceWorkLog
	sevenPace.onlyRequestFor(http.MethodPost, sevenPaceWorkLogPath).decodeBody(t, &posted)

	if posted.WorkItemID == nil || *posted.WorkItemID != 99 {
		t.Errorf("work item id: got %v, want 99", posted.WorkItemID)
	}
	if want := 5400; posted.Length != want {
		t.Errorf("length: got %d, want %d", posted.Length, want)
	}
	if want := "2024-03-04T13:15:00+09:00"; posted.Timestamp != want {
		t.Errorf("timestamp: got %q, want %q", posted.Timestamp, want)
	}
	if !strings.Contains(out, "Posted worklog: 01:30:00 for 2024-03-04 13:15") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestSevenPaceAdd_RequiresWorkItemOrComment(t *testing.T) {
	sevenPace := setupSevenPaceTest(t, nil)

	_, _, err := executeCommand(t, "7pace", "add", "--duration", "30m")
	if err == nil {
		t.Fatal("expected an error without --work-item or --comment")
	}
	if !strings.Contains(err.Error(), "either --work-item or --comment") {
		t.Errorf("unexpected error: %v", err)
	}
	if posted := sevenPace.requestsFor(http.MethodPost, sevenPaceWorkLogPath); len(posted) != 0 {
		t.Errorf("invalid worklog posted %d time(s), want 0", len(posted))
	}
}
