package core

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	// 创建测试缓冲区
	var buf bytes.Buffer
	logger := NewLogger(slog.LevelDebug, &buf)
	
	// 测试 Info
	logger.Info("test message", "key", "value")
	output := buf.String()
	
	if !strings.Contains(output, "test message") {
		t.Errorf("日志应该包含消息")
	}
	if !strings.Contains(output, "key") {
		t.Errorf("日志应该包含键值对")
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(slog.LevelWarn, &buf)
	
	// Info 应该被过滤
	logger.Info("info message")
	if buf.Len() > 0 {
		t.Errorf("INFO 日志应该被过滤")
	}
	
	// Warn 应该输出
	logger.Warn("warn message")
	if buf.Len() == 0 {
		t.Errorf("WARN 日志应该输出")
	}
}

func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(slog.LevelInfo, &buf)
	
	// 测试 With
	contextLogger := logger.With("request_id", "12345")
	contextLogger.Info("test with context")
	
	output := buf.String()
	if !strings.Contains(output, "request_id") {
		t.Errorf("日志应该包含上下文")
	}
	if !strings.Contains(output, "12345") {
		t.Errorf("日志应该包含上下文值")
	}
}

func TestMultipleLevels(t *testing.T) {
	tests := []struct {
		name     string
		logLevel slog.Level
		logFunc  func(Logger, string)
		shouldLog bool
	}{
		{"Debug with Debug level", slog.LevelDebug, func(l Logger, msg string) { l.Debug(msg) }, true},
		{"Info with Debug level", slog.LevelDebug, func(l Logger, msg string) { l.Info(msg) }, true},
		{"Debug with Info level", slog.LevelInfo, func(l Logger, msg string) { l.Debug(msg) }, false},
		{"Info with Info level", slog.LevelInfo, func(l Logger, msg string) { l.Info(msg) }, true},
		{"Warn with Info level", slog.LevelInfo, func(l Logger, msg string) { l.Warn(msg) }, true},
		{"Error with Warn level", slog.LevelWarn, func(l Logger, msg string) { l.Error(msg) }, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(tt.logLevel, &buf)
			
			tt.logFunc(logger, "test message")
			
			hasOutput := buf.Len() > 0
			if hasOutput != tt.shouldLog {
				t.Errorf("期望输出=%v, 实际输出=%v", tt.shouldLog, hasOutput)
			}
		})
	}
}

