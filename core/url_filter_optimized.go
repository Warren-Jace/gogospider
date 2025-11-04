package core

import (
	"hash/fnv"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
	
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/bits-and-blooms/bloom/v3"
)

// ============================================================================
// 优化1: 智能FilterContext - 解决URL重复解析问题
// ============================================================================

// OptimizedFilterContext 优化的过滤上下文
// 核心优化: URL只解析一次, 结果被缓存并在所有过滤器间共享
type OptimizedFilterContext struct {
	// 原始URL
	rawURL string
	
	// 解析结果(延迟初始化, 只解析一次)
	parsedURL  *url.URL
	parseError error
	parseOnce  sync.Once
	
	// 预缓存的常用字段(避免重复访问)
	hostname    string
	path        string
	queryParams url.Values
	scheme      string
	
	// 元数据
	Depth      int
	Method     string
	SourceType string
	CustomData map[string]interface{}
	
	// 性能统计
	ParseTime    time.Duration
	TotalFilters int
	StartTime    time.Time
}

// NewOptimizedFilterContext 创建优化的上下文
func NewOptimizedFilterContext(rawURL string) *OptimizedFilterContext {
	return &OptimizedFilterContext{
		rawURL:     rawURL,
		CustomData: make(map[string]interface{}),
		StartTime:  time.Now(),
	}
}

// GetParsedURL 获取解析后的URL (只解析一次, 线程安全)
func (ctx *OptimizedFilterContext) GetParsedURL() (*url.URL, error) {
	ctx.parseOnce.Do(func() {
		start := time.Now()
		ctx.parsedURL, ctx.parseError = url.Parse(ctx.rawURL)
		ctx.ParseTime = time.Since(start)
		
		// 预缓存常用字段
		if ctx.parsedURL != nil {
			ctx.hostname = ctx.parsedURL.Hostname()
			ctx.path = ctx.parsedURL.Path
			ctx.queryParams = ctx.parsedURL.Query()
			ctx.scheme = ctx.parsedURL.Scheme
		}
	})
	return ctx.parsedURL, ctx.parseError
}

// Hostname 快速获取主机名 (无需检查错误)
func (ctx *OptimizedFilterContext) Hostname() string {
	ctx.GetParsedURL()
	return ctx.hostname
}

// Path 快速获取路径
func (ctx *OptimizedFilterContext) Path() string {
	ctx.GetParsedURL()
	return ctx.path
}

// QueryParams 快速获取查询参数
func (ctx *OptimizedFilterContext) QueryParams() url.Values {
	ctx.GetParsedURL()
	return ctx.queryParams
}

// Scheme 快速获取协议
func (ctx *OptimizedFilterContext) Scheme() string {
	ctx.GetParsedURL()
	return ctx.scheme
}

// ElapsedTime 获取总处理时间
func (ctx *OptimizedFilterContext) ElapsedTime() time.Duration {
	return time.Since(ctx.StartTime)
}

// ============================================================================
// 优化2: FilterContext对象池 - 减少内存分配
// ============================================================================

var filterContextPool = sync.Pool{
	New: func() interface{} {
		return &OptimizedFilterContext{
			CustomData: make(map[string]interface{}),
		}
	},
}

// AcquireFilterContext 从池中获取上下文对象
func AcquireFilterContext(rawURL string) *OptimizedFilterContext {
	ctx := filterContextPool.Get().(*OptimizedFilterContext)
	
	// 重置状态
	ctx.rawURL = rawURL
	ctx.parsedURL = nil
	ctx.parseError = nil
	ctx.parseOnce = sync.Once{}
	ctx.hostname = ""
	ctx.path = ""
	ctx.queryParams = nil
	ctx.scheme = ""
	ctx.TotalFilters = 0
	ctx.StartTime = time.Now()
	
	// 清空CustomData但复用底层map
	for k := range ctx.CustomData {
		delete(ctx.CustomData, k)
	}
	
	return ctx
}

// ReleaseFilterContext 归还上下文对象到池
func ReleaseFilterContext(ctx *OptimizedFilterContext) {
	filterContextPool.Put(ctx)
}

// ============================================================================
// 优化3: 分段锁Filter - 解决全局锁竞争
// ============================================================================

const (
	DefaultShardCount = 256 // 256个分片, 锁竞争降低256倍
)

// ShardedURLFilter 分片过滤器
// 核心优化: 使用256个独立的分片, 每个分片有自己的锁
type ShardedURLFilter struct {
	shards    [DefaultShardCount]*FilterShardOptimized
	numShards int
	
	// 只读配置 (不需要锁)
	config interface{}
	name   string
}

// FilterShardOptimized 单个优化分片 (renamed to avoid conflict)
type FilterShardOptimized struct {
	mu   sync.RWMutex  // 读写锁 (读操作不互斥)
	data map[string]interface{}
	
	// 分片级别的统计
	totalChecked  atomic.Uint64
	totalAllowed  atomic.Uint64
	totalRejected atomic.Uint64
}

// NewShardedURLFilter 创建分片过滤器
func NewShardedURLFilter(name string) *ShardedURLFilter {
	f := &ShardedURLFilter{
		numShards: DefaultShardCount,
		name:      name,
	}
	
	// 初始化所有分片
	for i := 0; i < DefaultShardCount; i++ {
		f.shards[i] = &FilterShardOptimized{
			data: make(map[string]interface{}),
		}
	}
	
	return f
}

// getShard 根据URL hash选择分片
func (f *ShardedURLFilter) getShard(url string) *FilterShardOptimized {
	// 使用FNV-1a hash (快速且分布均匀)
	h := fnv.New32a()
	h.Write([]byte(url))
	index := h.Sum32() & uint32(DefaultShardCount-1) // 位运算比取模快
	return f.shards[index]
}

// Get 获取数据 (使用读锁)
func (f *ShardedURLFilter) Get(url string) (interface{}, bool) {
	shard := f.getShard(url)
	
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	
	value, exists := shard.data[url]
	return value, exists
}

// Set 设置数据 (使用写锁)
func (f *ShardedURLFilter) Set(url string, value interface{}) {
	shard := f.getShard(url)
	
	shard.mu.Lock()
	defer shard.mu.Unlock()
	
	shard.data[url] = value
}

// GetOrCompute 获取或计算 (double-check模式)
func (f *ShardedURLFilter) GetOrCompute(url string, compute func() interface{}) (interface{}, bool) {
	shard := f.getShard(url)
	
	// 快速路径: 使用读锁
	shard.mu.RLock()
	value, exists := shard.data[url]
	shard.mu.RUnlock()
	
	if exists {
		return value, false // false表示使用的缓存值
	}
	
	// 慢速路径: 需要计算, 使用写锁
	shard.mu.Lock()
	defer shard.mu.Unlock()
	
	// Double-check (可能在等待锁期间被其他goroutine设置)
	value, exists = shard.data[url]
	if exists {
		return value, false
	}
	
	// 计算新值
	value = compute()
	shard.data[url] = value
	return value, true // true表示是新计算的
}

// GetStats 获取全局统计
func (f *ShardedURLFilter) GetStats() map[string]interface{} {
	var totalChecked, totalAllowed, totalRejected uint64
	
	for i := 0; i < DefaultShardCount; i++ {
		totalChecked += f.shards[i].totalChecked.Load()
		totalAllowed += f.shards[i].totalAllowed.Load()
		totalRejected += f.shards[i].totalRejected.Load()
	}
	
	return map[string]interface{}{
		"total_checked":  totalChecked,
		"total_allowed":  totalAllowed,
		"total_rejected": totalRejected,
		"num_shards":     f.numShards,
	}
}

// ============================================================================
// 优化4: 混合去重策略 - 解决内存无限增长
// ============================================================================

// HybridDeduplicator 混合去重器
// 第1层: Bloom Filter (快速判断, 内存极小, 可能误报)
// 第2层: LRU Cache (精确判断, 内存有界)
// 第3层: 可选的持久化存储
type HybridDeduplicator struct {
	// 第1层: Bloom Filter
	bloomFilter *bloom.BloomFilter
	bloomSize   uint
	bloomFPR    float64
	
	// 第2层: LRU Cache
	recentURLs *lru.Cache[string, bool]
	cacheSize  int
	
	// 统计
	totalChecked    atomic.Uint64
	bloomHits       atomic.Uint64  // Bloom Filter拦截的
	cacheHits       atomic.Uint64  // 缓存命中的
	trueDuplicates  atomic.Uint64  // 真正的重复
	bloomFalsePos   atomic.Uint64  // Bloom Filter误报
	
	mu sync.RWMutex
}

// NewHybridDeduplicator 创建混合去重器
// bloomSize: Bloom Filter容量 (如1000万)
// cachesSize: LRU缓存大小 (如10000)
func NewHybridDeduplicator(bloomSize uint, cacheSize int) *HybridDeduplicator {
	// Bloom Filter: 误报率0.01 (1%)
	bf := bloom.NewWithEstimates(bloomSize, 0.01)
	
	// LRU Cache
	cache, _ := lru.New[string, bool](cacheSize)
	
	return &HybridDeduplicator{
		bloomFilter: bf,
		bloomSize:   bloomSize,
		bloomFPR:    0.01,
		recentURLs:  cache,
		cacheSize:   cacheSize,
	}
}

// IsDuplicate 检查URL是否重复
// 返回: (是否重复, 检查层级)
func (d *HybridDeduplicator) IsDuplicate(url string) (bool, string) {
	d.totalChecked.Add(1)
	
	// 第1层: Bloom Filter快速判断
	// 如果Bloom Filter说不存在, 则绝对不存在 (100%准确)
	if !d.bloomFilter.TestString(url) {
		d.bloomFilter.AddString(url)
		d.bloomHits.Add(1)
		return false, "bloom_miss"
	}
	
	// 到这里, Bloom Filter说"可能存在"
	// 可能是真的存在, 也可能是误报 (1%概率)
	
	// 第2层: LRU缓存精确判断
	d.mu.RLock()
	_, exists := d.recentURLs.Get(url)
	d.mu.RUnlock()
	
	if exists {
		// 确实是重复的
		d.cacheHits.Add(1)
		d.trueDuplicates.Add(1)
		return true, "cache_hit"
	}
	
	// Bloom Filter误报, 实际不是重复
	d.bloomFalsePos.Add(1)
	
	// 加入LRU缓存
	d.mu.Lock()
	d.recentURLs.Add(url, true)
	d.mu.Unlock()
	
	return false, "bloom_false_positive"
}

// Mark 标记URL为已处理 (如果知道不是重复)
func (d *HybridDeduplicator) Mark(url string) {
	d.bloomFilter.AddString(url)
	
	d.mu.Lock()
	d.recentURLs.Add(url, true)
	d.mu.Unlock()
}

// GetStats 获取统计信息
func (d *HybridDeduplicator) GetStats() map[string]interface{} {
	totalChecked := d.totalChecked.Load()
	bloomHits := d.bloomHits.Load()
	cacheHits := d.cacheHits.Load()
	trueDuplicates := d.trueDuplicates.Load()
	bloomFalsePos := d.bloomFalsePos.Load()
	
	var bloomHitRate, cacheHitRate, falsePositiveRate float64
	if totalChecked > 0 {
		bloomHitRate = float64(bloomHits) / float64(totalChecked) * 100
		cacheHitRate = float64(cacheHits) / float64(totalChecked) * 100
		falsePositiveRate = float64(bloomFalsePos) / float64(totalChecked) * 100
	}
	
	return map[string]interface{}{
		"total_checked":         totalChecked,
		"bloom_hits":            bloomHits,
		"cache_hits":            cacheHits,
		"true_duplicates":       trueDuplicates,
		"bloom_false_positives": bloomFalsePos,
		"bloom_hit_rate":        bloomHitRate,
		"cache_hit_rate":        cacheHitRate,
		"false_positive_rate":   falsePositiveRate,
		"bloom_size":            d.bloomSize,
		"cache_size":            d.cacheSize,
		"estimated_memory_mb":   d.estimateMemoryUsage(),
	}
}

// estimateMemoryUsage 估算内存占用
func (d *HybridDeduplicator) estimateMemoryUsage() float64 {
	// Bloom Filter内存: n个元素, m个bit, k个hash函数
	// m = -n*ln(p) / (ln(2)^2)
	// 对于1000万元素, 1%误报率: 约 11.98 MB
	bloomMemoryMB := float64(d.bloomFilter.Cap()) / 8 / 1024 / 1024
	
	// LRU Cache内存: 每个条目约100字节 (key+value+overhead)
	cacheMemoryMB := float64(d.cacheSize) * 100 / 1024 / 1024
	
	return bloomMemoryMB + cacheMemoryMB
}

// ============================================================================
// 优化5: 批量处理API - 提升吞吐量
// ============================================================================

// BatchFilterResult 批量过滤结果
type BatchFilterResult struct {
	URL    string
	Result FilterResult
	Index  int // 原始索引
}

// FilterBatch方法已存在于url_filter_manager.go中，这里移除重复定义
// 如需使用批量过滤，请直接调用URLFilterManager.FilterBatch()

// ============================================================================
// 优化6: 结果缓存 - 避免重复计算
// ============================================================================

// CachedFilterManager 带缓存的过滤管理器
type CachedFilterManager struct {
	*URLFilterManager
	
	// 结果缓存
	resultCache *lru.Cache[string, FilterResult]
	cacheSize   int
	
	// 缓存统计
	cacheHits   atomic.Uint64
	cacheMisses atomic.Uint64
}

// NewCachedFilterManager 创建带缓存的管理器
func NewCachedFilterManager(manager *URLFilterManager, cacheSize int) *CachedFilterManager {
	cache, _ := lru.New[string, FilterResult](cacheSize)
	
	return &CachedFilterManager{
		URLFilterManager: manager,
		resultCache:      cache,
		cacheSize:        cacheSize,
	}
}

// Filter 带缓存的过滤方法
func (m *CachedFilterManager) Filter(rawURL string, context map[string]interface{}) FilterResult {
	// 生成缓存key (包含URL和重要的上下文)
	cacheKey := m.generateCacheKey(rawURL, context)
	
	// 尝试从缓存获取
	if result, ok := m.resultCache.Get(cacheKey); ok {
		m.cacheHits.Add(1)
		return result
	}
	
	// 缓存未命中, 执行真正的过滤
	m.cacheMisses.Add(1)
	
	// 直接使用context参数，不需要转换
	result := m.URLFilterManager.Filter(rawURL, context)
	
	// 存入缓存
	m.resultCache.Add(cacheKey, result)
	
	return result
}

// generateCacheKey 生成缓存key
func (m *CachedFilterManager) generateCacheKey(url string, context map[string]interface{}) string {
	// 简单实现: URL + depth + method
	key := url
	if context != nil {
		if depth, ok := context["depth"].(int); ok {
			key += ":" + string(rune(depth))
		}
		if method, ok := context["method"].(string); ok {
			key += ":" + method
		}
	}
	return key
}

// GetCacheStats 获取缓存统计
func (m *CachedFilterManager) GetCacheStats() map[string]interface{} {
	hits := m.cacheHits.Load()
	misses := m.cacheMisses.Load()
	total := hits + misses
	
	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	
	return map[string]interface{}{
		"cache_hits":     hits,
		"cache_misses":   misses,
		"cache_hit_rate": hitRate,
		"cache_size":     m.cacheSize,
		"cache_len":      m.resultCache.Len(),
	}
}

// ============================================================================
// 性能基准测试辅助
// ============================================================================

// BenchmarkHelper 性能测试辅助
type BenchmarkHelper struct {
	StartTime     time.Time
	URLsProcessed atomic.Uint64
	BytesAllocated atomic.Uint64
}

// NewBenchmarkHelper 创建测试辅助
func NewBenchmarkHelper() *BenchmarkHelper {
	return &BenchmarkHelper{
		StartTime: time.Now(),
	}
}

// RecordURL 记录一个URL处理
func (b *BenchmarkHelper) RecordURL() {
	b.URLsProcessed.Add(1)
}

// GetThroughput 获取吞吐量 (URLs/秒)
func (b *BenchmarkHelper) GetThroughput() float64 {
	elapsed := time.Since(b.StartTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(b.URLsProcessed.Load()) / elapsed
}

// Report 生成性能报告
func (b *BenchmarkHelper) Report() map[string]interface{} {
	elapsed := time.Since(b.StartTime)
	urlsProcessed := b.URLsProcessed.Load()
	
	var throughput float64
	if elapsed.Seconds() > 0 {
		throughput = float64(urlsProcessed) / elapsed.Seconds()
	}
	
	return map[string]interface{}{
		"urls_processed": urlsProcessed,
		"elapsed_sec":    elapsed.Seconds(),
		"throughput":     throughput,
		"avg_time_us":    elapsed.Microseconds() / int64(urlsProcessed),
	}
}

