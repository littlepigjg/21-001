package config

import (
	"fmt"
	"strings"
)

// Validate 检查配置是否合法，返回所有问题描述。
//
// 若返回空切片表示配置合法。该函数用于在服务启动前暴露明显配置错误。
func (c *Config) Validate() []string {
	var problems []string

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		problems = append(problems, "server.port 必须在 1-65535 之间")
	}
	if c.Server.Host == "" {
		problems = append(problems, "server.host 不能为空")
	}
	if c.Server.ReadTimeoutSeconds <= 0 {
		problems = append(problems, "server.read_timeout_seconds 必须大于 0")
	}
	if c.Server.WriteTimeoutSeconds <= 0 {
		problems = append(problems, "server.write_timeout_seconds 必须大于 0")
	}
	if c.Server.ShutdownTimeoutSeconds <= 0 {
		problems = append(problems, "server.shutdown_timeout_seconds 必须大于 0")
	}

	if c.Storage.DataDir == "" {
		problems = append(problems, "storage.data_dir 不能为空")
	}
	if c.Storage.DocumentsFile == "" {
		problems = append(problems, "storage.documents_file 不能为空")
	}
	if c.Storage.IndexFile == "" {
		problems = append(problems, "storage.index_file 不能为空")
	}

	if c.Search.MaxPageSize <= 0 {
		problems = append(problems, "search.max_page_size 必须大于 0")
	}
	if c.Search.DefaultPageSize <= 0 || c.Search.DefaultPageSize > c.Search.MaxPageSize {
		problems = append(problems, "search.default_page_size 必须大于 0 且不超过 max_page_size")
	}
	if c.Search.BM25K1 <= 0 {
		problems = append(problems, "search.bm25_k1 必须大于 0")
	}
	if c.Search.BM25B < 0 || c.Search.BM25B > 1 {
		problems = append(problems, "search.bm25_b 必须在 [0,1] 之间")
	}

	if c.Upload.MaxUploadBytes <= 0 {
		problems = append(problems, "upload.max_upload_bytes 必须大于 0")
	}
	if len(c.Upload.AllowedFormats) == 0 {
		problems = append(problems, "upload.allowed_formats 不能为空")
	}

	if lvl := strings.ToLower(c.Log.Level); lvl != "debug" && lvl != "info" && lvl != "warn" && lvl != "error" {
		problems = append(problems, "log.level 取值非法，应为 debug/info/warn/error")
	}

	return problems
}

// ValidateString 返回配置校验结果的格式化文本。
func (c *Config) ValidateString() string {
	problems := c.Validate()
	if len(problems) == 0 {
		return "配置合法"
	}
	return fmt.Sprintf("配置存在问题：\n- %s", strings.Join(problems, "\n- "))
}
