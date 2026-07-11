package mime

import (
	"bytes"
	"testing"
)

func TestDetect(t *testing.T) {
	dicom := append(bytes.Repeat([]byte{0}, 128), []byte("DICM")...)
	cases := []struct {
		name string
		body []byte
		want string
	}{
		{"empty", nil, "application/octet-stream"},
		{"pdf", []byte("%PDF-1.7\n..."), "application/pdf"},
		{"jpeg", []byte{0xff, 0xd8, 0xff, 0xe0}, "image/jpeg"},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, "image/png"},
		{"zip", []byte("PK\x03\x04rest"), "application/zip"},
		{"dicom", dicom, "application/dicom"},
	}
	for _, c := range cases {
		if got := Detect(c.body); got != c.want {
			t.Errorf("Detect(%s) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestMatches(t *testing.T) {
	cases := []struct {
		declared, detected string
		want               bool
	}{
		{"application/pdf", "application/pdf", true},
		{"image/jpeg", "image/png", true}, // image/* ↔ image/*
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/zip", true},
		{"text/csv", "text/plain; charset=utf-8", true},
		{"application/pdf", "image/png", false},
		{"", "application/pdf", false}, // fail-closed
		{"application/pdf", "", false}, // fail-closed
		{"application/dicom", "image/jpeg", false},
	}
	for _, c := range cases {
		if got := Matches(c.declared, c.detected); got != c.want {
			t.Errorf("Matches(%q,%q) = %v, want %v", c.declared, c.detected, got, c.want)
		}
	}
}
