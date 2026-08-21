package service

import (
	"fmt"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/store"
)

// TestBugNil011_StatsNilMapIncrementView 验证缺陷 benzhi-nil-011：
// 新文档首次浏览时，统计 map 未初始化，向 nil map 自增会触发 panic。
//
// 缺陷存在（未修复）时：svc.IncrementView 触发 panic，打印 RED 并使测试失败。
// 缺陷修复后：svc.IncrementView 正常返回，打印 GREEN，测试通过。
func TestBugNil011_StatsNilMapIncrementView(t *testing.T) {
	st, err := store.NewStore(testBugNil011StorageConfig(t))
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	svc := New(st, config.Config{})

	var (
		panicked bool
		panicVal any
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicVal = r
			}
		}()
		_ = svc.IncrementView("doc-new-001")
	}()

	if panicked {
		fmt.Printf("RED（红灯，缺陷未修复）：新文档首次浏览触发 panic，panic=%v\n", panicVal)
		t.Fatalf("缺陷存在：期望新文档首次浏览正常，但发生 panic: %v", panicVal)
	}

	stats := svc.Store().GetStats("doc-new-001")
	fmt.Printf("GREEN（绿灯，缺陷已修复）：新文档首次浏览统计自增正常，ViewCount=%d\n", stats.ViewCount)
}

func testBugNil011StorageConfig(t *testing.T) config.StorageConfig {
	return config.StorageConfig{
		DataDir:  t.TempDir(),
		AutoSave: false,
	}
}
