package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sync"
	"testing"

	// The tests pin a timezone, so embed the zone database rather than relying
	// on the host having one installed.
	_ "time/tzdata"

	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/api"
)

const (
	testWorkspaceID = 4242
	// Asia/Tokyo is +09:00 all year, so no DST transition can shift an
	// expected timestamp out from under a test.
	testTimezone = "Asia/Tokyo"
)

// stubRequest is one request a command made against a stub server.
type stubRequest struct {
	Method string
	Path   string
	Query  url.Values
	Body   string
}

// decodeBody unmarshals a recorded JSON request body into v.
func (r stubRequest) decodeBody(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(r.Body), v); err != nil {
		t.Fatalf("decode %s %s body %q: %v", r.Method, r.Path, r.Body, err)
	}
}

// apiStub is a stand-in for a remote HTTP API. Routes are registered per
// "METHOD /path"; a request to any other route fails the test instead of
// quietly returning a 404 that the command would report as some other error.
type apiStub struct {
	t      *testing.T
	server *httptest.Server

	mu       sync.Mutex
	routes   map[string]http.HandlerFunc
	requests []stubRequest
}

func newAPIStub(t *testing.T) *apiStub {
	t.Helper()
	stub := &apiStub{t: t, routes: make(map[string]http.HandlerFunc)}
	stub.server = httptest.NewServer(stub)
	t.Cleanup(stub.server.Close)

	return stub
}

func (s *apiStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.t.Errorf("read request body: %v", err)
	}
	// Recording the body drains it; hand the handler a fresh reader.
	r.Body = io.NopCloser(bytes.NewReader(body))

	s.mu.Lock()
	s.requests = append(s.requests, stubRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.Query(),
		Body:   string(body),
	})
	handler, ok := s.routes[r.Method+" "+r.URL.Path]
	s.mu.Unlock()

	if !ok {
		s.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	handler(w, r)
}

// handle registers a handler for one route.
func (s *apiStub) handle(method, path string, handler http.HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes[method+" "+path] = handler
}

// respond registers a canned JSON response for one route.
func (s *apiStub) respond(method, path string, status int, body any) {
	s.handle(method, path, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body == nil {
			return
		}
		if err := json.NewEncoder(w).Encode(body); err != nil {
			s.t.Errorf("encode %s %s response: %v", method, path, err)
		}
	})
}

// requestsFor returns the recorded requests for one route, in order.
func (s *apiStub) requestsFor(method, path string) []stubRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	var matching []stubRequest
	for _, req := range s.requests {
		if req.Method == method && req.Path == path {
			matching = append(matching, req)
		}
	}

	return matching
}

// onlyRequestFor returns the single recorded request for a route, failing the
// test when the command made none or more than one.
func (s *apiStub) onlyRequestFor(method, path string) stubRequest {
	s.t.Helper()
	matching := s.requestsFor(method, path)
	if len(matching) != 1 {
		s.t.Fatalf("%s %s: got %d requests, want 1", method, path, len(matching))
	}

	return matching[0]
}

// stubHistory serves the entry list returned by GET /me/time_entries.
func (s *apiStub) stubHistory(entries ...api.TimeEntryItem) {
	s.respond(http.MethodGet, "/me/time_entries", http.StatusOK, entries)
}

// stubProjects serves the project list the commands use to resolve names.
func (s *apiStub) stubProjects(projects ...api.Project) {
	s.respond(http.MethodGet, fmt.Sprintf("/workspaces/%d/projects", testWorkspaceID), http.StatusOK, projects)
}

// setupCLITest isolates a command run from the developer's real config, cache
// and timezone, and returns the configuration to run commands with, pointing
// the Toggl client at stub.
func setupCLITest(t *testing.T, stub *apiStub) *viper.Viper {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))

	v := viper.New()
	v.Set("toggl.token", "test-token")
	v.Set("toggl.workspace_id", testWorkspaceID)
	v.Set("toggl.timezone", testTimezone)
	if stub != nil {
		v.Set("toggl.base_url", stub.server.URL)
	}

	return v
}

// executeCommand runs the CLI the way main() does — through a full command
// tree, so flag parsing, config lookup and output rendering are all exercised
// — and returns what the command wrote to stdout and stderr.
func executeCommand(t *testing.T, v *viper.Viper, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return executeCommandWithInput(t, v, "", args...)
}

// executeCommandWithInput is executeCommand with stdin wired to input, for the
// commands that prompt.
func executeCommandWithInput(t *testing.T, v *viper.Viper, input string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	root := newRootCmd(v)

	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetIn(bytes.NewBufferString(input))
	root.SetArgs(args)

	err = root.ExecuteContext(t.Context())

	return outBuf.String(), errBuf.String(), err
}
