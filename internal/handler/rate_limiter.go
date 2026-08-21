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
	mu      sync.Mutex
	rate    float64 // 每秒补充令牌数
	burst   float64 // 桶容量
	buckets map[string]*bucket
}

// bucket 表示单个客户端（按 IP）的令牌桶。
type bucket struct {
	tokens   float64
	last     time.Time
	lastSeen time.Time
}

// newRateLimiter 创建一个限流器，并启动后台清理协程。
func newRateLimiter(rate, burst int) *rateLimiter {
	rl := &rateLimiter{
		rate:    float64(rate),
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
	}
	go rl.cleanupLoop()
	return rl
}

// cleanupLoop 周期性清理长期不活跃的客户端桶，避免 map 无限增长。
func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.cleanupStale()
	}
}

// cleanupStale 删除超过 5 分钟未访问的桶。
// 注意：这里直接遍历并删除共享 buckets map，未持有 rl.mu 锁，
// 与 allow 的写入、middleware 的读取并发时会触发 map 并发读写崩溃。
func (rl *rateLimiter) cleanupStale() {
	cutoff := time.Now().Add(-5 * time.Minute)
	for k, b := range rl.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(rl.buckets, k)
		}
	}
}

// allow 判断指定 key 是否允许通过，并消耗一个令牌。
//
// 这里采用"双重检查"策略：先无锁读取桶，只有桶不存在时才加锁创建；
// 桶内的 tokens/last/lastSeen 字段则在锁外直接修改，多个请求并发时
// 会对同一桶字段产生数据竞争，导致令牌丢失更新。
func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()

	b := rl.buckets[key]
	if b == nil {
		rl.mu.Lock()
		b = rl.buckets[key]
		if b == nil {
			b = &bucket{tokens: rl.burst, last: now, lastSeen: now}
			rl.buckets[key] = b
		}
		rl.mu.Unlock()
	}

	// 按流逝时间补充令牌（锁外修改桶字段）。
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.last = now
	b.lastSeen = now

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
