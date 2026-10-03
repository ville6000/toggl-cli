package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/ville6000/toggl-cli/internal/api"
)

type mockProjectService struct {
	List      []api.Project
	Fresh     []api.Project
	Err       error
	refreshed bool
}

func (m *mockProjectService) Projects(_ context.Context, _ int) ([]api.Project, error) {
	return m.List, m.Err
}

func (m *mockProjectService) RefreshProjects(_ context.Context, _ int) ([]api.Project, error) {
	m.refreshed = true
	return m.Fresh, m.Err
}

func TestProjectListOutput_PrintsCorrectOutput(t *testing.T) {
	mock := &mockProjectService{
		List: []api.Project{
			{ID: 1, Name: "Project A"},
			{ID: 2, Name: "Project B"},
		},
	}

	var buf bytes.Buffer
	listErr := projectListOutput(t.Context(), &buf, mock, 1234, false)
	output := buf.String()

	if listErr != nil {
		t.Fatalf("expected no error, got: %v", listErr)
	}

	if !strings.Contains(output, "Project A") || !strings.Contains(output, "Project B") {
		t.Errorf("unexpected output: %s", output)
	}

	if !strings.Contains(output, "Project list") || !strings.Contains(output, "ID") || !strings.Contains(output, "PROJECT NAME") {
		t.Errorf("output doesn't contain expected table headers: %s", output)
	}
}

func TestProjectListOutput_ErrorHandling(t *testing.T) {
	mock := &mockProjectService{
		Err: errors.New("api error"),
	}

	err := projectListOutput(t.Context(), io.Discard, mock, 1234, false)
	if err == nil || !strings.Contains(err.Error(), "failed to get projects") {
		t.Errorf("expected error wrapping 'failed to get projects', got: %v", err)
	}
}

func TestProjectListOutput_RefreshBypassesTheCache(t *testing.T) {
	mock := &mockProjectService{
		List:  []api.Project{{ID: 1, Name: "Cached"}},
		Fresh: []api.Project{{ID: 1, Name: "Cached"}, {ID: 2, Name: "Brand New"}},
	}

	var buf bytes.Buffer
	if err := projectListOutput(t.Context(), &buf, mock, 1234, true); err != nil {
		t.Fatalf("projectListOutput: %v", err)
	}

	if !mock.refreshed {
		t.Error("--refresh did not fetch fresh projects")
	}
	if !strings.Contains(buf.String(), "Brand New") {
		t.Errorf("output missing the freshly fetched project:\n%s", buf.String())
	}
}
