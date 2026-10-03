package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestStatusError_Message(t *testing.T) {
	tests := []struct {
		name string
		err  statusError
		want string
	}{
		{"without body", statusError{Status: "400 Bad Request"}, "request failed: 400 Bad Request"},
		{"with body", statusError{Status: "400 Bad Request", Body: "bad project"}, "request failed: 400 Bad Request: bad project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDoRequest_ErrorIncludesResponseBody(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "workspace not found", http.StatusNotFound)
	}))

	_, err := client.Workspaces(t.Context())

	var statusErr *statusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected *statusError with 404, got %v", err)
	}
	if !strings.Contains(err.Error(), "workspace not found") {
		t.Errorf("error should include the response body, got %q", err)
	}
}

func TestRequests_StopWhenContextIsCancelled(t *testing.T) {
	client := newTestClient(t, jsonHandler(t, []Workspace{}))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := client.Workspaces(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
