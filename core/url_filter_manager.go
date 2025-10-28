package core

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// 核心接口定义
// ============================================================================

// URLFilter 统一的URL过滤器接口
type URLFilter interface {
	// Name 过滤器名称
	Name() string
	
	// Priority 优先级（数字越小越先执行）
	Priority() int
	
	// Filter 执行过滤
	// 返回：是否通过, 过滤原因, 元数据
	Filter(rawURL string, context *FilterContext) FilterResult
	
	// GetStats 获取统计信息
	GetStats() map[string]interface{}
	
	// Reset 重置统计
	Reset()
	
	// Enable 启用/禁用
	SetEnabled(enabled bool)
	IsEnabled() bool
}

// FilterContext 过滤上下文（共享信息，避免重复解析）
type FilterContext struct {
	// 解析后的URL（缓存）
	ParsedURL *url.URL
	
	// 元数据
	Depth         int                    // 当前深度
	Method        string                 // HTTP方法
	TargetDomain  string                 // 目标域名
	SourceType    string                 // 来源类型（html/js/api等）
	CustomData    map[string]interface{} // 自定义数据
	
	// 性能计数
	ParseTime     time.Duration // URL解析时间
	TotalFilters  int          // 执行的过滤器数量
}

// FilterResult 过滤结果
type FilterResult struct {
	Allowed  bool                   // 是否允许
	Action   FilterAction           // 动作（允许/拒绝/降级）
	Reason   string                 // 原因
	Score    float64                // 评分（0-100）
	Metadata map[string]interface{} // 元数据
}

// FilterAction 过滤动作
type FilterAction int

const (
	FilterAllow   FilterAction = iota // 允许
	FilterReject                      // 拒绝
	FilterDegrade                     // 降级（记录但不爬取）
)

// ============================================================================
// URL过滤管理器
// ============================================================================

// URLFilterManager 统一的URL过滤管理器
type URLFilterManager struct {
	mutex sync.RWMutex
	
	// 过滤器管道（按优先级排序）
	filters []URLFilter
	
	// 配置
	config FilterManagerConfig
	
	// 全局统计
	stats FilterManagerStats
	
	// 过滤链路追踪（用于调试）
	traceEnabled bool
	traceBuffer  []FilterTrace
	traceMaxSize int
}

// FilterManagerConfig 过滤管理器配置
type FilterManagerConfig struct {
	// 全局开关
	Enabled bool
	
	// 模式
	Mode FilterMode // Strict/Balanced/Loose
	
	// 性能优化
	EnableCaching    bool // 启用结果缓存
	CacheSize        int  // 缓存大小
	EnableEarlyStop  bool // 早停（第一个拒绝就停止）
	
	// 调试
	EnableTrace      bool // 启用链路追踪
	TraceBufferSize  int  // 追踪缓冲区大小
	VerboseLogging   bool // 详细日志
	
	// 目标域名（用于域名过滤）
	TargetDomain     string
}

// FilterMode 过滤模式
type FilterMode string

const (
	FilterModeStrict   FilterMode = "strict"   // 严格模式（过滤更多）
	FilterModeBalanced FilterMode = "balanced" // 平衡模式（推荐）
	FilterModeLoose    FilterMode = "loose"    // 宽松模式（尽量保留）
)

// FilterManagerStats 过滤管理器统计
type FilterManagerStats struct {
	TotalProcessed   int64         // 处理的URL总数
	TotalAllowed     int64         // 允许的URL数
	TotalRejected    int64         // 拒绝的URL数
	TotalDegraded    int64         // 降级的URL数
	AvgProcessTime   time.Duration // 平均处理时间
	LastUpdateTime   time.Time     // 最后更新时间
	
	// 按过滤器统计
	FilterStats      map[string]FilterStatInfo
}

// FilterStatInfo 单个过滤器统计信息
type FilterStatInfo struct {
	FilterName    string
	TotalChecked  int64
	TotalRejected int64
	AvgTime       time.Duration
}

// FilterTrace 过滤链路追踪
type FilterTrace struct {
	URL       string
	Timestamp time.Time
	Depth     int
	Steps     []FilterTraceStep
	Result    FilterResult
	Duration  time.Duration
}

// FilterTraceStep 追踪步骤
type FilterTraceStep struct {
	FilterName string
	Result     FilterResult
	Duration   time.Duration
}

// ============================================================================
// 构造函数
// ============================================================================

// NewURLFilterManager 创建URL过滤管理器
func NewURLFilterManager(config FilterManagerConfig) *URLFilterManager {
	// 设置默认值
	if config.CacheSize == 0 {
		config.CacheSize = 10000
	}
	if config.TraceBufferSize == 0 {
		config.TraceBufferSize = 100
	}
	if config.Mode == "" {
		config.Mode = FilterModeBalanced
	}
	
	mgr := &URLFilterManager{
		filters:      make([]URLFilter, 0),
		config:       config,
		traceEnabled: config.EnableTrace,
		traceBuffer:  make([]FilterTrace, 0, config.TraceBufferSize),
		traceMaxSize: config.TraceBufferSize,
		stats: FilterManagerStats{
			FilterStats: make(map[string]FilterStatInfo),
		},
	}
	
	return mgr
}

// ============================================================================
// 过滤器管理
// ============================================================================

// RegisterFilter 注册过滤器
func (m *URLFilterManager) RegisterFilter(filter URLFilter) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.filters = append(m.filters, filter)
	
	// 按优先级排序
	sort.Slice(m.filters, func(i, j int) bool {
		return m.filters[i].Priority() < m.filters[j].Priority()
	})
	
	// 初始化统计
	m.stats.FilterStats[filter.Name()] = FilterStatInfo{
		FilterName: filter.Name(),
	}
}

// UnregisterFilter 注销过滤器
func (m *URLFilterManager) UnregisterFilter(filterName string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	newFilters := make([]URLFilter, 0)
	for _, f := range m.filters {
		if f.Name() != filterName {
			newFilters = append(newFilters, f)
		}
	}
	m.filters = newFilters
	
	delete(m.stats.FilterStats, filterName)
}

// GetFilter 获取过滤器
func (m *URLFilterManager) GetFilter(filterName string) URLFilter {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	for _, f := range m.filters {
		if f.Name() == filterName {
			return f
		}
	}
	return nil
}

// ListFilters 列出所有过滤器
func (m *URLFilterManager) ListFilters() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	names := make([]string, 0, len(m.filters))
	for _, f := range m.filters {
		names = append(names, f.Name())
	}
	return names
}

// ============================================================================
// 核心过滤逻辑
// ============================================================================

// Filter 执行过滤（主入口）
func (m *URLFilterManager) Filter(rawURL string, customContext map[string]interface{}) FilterResult {
	startTime := time.Now()
	
	// 全局开关
	if !m.config.Enabled {
		return FilterResult{
			Allowed: true,
			Action:  FilterAllow,
			Reason:  "过滤管理器未启用",
		}
	}
	
	m.mutex.Lock()
	m.stats.TotalProcessed++
	m.mutex.Unlock()
	
	// 创建过滤上下文
	ctx := m.createContext(rawURL, customContext)
	
	// 链路追踪
	var trace *FilterTrace
	if m.traceEnabled {
		trace = &FilterTrace{
			URL:       rawURL,
			Timestamp: startTime,
			Depth:     ctx.Depth,
			Steps:     make([]FilterTraceStep, 0),
		}
	}
	
	// 遍历过滤器管道
	for _, filter := range m.filters {
		// 跳过禁用的过滤器
		if !filter.IsEnabled() {
			continue
		}
		
		stepStart := time.Now()
		ctx.TotalFilters++
		
		// 执行过滤
		result := filter.Filter(rawURL, ctx)
		
		stepDuration := time.Since(stepStart)
		
		// 更新统计
		m.updateFilterStats(filter.Name(), result, stepDuration)
		
		// 记录追踪
		if trace != nil {
			trace.Steps = append(trace.Steps, FilterTraceStep{
				FilterName: filter.Name(),
				Result:     result,
				Duration:   stepDuration,
			})
		}
		
		// 处理结果
		if result.Action == FilterReject {
			// 早停优化
			if m.config.EnableEarlyStop {
				m.recordTrace(trace, result, time.Since(startTime))
				m.mutex.Lock()
				m.stats.TotalRejected++
				m.mutex.Unlock()
				return result
			}
		}
		
		if result.Action == FilterDegrade {
			m.recordTrace(trace, result, time.Since(startTime))
			m.mutex.Lock()
			m.stats.TotalDegraded++
			m.mutex.Unlock()
			return result
		}
	}
	
	// 所有过滤器都通过
	finalResult := FilterResult{
		Allowed: true,
		Action:  FilterAllow,
		Reason:  "通过所有过滤器",
	}
	
	m.recordTrace(trace, finalResult, time.Since(startTime))
	
	m.mutex.Lock()
	m.stats.TotalAllowed++
	
	// 更新平均处理时间
	totalTime := m.stats.AvgProcessTime * time.Duration(m.stats.TotalProcessed-1)
	m.stats.AvgProcessTime = (totalTime + time.Since(startTime)) / time.Duration(m.stats.TotalProcessed)
	m.stats.LastUpdateTime = time.Now()
	m.mutex.Unlock()
	
	return finalResult
}

// FilterBatch 批量过滤
func (m *URLFilterManager) FilterBatch(urls []string, customContext map[string]interface{}) map[string]FilterResult {
	results := make(map[string]FilterResult, len(urls))
	
	for _, u := range urls {
		results[u] = m.Filter(u, customContext)
	}
	
	return results
}

// ShouldCrawl 简化的过滤接口（返回bool）
func (m *URLFilterManager) ShouldCrawl(rawURL string) bool {
	result := m.Filter(rawURL, nil)
	return result.Allowed && result.Action == FilterAllow
}

// ============================================================================
// 辅助方法
// ============================================================================

// createContext 创建过滤上下文
func (m *URLFilterManager) createContext(rawURL string, customContext map[string]interface{}) *FilterContext {
	ctx := &FilterContext{
		CustomData:   customContext,
		TargetDomain: m.config.TargetDomain,
	}
	
	// 解析URL（缓存）
	parseStart := time.Now()
	parsed, err := url.Parse(rawURL)
	if err == nil {
		ctx.ParsedURL = parsed
	}
	ctx.ParseTime = time.Since(parseStart)
	
	// 从自定义上下文提取信息
	if customContext != nil {
		if depth, ok := customContext["depth"].(int); ok {
			ctx.Depth = depth
		}
		if method, ok := customContext["method"].(string); ok {
			ctx.Method = method
		}
		if sourceType, ok := customContext["source_type"].(string); ok {
			ctx.SourceType = sourceType
		}
	}
	
	return ctx
}

// updateFilterStats 更新过滤器统计
func (m *URLFilterManager) updateFilterStats(filterName string, result FilterResult, duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	stats, exists := m.stats.FilterStats[filterName]
	if !exists {
		stats = FilterStatInfo{FilterName: filterName}
	}
	
	stats.TotalChecked++
	if !result.Allowed || result.Action == FilterReject {
		stats.TotalRejected++
	}
	
	// 更新平均时间
	totalTime := stats.AvgTime * time.Duration(stats.TotalChecked-1)
	stats.AvgTime = (totalTime + duration) / time.Duration(stats.TotalChecked)
	
	m.stats.FilterStats[filterName] = stats
}

// recordTrace 记录追踪
func (m *URLFilterManager) recordTrace(trace *FilterTrace, result FilterResult, duration time.Duration) {
	if !m.traceEnabled || trace == nil {
		return
	}
	
	trace.Result = result
	trace.Duration = duration
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// 添加到缓冲区
	if len(m.traceBuffer) >= m.traceMaxSize {
		// 移除最旧的
		m.traceBuffer = m.traceBuffer[1:]
	}
	m.traceBuffer = append(m.traceBuffer, *trace)
}

// ============================================================================
// 查询和调试
// ============================================================================

// GetStatistics 获取统计信息
func (m *URLFilterManager) GetStatistics() FilterManagerStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.stats
}

// GetRecentTraces 获取最近的追踪记录
func (m *URLFilterManager) GetRecentTraces(limit int) []FilterTrace {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if limit <= 0 || limit > len(m.traceBuffer) {
		limit = len(m.traceBuffer)
	}
	
	// 返回最近的记录
	start := len(m.traceBuffer) - limit
	traces := make([]FilterTrace, limit)
	copy(traces, m.traceBuffer[start:])
	
	return traces
}

// ExplainURL 解释URL为什么被过滤（详细调试）
func (m *URLFilterManager) ExplainURL(rawURL string) string {
	// 临时启用追踪
	oldTrace := m.traceEnabled
	m.traceEnabled = true
	defer func() { m.traceEnabled = oldTrace }()
	
	// 执行过滤
	result := m.Filter(rawURL, nil)
	
	// 获取最后一条追踪
	traces := m.GetRecentTraces(1)
	if len(traces) == 0 {
		return "无追踪信息"
	}
	
	trace := traces[0]
	
	// 构建说明
	var sb strings.Builder
	sb.WriteString("═══════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("URL: %s\n", rawURL))
	sb.WriteString(fmt.Sprintf("最终结果: %s (%s)\n", result.Reason, formatAction(result.Action)))
	sb.WriteString(fmt.Sprintf("处理时间: %v\n", trace.Duration))
	sb.WriteString(fmt.Sprintf("执行过滤器数: %d\n", len(trace.Steps)))
	sb.WriteString("═══════════════════════════════════════════════════════════════\n")
	sb.WriteString("过滤链路:\n")
	
	for i, step := range trace.Steps {
		icon := "✓"
		if !step.Result.Allowed || step.Result.Action != FilterAllow {
			icon = "✗"
		}
		
		sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, icon, step.FilterName))
		sb.WriteString(fmt.Sprintf("     动作: %s\n", formatAction(step.Result.Action)))
		sb.WriteString(fmt.Sprintf("     原因: %s\n", step.Result.Reason))
		if step.Result.Score > 0 {
			sb.WriteString(fmt.Sprintf("     评分: %.1f\n", step.Result.Score))
		}
		sb.WriteString(fmt.Sprintf("     耗时: %v\n", step.Duration))
	}
	
	sb.WriteString("═══════════════════════════════════════════════════════════════\n")
	
	return sb.String()
}

// PrintStatistics 打印统计信息
func (m *URLFilterManager) PrintStatistics() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	stats := m.stats
	
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              URL过滤管理器 - 统计报告                         ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ 模式: %-10s | 启用: %-5v | 早停: %-5v              ║\n",
		m.config.Mode, m.config.Enabled, m.config.EnableEarlyStop)
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	
	// 总体统计
	total := stats.TotalProcessed
	if total > 0 {
		allowRate := float64(stats.TotalAllowed) / float64(total) * 100
		rejectRate := float64(stats.TotalRejected) / float64(total) * 100
		degradeRate := float64(stats.TotalDegraded) / float64(total) * 100
		
		fmt.Printf("║ 总处理:   %-10d | 平均耗时: %-10v               ║\n", 
			total, stats.AvgProcessTime)
		fmt.Printf("║ 允许:     %-10d (%.1f%%)                              ║\n", 
			stats.TotalAllowed, allowRate)
		fmt.Printf("║ 拒绝:     %-10d (%.1f%%)                              ║\n", 
			stats.TotalRejected, rejectRate)
		fmt.Printf("║ 降级:     %-10d (%.1f%%)                              ║\n", 
			stats.TotalDegraded, degradeRate)
	} else {
		fmt.Println("║ 暂无数据                                                       ║")
	}
	
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Println("║ 过滤器详情                                                     ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	
	// 按拒绝率排序
	filterStats := make([]FilterStatInfo, 0, len(stats.FilterStats))
	for _, fs := range stats.FilterStats {
		filterStats = append(filterStats, fs)
	}
	sort.Slice(filterStats, func(i, j int) bool {
		return filterStats[i].TotalRejected > filterStats[j].TotalRejected
	})
	
	for _, fs := range filterStats {
		if fs.TotalChecked > 0 {
			rejectRate := float64(fs.TotalRejected) / float64(fs.TotalChecked) * 100
			fmt.Printf("║ • %-20s                                     ║\n", fs.FilterName)
			fmt.Printf("║   检查: %-8d | 拒绝: %-8d (%.1f%%) | %8v  ║\n",
				fs.TotalChecked, fs.TotalRejected, rejectRate, fs.AvgTime)
		}
	}
	
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

// ResetStatistics 重置统计
func (m *URLFilterManager) ResetStatistics() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.stats = FilterManagerStats{
		FilterStats: make(map[string]FilterStatInfo),
	}
	
	// 重置所有过滤器的统计
	for _, filter := range m.filters {
		filter.Reset()
		m.stats.FilterStats[filter.Name()] = FilterStatInfo{
			FilterName: filter.Name(),
		}
	}
}

// ============================================================================
// 辅助函数
// ============================================================================

// formatAction 格式化动作
func formatAction(action FilterAction) string {
	switch action {
	case FilterAllow:
		return "允许"
	case FilterReject:
		return "拒绝"
	case FilterDegrade:
		return "降级"
	default:
		return "未知"
	}
}

// ============================================================================
// 便捷方法
// ============================================================================

// SetMode 设置过滤模式
func (m *URLFilterManager) SetMode(mode FilterMode) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.Mode = mode
}

// EnableFilter 启用过滤器
func (m *URLFilterManager) EnableFilter(filterName string) {
	filter := m.GetFilter(filterName)
	if filter != nil {
		filter.SetEnabled(true)
	}
}

// DisableFilter 禁用过滤器
func (m *URLFilterManager) DisableFilter(filterName string) {
	filter := m.GetFilter(filterName)
	if filter != nil {
		filter.SetEnabled(false)
	}
}

