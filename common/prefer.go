package common

import (
	"net/http"
	"strings"
)

// HasPreferDirective reports whether a Prefer header contains an exact,
// case-insensitive preference token. It handles repeated header fields,
// comma-separated directives, quoted values, and preference parameters.
func HasPreferDirective(header http.Header, directive string) bool {
	directive = strings.TrimSpace(directive)
	if directive == "" {
		return false
	}

	for _, value := range header.Values("Prefer") {
		segmentStart := 0
		inQuote := false
		escaped := false
		for i := 0; i <= len(value); i++ {
			atEnd := i == len(value)
			if !atEnd {
				switch value[i] {
				case '\\':
					if inQuote && !escaped {
						escaped = true
						continue
					}
				case '"':
					if !escaped {
						inQuote = !inQuote
					}
				case ',':
					if !inQuote {
						atEnd = true
					}
				}
				escaped = false
			}
			if !atEnd {
				continue
			}

			segment := strings.TrimSpace(value[segmentStart:i])
			if separator := strings.IndexByte(segment, ';'); separator >= 0 {
				segment = strings.TrimSpace(segment[:separator])
			}
			if strings.EqualFold(segment, directive) {
				return true
			}
			segmentStart = i + 1
		}
	}
	return false
}
