package list

import (
	"fmt"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
)

func ProjectListOutput(client api.ProjectService, workspaceId int) error {
	projects, err := client.GetProjects(workspaceId)
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
	utils.RenderTable("Project list", headers, rows, nil)

	return nil
}
