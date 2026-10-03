// Package config reads toggl-cli settings from a loaded configuration.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// EnvPrefix starts the name of every environment variable that sets a config
// key: toggl.token is read from TOGGL_CLI_TOGGL_TOKEN.
const EnvPrefix = "TOGGL_CLI"

var envKeyReplacer = strings.NewReplacer(".", "_")

// UseEnv makes v read config keys from the environment as well. A variable
// outranks the config file, so credentials can be supplied without one.
func UseEnv(v *viper.Viper) {
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(envKeyReplacer)
	v.AutomaticEnv()
}

// EnvVar returns the environment variable that sets key.
func EnvVar(key string) string {
	return EnvPrefix + "_" + strings.ToUpper(envKeyReplacer.Replace(key))
}

// missing returns the error for required keys that are not set.
func missing(keys ...string) error {
	vars := make([]string, len(keys))
	for i, key := range keys {
		vars[i] = EnvVar(key)
	}

	return fmt.Errorf("missing %s in config (or %s), please run 'toggl-cli config'",
		strings.Join(keys, " or "), strings.Join(vars, " / "))
}

// Token returns the configured Toggl API token.
func Token(v *viper.Viper) (string, error) {
	token := v.GetString("toggl.token")
	if token == "" {
		return "", missing("toggl.token")
	}

	return token, nil
}

// TokenAndWorkspace returns the configured Toggl API token and default
// workspace ID, both of which are required.
func TokenAndWorkspace(v *viper.Viper) (string, int, error) {
	token := v.GetString("toggl.token")
	if token == "" {
		return "", 0, missing("toggl.token")
	}

	workspaceID := v.GetInt("toggl.workspace_id")
	if workspaceID == 0 {
		return "", 0, missing("toggl.workspace_id")
	}

	return token, workspaceID, nil
}

// TogglBaseURL returns the toggl.base_url config override, or "" when the
// default Toggl API endpoint should be used. Pointing this at a stub server is
// how the command tests exercise the CLI end to end.
func TogglBaseURL(v *viper.Viper) string {
	return v.GetString("toggl.base_url")
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
