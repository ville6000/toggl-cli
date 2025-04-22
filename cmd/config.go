package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration settings",
	Long:  "Manage configuration settings for the Toggl CLI.",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Please enter your Toggl API token: ")
		token, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Error reading input", err)
		}
		token = token[:len(token)-1]

		fmt.Print("Please enter your default workspace ID: ")
		var workspaceID int
		_, err = fmt.Scanf("%d", &workspaceID)
		if err != nil {
			log.Fatal("Invalid workspace ID:", err)
			return
		}

		err = writeConfig(token, workspaceID)
		if err != nil {
			fmt.Println("Error saving configuration:", err)
		} else {
			fmt.Println("Configuration saved successfully!")
		}
	},
}

func writeConfig(token string, workspaceID int) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configPath := filepath.Join(home, ".toggl-cli.yaml")
	viper.SetConfigFile(configPath)

	viper.Set("toggl.token", token)
	viper.Set("toggl.workspace_id", workspaceID)

	writeErr := viper.WriteConfig()

	if writeErr == nil {
		return nil
	}

	var configFileNotFoundError viper.ConfigFileNotFoundError
	if errors.As(writeErr, &configFileNotFoundError) {
		err = viper.SafeWriteConfig()

		if err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
}
