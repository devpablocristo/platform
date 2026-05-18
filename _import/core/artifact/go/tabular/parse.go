package tabular

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

type Sheet struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// ParseCSV decodifica un CSV completo en headers + rows.
func ParseCSV(body []byte) (Sheet, error) {
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})))
	records, err := reader.ReadAll()
	if err != nil {
		if err == io.EOF {
			return Sheet{}, nil
		}
		return Sheet{}, err
	}
	if len(records) == 0 {
		return Sheet{}, nil
	}
	return Sheet{
		Headers: append([]string(nil), records[0]...),
		Rows:    cloneRows(records[1:]),
	}, nil
}

// ParseXLSX decodifica la primera hoja o una hoja explícita.
func ParseXLSX(body []byte, sheetName string) (Sheet, error) {
	file, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		return Sheet{}, err
	}
	defer func() { _ = file.Close() }()

	if sheetName == "" {
		sheets := file.GetSheetList()
		if len(sheets) == 0 {
			return Sheet{}, fmt.Errorf("xlsx workbook has no sheets")
		}
		sheetName = sheets[0]
	}

	rows, err := file.GetRows(sheetName)
	if err != nil {
		return Sheet{}, err
	}
	if len(rows) == 0 {
		return Sheet{}, nil
	}
	return Sheet{
		Headers: append([]string(nil), rows[0]...),
		Rows:    cloneRows(rows[1:]),
	}, nil
}

func cloneRows(rows [][]string) [][]string {
	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, append([]string(nil), row...))
	}
	return out
}
