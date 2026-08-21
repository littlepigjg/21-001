// 企业知识库全文检索系统 —— 服务入口。
//
// 该文件负责：加载配置、初始化日志与存储、构建 HTTP 服务，并在收到
// SIGINT/SIGTERM 信号时执行优雅关闭。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"benzhi/internal/config"
	"benzhi/internal/handler"
	"benzhi/internal/service"
	"benzhi/internal/store"
	"benzhi/pkg/logger"
)

func main() {
	configPath := flag.String("config", "", "配置文件路径（JSON）")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		// 配置文件缺失或损坏时回退到默认配置，保证服务可启动。
		cfg = config.Get()
		fmt.Fprintf(os.Stderr, "警告：加载配置失败，使用默认配置（%v）\n", err)
	}

	if err := initLogger(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败：%v\n", err)
		os.Exit(1)
	}

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		logger.Error("初始化存储失败", "err", err)
		os.Exit(1)
	}
	if err := st.Load(); err != nil {
		logger.Error("加载持久化数据失败", "err", err)
		os.Exit(1)
	}

	svc := service.New(st, cfg)
	h := handler.NewHandler(svc)
	router := handler.NewRouter(h)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}

	// 在独立 goroutine 中启动服务，主协程等待退出信号。
	go func() {
		logger.Info("服务启动", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("服务异常退出", "err", err)
			os.Exit(1)
		}
	}()

	// 优雅关闭：监听 SIGINT / SIGTERM。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("收到退出信号，开始优雅关闭", "signal", sig.String())
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeoutSeconds)*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP 服务关闭失败", "err", err)
	}
	if err := st.Close(); err != nil {
		logger.Error("存储落盘失败", "err", err)
	}
	logger.Info("服务已退出")
}

// initLogger 根据配置初始化全局日志器的级别与输出目标。
func initLogger(cfg config.Config) error {
	logger.SetLevel(logger.ParseLevel(cfg.Log.Level))

	switch cfg.Log.Output {
	case "", "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		f, err := os.OpenFile(cfg.Log.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("打开日志文件失败: %w", err)
		}
		logger.SetOutput(f)
	}
	return nil
}
