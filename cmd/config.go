package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration settings",
	Long:  "Manage configuration settings for the Toggl CLI.",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Please enter your Toggl API token: ")
		token, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}
		token = strings.TrimSpace(token)

		fmt.Print("Please enter your default workspace ID: ")
		wsLine, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}
		var workspaceID int
		if _, err = fmt.Sscanf(strings.TrimSpace(wsLine), "%d", &workspaceID); err != nil {
			return fmt.Errorf("invalid workspace ID: %w", err)
		}

		fmt.Printf("Please enter your timezone (leave empty for system default %q): ", time.Local)
		tz, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}
		tz = strings.TrimSpace(tz)

		if tz != "" {
			if _, err := time.LoadLocation(tz); err != nil {
				return fmt.Errorf("invalid timezone %q: %w", tz, err)
			}
		}

		if err = writeConfig(token, workspaceID, tz); err != nil {
			return fmt.Errorf("error saving configuration: %w", err)
		}

		fmt.Println("Configuration saved successfully!")
		return nil
	},
}

func writeConfig(token string, workspaceID int, timezone string) error {
	configPath, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}
	viper.SetConfigFile(configPath)

	viper.Set("toggl.token", token)
	viper.Set("toggl.workspace_id", workspaceID)
	viper.Set("toggl.timezone", timezone)

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

// ConfigPath returns the config file to use: the first existing candidate,
// or the preferred XDG path for new installs.
func ConfigPath() (string, error) {
	candidates, err := configCandidates()
	if err != nil {
		return "", err
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return candidates[0], nil
}

// configCandidates returns config file paths in priority order:
// XDG_CONFIG_HOME, then ~/.config/toggl-cli, then ~/.toggl-cli.yaml.
func configCandidates() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	candidates := []string{}

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "toggl-cli", "config.yaml"))
	}

	candidates = append(candidates,
		filepath.Join(home, ".config", "toggl-cli", "config.yaml"),
		filepath.Join(home, ".toggl-cli.yaml"),
	)

	return candidates, nil
}

func init() {
	rootCmd.AddCommand(configCmd)
}
