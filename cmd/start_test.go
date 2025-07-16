package cmd

import (
	"testing"
)

// Test getTicketNumberFromPath
func TestGetTicketNumberFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "path with numbers",
			path:     "ticket-123",
			expected: "123",
		},
		{
			name:     "path with mixed characters",
			path:     "abc-123-xyz",
			expected: "123",
		},
		{
			name:     "path with no numbers",
			path:     "no-numbers",
			expected: "",
		},
		{
			name:     "path with multiple number groups",
			path:     "abc-123-xyz-456",
			expected: "123456",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTicketNumberFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("getTicketNumberFromPath(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}

// Test getDescription with direct args
func TestGetDescriptionWithArgs(t *testing.T) {
	// Test with args provided
	args := []string{"test description"}
	result := getDescription(args)
	if result != "test description" {
		t.Errorf("getDescription() with args = %s, want %s", result, "test description")
	}

	// Test with empty args slice
	result = getDescription([]string{})
	// We can't test the exact result since it depends on detectDescriptionFromCurrentPath
	// which uses os.Getwd(), but we can at least ensure the function runs without panicking
	t.Logf("getDescription() with empty args returned: %s", result)

	// Test with nil args
	result = getDescription(nil)
	// Same as above, we can't test the exact result
	t.Logf("getDescription() with nil args returned: %s", result)
}
