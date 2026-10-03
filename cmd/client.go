package cmd

import (
	"github.com/spf13/viper"

	"github.com/ville6000/toggl-cli/internal/api"
	"github.com/ville6000/toggl-cli/internal/config"
)

// newTogglClient builds the Toggl client used by commands, honouring the
// optional toggl.base_url override in v.
func newTogglClient(v *viper.Viper, token string) *api.Client {
	var opts []api.ClientOption
	if baseURL := config.TogglBaseURL(v); baseURL != "" {
		opts = append(opts, api.WithBaseURL(baseURL))
	}

	return api.NewClient(token, opts...)
}
