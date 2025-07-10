package list

import (
	"fmt"
	"github.com/ville6000/toggl-cli/internal/api"
)

func ProjectListOutput(client api.ProjectService, workspaceId int, printer func(string)) error {
	projects, err := client.GetProjects(workspaceId)
	if err != nil {
		return fmt.Errorf("failed to get projects: %w", err)
	}

	for _, project := range projects {
		printer(fmt.Sprintf("ID: %d, Name: %s\n", project.ID, project.Name))
	}

	return nil
}
