package core

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// SmartRetryStrategy 智能重试策略
// 🔧 修复：提供指数退避重试和自适应超时
type SmartRetryStrategy struct {
	mutex sync.RWMutex
	
	// 重试配置
	maxRetries        int           // 最大重试次数
	baseTimeout       time.Duration // 基础超时时间
	maxTimeout        time.Duration // 最大超时时间
	backoffMultiplier float64       // 退避倍数
	
	// 自适应配置
	enableAdaptive    bool          // 是否启用自适应超时
	targetSuccessRate float64       // 目标成功率
	
	// 统计数据
	totalRequests     int           // 总请求数
	successRequests   int           // 成功请求数
	failedRequests    int           // 失败请求数
	totalRetries      int           // 总重试次数
	
	// 响应时间统计（用于自适应超时）
	responseTimes     []time.Duration // 最近的响应时间
	maxHistorySize    int             // 保留的历史记录数
	avgResponseTime   time.Duration   // 平均响应时间
}

// RetryDecision 重试决策
type RetryDecision struct {
	ShouldRetry bool          // 是否应该重试
	Delay       time.Duration // 重试延迟
	Timeout     time.Duration // 本次请求超时
	Reason      string        // 决策原因
}

// NewSmartRetryStrategy 创建智能重试策略
func NewSmartRetryStrategy() *SmartRetryStrategy {
	return &SmartRetryStrategy{
		maxRetries:        3,
		baseTimeout:       30 * time.Second,
		maxTimeout:        120 * time.Second,
		backoffMultiplier: 2.0,
		enableAdaptive:    true,
		targetSuccessRate: 0.90, // 目标成功率90%
		totalRequests:     0,
		successRequests:   0,
		failedRequests:    0,
		totalRetries:      0,
		responseTimes:     make([]time.Duration, 0, 100),
		maxHistorySize:    100,
		avgResponseTime:   30 * time.Second,
	}
}

// SetMaxRetries 设置最大重试次数
func (srs *SmartRetryStrategy) SetMaxRetries(max int) {
	srs.mutex.Lock()
	defer srs.mutex.Unlock()
	if max >= 0 {
		srs.maxRetries = max
	}
}

// SetBaseTimeout 设置基础超时时间
func (srs *SmartRetryStrategy) SetBaseTimeout(timeout time.Duration) {
	srs.mutex.Lock()
	defer srs.mutex.Unlock()
	if timeout > 0 {
		srs.baseTimeout = timeout
	}
}

// SetEnableAdaptive 设置是否启用自适应
func (srs *SmartRetryStrategy) SetEnableAdaptive(enable bool) {
	srs.mutex.Lock()
	defer srs.mutex.Unlock()
	srs.enableAdaptive = enable
}

// ShouldRetry 判断是否应该重试
// 参数：
//   - attemptNum: 当前尝试次数（1,2,3...）
//   - err: 错误信息
// 返回：重试决策
func (srs *SmartRetryStrategy) ShouldRetry(attemptNum int, err error) RetryDecision {
	srs.mutex.Lock()
	defer srs.mutex.Unlock()
	
	// 如果达到最大重试次数
	if attemptNum > srs.maxRetries {
		return RetryDecision{
			ShouldRetry: false,
			Delay:       0,
			Timeout:     srs.getCurrentTimeout(),
			Reason:      fmt.Sprintf("达到最大重试次数 (%d)", srs.maxRetries),
		}
	}
	
	// 检查错误类型是否可重试
	if !srs.isRetryableError(err) {
		return RetryDecision{
			ShouldRetry: false,
			Delay:       0,
			Timeout:     srs.getCurrentTimeout(),
			Reason:      "不可重试的错误类型",
		}
	}
	
	// 计算退避延迟（指数退避）
	delay := srs.calculateBackoffDelay(attemptNum)
	
	// 计算本次请求的超时时间
	timeout := srs.getCurrentTimeout()
	
	return RetryDecision{
		ShouldRetry: true,
		Delay:       delay,
		Timeout:     timeout,
		Reason:      fmt.Sprintf("第%d次重试（共%d次）", attemptNum, srs.maxRetries),
	}
}

// calculateBackoffDelay 计算退避延迟（指数退避）
func (srs *SmartRetryStrategy) calculateBackoffDelay(attemptNum int) time.Duration {
	// 基础延迟：1秒
	baseDelay := 1.0 * float64(time.Second)
	
	// 指数退避：delay = baseDelay * (multiplier ^ (attemptNum - 1))
	multiplier := math.Pow(srs.backoffMultiplier, float64(attemptNum-1))
	delay := time.Duration(baseDelay * multiplier)
	
	// 添加抖动（jitter）避免惊群效应
	// jitter: ±10%
	jitterFactor := 1.0
	if time.Now().UnixNano()%2 == 0 {
		jitterFactor = 0.9  // -10%
	} else {
		jitterFactor = 1.1  // +10%
	}
	delay = time.Duration(float64(delay) * jitterFactor)
	
	// 限制最大延迟为60秒
	maxDelay := 60 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	
	return delay
}

// isRetryableError 判断错误是否可重试
func (srs *SmartRetryStrategy) isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	
	// 可重试的错误模式
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"network is unreachable",
		"no route to host",
		"i/o timeout",
		"TLS handshake timeout",
		"EOF",
	}
	
	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}
	
	return false
}

// getCurrentTimeout 获取当前超时时间（自适应）
func (srs *SmartRetryStrategy) getCurrentTimeout() time.Duration {
	if !srs.enableAdaptive {
		return srs.baseTimeout
	}
	
	// 基于平均响应时间的自适应超时
	// timeout = avg_response_time * 3 + 10s（缓冲）
	adaptiveTimeout := srs.avgResponseTime*3 + 10*time.Second
	
	// 限制在基础超时和最大超时之间
	if adaptiveTimeout < srs.baseTimeout {
		adaptiveTimeout = srs.baseTimeout
	}
	if adaptiveTimeout > srs.maxTimeout {
		adaptiveTimeout = srs.maxTimeout
	}
	
	return adaptiveTimeout
}

// RecordSuccess 记录成功请求
func (srs *SmartRetryStrategy) RecordSuccess(responseTime time.Duration) {
	srs.mutex.Lock()
	defer srs.mutex.Unlock()
	
	srs.totalRequests++
	srs.successRequests++
	
	// 记录响应时间
	srs.recordResponseTime(responseTime)
}

// RecordFailure 记录失败请求
func (srs *SmartRetryStrategy) RecordFailure(wasRetried bool) {
	srs.mutex.Lock()
	defer srs.mutex.Unlock()
	
	srs.totalRequests++
	srs.failedRequests++
	
	if wasRetried {
		srs.totalRetries++
	}
}

// recordResponseTime 记录响应时间并更新平均值
func (srs *SmartRetryStrategy) recordResponseTime(responseTime time.Duration) {
	// 添加到历史记录
	srs.responseTimes = append(srs.responseTimes, responseTime)
	
	// 保持历史记录大小
	if len(srs.responseTimes) > srs.maxHistorySize {
		srs.responseTimes = srs.responseTimes[1:]
	}
	
	// 计算平均响应时间（移动平均）
	if len(srs.responseTimes) > 0 {
		total := time.Duration(0)
		for _, rt := range srs.responseTimes {
			total += rt
		}
		srs.avgResponseTime = total / time.Duration(len(srs.responseTimes))
	}
}

// GetStatistics 获取统计信息
func (srs *SmartRetryStrategy) GetStatistics() map[string]interface{} {
	srs.mutex.RLock()
	defer srs.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	
	stats["total_requests"] = srs.totalRequests
	stats["success_requests"] = srs.successRequests
	stats["failed_requests"] = srs.failedRequests
	stats["total_retries"] = srs.totalRetries
	
	if srs.totalRequests > 0 {
		successRate := float64(srs.successRequests) / float64(srs.totalRequests)
		stats["success_rate"] = successRate
		stats["success_percent"] = successRate * 100
		
		failRate := float64(srs.failedRequests) / float64(srs.totalRequests)
		stats["fail_rate"] = failRate
		stats["fail_percent"] = failRate * 100
		
		avgRetries := float64(srs.totalRetries) / float64(srs.totalRequests)
		stats["avg_retries_per_request"] = avgRetries
	}
	
	stats["avg_response_time_ms"] = srs.avgResponseTime.Milliseconds()
	stats["current_timeout_ms"] = srs.getCurrentTimeout().Milliseconds()
	stats["base_timeout_ms"] = srs.baseTimeout.Milliseconds()
	stats["adaptive_enabled"] = srs.enableAdaptive
	
	return stats
}

// PrintReport 打印重试策略报告
func (srs *SmartRetryStrategy) PrintReport() {
	srs.mutex.RLock()
	defer srs.mutex.RUnlock()
	
	if srs.totalRequests == 0 {
		return
	}
	
	stats := srs.GetStatistics()
	
	println()
	println("═══════════════════════════════════════════════════════════")
	println("🔄 智能重试策略报告")
	println("═══════════════════════════════════════════════════════════")
	
	println("【请求统计】")
	println("  总请求数:", srs.totalRequests)
	println("  成功请求:", srs.successRequests)
	println("  失败请求:", srs.failedRequests)
	println("  总重试次数:", srs.totalRetries)
	
	if srs.totalRequests > 0 {
		println("\n【成功率】")
		print("  成功率: ")
		print(stats["success_percent"].(float64))
		println("%")
		
		print("  失败率: ")
		print(stats["fail_percent"].(float64))
		println("%")
		
		print("  平均重试次数: ")
		print(stats["avg_retries_per_request"].(float64))
		println()
	}
	
	println("\n【超时配置】")
	println("  自适应超时:", srs.enableAdaptive)
	print("  基础超时: ")
	print(srs.baseTimeout.Seconds())
	println("秒")
	
	print("  当前超时: ")
	print(srs.getCurrentTimeout().Seconds())
	println("秒")
	
	print("  平均响应时间: ")
	print(srs.avgResponseTime.Milliseconds())
	println("ms")
	
	println("\n【重试策略】")
	println("  最大重试次数:", srs.maxRetries)
	print("  退避倍数: ")
	print(srs.backoffMultiplier)
	println()
	
	println("═══════════════════════════════════════════════════════════")
}

// Reset 重置统计
func (srs *SmartRetryStrategy) Reset() {
	srs.mutex.Lock()
	defer srs.mutex.Unlock()
	
	srs.totalRequests = 0
	srs.successRequests = 0
	srs.failedRequests = 0
	srs.totalRetries = 0
	srs.responseTimes = make([]time.Duration, 0, srs.maxHistorySize)
	srs.avgResponseTime = srs.baseTimeout
}

// 辅助函数

// contains 检查字符串是否包含子串（不区分大小写）
func contains(str, substr string) bool {
	str = toLower(str)
	substr = toLower(substr)
	return indexOf(str, substr) >= 0
}

// toLower 转换为小写
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// indexOf 查找子串位置
func indexOf(str, substr string) int {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

