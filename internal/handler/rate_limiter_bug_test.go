package handler

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestBugConcur006_RateLimiterUnlockedMap 验证限流器令牌桶 map 无锁并发访问缺陷。
//
// 缺陷存在时：
//   - rate_limiter.go 的 (*rateLimiter).allow 采用"双重检查"：只在桶不存在时加锁创建，
//     桶内 tokens/last/lastSeen 字段在锁外直接修改，并发调用会丢失令牌更新；
//   - rate_limiter.go 的 cleanupStale 未加锁遍历并删除共享 buckets map；
//   - middleware.go 的 loggingMiddleware 未加锁直接读取并遍历 apiLimiter.buckets；
//   并发调用 allow 会丢失令牌更新，导致桶内剩余令牌不等于 0，判定为 RED。
//
// 缺陷修复后（对共享 map 与桶字段统一加锁保护），令牌计数正确，判定为 GREEN。
func TestBugConcur006_RateLimiterUnlockedMap(t *testing.T) {
	const workers = 20000

	// rate=0 表示不补充令牌，避免时间流逝影响结果。
	rl := newRateLimiter(0, workers)

	const key = "fixed-client"
	// 预插入桶，避免并发 allow 触发不可恢复的 "concurrent map writes" fatal error，
	// 从而只观察令牌丢失更新这一可恢复、可判定的缺陷表现。
	rl.buckets[key] = &bucket{tokens: float64(workers), last: time.Now(), lastSeen: time.Now()}

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = rl.allow(key)
		}()
	}
	close(start)
	wg.Wait()

	left := rl.buckets[key].tokens
	if left != 0 {
		fmt.Printf("RED（红灯，缺陷未修复）：令牌桶丢失更新，剩余令牌=%v（期望 0）\n", left)
		t.Fatalf("rate limiter lost token updates: remaining=%v, want 0", left)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）：令牌桶并发访问安全，剩余令牌=0")
}
