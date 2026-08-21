package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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
