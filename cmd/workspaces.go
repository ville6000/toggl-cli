package cmd

import (
	"fmt"

	"github.com/ville6000/toggl-cli/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newWorkspacesCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspaces",
		Short: "List workspaces",
		Long:  "List all workspaces associated with the Toggl account.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			token, err := config.Token(v)
			if err != nil {
				return fmt.Errorf("failed to get API token: %w", err)
			}

			client := newTogglClient(v, token)
			workspaces, err := client.Workspaces(ctx)
			if err != nil {
				return fmt.Errorf("failed to get workspaces: %w", err)
			}

			for _, workspace := range workspaces {
				fmt.Fprintf(cmd.OutOrStdout(), "ID: %d, Name: %s\n", workspace.ID, workspace.Name)
			}

			return nil
		},
	}

	return cmd
}
