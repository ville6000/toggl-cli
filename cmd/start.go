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

		timeEntry := api.TimeEntry{
			CreatedWith: "API example code",
			Description: "Hello Toggl",
			Tags:        []string{},
			Billable:    false,
			WorkspaceID: workspaceId,
			Duration:    -1,
			Start:       time.Now().Format(time.RFC3339),
			Stop:        nil,
		}

		client := api.NewAPIClient(token)
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
