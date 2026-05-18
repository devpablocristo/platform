package qr

import (
	"strings"

	"github.com/devpablocristo/core/artifact/go"
	qrcode "github.com/skip2/go-qrcode"
)

const defaultSize = 256

// PNG genera un QR PNG reusable.
func PNG(content string, size int) ([]byte, error) {
	if size <= 0 {
		size = defaultSize
	}
	return qrcode.Encode(strings.TrimSpace(content), qrcode.Medium, size)
}

// PNGAsset genera un asset PNG listo para storage o response.
func PNGAsset(name, content string, size int, metadata map[string]string) (artifact.Asset, error) {
	body, err := PNG(content, size)
	if err != nil {
		return artifact.Asset{}, err
	}
	return artifact.New(name, artifact.FormatPNG, body, metadata), nil
}
