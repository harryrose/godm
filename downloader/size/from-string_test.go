package size

import (
	"testing"
)

// Size represents a size in bytes.

func TestFromString(t *testing.T) {
	tests := []struct {
		input     string
		expected  Size
		expectErr bool
	}{
		{"123", 123, false},
		{"123B", 123, false},
		{"1KB", 1024, false},
		{"1KB ", 1024, false},
		{"1.5KB", 1536, false},
		{"2MB", 2 * 1024 * 1024, false},
		{"0.5MB", 524288, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"1.5GB", Size(1.5 * 1024 * 1024 * 1024), false},
		{"1TB", 1024 * 1024 * 1024 * 1024, false},
		{"1.5TB", Size(1.5 * 1024 * 1024 * 1024 * 1024), false},
		{"100", 100, false},
		{"100 B", 100, false},
		{"100K", 102400, false},
		{"100M", 104857600, false},
		{"100G", 107374182400, false},
		{"100T", 109951162777600, false},
		{"1PB", 0, true},     // unsupported suffix
		{"abc", 0, true},     // invalid number
		{"1XB", 0, true},     // invalid suffix
		{"1.2.3KB", 0, true}, // invalid number
		{"1KBB", 0, true},    // invalid suffix
	}

	for _, tt := range tests {
		result, err := FromString(tt.input)
		if tt.expectErr {
			if err == nil {
				t.Errorf("FromString(%q) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("FromString(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("FromString(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}
