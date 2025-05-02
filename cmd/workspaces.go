package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
)

var workspacesCmd = &cobra.Command{
	Use:   "workspaces",
	Short: "List workspaces",
	Long:  "List all workspaces associated with the Toggl account.",
	Run: func(cmd *cobra.Command, args []string) {
		token := viper.GetString("toggl.token")
		if token == "" {
			log.Fatal("Missing toggl.token in config file")
		}

		client := api.NewAPIClient(token)
		workspaces, err := client.GetWorkspaces()
		if err != nil {
			log.Println("Failed to get workspaces:", err)
		}

		for _, workspace := range workspaces {
			fmt.Printf("ID: %d, Name: %s\n", workspace.ID, workspace.Name)
		}
	},
}

func init() {
	rootCmd.AddCommand(workspacesCmd)
}
