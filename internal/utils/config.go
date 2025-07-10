package utils

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

func GetTogglConfig() (string, int) {
	token := viper.GetString("toggl.token")
	if token == "" {
		log.Fatal("Missing toggl.token in config file")
	}

	workspaceId := viper.GetInt("toggl.workspace_id")
	if workspaceId == 0 {
		log.Fatal("Missing toggl.workspace_id in config file")
	}

	return token, workspaceId
}

func GetConfig() (string, int, error) {
	token, workspaceId := GetTogglConfig()

	if token == "" || workspaceId == 0 {
		return "", 0, fmt.Errorf("Invalid configuration, please run 'toggl-cli config'")
	}

	return token, workspaceId, nil
}
