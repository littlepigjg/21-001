// Package logger 提供结构化日志能力。
//
// 日志以 JSON 行格式输出，每行一条记录，包含时间戳、级别、消息以及可选的
// 结构化字段。该实现完全基于标准库，不依赖任何第三方日志库。
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level 表示日志级别。
type Level int

// 预定义日志级别，数值越大级别越高。
const (
	// LevelDebug 用于开发期调试信息。
	LevelDebug Level = iota
	// LevelInfo 用于常规运行信息。
	LevelInfo
	// LevelWarn 用于可恢复的异常提示。
	LevelWarn
	// LevelError 用于需要关注的错误。
	LevelError
)

// String 返回日志级别的字符串表示。
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "info"
	}
}

// ParseLevel 将字符串解析为日志级别，非法值返回 LevelInfo。
func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Logger 是结构化日志器。
type Logger struct {
	mu    sync.Mutex
	out   io.Writer
	level Level
}

// 全局默认日志器，保证在未初始化时也能输出。
var defaultLogger = New(LevelInfo, os.Stdout)

// New 创建一个新的日志器。
func New(level Level, out io.Writer) *Logger {
	return &Logger{level: level, out: out}
}

// SetOutput 设置全局日志器的输出目标。
func SetOutput(out io.Writer) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.out = out
}

// SetLevel 设置全局日志器的级别。
func SetLevel(level Level) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = level
}

// log 是日志输出的统一内部实现。
func (l *Logger) log(level Level, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := map[string]interface{}{
		"time":  time.Now().Format(time.RFC3339Nano),
		"level": level.String(),
		"msg":   msg,
	}
	for k, v := range fields {
		entry[k] = v
	}

	data, err := json.Marshal(entry)
	if err != nil {
		// 极端情况下序列化失败，退化为直接输出消息。
		fmt.Fprintf(l.out, `{"level":"error","msg":"日志序列化失败: %s"}`+"\n", err.Error())
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.out.Write(append(data, '\n'))
}

// Debug 输出调试级别日志。
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.log(LevelDebug, msg, pairs(fields...))
}

// Info 输出信息级别日志。
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.log(LevelInfo, msg, pairs(fields...))
}

// Warn 输出警告级别日志。
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.log(LevelWarn, msg, pairs(fields...))
}

// Error 输出错误级别日志。
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.log(LevelError, msg, pairs(fields...))
}

// pairs 将变长的 key/value 参数转换为 map。
//
// 当参数个数为奇数时，最后一个 key 会以 "missing" 作为值补全，避免 panic。
func pairs(kv ...interface{}) map[string]interface{} {
	fields := make(map[string]interface{}, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", kv[i])
		}
		fields[key] = kv[i+1]
	}
	if len(kv)%2 != 0 {
		fields[fmt.Sprintf("%v", kv[len(kv)-1])] = "missing"
	}
	return fields
}

// 包级便捷函数，使用全局默认日志器。
func Debug(msg string, fields ...interface{}) { defaultLogger.Debug(msg, fields...) }
func Info(msg string, fields ...interface{})  { defaultLogger.Info(msg, fields...) }
func Warn(msg string, fields ...interface{})  { defaultLogger.Warn(msg, fields...) }
func Error(msg string, fields ...interface{}) { defaultLogger.Error(msg, fields...) }
