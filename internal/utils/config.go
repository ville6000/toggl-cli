package utils

import (
	"fmt"

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
