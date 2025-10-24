package improvements

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ==========================================
// 示例4: 监控和指标收集
// ==========================================

// Metrics 爬虫指标
type Metrics struct {
	// 基础计数器（使用 atomic 保证并发安全）
	totalRequests   int64
	successRequests int64
	failedRequests  int64
	totalURLs       int64
	totalForms      int64
	totalAPIs       int64
	totalLinks      int64
	
	// 时间统计
	startTime       time.Time
	totalDuration   time.Duration
	
	// 响应时间统计
	responseTimes   []time.Duration
	mu              sync.RWMutex
	
	// HTTP状态码统计
	statusCodes     map[int]int64
	statusMu        sync.RWMutex
	
	// 错误统计
	errorTypes      map[string]int64
	errorMu         sync.RWMutex
}

// NewMetrics 创建指标收集器
func NewMetrics() *Metrics {
	return &Metrics{
		startTime:    time.Now(),
		statusCodes:  make(map[int]int64),
		errorTypes:   make(map[string]int64),
		responseTimes: make([]time.Duration, 0, 1000),
	}
}

// IncrementRequests 增加请求计数
func (m *Metrics) IncrementRequests() {
	atomic.AddInt64(&m.totalRequests, 1)
}

// IncrementSuccess 增加成功计数
func (m *Metrics) IncrementSuccess() {
	atomic.AddInt64(&m.successRequests, 1)
}

// IncrementFailure 增加失败计数
func (m *Metrics) IncrementFailure() {
	atomic.AddInt64(&m.failedRequests, 1)
}

// AddURL 添加URL计数
func (m *Metrics) AddURL(count int) {
	atomic.AddInt64(&m.totalURLs, int64(count))
}

// AddForm 添加表单计数
func (m *Metrics) AddForm(count int) {
	atomic.AddInt64(&m.totalForms, int64(count))
}

// AddAPI 添加API计数
func (m *Metrics) AddAPI(count int) {
	atomic.AddInt64(&m.totalAPIs, int64(count))
}

// AddLink 添加链接计数
func (m *Metrics) AddLink(count int) {
	atomic.AddInt64(&m.totalLinks, int64(count))
}

// RecordResponseTime 记录响应时间
func (m *Metrics) RecordResponseTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responseTimes = append(m.responseTimes, duration)
}

// RecordStatusCode 记录HTTP状态码
func (m *Metrics) RecordStatusCode(code int) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.statusCodes[code]++
}

// RecordError 记录错误类型
func (m *Metrics) RecordError(errType string) {
	m.errorMu.Lock()
	defer m.errorMu.Unlock()
	m.errorTypes[errType]++
}

// GetSnapshot 获取指标快照
func (m *Metrics) GetSnapshot() *MetricsSnapshot {
	m.mu.RLock()
	m.statusMu.RLock()
	m.errorMu.RLock()
	defer m.mu.RUnlock()
	defer m.statusMu.RUnlock()
	defer m.errorMu.RUnlock()
	
	elapsed := time.Since(m.startTime)
	
	snapshot := &MetricsSnapshot{
		TotalRequests:   atomic.LoadInt64(&m.totalRequests),
		SuccessRequests: atomic.LoadInt64(&m.successRequests),
		FailedRequests:  atomic.LoadInt64(&m.failedRequests),
		TotalURLs:       atomic.LoadInt64(&m.totalURLs),
		TotalForms:      atomic.LoadInt64(&m.totalForms),
		TotalAPIs:       atomic.LoadInt64(&m.totalAPIs),
		TotalLinks:      atomic.LoadInt64(&m.totalLinks),
		Elapsed:         elapsed,
		StatusCodes:     make(map[int]int64),
		ErrorTypes:      make(map[string]int64),
	}
	
	// 复制状态码统计
	for code, count := range m.statusCodes {
		snapshot.StatusCodes[code] = count
	}
	
	// 复制错误类型统计
	for errType, count := range m.errorTypes {
		snapshot.ErrorTypes[errType] = count
	}
	
	// 计算响应时间统计
	if len(m.responseTimes) > 0 {
		snapshot.AvgResponseTime = m.calculateAverage()
		snapshot.MinResponseTime = m.calculateMin()
		snapshot.MaxResponseTime = m.calculateMax()
		snapshot.P50ResponseTime = m.calculatePercentile(0.5)
		snapshot.P95ResponseTime = m.calculatePercentile(0.95)
		snapshot.P99ResponseTime = m.calculatePercentile(0.99)
	}
	
	// 计算速率
	if elapsed.Seconds() > 0 {
		snapshot.RequestsPerSec = float64(snapshot.TotalRequests) / elapsed.Seconds()
		snapshot.URLsPerSec = float64(snapshot.TotalURLs) / elapsed.Seconds()
	}
	
	return snapshot
}

// calculateAverage 计算平均响应时间
func (m *Metrics) calculateAverage() time.Duration {
	if len(m.responseTimes) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range m.responseTimes {
		total += d
	}
	return total / time.Duration(len(m.responseTimes))
}

// calculateMin 计算最小响应时间
func (m *Metrics) calculateMin() time.Duration {
	if len(m.responseTimes) == 0 {
		return 0
	}
	min := m.responseTimes[0]
	for _, d := range m.responseTimes {
		if d < min {
			min = d
		}
	}
	return min
}

// calculateMax 计算最大响应时间
func (m *Metrics) calculateMax() time.Duration {
	if len(m.responseTimes) == 0 {
		return 0
	}
	max := m.responseTimes[0]
	for _, d := range m.responseTimes {
		if d > max {
			max = d
		}
	}
	return max
}

// calculatePercentile 计算百分位数（简化版）
func (m *Metrics) calculatePercentile(p float64) time.Duration {
	if len(m.responseTimes) == 0 {
		return 0
	}
	
	// 简化：不排序，仅作示例
	// 生产环境应使用更高效的算法
	index := int(float64(len(m.responseTimes)) * p)
	if index >= len(m.responseTimes) {
		index = len(m.responseTimes) - 1
	}
	return m.responseTimes[index]
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalURLs       int64
	TotalForms      int64
	TotalAPIs       int64
	TotalLinks      int64
	
	Elapsed         time.Duration
	RequestsPerSec  float64
	URLsPerSec      float64
	
	AvgResponseTime time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
	P50ResponseTime time.Duration
	P95ResponseTime time.Duration
	P99ResponseTime time.Duration
	
	StatusCodes     map[int]int64
	ErrorTypes      map[string]int64
}

// Print 打印指标
func (s *MetricsSnapshot) Print() {
	fmt.Println("\n" + "="*70)
	fmt.Println("                       爬虫运行指标")
	fmt.Println("="*70)
	
	// 基础统计
	fmt.Printf("总请求数:        %d\n", s.TotalRequests)
	fmt.Printf("成功请求:        %d (%.1f%%)\n", 
		s.SuccessRequests, 
		float64(s.SuccessRequests)/float64(s.TotalRequests)*100)
	fmt.Printf("失败请求:        %d (%.1f%%)\n", 
		s.FailedRequests, 
		float64(s.FailedRequests)/float64(s.TotalRequests)*100)
	fmt.Println()
	
	// 发现统计
	fmt.Printf("发现URL数:       %d\n", s.TotalURLs)
	fmt.Printf("发现表单数:      %d\n", s.TotalForms)
	fmt.Printf("发现API数:       %d\n", s.TotalAPIs)
	fmt.Printf("发现链接数:      %d\n", s.TotalLinks)
	fmt.Println()
	
	// 性能统计
	fmt.Printf("运行时间:        %.2f秒\n", s.Elapsed.Seconds())
	fmt.Printf("请求速率:        %.2f 请求/秒\n", s.RequestsPerSec)
	fmt.Printf("URL发现速率:     %.2f URL/秒\n", s.URLsPerSec)
	fmt.Println()
	
	// 响应时间统计
	if s.AvgResponseTime > 0 {
		fmt.Printf("平均响应时间:    %v\n", s.AvgResponseTime)
		fmt.Printf("最小响应时间:    %v\n", s.MinResponseTime)
		fmt.Printf("最大响应时间:    %v\n", s.MaxResponseTime)
		fmt.Printf("P50响应时间:     %v\n", s.P50ResponseTime)
		fmt.Printf("P95响应时间:     %v\n", s.P95ResponseTime)
		fmt.Printf("P99响应时间:     %v\n", s.P99ResponseTime)
		fmt.Println()
	}
	
	// HTTP状态码分布
	if len(s.StatusCodes) > 0 {
		fmt.Println("HTTP状态码分布:")
		for code, count := range s.StatusCodes {
			fmt.Printf("  %d: %d (%.1f%%)\n", 
				code, count, 
				float64(count)/float64(s.TotalRequests)*100)
		}
		fmt.Println()
	}
	
	// 错误类型分布
	if len(s.ErrorTypes) > 0 {
		fmt.Println("错误类型分布:")
		for errType, count := range s.ErrorTypes {
			fmt.Printf("  %s: %d (%.1f%%)\n", 
				errType, count, 
				float64(count)/float64(s.FailedRequests)*100)
		}
	}
	
	fmt.Println("="*70)
}

// ==========================================
// 使用示例
// ==========================================

func ExampleMetrics() {
	// 创建指标收集器
	metrics := NewMetrics()
	
	// 模拟爬取过程
	for i := 0; i < 100; i++ {
		metrics.IncrementRequests()
		
		// 模拟请求
		start := time.Now()
		time.Sleep(time.Millisecond * 10)
		elapsed := time.Since(start)
		
		metrics.RecordResponseTime(elapsed)
		
		// 随机成功/失败
		if i%10 != 0 {
			metrics.IncrementSuccess()
			metrics.RecordStatusCode(200)
			metrics.AddURL(3)
			metrics.AddLink(5)
		} else {
			metrics.IncrementFailure()
			metrics.RecordStatusCode(500)
			metrics.RecordError("timeout")
		}
		
		// 定期打印指标
		if (i+1)%25 == 0 {
			snapshot := metrics.GetSnapshot()
			snapshot.Print()
		}
	}
	
	// 最终指标
	finalSnapshot := metrics.GetSnapshot()
	finalSnapshot.Print()
}

