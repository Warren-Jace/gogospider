package improvements

import (
	"errors"
	"fmt"
)

// ==========================================
// 示例3: 改进的错误处理
// ==========================================

// 定义错误类型
var (
	ErrInvalidURL     = errors.New("无效的URL")
	ErrTimeout        = errors.New("请求超时")
	ErrRateLimited    = errors.New("请求被限流")
	ErrForbidden      = errors.New("访问被禁止")
	ErrNotFound       = errors.New("资源不存在")
	ErrServerError    = errors.New("服务器错误")
	ErrNetworkError   = errors.New("网络错误")
	ErrParseError     = errors.New("解析错误")
)

// CrawlError 爬虫错误（带上下文）
type CrawlError struct {
	URL       string
	Err       error
	Retryable bool
	StatusCode int
}

func (e *CrawlError) Error() string {
	return fmt.Sprintf("爬取失败 [%s]: %v (状态码: %d, 可重试: %v)",
		e.URL, e.Err, e.StatusCode, e.Retryable)
}

func (e *CrawlError) Unwrap() error {
	return e.Err
}

// NewCrawlError 创建爬虫错误
func NewCrawlError(url string, err error, statusCode int) *CrawlError {
	return &CrawlError{
		URL:        url,
		Err:        err,
		StatusCode: statusCode,
		Retryable:  isRetryable(statusCode),
	}
}

// isRetryable 判断是否可重试
func isRetryable(statusCode int) bool {
	return statusCode == 429 || statusCode == 503 || statusCode == 504 || statusCode >= 500
}

// ErrorHandler 错误处理器
type ErrorHandler struct {
	logger     Logger
	maxRetries int
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler(logger Logger, maxRetries int) *ErrorHandler {
	return &ErrorHandler{
		logger:     logger,
		maxRetries: maxRetries,
	}
}

// Handle 处理错误
func (h *ErrorHandler) Handle(err error, url string) error {
	if err == nil {
		return nil
	}
	
	// 检查错误类型
	var crawlErr *CrawlError
	if errors.As(err, &crawlErr) {
		return h.handleCrawlError(crawlErr)
	}
	
	// 检查已知错误
	switch {
	case errors.Is(err, ErrTimeout):
		h.logger.Warn("请求超时",
			"url", url,
			"error", err,
			"suggestion", "考虑增加超时时间或使用代理")
		return err
		
	case errors.Is(err, ErrRateLimited):
		h.logger.Warn("触发限流",
			"url", url,
			"error", err,
			"suggestion", "建议降低请求频率")
		return err
		
	case errors.Is(err, ErrForbidden):
		h.logger.Error("访问被禁止",
			"url", url,
			"error", err,
			"suggestion", "检查User-Agent和Cookie")
		return err
		
	default:
		h.logger.Error("未知错误",
			"url", url,
			"error", err)
		return err
	}
}

// handleCrawlError 处理爬虫错误
func (h *ErrorHandler) handleCrawlError(err *CrawlError) error {
	if err.Retryable {
		h.logger.Warn("爬取失败（可重试）",
			"url", err.URL,
			"status_code", err.StatusCode,
			"error", err.Err,
			"max_retries", h.maxRetries)
	} else {
		h.logger.Error("爬取失败（不可重试）",
			"url", err.URL,
			"status_code", err.StatusCode,
			"error", err.Err)
	}
	return err
}

// ==========================================
// 重试机制
// ==========================================

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries int
	Backoff    BackoffStrategy
}

// BackoffStrategy 退避策略
type BackoffStrategy interface {
	Next(attempt int) int // 返回等待时间（秒）
}

// ExponentialBackoff 指数退避
type ExponentialBackoff struct {
	InitialDelay int
	Multiplier   float64
	MaxDelay     int
}

func (b *ExponentialBackoff) Next(attempt int) int {
	delay := float64(b.InitialDelay)
	for i := 0; i < attempt; i++ {
		delay *= b.Multiplier
		if int(delay) > b.MaxDelay {
			return b.MaxDelay
		}
	}
	return int(delay)
}

// WithRetry 执行带重试的操作
func WithRetry(config RetryConfig, fn func() error) error {
	var lastErr error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		// 检查是否可重试
		var crawlErr *CrawlError
		if errors.As(err, &crawlErr) && !crawlErr.Retryable {
			return err // 不可重试，直接返回
		}
		
		if attempt < config.MaxRetries {
			waitTime := config.Backoff.Next(attempt)
			fmt.Printf("第 %d 次重试，等待 %d 秒...\n", attempt+1, waitTime)
			// time.Sleep(time.Duration(waitTime) * time.Second)
		}
	}
	
	return fmt.Errorf("重试 %d 次后仍然失败: %w", config.MaxRetries, lastErr)
}

// ==========================================
// 使用示例
// ==========================================

func ExampleErrorHandling() {
	logger := NewLogger(LevelInfo)
	handler := NewErrorHandler(logger, 3)
	
	// 示例1: 基本错误处理
	err1 := NewCrawlError("https://example.com", ErrTimeout, 0)
	handler.Handle(err1, "https://example.com")
	
	// 示例2: 带重试的爬取
	config := RetryConfig{
		MaxRetries: 3,
		Backoff: &ExponentialBackoff{
			InitialDelay: 1,
			Multiplier:   2,
			MaxDelay:     30,
		},
	}
	
	err := WithRetry(config, func() error {
		// 模拟爬取
		return NewCrawlError("https://example.com", ErrRateLimited, 429)
	})
	
	if err != nil {
		logger.Error("爬取最终失败", "error", err)
	}
	
	// 示例3: 错误分类处理
	urls := []string{
		"invalid://url",
		"https://example.com/forbidden",
		"https://example.com/timeout",
	}
	
	for _, url := range urls {
		var err error
		// 模拟不同的错误
		switch url {
		case "invalid://url":
			err = fmt.Errorf("%w: %s", ErrInvalidURL, url)
		case "https://example.com/forbidden":
			err = NewCrawlError(url, ErrForbidden, 403)
		case "https://example.com/timeout":
			err = NewCrawlError(url, ErrTimeout, 0)
		}
		
		if err != nil {
			handler.Handle(err, url)
		}
	}
}

