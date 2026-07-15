package strutil

import (
	"reflect"
	"testing"
)

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "only whitespace",
			input:    "   ,  , ",
			expected: nil,
		},
		{
			name:     "single element",
			input:    "openid",
			expected: []string{"openid"},
		},
		{
			name:     "multiple elements",
			input:    "openid,profile,email",
			expected: []string{"openid", "profile", "email"},
		},
		{
			name:     "with whitespace",
			input:    " openid ,  profile, email  ",
			expected: []string{"openid", "profile", "email"},
		},
		{
			name:     "with empty elements",
			input:    "openid,,profile,,email",
			expected: []string{"openid", "profile", "email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommaSeparated(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseCommaSeparated(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}
