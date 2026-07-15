package strutil

import "strings"

// ParseCommaSeparated splits a string by comma, trims whitespace from each element,
// and returns a slice containing only non-empty elements.
func ParseCommaSeparated(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
