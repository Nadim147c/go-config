package config

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cast"
)

// KeyKind is...
type KeyKind int

const (
	// SelfKey is name of the key
	SelfKey KeyKind = iota
	// StringKey is name of the key
	StringKey
	// IndexKey is the index of a slice
	IndexKey
)

// Key parsed key for easy uses
type Key struct {
	Raw   string
	Parts []KeyPart
}

// Len returns length of the key
func (k Key) Len() int {
	return len(k.Parts)
}

// LastIndex returns last index of key parts. Returns -1 if length is 0.
func (k Key) LastIndex() int {
	return len(k.Parts) - 1
}

// EnvKey return list of possible env key
func (k Key) EnvKey(prefix string) string {
	// Sanitize each part of the nested key
	sanitizedParts := make([]string, k.Len())
	for i, part := range k.Parts {
		sanitizedParts[i] = sanitizeEnvKeyPart(part.String())
	}
	if prefix != "" {
		return strings.ToUpper(prefix) + "_" + strings.Join(sanitizedParts, "__")
	}
	return strings.Join(sanitizedParts, "__")
}

// sanitizeEnvKeyPart replaces special characters with underscores and ensures valid env var format
func sanitizeEnvKeyPart(part string) string {
	// Replace all special characters with underscores
	var result strings.Builder
	for _, r := range part {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			result.WriteRune(r)
		default:
			result.WriteRune('_')
		}
	}

	// Ensure the part doesn't start with a number
	if result.Len() > 0 && unicode.IsDigit(rune(result.String()[0])) {
		return "_" + result.String()
	}

	return strings.ToUpper(result.String())
}

// KeyPart is a single key
type KeyPart struct {
	Kind      KeyKind
	Interface any
}

func (kp KeyPart) String() string {
	return Must(cast.ToStringE(kp.Interface))
}

// Int converts the KeyPart in an int for array index use
func (kp KeyPart) Int() int {
	return Must(cast.ToIntE(kp.Interface))
}

// KeySplit parses a dotted key path into parts, respecting quotes.
// Example:
//
// "a.b.c"         -> {"a", "b", "c"}
// "a.'b.c'.\"c\"" -> {"a", "b.c", "c"}
// "'a.b'.c"       -> {"a.b", "c"}
func KeySplit(key string) (Key, error) {
	out := Key{
		Raw:   key,
		Parts: []KeyPart{},
	}
	var buf strings.Builder

	if key == "." {
		out.Parts = append(out.Parts, KeyPart{SelfKey, key})
		return out, nil
	}

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
			out.Parts = append(out.Parts, KeyPart{StringKey, buf.String()})
			buf.Reset()

		case r == '\\':
			// Handle escapes
			if i+1 >= len(key) {
				return out, fmt.Errorf("dangling escape at position %d", i)
			}
			nextRune, width := utf8.DecodeRuneInString(key[i+1:])
			buf.WriteRune(nextRune)
			i += width - 1 // skip consumed rune

		default:
			buf.WriteRune(r)
		}
	}

	if inQuotes {
		return out, errors.New("unclosed quote in key")
	}

	// Last part
	out.Parts = append(out.Parts, KeyPart{StringKey, buf.String()})
	return out, nil
}
