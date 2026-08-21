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
// 遍历与删除均持有 rl.mu，避免与 allow 的写入并发触发
// "concurrent map iteration and map write"。
// （Go 允许在 range 循环中 delete 当前 key，无需额外处理。）
func (rl *rateLimiter) cleanupStale() {
	cutoff := time.Now().Add(-5 * time.Minute)
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k, b := range rl.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(rl.buckets, k)
		}
	}
}

// allow 判断指定 key 是否允许通过，并消耗一个令牌。
//
// 整个"取桶-补令牌-扣令牌-写回"过程都持有 rl.mu，避免：
//   - 无锁读 map 与写 map 并发触发 "concurrent map read and map write"；
//   - 桶字段在锁外修改导致令牌丢失更新（放行数超出桶容量、计数乱跳）。
func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	b := rl.buckets[key]
	if b == nil {
		b = &bucket{tokens: rl.burst, last: now, lastSeen: now}
		rl.buckets[key] = b
	}

	// 按流逝时间补充令牌。
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

// snapshot 返回用于观测的桶统计快照（指定 key 的剩余令牌、活跃桶数、令牌总量）。
// 持有 rl.mu 读取，避免与 allow/cleanupStale 并发触发 map 并发读写。
func (rl *rateLimiter) snapshot(key string) (left float64, active int, totalTokens float64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if b, ok := rl.buckets[key]; ok {
		left = b.tokens
	}
	for _, b := range rl.buckets {
		active++
		totalTokens += b.tokens
	}
	return left, active, totalTokens
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
