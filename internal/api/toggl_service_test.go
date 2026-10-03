package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient creates a Client whose BaseURL points at the given test server.
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		AuthToken:  "test-token",
		Cache:      &ProjectCache{CacheDir: t.TempDir()},
	}
	return client
}

// jsonHandler responds 200 OK with body encoded as JSON.
func jsonHandler(t *testing.T, body any) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if body != nil {
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Errorf("jsonHandler encode: %v", err)
			}
		}
	}
}

func errorHandler(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}
}

// ---------- NewTimeEntry ----------

func TestNewTimeEntry(t *testing.T) {
	// Truncate to seconds because NewTimeEntry formats Start with RFC3339 (second precision).
	before := time.Now().Truncate(time.Second)
	e := NewTimeEntry("my desc", 10, 20, true)
	after := time.Now().Add(time.Second).Truncate(time.Second)

	if e.Description != "my desc" {
		t.Errorf("Description: got %q", e.Description)
	}
	if e.WorkspaceID != 10 {
		t.Errorf("WorkspaceID: got %d", e.WorkspaceID)
	}
	if e.ProjectID != 20 {
		t.Errorf("ProjectID: got %d", e.ProjectID)
	}
	if !e.Billable {
		t.Error("expected Billable=true")
	}
	if e.Duration != -1 {
		t.Errorf("Duration: got %d, want -1", e.Duration)
	}
	if e.CreatedWith != "toggl-cli" {
		t.Errorf("CreatedWith: got %q", e.CreatedWith)
	}
	if e.Stop != nil {
		t.Errorf("Stop: expected nil, got %v", e.Stop)
	}
	if len(e.Tags) != 0 {
		t.Errorf("Tags: expected empty slice, got %v", e.Tags)
	}

	start, err := time.Parse(time.RFC3339, e.Start)
	if err != nil {
		t.Fatalf("Start not valid RFC3339: %v", err)
	}
	if start.Before(before) || start.After(after) {
		t.Errorf("Start %v outside expected range [%v, %v]", start, before, after)
	}
}

func TestNewTimeEntry_NotBillable(t *testing.T) {
	e := NewTimeEntry("desc", 1, 2, false)
	if e.Billable {
		t.Error("expected Billable=false")
	}
}

// ---------- Auth header encoding ----------

func TestAuthHeaderEncoding(t *testing.T) {
	var capturedAuth string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		if err := json.NewEncoder(w).Encode([]Workspace{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)
	client.AuthToken = "mytoken"

	if _, err := client.Workspaces(t.Context()); err != nil {
		t.Fatalf("Workspaces: %v", err)
	}

	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("mytoken:api_token"))
	if capturedAuth != expected {
		t.Errorf("Authorization header: got %q, want %q", capturedAuth, expected)
	}
}

func TestContentTypeHeader(t *testing.T) {
	var capturedContentType string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		if err := json.NewEncoder(w).Encode([]Workspace{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	if _, err := client.Workspaces(t.Context()); err != nil {
		t.Fatalf("Workspaces: %v", err)
	}

	if capturedContentType != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", capturedContentType)
	}
}

// ---------- Workspaces ----------

func TestWorkspaces_Success(t *testing.T) {
	workspaces := []Workspace{{ID: 1, Name: "Main"}, {ID: 2, Name: "Side"}}
	client := newTestClient(t, jsonHandler(t, workspaces))

	got, err := client.Workspaces(t.Context())
	if err != nil {
		t.Fatalf("Workspaces: %v", err)
	}
	if len(got) != 2 || got[0].Name != "Main" || got[1].Name != "Side" {
		t.Errorf("unexpected workspaces: %+v", got)
	}
}

func TestWorkspaces_HTTPError(t *testing.T) {
	client := newTestClient(t, errorHandler(http.StatusUnauthorized))
	if _, err := client.Workspaces(t.Context()); err == nil {
		t.Error("expected error for HTTP 401")
	}
}

// ---------- CurrentTimeEntry ----------

func TestCurrentTimeEntry_Success(t *testing.T) {
	entry := TimeEntryItem{ID: 99, Description: "current work"}
	client := newTestClient(t, jsonHandler(t, entry))

	got, err := client.CurrentTimeEntry(t.Context())
	if err != nil {
		t.Fatalf("CurrentTimeEntry: %v", err)
	}
	if got.ID != 99 || got.Description != "current work" {
		t.Errorf("unexpected entry: %+v", got)
	}
}

func TestCurrentTimeEntry_HTTPError(t *testing.T) {
	client := newTestClient(t, errorHandler(http.StatusNotFound))
	if _, err := client.CurrentTimeEntry(t.Context()); err == nil {
		t.Error("expected error for HTTP 404")
	}
}

// ---------- CreateTimeEntry ----------

func TestTimeEntry_FetchesByID(t *testing.T) {
	var gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewEncoder(w).Encode(TimeEntryItem{ID: 4123, Description: "old"}); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))

	entry, err := client.TimeEntry(t.Context(), 4123)
	if err != nil {
		t.Fatalf("TimeEntry: %v", err)
	}
	if gotPath != "/me/time_entries/4123" {
		t.Errorf("path: got %q", gotPath)
	}
	if entry.ID != 4123 || entry.Description != "old" {
		t.Errorf("got %+v", entry)
	}
}

func TestTimeEntry_HTTPError(t *testing.T) {
	client := newTestClient(t, errorHandler(http.StatusNotFound))

	if _, err := client.TimeEntry(t.Context(), 1); err == nil {
		t.Error("expected error for HTTP 404")
	}
}

func TestCreateTimeEntry_Success(t *testing.T) {
	input := TimeEntry{Description: "coding", WorkspaceID: 5, Duration: -1}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := json.NewEncoder(w).Encode(input); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	got, err := client.CreateTimeEntry(t.Context(), 5, input)
	if err != nil {
		t.Fatalf("CreateTimeEntry: %v", err)
	}
	if got.Description != "coding" {
		t.Errorf("Description: got %q, want %q", got.Description, "coding")
	}
}

func TestCreateTimeEntry_HTTPError(t *testing.T) {
	client := newTestClient(t, errorHandler(http.StatusInternalServerError))
	if _, err := client.CreateTimeEntry(t.Context(), 1, TimeEntry{}); err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestCreateTimeEntry_URLContainsWorkspaceID(t *testing.T) {
	var capturedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := json.NewEncoder(w).Encode(TimeEntry{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	_, _ = client.CreateTimeEntry(t.Context(), 42, TimeEntry{})
	if !strings.Contains(capturedPath, "42") {
		t.Errorf("URL path %q does not contain workspace ID 42", capturedPath)
	}
}

// ---------- StopTimeEntry ----------

func TestStopTimeEntry_Success(t *testing.T) {
	entry := TimeEntryItem{ID: 77}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if err := json.NewEncoder(w).Encode(entry); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	got, err := client.StopTimeEntry(t.Context(), 1, 77)
	if err != nil {
		t.Fatalf("StopTimeEntry: %v", err)
	}
	if got.ID != 77 {
		t.Errorf("ID: got %d, want 77", got.ID)
	}
}

func TestStopTimeEntry_HTTPError(t *testing.T) {
	client := newTestClient(t, errorHandler(http.StatusNotFound))
	if _, err := client.StopTimeEntry(t.Context(), 1, 99); err == nil {
		t.Error("expected error for HTTP 404")
	}
}

func TestStopTimeEntry_URLContainsIDs(t *testing.T) {
	var capturedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := json.NewEncoder(w).Encode(TimeEntryItem{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	_, _ = client.StopTimeEntry(t.Context(), 10, 20)
	if !strings.Contains(capturedPath, "10") || !strings.Contains(capturedPath, "20") {
		t.Errorf("URL path %q does not contain workspace/entry IDs", capturedPath)
	}
	if !strings.HasSuffix(capturedPath, "/stop") {
		t.Errorf("URL path %q does not end with /stop", capturedPath)
	}
}

// ---------- UpdateTimeEntry ----------

func TestUpdateTimeEntry_Success(t *testing.T) {
	result := TimeEntryItem{ID: 5, Description: "updated"}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if err := json.NewEncoder(w).Encode(result); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	got, err := client.UpdateTimeEntry(t.Context(), 1, 5, TimeEntry{Description: "updated"})
	if err != nil {
		t.Fatalf("UpdateTimeEntry: %v", err)
	}
	if got.Description != "updated" {
		t.Errorf("Description: got %q, want %q", got.Description, "updated")
	}
}

func TestUpdateTimeEntry_URLContainsIDs(t *testing.T) {
	var capturedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := json.NewEncoder(w).Encode(TimeEntryItem{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	_, _ = client.UpdateTimeEntry(t.Context(), 10, 20, TimeEntry{})
	const wantPath = "/workspaces/10/time_entries/20"
	if capturedPath != wantPath {
		t.Errorf("URL path: got %q, want %q", capturedPath, wantPath)
	}
}

func TestUpdateTimeEntry_HTTPError(t *testing.T) {
	client := newTestClient(t, errorHandler(http.StatusNotFound))
	if _, err := client.UpdateTimeEntry(t.Context(), 1, 99, TimeEntry{}); err == nil {
		t.Error("expected error for HTTP 404")
	}
}

// ---------- Projects ----------

func TestProjects_FetchesFromAPI(t *testing.T) {
	projects := []Project{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}
	client := newTestClient(t, jsonHandler(t, projects))

	got, err := client.Projects(t.Context(), 10)
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(got) != 2 || got[0].Name != "Alpha" || got[1].Name != "Beta" {
		t.Errorf("unexpected projects: %+v", got)
	}
}

func TestProjects_UsesCache(t *testing.T) {
	callCount := 0
	projects := []Project{{ID: 1, Name: "Cached"}}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if err := json.NewEncoder(w).Encode(projects); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	if _, err := client.Projects(t.Context(), 10); err != nil {
		t.Fatalf("first Projects: %v", err)
	}
	if _, err := client.Projects(t.Context(), 10); err != nil {
		t.Fatalf("second Projects: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call (cache hit on second), got %d", callCount)
	}
}

func TestProjects_HTTPError(t *testing.T) {
	client := newTestClient(t, errorHandler(http.StatusUnauthorized))
	if _, err := client.Projects(t.Context(), 10); err == nil {
		t.Error("expected error for HTTP 401")
	}
}

func TestProjects_URLContainsWorkspaceID(t *testing.T) {
	var capturedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := json.NewEncoder(w).Encode([]Project{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	_, _ = client.Projects(t.Context(), 99)
	if !strings.Contains(capturedPath, "99") {
		t.Errorf("URL path %q does not contain workspace ID 99", capturedPath)
	}
}

// ---------- ProjectIDByName ----------

func TestProjectIDByName_Found(t *testing.T) {
	projects := []Project{{ID: 5, Name: "MyProject"}}
	client := newTestClient(t, jsonHandler(t, projects))

	id, err := client.ProjectIDByName(t.Context(), 10, "MyProject")
	if err != nil {
		t.Fatalf("ProjectIDByName: %v", err)
	}
	if id != 5 {
		t.Errorf("got id %d, want 5", id)
	}
}

func TestProjectIDByName_CaseInsensitive(t *testing.T) {
	projects := []Project{{ID: 5, Name: "MyProject"}}
	client := newTestClient(t, jsonHandler(t, projects))

	id, err := client.ProjectIDByName(t.Context(), 10, "myproject")
	if err != nil {
		t.Fatalf("ProjectIDByName: %v", err)
	}
	if id != 5 {
		t.Errorf("case-insensitive match: got id %d, want 5", id)
	}
}

func TestProjectIDByName_NotFound(t *testing.T) {
	projects := []Project{{ID: 5, Name: "MyProject"}}
	client := newTestClient(t, jsonHandler(t, projects))

	if _, err := client.ProjectIDByName(t.Context(), 10, "nonexistent"); err == nil {
		t.Error("expected error for missing project")
	}
}

func TestProjectIDByName_EmptyProjects(t *testing.T) {
	client := newTestClient(t, jsonHandler(t, []Project{}))

	if _, err := client.ProjectIDByName(t.Context(), 10, "any"); err == nil {
		t.Error("expected error with empty project list")
	}
}

func TestProjectIDByName_APIError(t *testing.T) {
	client := newTestClient(t, errorHandler(http.StatusUnauthorized))
	if _, err := client.ProjectIDByName(t.Context(), 10, "any"); err == nil {
		t.Error("expected error when API fails")
	}
}

// ---------- TimeEntries ----------

func TestTimeEntries_NoParams(t *testing.T) {
	entries := []TimeEntryItem{{ID: 1, Description: "work"}}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}
		if err := json.NewEncoder(w).Encode(entries); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	got, err := client.TimeEntries(t.Context(), nil, nil)
	if err != nil {
		t.Fatalf("TimeEntries: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Errorf("unexpected entries: %+v", got)
	}
}

func TestTimeEntries_WithBothDates(t *testing.T) {
	loc := time.FixedZone("EET", 2*60*60)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, loc)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("start_date") != "2024-01-01T00:00:00+02:00" {
			t.Errorf("start_date: got %q, want 2024-01-01T00:00:00+02:00", q.Get("start_date"))
		}
		if q.Get("end_date") != "2024-01-31T00:00:00+02:00" {
			t.Errorf("end_date: got %q, want 2024-01-31T00:00:00+02:00", q.Get("end_date"))
		}
		if err := json.NewEncoder(w).Encode([]TimeEntryItem{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	if _, err := client.TimeEntries(t.Context(), &from, &to); err != nil {
		t.Fatalf("TimeEntries: %v", err)
	}
}

func TestTimeEntries_OnlyFromDate(t *testing.T) {
	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("start_date") != "2024-06-01T00:00:00Z" {
			t.Errorf("start_date: got %q", q.Get("start_date"))
		}
		if q.Get("end_date") != "" {
			t.Errorf("end_date should be absent, got %q", q.Get("end_date"))
		}
		if err := json.NewEncoder(w).Encode([]TimeEntryItem{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	if _, err := client.TimeEntries(t.Context(), &from, nil); err != nil {
		t.Fatalf("TimeEntries: %v", err)
	}
}

func TestTimeEntries_OnlyToDate(t *testing.T) {
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("end_date") != "2024-06-30T00:00:00Z" {
			t.Errorf("end_date: got %q", q.Get("end_date"))
		}
		if q.Get("start_date") != "" {
			t.Errorf("start_date should be absent, got %q", q.Get("start_date"))
		}
		if err := json.NewEncoder(w).Encode([]TimeEntryItem{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	if _, err := client.TimeEntries(t.Context(), nil, &to); err != nil {
		t.Fatalf("TimeEntries: %v", err)
	}
}

func TestTimeEntries_HTTPError(t *testing.T) {
	client := newTestClient(t, errorHandler(http.StatusForbidden))
	if _, err := client.TimeEntries(t.Context(), nil, nil); err == nil {
		t.Error("expected error for HTTP 403")
	}
}

// ---------- ProjectNames ----------

func TestProjectNames_Success(t *testing.T) {
	projects := []Project{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}
	client := newTestClient(t, jsonHandler(t, projects))

	lookup, err := client.ProjectNames(t.Context(), 10)
	if err != nil {
		t.Fatalf("ProjectNames: %v", err)
	}
	if lookup[1] != "Alpha" || lookup[2] != "Beta" {
		t.Errorf("unexpected lookup: %+v", lookup)
	}
}

func TestProjectNames_Empty(t *testing.T) {
	client := newTestClient(t, jsonHandler(t, []Project{}))

	lookup, err := client.ProjectNames(t.Context(), 10)
	if err != nil {
		t.Fatalf("ProjectNames: %v", err)
	}
	if len(lookup) != 0 {
		t.Errorf("expected empty map, got %+v", lookup)
	}
}

func TestProjectNames_APIError(t *testing.T) {
	client := newTestClient(t, errorHandler(http.StatusUnauthorized))
	if _, err := client.ProjectNames(t.Context(), 10); err == nil {
		t.Error("expected error when API fails")
	}
}
