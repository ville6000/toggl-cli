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
