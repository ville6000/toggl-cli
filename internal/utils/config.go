package utils

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

func GetToken() (string, error) {
	token := viper.GetString("toggl.token")
	if token == "" {
		return "", fmt.Errorf("missing toggl.token in config, please run 'toggl-cli config'")
	}

	return token, nil
}

func GetConfig() (string, int, error) {
	token := viper.GetString("toggl.token")
	if token == "" {
		return "", 0, fmt.Errorf("missing toggl.token in config, please run 'toggl-cli config'")
	}

	workspaceId := viper.GetInt("toggl.workspace_id")
	if workspaceId == 0 {
		return "", 0, fmt.Errorf("missing toggl.workspace_id in config, please run 'toggl-cli config'")
	}

	return token, workspaceId, nil
}

// SevenPaceConfig holds the settings for talking to an on-prem 7pace
// Timetracker instance using NTLM (Windows) authentication.
type SevenPaceConfig struct {
	BaseURL         string
	Domain          string
	Username        string
	Password        string
	ActivityTypeID  string
	InsecureSkipTLS bool
}

// GetSevenPaceConfig reads the 7pace configuration from viper. Base URL,
// username and password are required; domain and activity type are optional.
func GetSevenPaceConfig() (SevenPaceConfig, error) {
	cfg := SevenPaceConfig{
		BaseURL:         viper.GetString("sevenpace.base_url"),
		Domain:          viper.GetString("sevenpace.domain"),
		Username:        viper.GetString("sevenpace.username"),
		Password:        viper.GetString("sevenpace.password"),
		ActivityTypeID:  viper.GetString("sevenpace.activity_type_id"),
		InsecureSkipTLS: viper.GetBool("sevenpace.insecure_skip_verify"),
	}

	if cfg.BaseURL == "" {
		return SevenPaceConfig{}, fmt.Errorf("missing sevenpace.base_url in config, please run 'toggl-cli config'")
	}
	if cfg.Username == "" || cfg.Password == "" {
		return SevenPaceConfig{}, fmt.Errorf("missing sevenpace.username or sevenpace.password in config, please run 'toggl-cli config'")
	}

	return cfg, nil
}

func GetTimezone() (*time.Location, error) {
	tz := viper.GetString("toggl.timezone")
	if tz == "" {
		return time.Local, nil
	}

	location, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q in config: %w", tz, err)
	}

	return location, nil
}
