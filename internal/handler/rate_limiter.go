package handler

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimiter 是基于令牌桶算法的简单限流器。
type rateLimiter struct {
	mu    sync.Mutex
	rate  float64 // 每秒补充令牌数
	burst float64 // 桶容量
	buckets map[string]*bucket
}

// bucket 表示单个客户端（按 IP）的令牌桶。
type bucket struct {
	tokens float64
	last   time.Time
}

// newRateLimiter 创建一个限流器。
func newRateLimiter(rate, burst int) *rateLimiter {
	return &rateLimiter{
		rate:    float64(rate),
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
	}
}

// allow 判断指定 key 是否允许通过，并消耗一个令牌。
func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, last: now}
		rl.buckets[key] = b
	}
	// 按流逝时间补充令牌。
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// apiLimiter 是全局 API 限流器。
var apiLimiter = newRateLimiter(100, 200)

// rateLimitMiddleware 按客户端 IP 限制请求频率。
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !apiLimiter.allow(clientIP(r)) {
			http.Error(w, `{"code":42900,"message":"请求过于频繁"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP 提取客户端 IP，优先考虑代理转发的 X-Forwarded-For。
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
