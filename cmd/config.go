package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		token = strings.TrimSpace(token)

		fmt.Print("Please enter your default workspace ID: ")
		var workspaceID int
		_, err = fmt.Scanf("%d", &workspaceID)
		if err != nil {
			log.Fatal("Invalid workspace ID:", err)
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
	configPath, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}
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
		} else {
			return nil
		}
	}

	return writeErr
}

// ConfigPath returns the path to the configuration file.
// It checks for the XDG_CONFIG_HOME environment variable first,
// then checks for the $HOME/.config directory, and finally defaults to $HOME/.toggl-cli.yaml.
// It returns an error if the home directory cannot be determined.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check if XDG_CONFIG_HOME is set and use it if available
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "toggl-cli", "config.yaml"), nil
	}

	// Check if $HOME/.config directory exists. and use it if available
	if homeConfig, _ := os.Stat(filepath.Join(home, ".config")); homeConfig != nil && homeConfig.IsDir() {
		return filepath.Join(home, ".config", "toggl-cli", "config.yaml"), nil
	}

	return filepath.Join(home, ".toggle-cli.yaml"), nil
}

func init() {
	rootCmd.AddCommand(configCmd)
}
