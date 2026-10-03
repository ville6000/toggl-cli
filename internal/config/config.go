// Package config reads toggl-cli settings from a loaded configuration.
package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Token returns the configured Toggl API token.
func Token(v *viper.Viper) (string, error) {
	token := v.GetString("toggl.token")
	if token == "" {
		return "", errors.New("missing toggl.token in config, please run 'toggl-cli config'")
	}

	return token, nil
}

// TokenAndWorkspace returns the configured Toggl API token and default
// workspace ID, both of which are required.
func TokenAndWorkspace(v *viper.Viper) (string, int, error) {
	token := v.GetString("toggl.token")
	if token == "" {
		return "", 0, errors.New("missing toggl.token in config, please run 'toggl-cli config'")
	}

	workspaceID := v.GetInt("toggl.workspace_id")
	if workspaceID == 0 {
		return "", 0, errors.New("missing toggl.workspace_id in config, please run 'toggl-cli config'")
	}

	return token, workspaceID, nil
}

// TogglBaseURL returns the toggl.base_url config override, or "" when the
// default Toggl API endpoint should be used. Pointing this at a stub server is
// how the command tests exercise the CLI end to end.
func TogglBaseURL(v *viper.Viper) string {
	return v.GetString("toggl.base_url")
}

// SevenPace holds the settings for talking to an on-prem 7pace
// Timetracker instance using NTLM (Windows) authentication.
type SevenPace struct {
	BaseURL         string
	Domain          string
	Username        string
	Password        string
	ActivityTypeID  string
	InsecureSkipTLS bool
}

// LoadSevenPace reads the 7pace configuration from v. Base URL,
// username and password are required; domain and activity type are optional.
func LoadSevenPace(v *viper.Viper) (SevenPace, error) {
	cfg := SevenPace{
		BaseURL:         v.GetString("sevenpace.base_url"),
		Domain:          v.GetString("sevenpace.domain"),
		Username:        v.GetString("sevenpace.username"),
		Password:        v.GetString("sevenpace.password"),
		ActivityTypeID:  v.GetString("sevenpace.activity_type_id"),
		InsecureSkipTLS: v.GetBool("sevenpace.insecure_skip_verify"),
	}

	if cfg.BaseURL == "" {
		return SevenPace{}, errors.New("missing sevenpace.base_url in config, please run 'toggl-cli config'")
	}
	if cfg.Username == "" || cfg.Password == "" {
		return SevenPace{}, errors.New("missing sevenpace.username or sevenpace.password in config, please run 'toggl-cli config'")
	}

	return cfg, nil
}

// Timezone returns the configured toggl.timezone, or the local time zone when
// none is set.
func Timezone(v *viper.Viper) (*time.Location, error) {
	tz := v.GetString("toggl.timezone")
	if tz == "" {
		return time.Local, nil
	}

	location, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q in config: %w", tz, err)
	}

	return location, nil
}
