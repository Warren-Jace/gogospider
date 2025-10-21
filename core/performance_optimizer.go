package core

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// PerformanceOptimizer 性能优化器
type PerformanceOptimizer struct {
	// 对象池
	bufferPool    *sync.Pool
	requestPool   *sync.Pool
	responsePool  *sync.Pool
	
	// HTTP客户端（带连接池）
	httpClient    *http.Client
	
	// 内存管理
	maxMemoryMB   int64
	currentMemory int64
	memoryMutex   sync.Mutex
	
	// 统计信息
	stats         *PerformanceStats
}

// PerformanceStats 性能统计
type PerformanceStats struct {
	BufferPoolHits   int64
	BufferPoolMisses int64
	RequestPoolHits  int64
	RequestPoolMisses int64
	ConnectionReused int64
	ConnectionCreated int64
	TotalRequests    int64
	AvgResponseTime  time.Duration
	mutex            sync.Mutex
}

// NewPerformanceOptimizer 创建性能优化器
func NewPerformanceOptimizer(maxMemoryMB int64) *PerformanceOptimizer {
	po := &PerformanceOptimizer{
		maxMemoryMB: maxMemoryMB,
		stats:       &PerformanceStats{},
	}
	
	// 初始化Buffer对象池
	po.bufferPool = &sync.Pool{
		New: func() interface{} {
			po.stats.mutex.Lock()
			po.stats.BufferPoolMisses++
			po.stats.mutex.Unlock()
			return new(bytes.Buffer)
		},
	}
	
	// 初始化Request对象池
	po.requestPool = &sync.Pool{
		New: func() interface{} {
			po.stats.mutex.Lock()
			po.stats.RequestPoolMisses++
			po.stats.mutex.Unlock()
			return &http.Request{}
		},
	}
	
	// 初始化Response对象池
	po.responsePool = &sync.Pool{
		New: func() interface{} {
			return &http.Response{}
		},
	}
	
	// 创建优化的HTTP客户端
	po.httpClient = po.createOptimizedHTTPClient()
	
	return po
}

// createOptimizedHTTPClient 创建优化的HTTP客户端
func (po *PerformanceOptimizer) createOptimizedHTTPClient() *http.Client {
	// 自定义Transport以实现连接池优化
	transport := &http.Transport{
		// 连接池配置
		MaxIdleConns:        100,              // 最大空闲连接数
		MaxIdleConnsPerHost: 20,               // 每个Host的最大空闲连接数
		MaxConnsPerHost:     50,               // 每个Host的最大连接数
		IdleConnTimeout:     90 * time.Second, // 空闲连接超时
		
		// 拨号器配置
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, // 连接超时
			KeepAlive: 30 * time.Second, // Keep-Alive探测间隔
		}).DialContext,
		
		// 其他优化
		ForceAttemptHTTP2:     true,                // 启用HTTP/2
		TLSHandshakeTimeout:   10 * time.Second,    // TLS握手超时
		ExpectContinueTimeout: 1 * time.Second,     // 100-continue超时
		ResponseHeaderTimeout: 30 * time.Second,    // 响应头超时
		DisableCompression:    false,               // 启用压缩
		DisableKeepAlives:     false,               // 启用Keep-Alive
	}
	
	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second, // 整体请求超时
	}
}

// GetBuffer 从对象池获取Buffer
func (po *PerformanceOptimizer) GetBuffer() *bytes.Buffer {
	po.stats.mutex.Lock()
	po.stats.BufferPoolHits++
	po.stats.mutex.Unlock()
	
	buf := po.bufferPool.Get().(*bytes.Buffer)
	buf.Reset() // 重置Buffer
	return buf
}

// PutBuffer 归还Buffer到对象池
func (po *PerformanceOptimizer) PutBuffer(buf *bytes.Buffer) {
	// 限制Buffer大小，避免内存泄漏
	if buf.Cap() > 64*1024 { // 64KB
		return // 过大的Buffer不回收
	}
	po.bufferPool.Put(buf)
}

// GetRequest 从对象池获取Request（预留，实际使用http.NewRequest）
func (po *PerformanceOptimizer) GetRequest() *http.Request {
	po.stats.mutex.Lock()
	po.stats.RequestPoolHits++
	po.stats.mutex.Unlock()
	
	return po.requestPool.Get().(*http.Request)
}

// PutRequest 归还Request到对象池
func (po *PerformanceOptimizer) PutRequest(req *http.Request) {
	po.requestPool.Put(req)
}

// DoRequest 执行HTTP请求（使用连接池）
func (po *PerformanceOptimizer) DoRequest(req *http.Request) (*http.Response, error) {
	po.stats.mutex.Lock()
	po.stats.TotalRequests++
	po.stats.mutex.Unlock()
	
	start := time.Now()
	resp, err := po.httpClient.Do(req)
	duration := time.Since(start)
	
	// 更新平均响应时间
	po.stats.mutex.Lock()
	if po.stats.AvgResponseTime == 0 {
		po.stats.AvgResponseTime = duration
	} else {
		// 计算移动平均
		po.stats.AvgResponseTime = (po.stats.AvgResponseTime + duration) / 2
	}
	po.stats.mutex.Unlock()
	
	return resp, err
}

// DoRequestWithContext 执行带超时控制的HTTP请求
func (po *PerformanceOptimizer) DoRequestWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	return po.DoRequest(req)
}

// AllocateMemory 分配内存（带限制）
func (po *PerformanceOptimizer) AllocateMemory(sizeMB int64) bool {
	po.memoryMutex.Lock()
	defer po.memoryMutex.Unlock()
	
	if po.currentMemory+sizeMB > po.maxMemoryMB {
		return false // 超过内存限制
	}
	
	po.currentMemory += sizeMB
	return true
}

// ReleaseMemory 释放内存
func (po *PerformanceOptimizer) ReleaseMemory(sizeMB int64) {
	po.memoryMutex.Lock()
	defer po.memoryMutex.Unlock()
	
	po.currentMemory -= sizeMB
	if po.currentMemory < 0 {
		po.currentMemory = 0
	}
}

// GetMemoryUsage 获取当前内存使用
func (po *PerformanceOptimizer) GetMemoryUsage() (current int64, max int64, percentage float64) {
	po.memoryMutex.Lock()
	defer po.memoryMutex.Unlock()
	
	current = po.currentMemory
	max = po.maxMemoryMB
	
	if max > 0 {
		percentage = float64(current) / float64(max) * 100
	}
	
	return
}

// GetStatistics 获取性能统计
func (po *PerformanceOptimizer) GetStatistics() map[string]interface{} {
	po.stats.mutex.Lock()
	defer po.stats.mutex.Unlock()
	
	stats := make(map[string]interface{})
	
	// 对象池统计
	bufferTotal := po.stats.BufferPoolHits + po.stats.BufferPoolMisses
	if bufferTotal > 0 {
		stats["buffer_pool_hit_rate"] = float64(po.stats.BufferPoolHits) / float64(bufferTotal) * 100
	} else {
		stats["buffer_pool_hit_rate"] = 0.0
	}
	stats["buffer_pool_hits"] = po.stats.BufferPoolHits
	stats["buffer_pool_misses"] = po.stats.BufferPoolMisses
	
	// 请求统计
	stats["total_requests"] = po.stats.TotalRequests
	stats["avg_response_time_ms"] = po.stats.AvgResponseTime.Milliseconds()
	
	// 内存统计
	current, max, percentage := po.GetMemoryUsage()
	stats["memory_current_mb"] = current
	stats["memory_max_mb"] = max
	stats["memory_usage_percent"] = percentage
	
	return stats
}

// Reset 重置统计
func (po *PerformanceOptimizer) Reset() {
	po.stats.mutex.Lock()
	defer po.stats.mutex.Unlock()
	
	po.stats.BufferPoolHits = 0
	po.stats.BufferPoolMisses = 0
	po.stats.RequestPoolHits = 0
	po.stats.RequestPoolMisses = 0
	po.stats.ConnectionReused = 0
	po.stats.ConnectionCreated = 0
	po.stats.TotalRequests = 0
	po.stats.AvgResponseTime = 0
	
	po.memoryMutex.Lock()
	po.currentMemory = 0
	po.memoryMutex.Unlock()
}

// Close 关闭并清理资源
func (po *PerformanceOptimizer) Close() {
	if po.httpClient != nil {
		po.httpClient.CloseIdleConnections()
	}
}

// OptimizedReadCloser 优化的ReadCloser（使用Buffer池）
type OptimizedReadCloser struct {
	buffer *bytes.Buffer
	po     *PerformanceOptimizer
}

// Read 实现io.Reader接口
func (orc *OptimizedReadCloser) Read(p []byte) (n int, err error) {
	return orc.buffer.Read(p)
}

// Close 实现io.Closer接口
func (orc *OptimizedReadCloser) Close() error {
	if orc.po != nil && orc.buffer != nil {
		orc.po.PutBuffer(orc.buffer)
	}
	return nil
}

// WrapWithOptimizedReader 包装为优化的Reader
func (po *PerformanceOptimizer) WrapWithOptimizedReader(data []byte) *OptimizedReadCloser {
	buf := po.GetBuffer()
	buf.Write(data)
	
	return &OptimizedReadCloser{
		buffer: buf,
		po:     po,
	}
}

// BatchProcessor 批处理器（用于批量处理URL）
type BatchProcessor struct {
	batchSize int
	processor func([]string) error
	buffer    []string
	mutex     sync.Mutex
}

// NewBatchProcessor 创建批处理器
func NewBatchProcessor(batchSize int, processor func([]string) error) *BatchProcessor {
	return &BatchProcessor{
		batchSize: batchSize,
		processor: processor,
		buffer:    make([]string, 0, batchSize),
	}
}

// Add 添加项目到批处理
func (bp *BatchProcessor) Add(item string) error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	bp.buffer = append(bp.buffer, item)
	
	// 达到批处理大小，执行处理
	if len(bp.buffer) >= bp.batchSize {
		return bp.flush()
	}
	
	return nil
}

// Flush 刷新缓冲区
func (bp *BatchProcessor) Flush() error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	return bp.flush()
}

// flush 内部刷新方法（无锁）
func (bp *BatchProcessor) flush() error {
	if len(bp.buffer) == 0 {
		return nil
	}
	
	err := bp.processor(bp.buffer)
	bp.buffer = bp.buffer[:0] // 清空buffer但保留容量
	
	return err
}

// ConnectionPoolMonitor 连接池监控器
type ConnectionPoolMonitor struct {
	transport *http.Transport
	interval  time.Duration
	stopChan  chan struct{}
}

// NewConnectionPoolMonitor 创建连接池监控器
func NewConnectionPoolMonitor(transport *http.Transport, interval time.Duration) *ConnectionPoolMonitor {
	return &ConnectionPoolMonitor{
		transport: transport,
		interval:  interval,
		stopChan:  make(chan struct{}),
	}
}

// Start 开始监控
func (cpm *ConnectionPoolMonitor) Start() {
	ticker := time.NewTicker(cpm.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// 定期清理空闲连接
			cpm.transport.CloseIdleConnections()
			
		case <-cpm.stopChan:
			return
		}
	}
}

// Stop 停止监控
func (cpm *ConnectionPoolMonitor) Stop() {
	close(cpm.stopChan)
}

// MemoryLimiter 内存限制器
type MemoryLimiter struct {
	maxBytes  int64
	current   int64
	mutex     sync.Mutex
	waitQueue chan struct{}
}

// NewMemoryLimiter 创建内存限制器
func NewMemoryLimiter(maxMB int64) *MemoryLimiter {
	return &MemoryLimiter{
		maxBytes:  maxMB * 1024 * 1024,
		waitQueue: make(chan struct{}, 1),
	}
}

// Acquire 获取内存
func (ml *MemoryLimiter) Acquire(bytes int64) bool {
	ml.mutex.Lock()
	defer ml.mutex.Unlock()
	
	if ml.current+bytes > ml.maxBytes {
		return false
	}
	
	ml.current += bytes
	return true
}

// Release 释放内存
func (ml *MemoryLimiter) Release(bytes int64) {
	ml.mutex.Lock()
	defer ml.mutex.Unlock()
	
	ml.current -= bytes
	if ml.current < 0 {
		ml.current = 0
	}
	
	// 通知等待的协程
	select {
	case ml.waitQueue <- struct{}{}:
	default:
	}
}

// GetUsage 获取使用情况
func (ml *MemoryLimiter) GetUsage() (current int64, max int64, percent float64) {
	ml.mutex.Lock()
	defer ml.mutex.Unlock()
	
	current = ml.current
	max = ml.maxBytes
	percent = float64(current) / float64(max) * 100
	
	return
}

