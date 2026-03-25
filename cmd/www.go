package cmd

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var wwwCmd = &cobra.Command{
	Use:   "www",
	Short: "Open Toggl in the browser",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := browser.OpenURL("https://track.toggl.com/timer"); err != nil {
			return fmt.Errorf("failed to open browser: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(wwwCmd)
}
