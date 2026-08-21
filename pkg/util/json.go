package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadJSON 从文件读取并反序列化 JSON 到目标结构。
//
// 若文件不存在则返回 false 而不报错，便于调用方判断是否需要初始化。
func LoadJSON(path string, target interface{}) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("读取文件 %s 失败: %w", path, err)
	}
	if len(data) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(data, target); err != nil {
		return false, fmt.Errorf("解析文件 %s 失败: %w", path, err)
	}
	return true, nil
}

// SaveJSON 将目标结构序列化并原子写入文件。
//
// 先写入临时文件再重命名，避免进程崩溃时损坏已有数据。
func SaveJSON(path string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("重命名文件失败: %w", err)
	}
	return nil
}
