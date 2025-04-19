package cmd

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
	"log"

	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects",
	Long:  "List all projects associated with the default workspace",
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("toggl.token")
		if token == "" {
			log.Fatal("Missing toggl.token in config file")
		}

		workspaceId := viper.GetInt("toggl.workspace_id")
		if workspaceId == 0 {
			log.Fatal("Missing toggl.workspace_id in config file")
		}

		client := api.NewAPIClient(token)
		projects, err := client.GetProjects(workspaceId)
		if err != nil {
			log.Println("Failed to get projects:", err)
			return
		}

		for _, project := range projects {
			fmt.Printf("ID: %d, Name: %s\n", project.ID, project.Name)
		}
	},
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}
