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
	Run: func(cmd *cobra.Command, args []string) {
		err := browser.OpenURL("https://track.toggl.com/timer")
		if err != nil {
			fmt.Println("Error opening browser:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(wwwCmd)
}
