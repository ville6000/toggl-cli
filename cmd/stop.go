package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the current timer entry",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		token, workspaceId := utils.GetTogglConfig()
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

		utils.RenderTable("Stopped timer entry", headers, rows, nil)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
