// Package config 负责读取并解析应用配置。
//
// 配置以 JSON 文件形式存储，默认路径为 ./config.json，也可通过命令行参数
// -config 指定。若配置文件不存在或解析失败，则回退到内置的默认配置，保证
// 服务在最小环境下依然可以启动。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Config 是应用运行时的全局配置聚合结构。
type Config struct {
	// Server 保存 HTTP 服务相关配置。
	Server ServerConfig `json:"server"`
	// Storage 保存数据持久化相关配置。
	Storage StorageConfig `json:"storage"`
	// Search 保存全文检索相关配置。
	Search SearchConfig `json:"search"`
	// Upload 保存文件上传相关配置。
	Upload UploadConfig `json:"upload"`
	// Log 保存日志相关配置。
	Log LogConfig `json:"log"`
}

// ServerConfig 描述 HTTP 服务器监听地址与超时策略。
type ServerConfig struct {
	// Host 是监听地址，例如 "0.0.0.0"。
	Host string `json:"host"`
	// Port 是监听端口，默认 8080。
	Port int `json:"port"`
	// ReadTimeoutSeconds 是读取请求体的超时秒数。
	ReadTimeoutSeconds int `json:"read_timeout_seconds"`
	// WriteTimeoutSeconds 是写入响应的超时秒数。
	WriteTimeoutSeconds int `json:"write_timeout_seconds"`
	// ShutdownTimeoutSeconds 是优雅关闭的最大等待秒数。
	ShutdownTimeoutSeconds int `json:"shutdown_timeout_seconds"`
}

// StorageConfig 描述数据目录与 JSON 持久化策略。
type StorageConfig struct {
	// DataDir 是所有持久化 JSON 文件所在目录。
	DataDir string `json:"data_dir"`
	// DocumentsFile 是文档主数据文件名。
	DocumentsFile string `json:"documents_file"`
	// IndexFile 是倒排索引文件名。
	IndexFile string `json:"index_file"`
	// TagsFile 是标签数据文件名。
	TagsFile string `json:"tags_file"`
	// CategoriesFile 是分类数据文件名。
	CategoriesFile string `json:"categories_file"`
	// StatsFile 是统计数据文件名。
	StatsFile string `json:"stats_file"`
	// AutoSave 是否在每次写操作后自动持久化。
	AutoSave bool `json:"auto_save"`
}

// SearchConfig 描述检索相关的可调参数。
type SearchConfig struct {
	// MaxPageSize 限制单次检索返回的最大条数。
	MaxPageSize int `json:"max_page_size"`
	// DefaultPageSize 是未指定时的默认页大小。
	DefaultPageSize int `json:"default_page_size"`
	// BM25K1 与 BM25B 是 BM25 排序算法的参数。
	BM25K1 float64 `json:"bm25_k1"`
	BM25B  float64 `json:"bm25_b"`
}

// UploadConfig 描述文件上传限制。
type UploadConfig struct {
	// MaxUploadBytes 限制单个上传文件的最大字节数。
	MaxUploadBytes int64 `json:"max_upload_bytes"`
	// AllowedFormats 是允许的上传格式列表。
	AllowedFormats []string `json:"allowed_formats"`
}

// LogConfig 描述日志输出相关配置。
type LogConfig struct {
	// Level 是日志级别：debug / info / warn / error。
	Level string `json:"level"`
	// Output 是日志输出目标：stdout / stderr / 文件路径。
	Output string `json:"output"`
}

var (
	mu     sync.RWMutex
	global = defaultConfig()
)

// Get 返回当前全局配置的只读副本。
//
// 通过返回副本而不是指针，避免调用方在运行时无意间修改全局配置。
func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return global
}

// Set 覆盖全局配置。主要用于启动阶段或测试阶段注入配置。
func Set(cfg Config) {
	mu.Lock()
	defer mu.Unlock()
	global = cfg
}

// Load 从指定路径加载 JSON 配置。若 path 为空则使用默认路径。
//
// 该函数会在读取后对配置做一次规范化（如填充缺失的默认值、校正非法参数）。
func Load(path string) (Config, error) {
	if path == "" {
		path = defaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("解析配置文件失败: %w", err)
	}

	cfg = normalize(cfg)
	Set(cfg)
	return cfg, nil
}

// defaultConfigPath 返回默认的配置文件路径，优先读取环境变量。
func defaultConfigPath() string {
	if p := os.Getenv("BENZHI_CONFIG"); p != "" {
		return p
	}
	return "./config.json"
}

// defaultConfig 返回一组安全的默认配置，保证开箱即用。
func defaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Host:                   "0.0.0.0",
			Port:                   8080,
			ReadTimeoutSeconds:     15,
			WriteTimeoutSeconds:    30,
			ShutdownTimeoutSeconds: 10,
		},
		Storage: StorageConfig{
			DataDir:        "./data",
			DocumentsFile:  "documents.json",
			IndexFile:      "index.json",
			TagsFile:       "tags.json",
			CategoriesFile: "categories.json",
			StatsFile:      "stats.json",
			AutoSave:       true,
		},
		Search: SearchConfig{
			MaxPageSize:      100,
			DefaultPageSize:  10,
			BM25K1:           1.2,
			BM25B:            0.75,
		},
		Upload: UploadConfig{
			MaxUploadBytes: 10 * 1024 * 1024,
			AllowedFormats: []string{"txt", "md", "markdown", "pdf"},
		},
		Log: LogConfig{
			Level:  "info",
			Output: "stdout",
		},
	}
}

// normalize 对配置做兜底校正，避免非法参数导致运行时异常。
func normalize(cfg Config) Config {
	def := defaultConfig()

	if cfg.Server.Port == 0 {
		cfg.Server.Port = def.Server.Port
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = def.Server.Host
	}
	if cfg.Server.ReadTimeoutSeconds <= 0 {
		cfg.Server.ReadTimeoutSeconds = def.Server.ReadTimeoutSeconds
	}
	if cfg.Server.WriteTimeoutSeconds <= 0 {
		cfg.Server.WriteTimeoutSeconds = def.Server.WriteTimeoutSeconds
	}
	if cfg.Server.ShutdownTimeoutSeconds <= 0 {
		cfg.Server.ShutdownTimeoutSeconds = def.Server.ShutdownTimeoutSeconds
	}

	if cfg.Storage.DataDir == "" {
		cfg.Storage.DataDir = def.Storage.DataDir
	}
	if cfg.Storage.DocumentsFile == "" {
		cfg.Storage.DocumentsFile = def.Storage.DocumentsFile
	}
	if cfg.Storage.IndexFile == "" {
		cfg.Storage.IndexFile = def.Storage.IndexFile
	}
	if cfg.Storage.TagsFile == "" {
		cfg.Storage.TagsFile = def.Storage.TagsFile
	}
	if cfg.Storage.CategoriesFile == "" {
		cfg.Storage.CategoriesFile = def.Storage.CategoriesFile
	}
	if cfg.Storage.StatsFile == "" {
		cfg.Storage.StatsFile = def.Storage.StatsFile
	}

	if cfg.Search.MaxPageSize <= 0 {
		cfg.Search.MaxPageSize = def.Search.MaxPageSize
	}
	if cfg.Search.DefaultPageSize <= 0 {
		cfg.Search.DefaultPageSize = def.Search.DefaultPageSize
	}
	if cfg.Search.BM25K1 <= 0 {
		cfg.Search.BM25K1 = def.Search.BM25K1
	}
	if cfg.Search.BM25B < 0 {
		cfg.Search.BM25B = def.Search.BM25B
	}

	if cfg.Upload.MaxUploadBytes <= 0 {
		cfg.Upload.MaxUploadBytes = def.Upload.MaxUploadBytes
	}
	if len(cfg.Upload.AllowedFormats) == 0 {
		cfg.Upload.AllowedFormats = def.Upload.AllowedFormats
	}

	if cfg.Log.Level == "" {
		cfg.Log.Level = def.Log.Level
	}
	if cfg.Log.Output == "" {
		cfg.Log.Output = def.Log.Output
	}

	return cfg
}
