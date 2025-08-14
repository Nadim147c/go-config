package config

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// KeySplit parses a dotted key path into parts, respecting quotes.
// Example:
//
// "a.b.c"         -> {"a", "b", "c"}
// "a.'b.c'.\"c\"" -> {"a", "b.c", "c"}
// "'a.b'.c"       -> {"a.b", "c"}
func KeySplit(key string) ([]string, error) {
	var parts []string
	var buf strings.Builder

	inQuotes := false
	quoteChar := rune(0)

	for i, r := range key {
		switch {
		case (r == '\'' || r == '"'):
			if inQuotes {
				if r == quoteChar {
					// End quote
					inQuotes = false
					quoteChar = 0
				} else {
					// Different quote inside quoted string
					buf.WriteRune(r)
				}
			} else {
				// Start quote
				inQuotes = true
				quoteChar = r
			}

		case r == '.' && !inQuotes:
			// Dot outside quotes = new part
			parts = append(parts, buf.String())
			buf.Reset()

		case r == '\\':
			// Handle escapes
			if i+1 < len(key) {
				nextRune, width := utf8.DecodeRuneInString(key[i+1:])
				buf.WriteRune(nextRune)
				i += width - 1 // skip consumed rune
			} else {
				return nil, fmt.Errorf("dangling escape at position %d", i)
			}

		default:
			buf.WriteRune(r)
		}
	}

	if inQuotes {
		return nil, fmt.Errorf("unclosed quote in key")
	}

	// Last part
	parts = append(parts, buf.String())
	return parts, nil
}
