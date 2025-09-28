package utils

import (
	"strings"
	"unicode"
)

// CamelToSnake converts a camelCase string into snake_case while preserving separators.
func CamelToSnake(s string) string {
	if s == "" {
		return ""
	}

	runes := []rune(s)
	var builder strings.Builder
	builder.Grow(len(runes) + len(runes)/2)

	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				var next rune
				if i+1 < len(runes) {
					next = runes[i+1]
				}

				if unicode.IsLower(prev) || unicode.IsDigit(prev) ||
					(unicode.IsUpper(prev) && unicode.IsLower(next)) {
					builder.WriteByte('_')
				}
			}

			builder.WriteRune(unicode.ToLower(r))
			continue
		}

		builder.WriteRune(r)
	}

	return builder.String()
}
