package utils

import (
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
