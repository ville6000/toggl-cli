package cmd

import (
	"log"

	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"

	"github.com/spf13/cobra"
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
			log.Println("No current timer entry.")
			return
		}

		stoppedEntry, err := client.StopTimeEntry(workspaceId, currentEntry.ID)
		if err != nil {
			log.Fatal("Failed to stop time entry:", err)
		}

		log.Println("Stopped time entry:", stoppedEntry.Description)
		duration := float64(stoppedEntry.Duration)
		log.Println("Duration:", api.FormatDuration(duration))
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
