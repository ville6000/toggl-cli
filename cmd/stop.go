package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
	"log"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the current timer entry",
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

		client := api.NewAPIClient(token)
		currentEntry, err := client.GetCurrentTimerEntry()
		if err != nil {
			log.Fatal("Failed to get current timer entry:", err)
		}

		if currentEntry == nil || currentEntry.ID == 0 {
			fmt.Println("No current timer entry.")
			return
		}

		stoppedEntry, err := client.StopTimeEntry(workspaceId, currentEntry.ID)
		if err != nil {
			log.Fatal("Failed to stop time entry:", err)
		}

		projectsMap, err := client.GetProjectsLookupMap(workspaceId)
		if err != nil {
			log.Fatal("Failed to get projects", err)
		}

		fmt.Println("Stopped time entry")

		headers := []interface{}{"#", "Started At", "Duration", "Description", "Project"}

		duration := float64(stoppedEntry.Duration)
		projectName := projectsMap[currentEntry.ProjectID]
		rows := [][]interface{}{
			{
				stoppedEntry.ID,
				stoppedEntry.Start.Format("02.01.2006 15:04"),
				api.FormatDuration(duration),
				stoppedEntry.Description,
				projectName,
			},
		}

		utils.RenderTable(headers, rows)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
