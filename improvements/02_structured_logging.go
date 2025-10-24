package improvements

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// ==========================================
// 示例2: 结构化日志系统
// ==========================================

// LogLevel 日志级别
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// Logger 日志接口（便于测试和替换）
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
func NewLogger(level LogLevel) Logger {
	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}
	
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 自定义时间格式
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
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

// ==========================================
// Spider 集成示例
// ==========================================

// SpiderWithLogger 带日志的爬虫
type SpiderWithLogger struct {
	logger Logger
	// ... 其他字段
}

// NewSpiderWithLogger 创建带日志的爬虫
func NewSpiderWithLogger(logLevel LogLevel) *SpiderWithLogger {
	return &SpiderWithLogger{
		logger: NewLogger(logLevel),
	}
}

// Crawl 爬取示例（带结构化日志）
func (s *SpiderWithLogger) Crawl(ctx context.Context, url string) error {
	// 创建带上下文的 logger
	logger := s.logger.With(
		"url", url,
		"correlation_id", generateCorrelationID(),
	)
	
	logger.Info("开始爬取",
		"depth", 3,
		"timeout", "30s",
	)
	
	startTime := time.Now()
	
	// 模拟爬取
	err := s.doRequest(ctx, url)
	
	elapsed := time.Since(startTime)
	
	if err != nil {
		logger.Error("爬取失败",
			"error", err,
			"elapsed", elapsed.Seconds(),
			"retry_count", 0,
		)
		return err
	}
	
	logger.Info("爬取成功",
		"elapsed", elapsed.Seconds(),
		"links_found", 42,
		"forms_found", 5,
	)
	
	return nil
}

// doRequest 执行请求（示例）
func (s *SpiderWithLogger) doRequest(ctx context.Context, url string) error {
	s.logger.Debug("发送HTTP请求",
		"method", "GET",
		"url", url,
		"user_agent", "Spider-Bot/1.0",
	)
	
	// 模拟请求
	time.Sleep(100 * time.Millisecond)
	
	return nil
}

// generateCorrelationID 生成关联ID
func generateCorrelationID() string {
	return time.Now().Format("20060102150405")
}

// ==========================================
// 使用示例
// ==========================================

func ExampleLogging() {
	// 1. 创建日志记录器
	logger := NewLogger(LevelInfo)
	
	// 2. 基本使用
	logger.Info("应用启动", "version", "2.5")
	logger.Debug("调试信息", "var", "value") // 不会输出（级别不够）
	logger.Warn("警告信息", "reason", "配置缺失")
	logger.Error("错误信息", "error", "连接失败")
	
	// 3. 带上下文的日志
	requestLogger := logger.With(
		"request_id", "req-123",
		"user_id", "user-456",
	)
	requestLogger.Info("处理请求", "path", "/api/data")
	
	// 4. 在 Spider 中使用
	spider := NewSpiderWithLogger(LevelDebug)
	ctx := context.Background()
	spider.Crawl(ctx, "https://example.com")
	
	// 输出示例：
	// {"timestamp":"2024-10-24T16:00:00+08:00","level":"INFO","msg":"开始爬取","url":"https://example.com","correlation_id":"20241024160000","depth":3,"timeout":"30s"}
	// {"timestamp":"2024-10-24T16:00:00+08:00","level":"DEBUG","msg":"发送HTTP请求","method":"GET","url":"https://example.com","user_agent":"Spider-Bot/1.0"}
	// {"timestamp":"2024-10-24T16:00:00+08:00","level":"INFO","msg":"爬取成功","elapsed":0.1,"links_found":42,"forms_found":5}
}

