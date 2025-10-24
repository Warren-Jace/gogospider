package core

import (
	"errors"
	"fmt"
)

// 定义常见错误类型
var (
	ErrInvalidURL     = errors.New("无效的URL")
	ErrTimeout        = errors.New("请求超时")
	ErrRateLimited    = errors.New("请求被限流")
	ErrForbidden      = errors.New("访问被禁止")
	ErrNotFound       = errors.New("资源不存在")
	ErrServerError    = errors.New("服务器错误")
	ErrNetworkError   = errors.New("网络错误")
	ErrParseError     = errors.New("解析错误")
	ErrDuplicate      = errors.New("URL已处理过")
	ErrClosed         = errors.New("爬虫已关闭")
)

// CrawlError 爬虫错误（带上下文信息）
type CrawlError struct {
	URL        string // 出错的URL
	Err        error  // 原始错误
	StatusCode int    // HTTP状态码
	Retryable  bool   // 是否可重试
}

// Error 实现 error 接口
func (e *CrawlError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("爬取失败 [%s]: %v (状态码: %d)", e.URL, e.Err, e.StatusCode)
	}
	return fmt.Sprintf("爬取失败 [%s]: %v", e.URL, e.Err)
}

// Unwrap 实现 errors.Unwrap 接口
func (e *CrawlError) Unwrap() error {
	return e.Err
}

// NewCrawlError 创建爬虫错误
func NewCrawlError(url string, err error, statusCode int) *CrawlError {
	return &CrawlError{
		URL:        url,
		Err:        err,
		StatusCode: statusCode,
		Retryable:  isRetryable(statusCode, err),
	}
}

// isRetryable 判断错误是否可重试
func isRetryable(statusCode int, err error) bool {
	// HTTP 状态码判断
	if statusCode == 429 || statusCode == 503 || statusCode == 504 || 
	   (statusCode >= 500 && statusCode < 600) {
		return true
	}
	
	// 错误类型判断
	if errors.Is(err, ErrTimeout) || errors.Is(err, ErrNetworkError) {
		return true
	}
	
	return false
}

// IsRetryable 检查错误是否可重试
func IsRetryable(err error) bool {
	var crawlErr *CrawlError
	if errors.As(err, &crawlErr) {
		return crawlErr.Retryable
	}
	
	// 检查已知的可重试错误
	return errors.Is(err, ErrTimeout) || errors.Is(err, ErrNetworkError) || 
	       errors.Is(err, ErrRateLimited)
}

