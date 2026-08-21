package util

import (
	"encoding/csv"
	"fmt"
	"io"
)

func WriteCSV(w io.Writer, headers []string, rows [][]string) (int, error) {
	cw := csv.NewWriter(w)
	if err := cw.Write(headers); err != nil {
		return 0, fmt.Errorf("写入表头失败: %w", err)
	}
	for _, row := range rows {
		if err := cw.Write(row); err != nil {
			return 0, fmt.Errorf("写入数据行失败: %w", err)
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return 0, fmt.Errorf("刷新 CSV 失败: %w", err)
	}
	return len(rows) + 1, nil
}

func ReadCSVAll(r io.Reader) ([][]string, error) {
	cr := csv.NewReader(r)
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("读取 CSV 失败: %w", err)
	}
	return records, nil
}
