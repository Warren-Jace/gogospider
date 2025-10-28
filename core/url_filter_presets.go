package core

import (
	"strings"
)

// ============================================================================
// URL过滤器预设配置
// ============================================================================

// FilterPreset 过滤器预设
type FilterPreset string

const (
	PresetStrict   FilterPreset = "strict"   // 严格模式：过滤更多，适合大型网站
	PresetBalanced FilterPreset = "balanced" // 平衡模式：默认推荐
	PresetLoose    FilterPreset = "loose"    // 宽松模式：保留更多URL
	PresetAPIOnly  FilterPreset = "api_only" // 只爬取API端点
	PresetDeepScan FilterPreset = "deep_scan" // 深度扫描：保留所有有价值的
)

// NewURLFilterManagerWithPreset 使用预设创建过滤管理器
func NewURLFilterManagerWithPreset(preset FilterPreset, targetDomain string) *URLFilterManager {
	var config FilterManagerConfig
	var filters []URLFilter
	
	switch preset {
	case PresetStrict:
		config = getStrictConfig()
		filters = createStrictFilters(targetDomain)
		
	case PresetBalanced:
		config = getBalancedConfig()
		filters = createBalancedFilters(targetDomain)
		
	case PresetLoose:
		config = getLooseConfig()
		filters = createLooseFilters(targetDomain)
		
	case PresetAPIOnly:
		config = getAPIOnlyConfig()
		filters = createAPIOnlyFilters(targetDomain)
		
	case PresetDeepScan:
		config = getDeepScanConfig()
		filters = createDeepScanFilters(targetDomain)
		
	default:
		// 默认使用平衡模式
		config = getBalancedConfig()
		filters = createBalancedFilters(targetDomain)
	}
	
	config.TargetDomain = targetDomain
	
	mgr := NewURLFilterManager(config)
	for _, filter := range filters {
		mgr.RegisterFilter(filter)
	}
	
	return mgr
}

// ============================================================================
// 严格模式（Strict）
// ============================================================================

func getStrictConfig() FilterManagerConfig {
	return FilterManagerConfig{
		Enabled:          true,
		Mode:             FilterModeStrict,
		EnableCaching:    true,
		CacheSize:        10000,
		EnableEarlyStop:  true,
		EnableTrace:      false,
		TraceBufferSize:  100,
		VerboseLogging:   false,
	}
}

func createStrictFilters(targetDomain string) []URLFilter {
	return []URLFilter{
		// 1. 基础格式（优先级10）
		NewBasicFormatFilter(),
		
		// 2. 黑名单（优先级20）
		NewBlacklistFilter(),
		
		// 3. 域名作用域（优先级30）
		NewScopeFilter(ScopeFilterConfig{
			TargetDomain:       targetDomain,
			AllowSubdomains:    true,
			AllowHTTP:          true,
			AllowHTTPS:         true,
			ExternalLinkAction: FilterReject, // 严格模式：拒绝外部链接
		}),
		
		// 4. 类型分类（优先级40）
		NewTypeClassifierFilter(TypeClassifierConfig{
			StaticResourceAction: FilterReject, // 严格模式：拒绝静态资源
			JSFileAction:         FilterAllow,  // JS文件允许（需要分析）
			CSSFileAction:        FilterReject, // CSS文件拒绝
		}),
		
		// 5. 业务价值（优先级50）
		NewBusinessValueFilter(40.0, 70.0), // 严格模式：最低分40
	}
}

// ============================================================================
// 平衡模式（Balanced）- 默认推荐
// ============================================================================

func getBalancedConfig() FilterManagerConfig {
	return FilterManagerConfig{
		Enabled:          true,
		Mode:             FilterModeBalanced,
		EnableCaching:    true,
		CacheSize:        10000,
		EnableEarlyStop:  true,
		EnableTrace:      false,
		TraceBufferSize:  100,
		VerboseLogging:   false,
	}
}

func createBalancedFilters(targetDomain string) []URLFilter {
	return []URLFilter{
		// 1. 基础格式（优先级10）
		NewBasicFormatFilter(),
		
		// 2. 黑名单（优先级20）
		NewBlacklistFilter(),
		
		// 3. 域名作用域（优先级30）
		NewScopeFilter(ScopeFilterConfig{
			TargetDomain:       targetDomain,
			AllowSubdomains:    true,
			AllowHTTP:          true,
			AllowHTTPS:         true,
			ExternalLinkAction: FilterDegrade, // 平衡模式：外部链接降级
		}),
		
		// 4. 类型分类（优先级40）
		NewTypeClassifierFilter(TypeClassifierConfig{
			StaticResourceAction: FilterDegrade, // 平衡模式：静态资源降级
			JSFileAction:         FilterAllow,   // JS文件允许
			CSSFileAction:        FilterDegrade, // CSS文件降级
		}),
		
		// 5. 业务价值（优先级50）
		NewBusinessValueFilter(30.0, 70.0), // 平衡模式：最低分30
	}
}

// ============================================================================
// 宽松模式（Loose）
// ============================================================================

func getLooseConfig() FilterManagerConfig {
	return FilterManagerConfig{
		Enabled:          true,
		Mode:             FilterModeLoose,
		EnableCaching:    true,
		CacheSize:        10000,
		EnableEarlyStop:  false, // 宽松模式：不早停，所有过滤器都执行
		EnableTrace:      false,
		TraceBufferSize:  100,
		VerboseLogging:   false,
	}
}

func createLooseFilters(targetDomain string) []URLFilter {
	return []URLFilter{
		// 1. 基础格式（优先级10）
		NewBasicFormatFilter(),
		
		// 2. 黑名单（优先级20）- 禁用，宽松模式不使用黑名单
		func() URLFilter {
			f := NewBlacklistFilter()
			f.SetEnabled(false)
			return f
		}(),
		
		// 3. 域名作用域（优先级30）
		NewScopeFilter(ScopeFilterConfig{
			TargetDomain:       targetDomain,
			AllowSubdomains:    true,
			AllowHTTP:          true,
			AllowHTTPS:         true,
			ExternalLinkAction: FilterDegrade, // 宽松模式：外部链接降级
		}),
		
		// 4. 类型分类（优先级40）
		NewTypeClassifierFilter(TypeClassifierConfig{
			StaticResourceAction: FilterDegrade, // 宽松模式：静态资源降级
			JSFileAction:         FilterAllow,   // JS文件允许
			CSSFileAction:        FilterAllow,   // CSS文件也允许
		}),
		
		// 5. 业务价值（优先级50）
		NewBusinessValueFilter(20.0, 70.0), // 宽松模式：最低分20
	}
}

// ============================================================================
// API专用模式（API Only）
// ============================================================================

func getAPIOnlyConfig() FilterManagerConfig {
	return FilterManagerConfig{
		Enabled:          true,
		Mode:             FilterModeBalanced,
		EnableCaching:    true,
		CacheSize:        5000,
		EnableEarlyStop:  true,
		EnableTrace:      false,
		TraceBufferSize:  100,
		VerboseLogging:   false,
	}
}

func createAPIOnlyFilters(targetDomain string) []URLFilter {
	return []URLFilter{
		// 1. 基础格式（优先级10）
		NewBasicFormatFilter(),
		
		// 2. 域名作用域（优先级30）
		NewScopeFilter(ScopeFilterConfig{
			TargetDomain:       targetDomain,
			AllowSubdomains:    true,
			AllowHTTP:          true,
			AllowHTTPS:         true,
			ExternalLinkAction: FilterReject, // API模式：拒绝外部链接
		}),
		
		// 3. 类型分类（优先级40）
		NewTypeClassifierFilter(TypeClassifierConfig{
			StaticResourceAction: FilterReject, // API模式：拒绝静态资源
			JSFileAction:         FilterAllow,  // JS文件允许（可能包含API定义）
			CSSFileAction:        FilterReject, // CSS文件拒绝
		}),
		
		// 4. API专用过滤器
		NewAPIOnlyFilter(),
	}
}

// APIOnlyFilter API专用过滤器
type APIOnlyFilter struct {
	enabled       bool
	totalChecked  int64
	totalRejected int64
}

func NewAPIOnlyFilter() *APIOnlyFilter {
	return &APIOnlyFilter{enabled: true}
}

func (f *APIOnlyFilter) Name() string                  { return "APIOnly" }
func (f *APIOnlyFilter) Priority() int                 { return 45 }
func (f *APIOnlyFilter) SetEnabled(enabled bool)       { f.enabled = enabled }
func (f *APIOnlyFilter) IsEnabled() bool               { return f.enabled }
func (f *APIOnlyFilter) Reset()                        { f.totalChecked = 0; f.totalRejected = 0 }
func (f *APIOnlyFilter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_checked":  f.totalChecked,
		"total_rejected": f.totalRejected,
	}
}

func (f *APIOnlyFilter) Filter(rawURL string, ctx *FilterContext) FilterResult {
	f.totalChecked++
	
	lowerURL := strings.ToLower(rawURL)
	
	// 检查是否包含API特征
	apiPatterns := []string{
		"/api/", "/rest/", "/v1/", "/v2/", "/v3/",
		"/graphql", "/json", "/xml",
		"api.", "rest.",
	}
	
	for _, pattern := range apiPatterns {
		if strings.Contains(lowerURL, pattern) {
			return FilterResult{
				Allowed: true,
				Action:  FilterAllow,
				Reason:  "API端点",
				Score:   100,
			}
		}
	}
	
	// 不是API，拒绝
	f.totalRejected++
	return FilterResult{
		Allowed: false,
		Action:  FilterReject,
		Reason:  "非API端点",
		Score:   0,
	}
}

// ============================================================================
// 深度扫描模式（Deep Scan）
// ============================================================================

func getDeepScanConfig() FilterManagerConfig {
	return FilterManagerConfig{
		Enabled:          true,
		Mode:             FilterModeBalanced,
		EnableCaching:    true,
		CacheSize:        20000, // 更大的缓存
		EnableEarlyStop:  false, // 不早停，确保完整评估
		EnableTrace:      true,  // 启用追踪
		TraceBufferSize:  200,   // 更大的追踪缓冲区
		VerboseLogging:   true,  // 详细日志
	}
}

func createDeepScanFilters(targetDomain string) []URLFilter {
	return []URLFilter{
		// 1. 基础格式（优先级10）
		NewBasicFormatFilter(),
		
		// 2. 域名作用域（优先级30）
		NewScopeFilter(ScopeFilterConfig{
			TargetDomain:       targetDomain,
			AllowSubdomains:    true,
			AllowHTTP:          true,
			AllowHTTPS:         true,
			ExternalLinkAction: FilterDegrade, // 深度扫描：外部链接也记录
		}),
		
		// 3. 类型分类（优先级40）
		NewTypeClassifierFilter(TypeClassifierConfig{
			StaticResourceAction: FilterDegrade, // 深度扫描：静态资源降级
			JSFileAction:         FilterAllow,   // JS文件允许
			CSSFileAction:        FilterAllow,   // CSS文件允许
		}),
		
		// 4. 业务价值（优先级50）- 更宽松的阈值
		NewBusinessValueFilter(15.0, 60.0), // 深度扫描：最低分15
	}
}

// ============================================================================
// 自定义配置构建器
// ============================================================================

// FilterManagerBuilder 过滤管理器构建器
type FilterManagerBuilder struct {
	config      FilterManagerConfig
	filters     []URLFilter
	targetDomain string
}

// NewFilterManagerBuilder 创建构建器
func NewFilterManagerBuilder(targetDomain string) *FilterManagerBuilder {
	return &FilterManagerBuilder{
		config: FilterManagerConfig{
			Enabled:         true,
			Mode:            FilterModeBalanced,
			EnableCaching:   true,
			CacheSize:       10000,
			EnableEarlyStop: true,
			EnableTrace:     false,
			TraceBufferSize: 100,
			TargetDomain:    targetDomain,
		},
		filters:      make([]URLFilter, 0),
		targetDomain: targetDomain,
	}
}

// WithMode 设置模式
func (b *FilterManagerBuilder) WithMode(mode FilterMode) *FilterManagerBuilder {
	b.config.Mode = mode
	return b
}

// WithCaching 设置缓存
func (b *FilterManagerBuilder) WithCaching(enabled bool, size int) *FilterManagerBuilder {
	b.config.EnableCaching = enabled
	b.config.CacheSize = size
	return b
}

// WithEarlyStop 设置早停
func (b *FilterManagerBuilder) WithEarlyStop(enabled bool) *FilterManagerBuilder {
	b.config.EnableEarlyStop = enabled
	return b
}

// WithTrace 设置追踪
func (b *FilterManagerBuilder) WithTrace(enabled bool, bufferSize int) *FilterManagerBuilder {
	b.config.EnableTrace = enabled
	b.config.TraceBufferSize = bufferSize
	return b
}

// AddFilter 添加过滤器
func (b *FilterManagerBuilder) AddFilter(filter URLFilter) *FilterManagerBuilder {
	b.filters = append(b.filters, filter)
	return b
}

// AddBasicFormat 添加基础格式过滤器
func (b *FilterManagerBuilder) AddBasicFormat() *FilterManagerBuilder {
	return b.AddFilter(NewBasicFormatFilter())
}

// AddBlacklist 添加黑名单过滤器
func (b *FilterManagerBuilder) AddBlacklist() *FilterManagerBuilder {
	return b.AddFilter(NewBlacklistFilter())
}

// AddScope 添加作用域过滤器
func (b *FilterManagerBuilder) AddScope(config ScopeFilterConfig) *FilterManagerBuilder {
	config.TargetDomain = b.targetDomain
	return b.AddFilter(NewScopeFilter(config))
}

// AddTypeClassifier 添加类型分类器
func (b *FilterManagerBuilder) AddTypeClassifier(config TypeClassifierConfig) *FilterManagerBuilder {
	return b.AddFilter(NewTypeClassifierFilter(config))
}

// AddBusinessValue 添加业务价值过滤器
func (b *FilterManagerBuilder) AddBusinessValue(minScore, highThreshold float64) *FilterManagerBuilder {
	return b.AddFilter(NewBusinessValueFilter(minScore, highThreshold))
}

// Build 构建过滤管理器
func (b *FilterManagerBuilder) Build() *URLFilterManager {
	mgr := NewURLFilterManager(b.config)
	for _, filter := range b.filters {
		mgr.RegisterFilter(filter)
	}
	return mgr
}

// ============================================================================
// 使用示例
// ============================================================================

/*
使用预设配置：

	// 1. 平衡模式（推荐）
	manager := NewURLFilterManagerWithPreset(PresetBalanced, "example.com")
	
	// 2. 严格模式
	manager := NewURLFilterManagerWithPreset(PresetStrict, "example.com")
	
	// 3. API专用模式
	manager := NewURLFilterManagerWithPreset(PresetAPIOnly, "api.example.com")

自定义配置：

	manager := NewFilterManagerBuilder("example.com").
		WithMode(FilterModeBalanced).
		WithCaching(true, 10000).
		WithEarlyStop(true).
		AddBasicFormat().
		AddBlacklist().
		AddScope(ScopeFilterConfig{
			AllowSubdomains:    true,
			ExternalLinkAction: FilterDegrade,
		}).
		AddTypeClassifier(TypeClassifierConfig{
			StaticResourceAction: FilterDegrade,
			JSFileAction:         FilterAllow,
		}).
		AddBusinessValue(30.0, 70.0).
		Build()

使用过滤管理器：

	// 单个URL过滤
	result := manager.Filter("https://example.com/api/users", nil)
	if result.Allowed && result.Action == FilterAllow {
		// 爬取该URL
	}
	
	// 批量过滤
	urls := []string{"url1", "url2", "url3"}
	results := manager.FilterBatch(urls, nil)
	
	// 简化接口
	if manager.ShouldCrawl("https://example.com/page") {
		// 爬取
	}
	
	// 调试URL
	explanation := manager.ExplainURL("https://example.com/test")
	fmt.Println(explanation)
	
	// 查看统计
	manager.PrintStatistics()
*/

