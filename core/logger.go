package core

import (
	"io"
	"log/slog"
	"os"
)

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// SlogLogger 基于 slog 的实现
type SlogLogger struct {
	logger *slog.Logger
}

// NewLogger 创建新的日志记录器
func NewLogger(level slog.Level, output io.Writer) Logger {
	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 自定义时间格式
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format("2006-01-02 15:04:05"))
			}
			return a
		},
	})
	
	return &SlogLogger{
		logger: slog.New(handler),
	}
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{
		logger: l.logger.With(args...),
	}
}

// 全局日志实例
var DefaultLogger Logger

func init() {
	DefaultLogger = NewLogger(slog.LevelInfo, os.Stdout)
}

// 便捷方法
func Debug(msg string, args ...any) { DefaultLogger.Debug(msg, args...) }
func Info(msg string, args ...any)  { DefaultLogger.Info(msg, args...) }
func Warn(msg string, args ...any)  { DefaultLogger.Warn(msg, args...) }
func Error(msg string, args ...any) { DefaultLogger.Error(msg, args...) }

