package output

import "testing"

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "00:00:00"},
		{1, "00:00:01"},
		{59, "00:00:59"},
		{60, "00:01:00"},
		{90, "00:01:30"},
		{3600, "01:00:00"},
		{3661, "01:01:01"},
		{7384, "02:03:04"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := FormatDuration(tt.seconds); got != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.seconds, got, tt.expected)
			}
		})
	}
}
