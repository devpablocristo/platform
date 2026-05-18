package pdf

import (
	"bytes"
	"strings"

	"github.com/devpablocristo/core/artifact/go"
	"github.com/go-pdf/fpdf"
)

// Line representa una línea simple de label + valor.
type Line struct {
	Label string
	Value string
}

// Document es un documento PDF simple y reusable.
type Document struct {
	Title    string
	Subtitle string
	Lines    []Line
	Footer   string
}

// Simple genera un PDF básico sin acoplarlo a un producto.
func Simple(doc Document) ([]byte, error) {
	file := fpdf.New("P", "mm", "A4", "")
	file.SetTitle(strings.TrimSpace(doc.Title), false)
	file.SetMargins(15, 15, 15)
	file.AddPage()

	file.SetFont("Helvetica", "B", 18)
	file.MultiCell(0, 10, strings.TrimSpace(doc.Title), "", "L", false)

	if subtitle := strings.TrimSpace(doc.Subtitle); subtitle != "" {
		file.SetFont("Helvetica", "", 11)
		file.SetTextColor(90, 90, 90)
		file.MultiCell(0, 6, subtitle, "", "L", false)
		file.Ln(2)
		file.SetTextColor(0, 0, 0)
	}

	file.SetFont("Helvetica", "", 11)
	for _, line := range doc.Lines {
		label := strings.TrimSpace(line.Label)
		value := strings.TrimSpace(line.Value)
		if label == "" && value == "" {
			continue
		}
		if label == "" {
			file.MultiCell(0, 6, value, "", "L", false)
			continue
		}
		file.SetFont("Helvetica", "B", 11)
		file.CellFormat(45, 6, label, "", 0, "L", false, 0, "")
		file.SetFont("Helvetica", "", 11)
		file.MultiCell(0, 6, value, "", "L", false)
	}

	if footer := strings.TrimSpace(doc.Footer); footer != "" {
		file.SetY(-20)
		file.SetFont("Helvetica", "", 9)
		file.SetTextColor(100, 100, 100)
		file.MultiCell(0, 5, footer, "", "C", false)
	}

	var buf bytes.Buffer
	if err := file.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SimpleAsset genera un asset PDF listo para storage o response.
func SimpleAsset(name string, doc Document, metadata map[string]string) (artifact.Asset, error) {
	body, err := Simple(doc)
	if err != nil {
		return artifact.Asset{}, err
	}
	return artifact.New(name, artifact.FormatPDF, body, metadata), nil
}
