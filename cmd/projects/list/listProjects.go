package list

import (
	"fmt"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
	"log"

	"github.com/spf13/cobra"
)

var ProjectsListCmd = &cobra.Command{
	Use:   "ls",
	Short: "List projects",
	Long:  "List all projects associated with the default workspace",
	Run: func(cmd *cobra.Command, args []string) {
		token, workspaceId := utils.GetTogglConfig()
		client := api.NewAPIClient(token)
		projects, err := client.GetProjects(workspaceId)
		if err != nil {
			log.Fatal("Failed to get projects:", err)
		}

		for _, project := range projects {
			fmt.Printf("ID: %d, Name: %s\n", project.ID, project.Name)
		}
	},
}
