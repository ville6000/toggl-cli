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
			formattedDuration := api.FormatDuration(duration)

			fmt.Printf("%d - %s - %s\n", currentEntry.ID, currentEntry.Description, formattedDuration)
		} else {
			fmt.Println("No current timer entry.")
		}
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
