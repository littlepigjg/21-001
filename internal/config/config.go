package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Config struct {
	Server ServerConfig `json:"server"`

	Storage StorageConfig `json:"storage"`

	Search SearchConfig `json:"search"`

	Upload UploadConfig `json:"upload"`

	Log LogConfig `json:"log"`
}

type ServerConfig struct {
	Host string `json:"host"`

	Port int `json:"port"`

	ReadTimeoutSeconds int `json:"read_timeout_seconds"`

	WriteTimeoutSeconds int `json:"write_timeout_seconds"`

	ShutdownTimeoutSeconds int `json:"shutdown_timeout_seconds"`
}

type StorageConfig struct {
	DataDir string `json:"data_dir"`

	DocumentsFile string `json:"documents_file"`

	IndexFile string `json:"index_file"`

	TagsFile string `json:"tags_file"`

	CategoriesFile string `json:"categories_file"`

	StatsFile string `json:"stats_file"`

	AutoSave bool `json:"auto_save"`
}

type SearchConfig struct {
	MaxPageSize int `json:"max_page_size"`

	DefaultPageSize int `json:"default_page_size"`

	BM25K1 float64 `json:"bm25_k1"`
	BM25B  float64 `json:"bm25_b"`
}

type UploadConfig struct {
	MaxUploadBytes int64 `json:"max_upload_bytes"`

	AllowedFormats []string `json:"allowed_formats"`
}

type LogConfig struct {
	Level string `json:"level"`

	Output string `json:"output"`
}

var (
	mu     sync.RWMutex
	global = defaultConfig()
)

func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return global
}

func Set(cfg Config) {
	mu.Lock()
	defer mu.Unlock()
	global = cfg
}

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

func defaultConfigPath() string {
	if p := os.Getenv("BENZHI_CONFIG"); p != "" {
		return p
	}
	return "./config.json"
}

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
			MaxPageSize:     100,
			DefaultPageSize: 10,
			BM25K1:          1.2,
			BM25B:           0.75,
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
