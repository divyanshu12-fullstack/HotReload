package cmdline

import (
	"fmt"
	"strings"
	"unicode"
)

// Fields splits a command line while preserving quoted segments.
func Fields(input string) ([]string, error) {
	var fields []string
	var current strings.Builder

	inQuote := rune(0)
	escaped := false

	flush := func() {
		if current.Len() == 0 {
			return
		}
		fields = append(fields, current.String())
		current.Reset()
	}

	for _, char := range input {
		switch {
		case escaped:
			current.WriteRune(char)
			escaped = false
		case char == '\\' && inQuote != '\'':
			escaped = true
		case inQuote != 0:
			if char == inQuote {
				inQuote = 0
				continue
			}
			current.WriteRune(char)
		case char == '\'' || char == '"':
			inQuote = char
		case unicode.IsSpace(char):
			flush()
		default:
			current.WriteRune(char)
		}
	}

	if escaped {
		current.WriteRune('\\')
	}
	if inQuote != 0 {
		return nil, fmt.Errorf("unterminated quote in command: %q", input)
	}

	flush()
	return fields, nil
}
