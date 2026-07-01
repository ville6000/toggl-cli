package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ville6000/toggl-cli/internal/data"
)

// newTestSevenPaceClient creates a SevenPaceClient pointing at the test server,
// bypassing the NTLM negotiator (which is only exercised against a real server).
func newTestSevenPaceClient(t *testing.T, handler http.Handler) *SevenPaceClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &SevenPaceClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Domain:     "CORP",
		Username:   "user",
		Password:   "secret",
	}
}

func TestCreateWorkLog_PostsToEndpoint(t *testing.T) {
	var capturedMethod, capturedPath, capturedQuery, capturedAuth string
	var capturedBody data.SevenPaceWorkLog
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		capturedAuth = r.Header.Get("Authorization")

		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &capturedBody); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}

		if err := json.NewEncoder(w).Encode(capturedBody); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestSevenPaceClient(t, handler)

	workItem := 1234
	input := data.SevenPaceWorkLog{
		Timestamp:  "2024-01-02T10:00:00Z",
		Length:     3600,
		WorkItemID: &workItem,
		Comment:    "#1234 do stuff",
	}

	got, err := client.CreateWorkLog(input)
	if err != nil {
		t.Fatalf("CreateWorkLog: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", capturedMethod)
	}
	if capturedPath != "/workLogs" {
		t.Errorf("path: got %q, want /workLogs", capturedPath)
	}
	if capturedQuery != "api-version=3.0" {
		t.Errorf("query: got %q, want api-version=3.0", capturedQuery)
	}
	if capturedAuth == "" {
		t.Error("expected Basic auth header to be set for NTLM negotiation")
	}
	if capturedBody.Length != 3600 || capturedBody.WorkItemID == nil || *capturedBody.WorkItemID != 1234 {
		t.Errorf("unexpected body: %+v", capturedBody)
	}
	if got.Comment != "#1234 do stuff" {
		t.Errorf("response comment: got %q", got.Comment)
	}
}

func TestCreateWorkLog_HTTPError(t *testing.T) {
	client := newTestSevenPaceClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	if _, err := client.CreateWorkLog(data.SevenPaceWorkLog{Length: 3600}); err == nil {
		t.Error("expected error for HTTP 400")
	}
}
