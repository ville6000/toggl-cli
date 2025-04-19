package cmd

import (
	"fmt"
	"github.com/ville6000/toggl-cli/internal/api"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new time entry",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("toggl.token")
		if token == "" {
			log.Fatal("Missing toggl.token in config file")
		}

		workspaceId := viper.GetInt("toggl.workspace_id")
		if workspaceId == 0 {
			log.Fatal("Missing toggl.workspace_id in config file")
		}

		description, err := cmd.Flags().GetString("description")
		if err != nil {
			log.Fatal("Error retrieving description flag:", err)
		}

		projectName, err := cmd.Flags().GetString("project")
		if err != nil {
			log.Fatal("Error retrieving project flag:", err)
		}

		client := api.NewAPIClient(token)

		var projectId int
		if projectName != "" {
			projectId, err = client.GetProjectIdByName(workspaceId, projectName)

			if err != nil {
				log.Println("Failed to get project ID:", err)
				return
			}
		}

		timeEntry := api.TimeEntry{
			CreatedWith: "API example code",
			Description: description,
			Tags:        []string{},
			Billable:    false,
			WorkspaceID: workspaceId,
			Duration:    -1,
			Start:       time.Now().Format(time.RFC3339),
			Stop:        nil,
			ProjectID:   projectId,
		}

		createdEntry, err := client.CreateTimeEntry(workspaceId, timeEntry)
		if err != nil {
			log.Println("Failed to create time entry:", err)
			return
		}

		fmt.Printf("Created time entry: %+v\n", createdEntry)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringP("description", "d", "", "Description for the time entry")
	startCmd.Flags().StringP("project", "p", "", "Project for the time entry")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
