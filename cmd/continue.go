package cmd

import (
	"fmt"
	"log"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"

	"github.com/spf13/cobra"
)

var continueCmd = &cobra.Command{
	Use:   "continue",
	Short: "Continue latest timer entry",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		token, workspaceId := utils.GetTogglConfig()
		client := api.NewAPIClient(token)
		timeEntries, err := client.GetHistory(nil, nil)
		if err != nil {
			log.Fatal("Failed to retrieve latest time entries:", err)
		}

		if len(timeEntries) == 0 {
			fmt.Println("No time entries found.")
			return
		}

		e := timeEntries[0]
		timeEntry := client.NewTimeEntry(e.Description,
			workspaceId,
			e.ProjectID,
			e.Billable,
		)
		_, err = client.CreateTimeEntry(workspaceId, timeEntry)
		if err != nil {
			log.Fatal("Failed to create time entry:", err)
		}

		fmt.Println("Continuing timer for:", e.Description)
	},
}

func init() {
	rootCmd.AddCommand(continueCmd)
}
