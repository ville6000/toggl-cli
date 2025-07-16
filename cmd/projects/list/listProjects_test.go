package list

import (
	"errors"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/data"
	"io"
	"os"
	"strings"
	"testing"
)

type mockClient struct {
	api.Client
	Projects []data.Project
	Err      error
}

func (m *mockClient) GetProjects(workspaceId int) ([]data.Project, error) {
	return m.Projects, m.Err
}

func TestProjectListOutput_PrintsCorrectOutput(t *testing.T) {
	mock := &mockClient{
		Projects: []data.Project{
			{ID: 1, Name: "Project A"},
			{ID: 2, Name: "Project B"},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ProjectListOutput(mock, 1234)

	w.Close()
	os.Stdout = oldStdout

	var buf strings.Builder
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "Project A") || !strings.Contains(output, "Project B") {
		t.Errorf("unexpected output: %s", output)
	}

	if !strings.Contains(output, "Project list") || !strings.Contains(output, "ID") || !strings.Contains(output, "PROJECT NAME") {
		t.Errorf("output doesn't contain expected table headers: %s", output)
	}
}

func TestListProjects_ErrorHandling(t *testing.T) {
	mock := &mockClient{
		Err: errors.New("api error"),
	}

	err := ProjectListOutput(mock, 1234)
	if err == nil || !strings.Contains(err.Error(), "failed to get projects") {
		t.Errorf("expected error wrapping 'failed to get projects', got: %v", err)
	}
}
