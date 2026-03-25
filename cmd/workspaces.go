package cmd

import (
	"fmt"

	"github.com/ville6000/toggl-cli/internal/utils"

	"github.com/spf13/cobra"
	"github.com/ville6000/toggl-cli/internal/api"
)

var workspacesCmd = &cobra.Command{
	Use:   "workspaces",
	Short: "List workspaces",
	Long:  "List all workspaces associated with the Toggl account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := utils.GetToken()
		if err != nil {
			return fmt.Errorf("failed to get API token: %w", err)
		}

		client := api.NewAPIClient(token)
		workspaces, err := client.GetWorkspaces()
		if err != nil {
			return fmt.Errorf("failed to get workspaces: %w", err)
		}

		for _, workspace := range workspaces {
			fmt.Printf("ID: %d, Name: %s\n", workspace.ID, workspace.Name)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(workspacesCmd)
}
