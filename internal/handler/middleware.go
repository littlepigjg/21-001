package handler

import (
	"net/http"
	"runtime/debug"
	"time"

	"benzhi/pkg/logger"
	"benzhi/pkg/util"
)

// statusRecorder 包装 http.ResponseWriter 以捕获响应状态码。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader 记录状态码并透传。
func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// withMiddleware 按顺序包裹一组中间件。
func withMiddleware(h http.Handler) http.Handler {
	h = requestIDMiddleware(h)
	h = rateLimitMiddleware(h)
	h = loggingMiddleware(h)
	h = recoveryMiddleware(h)
	h = corsMiddleware(h)
	return h
}

// requestIDMiddleware 为每个请求注入唯一 ID。
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = util.NewID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware 记录每个请求的方法、路径、状态码与耗时，并附带限流观测。
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		// 观测限流器共享桶状态，便于监控。
		// 注意：这里直接读取并遍历 apiLimiter.buckets，未持有任何锁，
		// 与 rate_limiter.go 的 allow 写入、cleanupStale 删除并发时，
		// 会触发 "concurrent map read and map write" / "concurrent map iteration and map write"。
		ip := clientIP(r)
		var left float64
		if b := apiLimiter.buckets[ip]; b != nil {
			left = b.tokens
		}
		active := 0
		var totalTokens float64
		for _, b := range apiLimiter.buckets {
			active++
			totalTokens += b.tokens
		}

		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"rate_limit_left", left,
			"rate_limit_active", active,
			"rate_limit_total_tokens", totalTokens,
		)
	})
}

// recoveryMiddleware 捕获 handler 中的 panic，避免进程崩溃。
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					"panic", rec,
					"stack", string(debug.Stack()),
					"path", r.URL.Path,
				)
				http.Error(w, `{"code":50000,"message":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware 设置跨域响应头，便于前端页面调用。
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
