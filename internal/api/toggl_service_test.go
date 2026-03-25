package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ville6000/toggl-cli/internal/cache"
	"github.com/ville6000/toggl-cli/internal/data"
)

// newTestClient creates a Client whose BaseURL points at the given test server.
func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		AuthToken:  "test-token",
		Cache:      &cache.CacheService{CacheDir: t.TempDir()},
	}
	return client, server
}

func jsonHandler(t *testing.T, status int, body any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Errorf("jsonHandler encode: %v", err)
			}
		}
	}
}

func errorHandler(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}
}

// ---------- FormatDuration ----------

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{0, "00:00:00"},
		{1, "00:00:01"},
		{59, "00:00:59"},
		{60, "00:01:00"},
		{90, "00:01:30"},
		{3600, "01:00:00"},
		{3661, "01:01:01"},
		{7384, "02:03:04"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := FormatDuration(tt.seconds); got != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.seconds, got, tt.expected)
			}
		})
	}
}

// ---------- NewTimeEntry ----------

func TestNewTimeEntry(t *testing.T) {
	c := &Client{}
	// Truncate to seconds because NewTimeEntry formats Start with RFC3339 (second precision).
	before := time.Now().Truncate(time.Second)
	e := c.NewTimeEntry("my desc", 10, 20, true)
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
	c := &Client{}
	e := c.NewTimeEntry("desc", 1, 2, false)
	if e.Billable {
		t.Error("expected Billable=false")
	}
}

// ---------- Auth header encoding ----------

func TestAuthHeaderEncoding(t *testing.T) {
	var capturedAuth string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		if err := json.NewEncoder(w).Encode([]data.Workspace{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)
	client.AuthToken = "mytoken"

	if _, err := client.GetWorkspaces(); err != nil {
		t.Fatalf("GetWorkspaces: %v", err)
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
		if err := json.NewEncoder(w).Encode([]data.Workspace{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	if _, err := client.GetWorkspaces(); err != nil {
		t.Fatalf("GetWorkspaces: %v", err)
	}

	if capturedContentType != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", capturedContentType)
	}
}

// ---------- GetWorkspaces ----------

func TestGetWorkspaces_Success(t *testing.T) {
	workspaces := []data.Workspace{{ID: 1, Name: "Main"}, {ID: 2, Name: "Side"}}
	client, _ := newTestClient(t, jsonHandler(t, http.StatusOK, workspaces))

	got, err := client.GetWorkspaces()
	if err != nil {
		t.Fatalf("GetWorkspaces: %v", err)
	}
	if len(got) != 2 || got[0].Name != "Main" || got[1].Name != "Side" {
		t.Errorf("unexpected workspaces: %+v", got)
	}
}

func TestGetWorkspaces_HTTPError(t *testing.T) {
	client, _ := newTestClient(t, errorHandler(http.StatusUnauthorized))
	if _, err := client.GetWorkspaces(); err == nil {
		t.Error("expected error for HTTP 401")
	}
}

// ---------- GetCurrentTimerEntry ----------

func TestGetCurrentTimerEntry_Success(t *testing.T) {
	entry := data.TimeEntryItem{ID: 99, Description: "current work"}
	client, _ := newTestClient(t, jsonHandler(t, http.StatusOK, entry))

	got, err := client.GetCurrentTimerEntry()
	if err != nil {
		t.Fatalf("GetCurrentTimerEntry: %v", err)
	}
	if got.ID != 99 || got.Description != "current work" {
		t.Errorf("unexpected entry: %+v", got)
	}
}

func TestGetCurrentTimerEntry_HTTPError(t *testing.T) {
	client, _ := newTestClient(t, errorHandler(http.StatusNotFound))
	if _, err := client.GetCurrentTimerEntry(); err == nil {
		t.Error("expected error for HTTP 404")
	}
}

// ---------- CreateTimeEntry ----------

func TestCreateTimeEntry_Success(t *testing.T) {
	input := data.TimeEntry{Description: "coding", WorkspaceID: 5, Duration: -1}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := json.NewEncoder(w).Encode(input); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	got, err := client.CreateTimeEntry(5, input)
	if err != nil {
		t.Fatalf("CreateTimeEntry: %v", err)
	}
	if got.Description != "coding" {
		t.Errorf("Description: got %q, want %q", got.Description, "coding")
	}
}

func TestCreateTimeEntry_HTTPError(t *testing.T) {
	client, _ := newTestClient(t, errorHandler(http.StatusInternalServerError))
	if _, err := client.CreateTimeEntry(1, data.TimeEntry{}); err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestCreateTimeEntry_URLContainsWorkspaceID(t *testing.T) {
	var capturedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := json.NewEncoder(w).Encode(data.TimeEntry{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	_, _ = client.CreateTimeEntry(42, data.TimeEntry{})
	if !strings.Contains(capturedPath, "42") {
		t.Errorf("URL path %q does not contain workspace ID 42", capturedPath)
	}
}

// ---------- StopTimeEntry ----------

func TestStopTimeEntry_Success(t *testing.T) {
	entry := data.TimeEntryItem{ID: 77}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if err := json.NewEncoder(w).Encode(entry); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	got, err := client.StopTimeEntry(1, 77)
	if err != nil {
		t.Fatalf("StopTimeEntry: %v", err)
	}
	if got.ID != 77 {
		t.Errorf("ID: got %d, want 77", got.ID)
	}
}

func TestStopTimeEntry_HTTPError(t *testing.T) {
	client, _ := newTestClient(t, errorHandler(http.StatusNotFound))
	if _, err := client.StopTimeEntry(1, 99); err == nil {
		t.Error("expected error for HTTP 404")
	}
}

func TestStopTimeEntry_URLContainsIDs(t *testing.T) {
	var capturedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := json.NewEncoder(w).Encode(data.TimeEntryItem{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	_, _ = client.StopTimeEntry(10, 20)
	if !strings.Contains(capturedPath, "10") || !strings.Contains(capturedPath, "20") {
		t.Errorf("URL path %q does not contain workspace/entry IDs", capturedPath)
	}
	if !strings.HasSuffix(capturedPath, "/stop") {
		t.Errorf("URL path %q does not end with /stop", capturedPath)
	}
}

// ---------- GetProjects ----------

func TestGetProjects_FetchesFromAPI(t *testing.T) {
	projects := []data.Project{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}
	client, _ := newTestClient(t, jsonHandler(t, http.StatusOK, projects))

	got, err := client.GetProjects(10)
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if len(got) != 2 || got[0].Name != "Alpha" || got[1].Name != "Beta" {
		t.Errorf("unexpected projects: %+v", got)
	}
}

func TestGetProjects_UsesCache(t *testing.T) {
	callCount := 0
	projects := []data.Project{{ID: 1, Name: "Cached"}}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if err := json.NewEncoder(w).Encode(projects); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	if _, err := client.GetProjects(10); err != nil {
		t.Fatalf("first GetProjects: %v", err)
	}
	if _, err := client.GetProjects(10); err != nil {
		t.Fatalf("second GetProjects: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call (cache hit on second), got %d", callCount)
	}
}

func TestGetProjects_HTTPError(t *testing.T) {
	client, _ := newTestClient(t, errorHandler(http.StatusUnauthorized))
	if _, err := client.GetProjects(10); err == nil {
		t.Error("expected error for HTTP 401")
	}
}

func TestGetProjects_URLContainsWorkspaceID(t *testing.T) {
	var capturedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := json.NewEncoder(w).Encode([]data.Project{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	_, _ = client.GetProjects(99)
	if !strings.Contains(capturedPath, "99") {
		t.Errorf("URL path %q does not contain workspace ID 99", capturedPath)
	}
}

// ---------- GetProjectIdByName ----------

func TestGetProjectIdByName_Found(t *testing.T) {
	projects := []data.Project{{ID: 5, Name: "MyProject"}}
	client, _ := newTestClient(t, jsonHandler(t, http.StatusOK, projects))

	id, err := client.GetProjectIdByName(10, "MyProject")
	if err != nil {
		t.Fatalf("GetProjectIdByName: %v", err)
	}
	if id != 5 {
		t.Errorf("got id %d, want 5", id)
	}
}

func TestGetProjectIdByName_CaseInsensitive(t *testing.T) {
	projects := []data.Project{{ID: 5, Name: "MyProject"}}
	client, _ := newTestClient(t, jsonHandler(t, http.StatusOK, projects))

	id, err := client.GetProjectIdByName(10, "myproject")
	if err != nil {
		t.Fatalf("GetProjectIdByName: %v", err)
	}
	if id != 5 {
		t.Errorf("case-insensitive match: got id %d, want 5", id)
	}
}

func TestGetProjectIdByName_NotFound(t *testing.T) {
	projects := []data.Project{{ID: 5, Name: "MyProject"}}
	client, _ := newTestClient(t, jsonHandler(t, http.StatusOK, projects))

	if _, err := client.GetProjectIdByName(10, "nonexistent"); err == nil {
		t.Error("expected error for missing project")
	}
}

func TestGetProjectIdByName_EmptyProjects(t *testing.T) {
	client, _ := newTestClient(t, jsonHandler(t, http.StatusOK, []data.Project{}))

	if _, err := client.GetProjectIdByName(10, "any"); err == nil {
		t.Error("expected error with empty project list")
	}
}

func TestGetProjectIdByName_APIError(t *testing.T) {
	client, _ := newTestClient(t, errorHandler(http.StatusUnauthorized))
	if _, err := client.GetProjectIdByName(10, "any"); err == nil {
		t.Error("expected error when API fails")
	}
}

// ---------- GetHistory ----------

func TestGetHistory_NoParams(t *testing.T) {
	entries := []data.TimeEntryItem{{ID: 1, Description: "work"}}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}
		if err := json.NewEncoder(w).Encode(entries); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	got, err := client.GetHistory(nil, nil)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Errorf("unexpected entries: %+v", got)
	}
}

func TestGetHistory_WithBothDates(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("start_date") != "2024-01-01" {
			t.Errorf("start_date: got %q, want 2024-01-01", q.Get("start_date"))
		}
		if q.Get("end_date") != "2024-01-31" {
			t.Errorf("end_date: got %q, want 2024-01-31", q.Get("end_date"))
		}
		if err := json.NewEncoder(w).Encode([]data.TimeEntryItem{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	if _, err := client.GetHistory(&from, &to); err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
}

func TestGetHistory_OnlyFromDate(t *testing.T) {
	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("start_date") != "2024-06-01" {
			t.Errorf("start_date: got %q", q.Get("start_date"))
		}
		if q.Get("end_date") != "" {
			t.Errorf("end_date should be absent, got %q", q.Get("end_date"))
		}
		if err := json.NewEncoder(w).Encode([]data.TimeEntryItem{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	if _, err := client.GetHistory(&from, nil); err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
}

func TestGetHistory_OnlyToDate(t *testing.T) {
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("end_date") != "2024-06-30" {
			t.Errorf("end_date: got %q", q.Get("end_date"))
		}
		if q.Get("start_date") != "" {
			t.Errorf("start_date should be absent, got %q", q.Get("start_date"))
		}
		if err := json.NewEncoder(w).Encode([]data.TimeEntryItem{}); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client, _ := newTestClient(t, handler)

	if _, err := client.GetHistory(nil, &to); err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
}

func TestGetHistory_HTTPError(t *testing.T) {
	client, _ := newTestClient(t, errorHandler(http.StatusForbidden))
	if _, err := client.GetHistory(nil, nil); err == nil {
		t.Error("expected error for HTTP 403")
	}
}

// ---------- GetProjectsLookupMap ----------

func TestGetProjectsLookupMap_Success(t *testing.T) {
	projects := []data.Project{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}
	client, _ := newTestClient(t, jsonHandler(t, http.StatusOK, projects))

	lookup, err := client.GetProjectsLookupMap(10)
	if err != nil {
		t.Fatalf("GetProjectsLookupMap: %v", err)
	}
	if lookup[1] != "Alpha" || lookup[2] != "Beta" {
		t.Errorf("unexpected lookup: %+v", lookup)
	}
}

func TestGetProjectsLookupMap_Empty(t *testing.T) {
	client, _ := newTestClient(t, jsonHandler(t, http.StatusOK, []data.Project{}))

	lookup, err := client.GetProjectsLookupMap(10)
	if err != nil {
		t.Fatalf("GetProjectsLookupMap: %v", err)
	}
	if len(lookup) != 0 {
		t.Errorf("expected empty map, got %+v", lookup)
	}
}

func TestGetProjectsLookupMap_APIError(t *testing.T) {
	client, _ := newTestClient(t, errorHandler(http.StatusUnauthorized))
	if _, err := client.GetProjectsLookupMap(10); err == nil {
		t.Error("expected error when API fails")
	}
}
