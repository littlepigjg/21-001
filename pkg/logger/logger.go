package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota

	LevelInfo

	LevelWarn

	LevelError
)

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

type Logger struct {
	mu    sync.Mutex
	out   io.Writer
	level Level
}

var defaultLogger = New(LevelInfo, os.Stdout)

func New(level Level, out io.Writer) *Logger {
	return &Logger{level: level, out: out}
}

func SetOutput(out io.Writer) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.out = out
}

func SetLevel(level Level) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = level
}

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

		fmt.Fprintf(l.out, `{"level":"error","msg":"日志序列化失败: %s"}`+"\n", err.Error())
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.out.Write(append(data, '\n'))
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.log(LevelDebug, msg, pairs(fields...))
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	l.log(LevelInfo, msg, pairs(fields...))
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.log(LevelWarn, msg, pairs(fields...))
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	l.log(LevelError, msg, pairs(fields...))
}

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

func Debug(msg string, fields ...interface{}) { defaultLogger.Debug(msg, fields...) }
func Info(msg string, fields ...interface{})  { defaultLogger.Info(msg, fields...) }
func Warn(msg string, fields ...interface{})  { defaultLogger.Warn(msg, fields...) }
func Error(msg string, fields ...interface{}) { defaultLogger.Error(msg, fields...) }
