package stringutil

import (
	"regexp"
	"strings"
	"unicode"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// NormalizeString lowercases, removes accents, and keeps only a-z characters.
func NormalizeString(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	// Descomponer Unicode NFD y filtrar diacríticos manualmente (sin x/text)
	clean := make([]rune, 0, len(input))
	for _, r := range input {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r >= 'a' && r <= 'z' {
			clean = append(clean, r)
		}
	}
	return string(clean)
}

// BasicInputSanitizer trims whitespace and removes all HTML/XML tags.
func BasicInputSanitizer(input string) string {
	input = strings.TrimSpace(input)
	return htmlTagRe.ReplaceAllString(input, "")
}

// NormalizeTags deduplica y limpia una lista de tags (trim + elimina vacíos + únicos).
func NormalizeTags(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
