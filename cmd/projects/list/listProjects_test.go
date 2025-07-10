package list

import (
	"errors"
	"github.com/ville6000/toggl-cli/internal/api"
	"strings"
	"testing"
)

type mockClient struct {
	api.Client
	Projects []api.Project
	Err      error
}

func (m *mockClient) GetProjects(workspaceId int) ([]api.Project, error) {
	return m.Projects, m.Err
}

func TestProjectListOutput_PrintsCorrectOutput(t *testing.T) {
	mock := &mockClient{
		Projects: []api.Project{
			{ID: 1, Name: "Project A"},
			{ID: 2, Name: "Project B"},
		},
	}

	var output strings.Builder
	err := ProjectListOutput(mock, 1234, func(s string) {
		output.WriteString(s)
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got := output.String()
	if !strings.Contains(got, "Project A") || !strings.Contains(got, "Project B") {
		t.Errorf("unexpected output: %s", got)
	}
}

func TestListProjects_ErrorHandling(t *testing.T) {
	mock := &mockClient{
		Err: errors.New("api error"),
	}

	err := ProjectListOutput(mock, 1234, func(s string) {})
	if err == nil || !strings.Contains(err.Error(), "failed to get projects") {
		t.Errorf("expected error wrapping 'failed to get projects', got: %v", err)
	}
}
