package output

import "fmt"

// FormatDuration formats a number of seconds as HH:MM:SS.
func FormatDuration(seconds int) string {
	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
}
