package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
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

		client := api.NewAPIClient(token)
		currentEntry, err := client.GetCurrentTimerEntry()
		if err != nil {
			log.Println("Failed to get current timer entry:", err)
			return
		}

		if currentEntry != nil {
			duration := time.Since(currentEntry.Start).Seconds()
			formattedDuration := formatDuration(duration)

			fmt.Printf("%d - %s - %s\n", currentEntry.ID, currentEntry.Description, formattedDuration)
		} else {
			fmt.Println("No current timer entry.")
		}
	},
}

func formatDuration(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
