package artifact

import (
	"strings"
	"time"
	"unicode"
)

type Format string

const (
	FormatCSV  Format = "csv"
	FormatXLSX Format = "xlsx"
	FormatPDF  Format = "pdf"
	FormatJSON Format = "json"
	FormatTXT  Format = "txt"
	FormatPNG  Format = "png"
	FormatJPG  Format = "jpg"
)

type Asset struct {
	Name        string            `json:"name"`
	Format      Format            `json:"format"`
	ContentType string            `json:"content_type"`
	Body        []byte            `json:"body"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func New(name string, format Format, body []byte, metadata map[string]string) Asset {
	return Asset{
		Name:        NormalizeFilename(name, format),
		Format:      format,
		ContentType: ContentType(format),
		Body:        append([]byte(nil), body...),
		Metadata:    cloneMetadata(metadata),
	}
}

func (a Asset) Size() int {
	return len(a.Body)
}

func BuildFilename(parts []string, format Format, now time.Time) string {
	clean := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		if slug := Slug(part); slug != "" {
			clean = append(clean, slug)
		}
	}
	if !now.IsZero() {
		clean = append(clean, now.UTC().Format("2006-01-02"))
	}
	if len(clean) == 0 {
		clean = []string{"artifact"}
	}
	return NormalizeFilename(strings.Join(clean, "_"), format)
}

func NormalizeFilename(name string, format Format) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "artifact"
	}
	ext := Extension(format)
	if ext == "" {
		return name
	}
	if strings.HasSuffix(strings.ToLower(name), "."+ext) {
		return name
	}
	return name + "." + ext
}

func Extension(format Format) string {
	switch format {
	case FormatCSV:
		return "csv"
	case FormatXLSX:
		return "xlsx"
	case FormatPDF:
		return "pdf"
	case FormatJSON:
		return "json"
	case FormatTXT:
		return "txt"
	case FormatPNG:
		return "png"
	case FormatJPG:
		return "jpg"
	default:
		return string(format)
	}
}

func ContentType(format Format) string {
	switch format {
	case FormatCSV:
		return "text/csv; charset=utf-8"
	case FormatXLSX:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case FormatPDF:
		return "application/pdf"
	case FormatJSON:
		return "application/json"
	case FormatTXT:
		return "text/plain; charset=utf-8"
	case FormatPNG:
		return "image/png"
	case FormatJPG:
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

func Slug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}

	var out []rune
	lastSep := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			out = append(out, r)
			lastSep = false
		case r == '-' || r == '_' || unicode.IsSpace(r) || r == '/' || r == '.':
			if !lastSep && len(out) > 0 {
				out = append(out, '_')
				lastSep = true
			}
		}
	}

	return strings.Trim(string(out), "_")
}

func cloneMetadata(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
