package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
	"log"
	"time"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Get the current timer entry",
	Long:  "Get the current timer entry from Toggl.",
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
		currentEntry, err := client.GetCurrentTimerEntry()
		if err != nil {
			log.Println("Failed to get current timer entry:", err)
			return
		}

		projectsMap, err := client.GetProjectsLookupMap(workspaceId)
		if err != nil {
			log.Fatal("Failed to get projects", err)
		}

		if currentEntry == nil || currentEntry.ID == 0 {
			fmt.Println("No current timer entry.")
			return
		}

		headers := []interface{}{"#", "Started At", "Duration", "Description", "Project"}
		duration := time.Since(currentEntry.Start).Seconds()
		projectName := projectsMap[currentEntry.ProjectID]
		rows := [][]interface{}{
			{currentEntry.ID, currentEntry.Start.Format("02.01.2006 15:04"), api.FormatDuration(duration), currentEntry.Description, projectName},
		}

		utils.RenderTable(headers, rows)
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
