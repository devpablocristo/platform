// Package mime detecta el content-type de un objeto a partir de sus magic
// bytes, y verifica que un tipo declarado por el cliente sea compatible con el
// detectado. Uso típico: validación server-side tras un upload (evitar que el
// cliente suba un PDF declarando "image/jpeg") y defensa en profundidad en
// pipelines de ingesta. Solo-stdlib.
package mime

import (
	"bytes"
	"net/http"
	"strings"
)

// Detect identifica el content-type de un buffer leído desde el inicio del
// archivo. Se recomienda pasar al menos 512 bytes; con 8KB cubre todos los
// formatos detectables por magic bytes que aceptamos. Si no matchea nada
// específico, delega en `http.DetectContentType`.
func Detect(body []byte) string {
	if len(body) == 0 {
		return "application/octet-stream"
	}
	// DICOM (Parte 10): preámbulo de 128 bytes + marcador "DICM" en offset 128.
	if len(body) >= 132 && string(body[128:132]) == "DICM" {
		return "application/dicom"
	}
	switch {
	case bytes.HasPrefix(body, []byte("%PDF-")):
		return "application/pdf"
	case bytes.HasPrefix(body, []byte{0xff, 0xd8, 0xff}):
		return "image/jpeg"
	case bytes.HasPrefix(body, []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}):
		return "image/png"
	case bytes.HasPrefix(body, []byte("RIFF")) && len(body) > 12 && string(body[8:12]) == "WEBP":
		return "image/webp"
	case bytes.HasPrefix(body, []byte("II*\x00")) || bytes.HasPrefix(body, []byte("MM\x00*")):
		return "image/tiff"
	case bytes.HasPrefix(body, []byte("PK\x03\x04")):
		return "application/zip"
	case bytes.HasPrefix(body, []byte("ID3")):
		return "audio/mpeg"
	case bytes.HasPrefix(body, []byte("fLaC")):
		return "audio/flac"
	case bytes.HasPrefix(body, []byte("OggS")):
		return "audio/ogg"
	case bytes.HasPrefix(body, []byte("RIFF")) && len(body) > 12 && string(body[8:12]) == "WAVE":
		return "audio/wav"
	case bytes.HasPrefix(body, []byte("RIFF")) && len(body) > 12 && string(body[8:12]) == "AVI ":
		return "video/x-msvideo"
	case len(body) > 12 && string(body[4:8]) == "ftyp":
		brand := string(body[8:12])
		if strings.HasPrefix(brand, "qt") {
			return "video/quicktime"
		}
		return "video/mp4"
	default:
		return http.DetectContentType(body)
	}
}

// Matches devuelve true si el content-type detectado es compatible con el
// declarado por el cliente. Acepta equivalencias razonables:
//
//   - docx/xlsx detectados como zip (son contenedores OOXML).
//   - text/markdown y text/csv detectados como text/plain.
//   - audio/mp4 y audio/x-m4a detectados como video/mp4 (mismo container).
//   - cualquier image/* coincide con cualquier image/* (defensa: el cliente
//     puede declarar "image/jpeg" y subir "image/png"; lo aceptamos).
//
// Si declared o detected están vacíos, devuelve false (fail-closed por
// definición — el caller decide si rechazar o pasar a quarantine).
func Matches(declared, detected string) bool {
	declared = strings.TrimSpace(declared)
	detected = strings.TrimSpace(detected)
	if declared == "" || detected == "" {
		return false
	}
	if declared == detected {
		return true
	}
	switch declared {
	case "application/dicom":
		return detected == "application/dicom"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return detected == "application/zip"
	case "text/plain", "text/markdown", "text/csv":
		return strings.HasPrefix(detected, "text/plain")
	case "audio/mp4", "audio/x-m4a":
		return detected == "video/mp4"
	}
	if strings.HasPrefix(declared, "image/") {
		return strings.HasPrefix(detected, "image/")
	}
	if strings.HasPrefix(declared, "audio/") {
		return strings.HasPrefix(detected, "audio/")
	}
	if strings.HasPrefix(declared, "video/") {
		return strings.HasPrefix(detected, "video/")
	}
	return false
}
