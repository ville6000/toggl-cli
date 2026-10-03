package list

import (
	"fmt"
	"io"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/output"
)

func ProjectListOutput(out io.Writer, client api.ProjectService, workspaceID int) error {
	projects, err := client.GetProjects(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get projects: %w", err)
	}

	var rows [][]interface{}
	for _, project := range projects {
		rows = append(rows, []interface{}{
			project.ID,
			project.Name,
		})
	}

	headers := []interface{}{"ID", "Project Name"}
	output.RenderTable(out, "Project list", headers, rows, nil)

	return nil
}
