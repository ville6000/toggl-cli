package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func newConfigCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration settings",
		Long:  "Manage configuration settings for the Toggl CLI.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			in := cmd.InOrStdin()
			reader := bufio.NewReader(in)

			fmt.Fprint(out, "Please enter your Toggl API token: ")
			token, err := readSecret(out, in, reader)
			if err != nil {
				return err
			}

			fmt.Fprint(out, "Please enter your default workspace ID: ")
			wsLine, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}
			var workspaceID int
			if _, err = fmt.Sscanf(strings.TrimSpace(wsLine), "%d", &workspaceID); err != nil {
				return fmt.Errorf("invalid workspace ID: %w", err)
			}

			fmt.Fprintf(out, "Please enter your timezone (leave empty for system default %q): ", time.Now().Location().String())
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

			if err = writeConfig(v, token, workspaceID, tz); err != nil {
				return fmt.Errorf("error saving configuration: %w", err)
			}

			fmt.Fprintln(out, "Configuration saved successfully!")
			return nil
		},
	}

	return cmd
}

// Terminal access used by readSecret, replaced in tests.
var (
	isTerminal   = term.IsTerminal
	readPassword = term.ReadPassword
)

// readSecret reads a line without echoing it when in is a terminal, so a token
// or password doesn't end up on screen or in scrollback. Otherwise (piped
// input, tests) it reads the next line from reader like any other prompt.
func readSecret(out io.Writer, in io.Reader, reader *bufio.Reader) (string, error) {
	if f, ok := in.(*os.File); ok && isTerminal(int(f.Fd())) {
		secret, err := readPassword(int(f.Fd()))
		// The Enter that ended the input wasn't echoed either.
		fmt.Fprintln(out)
		if err != nil {
			return "", fmt.Errorf("error reading input: %w", err)
		}
		return strings.TrimSpace(string(secret)), nil
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func writeConfig(v *viper.Viper, token string, workspaceID int, timezone string) error {
	configPath, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// The directory does not exist yet on a fresh install, and viper only
	// creates the file. Keep it private: the file holds an API token.
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	v.SetConfigFile(configPath)

	v.Set("toggl.token", token)
	v.Set("toggl.workspace_id", workspaceID)
	v.Set("toggl.timezone", timezone)

	writeErr := v.WriteConfig()

	if writeErr != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](writeErr); !ok {
			return writeErr
		}

		if err := v.SafeWriteConfig(); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}

	return restrictConfigFile(configPath)
}

// restrictConfigFile makes the config file readable by its owner only. It holds
// an API token, and viper writes files with the process umask, which can leave
// them world-readable.
func restrictConfigFile(path string) error {
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("failed to restrict config file permissions: %w", err)
	}

	return nil
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
