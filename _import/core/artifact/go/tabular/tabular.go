package tabular

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/devpablocristo/core/artifact/go"
	"github.com/xuri/excelize/v2"
)

const (
	// CSVContentType es el content type recomendado para exportes CSV con BOM.
	CSVContentType = "text/csv; charset=utf-8"
	// XLSXContentType es el content type recomendado para exportes XLSX.
	XLSXContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
)

// CSV construye un CSV UTF-8 con BOM a partir de headers y rows.
func CSV(headers []string, rows [][]string) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(buf)
	if err := writer.Write(headers); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buf.Bytes(), writer.Error()
}

// CSVAsset construye un asset CSV listo para storage o response.
func CSVAsset(name string, headers []string, rows [][]string, metadata map[string]string) (artifact.Asset, error) {
	body, err := CSV(headers, rows)
	if err != nil {
		return artifact.Asset{}, err
	}
	return artifact.New(name, artifact.FormatCSV, body, metadata), nil
}

// XLSX construye un workbook con una sola hoja a partir de headers y rows.
func XLSX(headers []string, rows [][]string) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	for idx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return nil, err
		}
	}

	style, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	if len(headers) > 0 {
		end := fmt.Sprintf("%s1", columnName(len(headers)))
		_ = f.SetCellStyle(sheet, "A1", end, style)
	}

	for rowIdx, row := range rows {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
	}

	for idx := range headers {
		col := columnName(idx + 1)
		_ = f.SetColWidth(sheet, col, col, 18)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// XLSXAsset construye un asset XLSX listo para storage o response.
func XLSXAsset(name string, headers []string, rows [][]string, metadata map[string]string) (artifact.Asset, error) {
	body, err := XLSX(headers, rows)
	if err != nil {
		return artifact.Asset{}, err
	}
	return artifact.New(name, artifact.FormatXLSX, body, metadata), nil
}

func columnName(index int) string {
	name, _ := excelize.ColumnNumberToName(index)
	return name
}
