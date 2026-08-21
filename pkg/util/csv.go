package util

import (
	"encoding/csv"
	"fmt"
	"io"
)

// WriteCSV 将表头与数据行写入 io.Writer，返回写入的行数（含表头）。
//
// 该函数基于标准库 encoding/csv，自动处理字段引用与转义。
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

// ReadCSVAll 从 io.Reader 读取全部 CSV 记录（含表头）。
func ReadCSVAll(r io.Reader) ([][]string, error) {
	cr := csv.NewReader(r)
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("读取 CSV 失败: %w", err)
	}
	return records, nil
}
