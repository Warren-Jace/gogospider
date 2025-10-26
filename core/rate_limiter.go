package core

import (
	"context"
	"sync"
	"time"
)

// RateLimiter 速率限制器（参考Katana设计）
type RateLimiter struct {
	// 配置
	requestsPerSecond int           // 每秒请求数
	burstSize         int           // 突发大小
	requestDelay      time.Duration // 每个请求之间的延迟
	
	// 状态
	ticker      *time.Ticker
	tokens      chan struct{}
	lastRequest time.Time
	mutex       sync.Mutex
	
	// 统计
	totalRequests   int64
	blockedRequests int64
	
	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

// RateLimiterConfig 速率限制器配置
type RateLimiterConfig struct {
	RequestsPerSecond int           // 每秒最大请求数（0表示不限制）
	BurstSize         int           // 允许的突发请求数
	MinDelay          time.Duration // 最小请求间隔
	MaxDelay          time.Duration // 最大请求间隔
	Enabled           bool          // 是否启用
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if !config.Enabled {
		return nil
	}
	
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 100 // 默认100 req/s
	}
	
	if config.BurstSize <= 0 {
		config.BurstSize = config.RequestsPerSecond / 10 // 默认10%突发
		if config.BurstSize < 1 {
			config.BurstSize = 1
		}
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	rl := &RateLimiter{
		requestsPerSecond: config.RequestsPerSecond,
		burstSize:         config.BurstSize,
		requestDelay:      config.MinDelay,
		tokens:            make(chan struct{}, config.BurstSize),
		lastRequest:       time.Now(),
		ctx:               ctx,
		cancel:            cancel,
	}
	
	// 初始化token bucket
	for i := 0; i < config.BurstSize; i++ {
		rl.tokens <- struct{}{}
	}
	
	// 启动token补充器
	go rl.refillTokens()
	
	return rl
}

// Wait 等待获取许可（阻塞直到可以发送请求）
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if rl == nil {
		return nil // 未启用速率限制
	}
	
	rl.mutex.Lock()
	rl.totalRequests++
	rl.mutex.Unlock()
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.ctx.Done():
		return rl.ctx.Err()
	case <-rl.tokens:
		// 获取到token
		
		// 如果配置了最小延迟，确保延迟
		if rl.requestDelay > 0 {
			rl.mutex.Lock()
			timeSinceLastRequest := time.Since(rl.lastRequest)
			if timeSinceLastRequest < rl.requestDelay {
				sleepTime := rl.requestDelay - timeSinceLastRequest
				rl.mutex.Unlock()
				
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(sleepTime):
				}
				
				rl.mutex.Lock()
			}
			rl.lastRequest = time.Now()
			rl.mutex.Unlock()
		}
		
		return nil
	}
}

// TryAcquire 尝试获取许可（非阻塞）
func (rl *RateLimiter) TryAcquire() bool {
	if rl == nil {
		return true
	}
	
	select {
	case <-rl.tokens:
		rl.mutex.Lock()
		rl.totalRequests++
		rl.lastRequest = time.Now()
		rl.mutex.Unlock()
		return true
	default:
		rl.mutex.Lock()
		rl.blockedRequests++
		rl.mutex.Unlock()
		return false
	}
}

// refillTokens 补充tokens（后台goroutine）
func (rl *RateLimiter) refillTokens() {
	// 计算补充间隔
	interval := time.Second / time.Duration(rl.requestsPerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-rl.ctx.Done():
			return
		case <-ticker.C:
			// 尝试补充一个token
			select {
			case rl.tokens <- struct{}{}:
				// 成功补充
			default:
				// token bucket已满，跳过
			}
		}
	}
}

// GetStats 获取统计信息
func (rl *RateLimiter) GetStats() RateLimiterStats {
	if rl == nil {
		return RateLimiterStats{}
	}
	
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	return RateLimiterStats{
		TotalRequests:   rl.totalRequests,
		BlockedRequests: rl.blockedRequests,
		CurrentTokens:   len(rl.tokens),
		MaxTokens:       rl.burstSize,
	}
}

// RateLimiterStats 速率限制器统计
type RateLimiterStats struct {
	TotalRequests   int64
	BlockedRequests int64
	CurrentTokens   int
	MaxTokens       int
}

// Stop 停止速率限制器
func (rl *RateLimiter) Stop() {
	if rl == nil {
		return
	}
	rl.cancel()
}

// AdaptiveRateLimiter 自适应速率限制器
type AdaptiveRateLimiter struct {
	*RateLimiter
	
	// 自适应参数
	minRate         int           // 最小速率
	maxRate         int           // 最大速率
	currentRate     int           // 当前速率
	
	// 错误追踪
	errorCount      int           // 错误计数
	successCount    int           // 成功计数
	errorThreshold  int           // 错误阈值
	
	// 调整参数
	increaseStep    int           // 增加步长
	decreaseStep    int           // 减少步长
	adjustInterval  time.Duration // 调整间隔
	
	mutex           sync.Mutex
}

// NewAdaptiveRateLimiter 创建自适应速率限制器
func NewAdaptiveRateLimiter(minRate, maxRate int) *AdaptiveRateLimiter {
	initialRate := (minRate + maxRate) / 2
	
	config := RateLimiterConfig{
		RequestsPerSecond: initialRate,
		BurstSize:         initialRate / 10,
		Enabled:           true,
	}
	
	return &AdaptiveRateLimiter{
		RateLimiter:    NewRateLimiter(config),
		minRate:        minRate,
		maxRate:        maxRate,
		currentRate:    initialRate,
		errorThreshold: 10,
		increaseStep:   5,
		decreaseStep:   10,
		adjustInterval: 10 * time.Second,
	}
}

// ReportError 报告错误（会降低速率）
func (arl *AdaptiveRateLimiter) ReportError() {
	arl.mutex.Lock()
	defer arl.mutex.Unlock()
	
	arl.errorCount++
	arl.successCount = 0 // 重置成功计数
	
	// 如果错误过多，降低速率
	if arl.errorCount >= arl.errorThreshold {
		arl.decreaseRate()
		arl.errorCount = 0
	}
}

// ReportSuccess 报告成功（会增加速率）
func (arl *AdaptiveRateLimiter) ReportSuccess() {
	arl.mutex.Lock()
	defer arl.mutex.Unlock()
	
	arl.successCount++
	arl.errorCount = 0 // 重置错误计数
	
	// 如果持续成功，尝试增加速率
	if arl.successCount >= 50 {
		arl.increaseRate()
		arl.successCount = 0
	}
}

// decreaseRate 降低速率
func (arl *AdaptiveRateLimiter) decreaseRate() {
	newRate := arl.currentRate - arl.decreaseStep
	if newRate < arl.minRate {
		newRate = arl.minRate
	}
	
	if newRate != arl.currentRate {
		arl.currentRate = newRate
		arl.updateRate(newRate)
	}
}

// increaseRate 增加速率
func (arl *AdaptiveRateLimiter) increaseRate() {
	newRate := arl.currentRate + arl.increaseStep
	if newRate > arl.maxRate {
		newRate = arl.maxRate
	}
	
	if newRate != arl.currentRate {
		arl.currentRate = newRate
		arl.updateRate(newRate)
	}
}

// updateRate 更新速率（重新创建RateLimiter）
func (arl *AdaptiveRateLimiter) updateRate(newRate int) {
	// 停止旧的限制器
	if arl.RateLimiter != nil {
		arl.RateLimiter.Stop()
	}
	
	// 创建新的限制器
	config := RateLimiterConfig{
		RequestsPerSecond: newRate,
		BurstSize:         newRate / 10,
		Enabled:           true,
	}
	arl.RateLimiter = NewRateLimiter(config)
}

// GetCurrentRate 获取当前速率
func (arl *AdaptiveRateLimiter) GetCurrentRate() int {
	arl.mutex.Lock()
	defer arl.mutex.Unlock()
	return arl.currentRate
}

