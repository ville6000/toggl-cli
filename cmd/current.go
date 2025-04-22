package cmd

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
	"log"
	"os"
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

		projectsMap := client.GetProjectsLookupMap(workspaceId)

		if currentEntry == nil || currentEntry.ID == 0 {
			fmt.Println("No current timer entry.")
			return
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"#", "Started At", "Duration", "Description", "Project"})

		duration := time.Since(currentEntry.Start).Seconds()
		formattedDuration := api.FormatDuration(duration)
		projectName := projectsMap[currentEntry.ProjectID]

		t.AppendRow([]interface{}{currentEntry.ID, currentEntry.Start.Format("02.01.2006 15:04"), formattedDuration, currentEntry.Description, projectName})
		t.Render()
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
