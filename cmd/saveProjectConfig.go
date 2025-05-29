package cmd

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/utils"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var saveProjectConfigCmd = &cobra.Command{
	Use:   "saveProjectConfig",
	Short: "Save project path to be used with start command",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		token, workspaceId := utils.GetTogglConfig()
		projectName := args[0]
		currentPath, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current path:", err)
			return
		}

		client := api.NewAPIClient(token)

		var projectId int
		if projectName != "" {
			projectId, err = client.GetProjectIdByName(workspaceId, projectName)
			if err != nil {
				log.Fatal("Failed to get project ID:", err)
			}
		}

		viper.Set(fmt.Sprintf("projects.%s", projectName), projectId)
		viper.Set(fmt.Sprintf("projects.%s.path", projectName), currentPath)
		if err := viper.WriteConfig(); err != nil {
			log.Fatal("Error saving configuration:", err)
		}

		fmt.Println("Configuration saved successfully!")
	},
}

func init() {
	rootCmd.AddCommand(saveProjectConfigCmd)
}
