package config

import (
	"fmt"
	"strings"
	"time"
)

// Config 爬虫配置结构体
type Config struct {
	// 目标URL
	TargetURL string `json:"target_url"`

	// 爬取深度设置
	DepthSettings DepthSettings `json:"depth_settings"`

	// 爬取策略设置
	StrategySettings StrategySettings `json:"strategy_settings"`

	// 反爬设置
	AntiDetectionSettings AntiDetectionSettings `json:"anti_detection_settings"`

	// 去重设置
	DeduplicationSettings DeduplicationSettings `json:"deduplication_settings"`

	// 日志设置（v2.6 新增）
	LogSettings LogSettings `json:"log_settings"`
	
	// 🆕 v2.9 新增功能
	OutputSettings OutputSettings         `json:"output_settings"` // 输出设置
	RateLimitSettings RateLimitSettings   `json:"rate_limit_settings"` // 速率限制设置
	ExternalSourceSettings ExternalSourceSettings `json:"external_source_settings"` // 外部数据源设置
	ScopeSettings ScopeSettings           `json:"scope_settings"` // Scope设置
	PipelineSettings PipelineSettings     `json:"pipeline_settings"` // 管道模式设置
	
	// 🆕 敏感信息检测设置
	SensitiveDetectionSettings SensitiveDetectionSettings `json:"sensitive_detection_settings"` // 敏感信息检测设置
	
	// 🆕 v3.0 新增功能
	BlacklistSettings BlacklistSettings   `json:"blacklist_settings"` // 黑名单设置
	BatchScanSettings BatchScanSettings   `json:"batch_scan_settings"` // 批量扫描设置
	
	// 🆕 v3.4 新增功能
	SchedulingSettings SchedulingSettings `json:"scheduling_settings"` // 调度策略设置
	AdvancedSettings   AdvancedSettings   `json:"advanced_settings"`   // 高级功能设置
	OutputAdvanced     OutputAdvanced     `json:"output_advanced"`     // 输出增强配置
	
	// 🆕 v4.2 新增功能：统一URL过滤管理器
	FilterSettings     FilterSettings     `json:"filter_settings"`     // URL过滤设置
	
	// 🆕 v4.3: 性能优化开关
	EnablePerformanceOptimizations bool `json:"enable_performance_optimizations"` // 启用性能优化(URL解析缓存+分片锁+混合去重)
	
	// 🆕 v4.4: 请求日志开关
	EnableRequestLogging bool `json:"enable_request_logging"` // 启用请求日志记录(用于调试优化)
}

// DepthSettings 爬取深度设置
type DepthSettings struct {
	// 最大深度
	MaxDepth int `json:"max_depth"`

	// 是否深度爬取
	DeepCrawling bool `json:"deep_crawling"`

	// 调度算法 DFS/BFS
	SchedulingAlgorithm string `json:"scheduling_algorithm"`
}

// StrategySettings 爬取策略设置
type StrategySettings struct {
	// 是否启用静态爬虫
	EnableStaticCrawler bool `json:"enable_static_crawler"`

	// 是否启用动态爬虫
	EnableDynamicCrawler bool `json:"enable_dynamic_crawler"`

	// 是否启用JS分析
	EnableJSAnalysis bool `json:"enable_js_analysis"`

	// 是否启用API推测
	EnableAPIInference bool `json:"enable_api_inference"`

	// 参数爆破功能已移除（专注于纯爬虫功能）
	// EnableParamFuzzing bool (已废弃)
	// ParamFuzzLimit int (已废弃)
	// EnablePOSTParamFuzzing bool (已废弃)
	// POSTParamFuzzLimit int (已废弃)

	// 域名范围限制
	DomainScope string `json:"domain_scope"`
	
	// 🆕 v2.8 新增配置（已废弃，使用SchedulingSettings替代）
	UsePriorityQueue     bool `json:"use_priority_queue"`      // 是否使用优先级队列模式（默认false，使用BFS）
	EnableCommonPathScan bool `json:"enable_common_path_scan"` // 是否启用200个常见路径扫描（默认false，性能考虑）
}

// SchedulingSettings 调度策略设置（v3.4新增）
type SchedulingSettings struct {
	// 调度算法: BFS, DFS, PRIORITY_QUEUE, HYBRID
	Algorithm string `json:"algorithm"`
	
	// 混合策略配置
	HybridConfig HybridSchedulingConfig `json:"hybrid_config"`
	
	// 性能配置
	PerformanceConfig PerformanceConfig `json:"performance_config"`
}

// HybridSchedulingConfig 混合调度策略配置
type HybridSchedulingConfig struct {
	// 是否启用自适应学习
	EnableAdaptiveLearning bool `json:"enable_adaptive_learning"`
	
	// 优先级权重
	PriorityWeights PriorityWeights `json:"priority_weights"`
	
	// 每层最多爬取数量（0=不限制）
	MaxURLsPerLayer int `json:"max_urls_per_layer"`
	
	// 高价值URL阈值（高于此值的总是优先）
	HighValueThreshold float64 `json:"high_value_threshold"`
	
	// 学习率（自适应调整的速度，0.1-0.5）
	LearningRate float64 `json:"learning_rate"`
}

// PriorityWeights 优先级权重配置
type PriorityWeights struct {
	Depth         float64 `json:"depth"`          // 深度因子权重（浅层优先）
	Internal      float64 `json:"internal"`       // 域内链接权重
	Params        float64 `json:"params"`         // 参数权重（带参数的URL更重要）
	Recent        float64 `json:"recent"`         // 新鲜度权重（新发现的URL）
	PathValue     float64 `json:"path_value"`     // 路径价值权重（/admin, /api等）
	BusinessValue float64 `json:"business_value"` // 业务价值权重（结合业务感知过滤器）
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	MaxConcurrentRequests int  `json:"max_concurrent_requests"` // 最大并发请求数
	RequestTimeout        int  `json:"request_timeout"`         // 请求超时时间（秒）
	MaxRetry              int  `json:"max_retry"`               // 最大重试次数
	EnableConnectionPool  bool `json:"enable_connection_pool"`  // 启用连接池
	MaxMemoryMB           int  `json:"max_memory_mb"`           // 最大内存使用（MB）
	EnableDiskCache       bool `json:"enable_disk_cache"`       // 启用磁盘缓存
}

// AdvancedSettings 高级功能设置（v3.4新增）
type AdvancedSettings struct {
	EnableSmartThrottling        bool `json:"enable_smart_throttling"`         // 智能限速
	EnableCDNOptimization        bool `json:"enable_cdn_optimization"`         // CDN优化
	EnableGraphQLDetection       bool `json:"enable_graphql_detection"`        // GraphQL检测
	EnableWebSocketMonitoring    bool `json:"enable_websocket_monitoring"`     // WebSocket监控
	EnableAPIVersioningDetection bool `json:"enable_api_versioning_detection"` // API版本检测
}

// OutputAdvanced 输出增强配置（v3.4新增）
type OutputAdvanced struct {
	SaveCrawlTimeline          bool `json:"save_crawl_timeline"`           // 保存爬取时间线
	SavePriorityDistribution   bool `json:"save_priority_distribution"`    // 保存优先级分布
	SaveBusinessValueAnalysis  bool `json:"save_business_value_analysis"`  // 保存业务价值分析
	EnableRealtimeDashboard    bool `json:"enable_realtime_dashboard"`     // 启用实时仪表板
	DashboardPort              int  `json:"dashboard_port"`                // 仪表板端口
}

// AntiDetectionSettings 反爬设置
type AntiDetectionSettings struct {
	// 请求间隔
	RequestDelay time.Duration `json:"request_delay"`

	// User-Agent列表
	UserAgents []string `json:"user_agents"`

	// 代理列表
	Proxies []string `json:"proxies"`

	// 是否启用表单自动填充
	EnableFormAutoFill bool `json:"enable_form_auto_fill"`
	
	// ✅ 修复2: Cookie配置（统一在配置文件中管理）
	CookieFile   string `json:"cookie_file"`   // Cookie文件路径（JSON或文本格式）
	CookieString string `json:"cookie_string"` // Cookie字符串（格式：name1=value1; name2=value2）
	
	// ✅ 修复5: HTTPS证书验证配置
	InsecureSkipVerify bool `json:"insecure_skip_verify"` // 是否忽略HTTPS证书错误（默认false）
}

// DeduplicationSettings 去重设置
type DeduplicationSettings struct {
	// 相似度阈值
	SimilarityThreshold float64 `json:"similarity_threshold"`

	// 是否启用DOM相似度去重
	EnableDOMDeduplication bool `json:"enable_dom_deduplication"`

	// 是否启用URL模式识别
	EnableURLPatternRecognition bool `json:"enable_url_pattern_recognition"`

	// 是否启用智能参数值去重（v2.6.1 新增）
	EnableSmartParamDedup bool `json:"enable_smart_param_dedup"`

	// 每个参数值特征组最多爬取数量（v2.6.1 新增）
	MaxParamValueVariantsPerGroup int `json:"max_param_value_variants_per_group"`
	
	// 🆕 v4.5: 是否启用URL模式+DOM相似度去重（更精准的去重策略）
	EnableURLPatternDOMDedup bool `json:"enable_url_pattern_dom_dedup"`
	
	// 🆕 v4.5: URL模式采样次数（默认3次）
	URLPatternDOMSampleCount int `json:"url_pattern_dom_sample_count"`
	
	// 🆕 v4.5: URL模式DOM相似度阈值（默认0.85）
	URLPatternDOMThreshold float64 `json:"url_pattern_dom_threshold"`
	
	// 是否启用业务感知过滤（v2.7 新增）
	EnableBusinessAwareFilter bool `json:"enable_business_aware_filter"`
	
	// 业务感知过滤配置（v2.7 新增）
	BusinessFilterMinScore        float64 `json:"business_filter_min_score"` // 最低业务分数 (0-100)
	BusinessFilterHighValueThreshold float64 `json:"business_filter_high_value_threshold"` // 高价值URL阈值
	BusinessFilterMaxLowValue     int     `json:"business_filter_max_low_value"` // 低价值URL同模式最大数量
	BusinessFilterMaxMidValue     int     `json:"business_filter_max_mid_value"` // 中等价值URL同模式最大数量
	BusinessFilterMaxHighValue    int     `json:"business_filter_max_high_value"` // 高价值URL同模式最大数量
	BusinessFilterAdaptiveLearning bool   `json:"business_filter_adaptive_learning"` // 是否启用自适应学习
	
	// 智能参数验证（v2.8 新增）
	EnableParamValidation      bool    `json:"enable_param_validation"` // 是否启用参数验证
	ParamValidationSimilarity  float64 `json:"param_validation_similarity"` // 响应相似度阈值 (0-1)
	ParamValidationMaxSimilar  int     `json:"param_validation_max_similar"` // 最大相同响应次数
	ParamValidationMinDiff     int     `json:"param_validation_min_diff"` // 最小响应差异（字节）
}

// LogSettings 日志设置（v2.6 新增）
type LogSettings struct {
	// 日志级别: DEBUG, INFO, WARN, ERROR
	Level string `json:"level"`

	// 日志文件路径，空表示 stdout
	OutputFile string `json:"output_file"`

	// 日志格式: json, text
	Format string `json:"format"`

	// 是否显示实时指标
	ShowMetrics bool `json:"show_metrics"`
}

// OutputSettings 输出设置（v2.9 新增）
type OutputSettings struct {
	// 输出格式: text, json, jsonl
	Format string
	
	// 输出文件（为空则输出到stdout）
	OutputFile string
	
	// JSON输出模式: compact, pretty, line
	JSONMode string
	
	// 是否包含所有字段
	IncludeAll bool
	
	// 是否输出详细信息
	Verbose bool
}

// RateLimitSettings 速率限制设置（v2.9 新增）
type RateLimitSettings struct {
	// 是否启用速率限制
	Enabled bool
	
	// 每秒最大请求数
	RequestsPerSecond int
	
	// 突发请求数
	BurstSize int
	
	// 最小请求间隔（毫秒）
	MinDelay int
	
	// 最大请求间隔（毫秒）
	MaxDelay int
	
	// 是否启用自适应速率
	Adaptive bool
	
	// 自适应速率范围
	AdaptiveMinRate int
	AdaptiveMaxRate int
}

// ExternalSourceSettings 外部数据源设置（v2.9 新增）
type ExternalSourceSettings struct {
	// 是否启用外部数据源
	Enabled bool
	
	// 启用Wayback Machine
	EnableWaybackMachine bool
	
	// 启用VirusTotal
	EnableVirusTotal bool
	VirusTotalAPIKey string
	
	// 启用CommonCrawl
	EnableCommonCrawl bool
	
	// 每个数据源最大结果数
	MaxResultsPerSource int
	
	// 超时时间（秒）
	Timeout int
}

// ScopeSettings Scope设置（v2.9 新增）
type ScopeSettings struct {
	// 是否启用Scope控制
	Enabled bool `json:"enabled"`
	
	// 包含的域名
	IncludeDomains []string `json:"include_domains"`
	
	// 排除的域名
	ExcludeDomains []string `json:"exclude_domains"`
	
	// 包含的路径模式
	IncludePaths []string `json:"include_paths"`
	
	// 排除的路径模式
	ExcludePaths []string `json:"exclude_paths"`
	
	// 包含的URL正则
	IncludeRegex string `json:"include_regex"`
	
	// 排除的URL正则
	ExcludeRegex string `json:"exclude_regex"`
	
	// 包含的文件扩展名
	IncludeExtensions []string `json:"include_extensions"`
	
	// 排除的文件扩展名
	ExcludeExtensions []string `json:"exclude_extensions"`
	
	// 允许子域名
	AllowSubdomains bool `json:"allow_subdomains"`
	
	// 限制在同一域名内
	StayInDomain bool `json:"stay_in_domain"`
	
	// 允许HTTP
	AllowHTTP bool `json:"allow_http"`
	
	// 允许HTTPS
	AllowHTTPS bool `json:"allow_https"`
}

// PipelineSettings 管道模式设置（v2.9 新增）
type PipelineSettings struct {
	// 是否启用管道模式
	Enabled bool
	
	// 启用标准输入
	EnableStdin bool
	
	// 启用标准输出
	EnableStdout bool
	
	// 输入格式: text, json
	InputFormat string
	
	// 输出格式: text, json, jsonl
	OutputFormat string
	
	// 静默模式（不输出日志到stderr）
	Quiet bool
}

// SensitiveDetectionSettings 敏感信息检测设置（v2.10 新增）
type SensitiveDetectionSettings struct {
	// 是否启用敏感信息检测
	Enabled bool `json:"enabled"`
	
	// 是否扫描HTTP响应体
	ScanResponseBody bool `json:"scan_response_body"`
	
	// 是否扫描HTTP响应头
	ScanResponseHeaders bool `json:"scan_response_headers"`
	
	// 最低严重级别: LOW, MEDIUM, HIGH
	MinSeverity string `json:"min_severity"`
	
	// 是否启用自定义模式
	EnableCustomPatterns bool `json:"enable_custom_patterns"`
	
	// 自定义检测模式列表（正则表达式）
	CustomPatterns []CustomPattern `json:"custom_patterns"`
	
	// 是否保存完整敏感值（默认false，只保存脱敏值）
	SaveFullValue bool `json:"save_full_value"`
	
	// 敏感信息输出文件（为空则只在内存中保存）
	OutputFile string `json:"output_file"`
	
	// 是否实时输出敏感信息发现
	RealTimeOutput bool `json:"realtime_output"`
	
	// 排除的URL模式（不检测这些URL）
	ExcludeURLPatterns []string `json:"exclude_url_patterns"`
	
	// 敏感信息规则文件路径
	RulesFile string `json:"rules_file"`
}

// BlacklistSettings 黑名单设置（v3.0 新增）
type BlacklistSettings struct {
	// 是否启用黑名单
	Enabled bool
	
	// 黑名单域名列表（支持通配符，如 *.gov.cn）
	Domains []string
	
	// 黑名单域名模式（如 *bank*, *payment*）
	DomainPatterns []string
	
	// 严格模式：true=完全拒绝访问，false=只记录警告
	StrictMode bool
}

// BatchScanSettings 批量扫描设置（v3.0 新增）
type BatchScanSettings struct {
	// 是否启用批量扫描
	Enabled bool
	
	// 输入文件路径（每行一个URL）
	InputFile string
	
	// 并发扫描数量
	Concurrency int
	
	// 输出目录
	OutputDir string
	
	// 每个目标的超时时间（秒）
	PerTargetTimeout int
	
	// 遇到错误时是否继续
	ContinueOnError bool
	
	// 是否为每个目标保存单独的报告
	SaveIndividualReports bool
	
	// 是否保存汇总报告
	SaveSummaryReport bool
}

// CustomPattern 自定义检测模式
type CustomPattern struct {
	Name     string // 模式名称
	Pattern  string // 正则表达式
	Severity string // 严重程度: HIGH/MEDIUM/LOW
	Mask     bool   // 是否需要脱敏
}

// FilterSettings URL过滤设置（v4.2新增）
type FilterSettings struct {
	// 是否启用新的过滤管理器
	Enabled bool `json:"enabled"`
	
	// 预设模式: strict/balanced/loose/api_only/deep_scan
	Preset string `json:"preset"`
	
	// 过滤模式: strict/balanced/loose
	Mode string `json:"mode"`
	
	// 性能优化
	EnableCaching   bool `json:"enable_caching"`
	CacheSize       int  `json:"cache_size"`
	EnableEarlyStop bool `json:"enable_early_stop"`
	
	// 调试
	EnableTrace     bool `json:"enable_trace"`
	TraceBufferSize int  `json:"trace_buffer_size"`
	VerboseLogging  bool `json:"verbose_logging"`
	
	// 外部链接处理: allow/reject/degrade
	ExternalLinkAction string `json:"external_link_action"`
	
	// 静态资源处理: allow/reject/degrade
	StaticResourceAction string `json:"static_resource_action"`
	
	// 业务价值评估
	MinBusinessScore    float64 `json:"min_business_score"`
	HighValueThreshold  float64 `json:"high_value_threshold"`
}

// NewDefaultConfig 创建默认配置（优化版 - 超越crawlergo）
func NewDefaultConfig() *Config {
	return &Config{
		DepthSettings: DepthSettings{
			MaxDepth:            5,     // 增加到5层深度
			DeepCrawling:        true,  // 启用深度爬取
			SchedulingAlgorithm: "BFS", // 广度优先，确保覆盖全面
		},
		StrategySettings: StrategySettings{
			EnableStaticCrawler:   true,  // 启用静态爬虫
			EnableDynamicCrawler:  true,  // 启用动态爬虫（已优化）
			EnableJSAnalysis:      true,  // 启用JS分析
			EnableAPIInference:    true,  // 启用API推测
			DomainScope:           "",    // 默认不限制
			UsePriorityQueue:      false, // 默认使用BFS
			EnableCommonPathScan:  false, // 🔧 默认禁用（性能考虑）
			// 参数爆破功能已移除（专注于纯爬虫）
		},
		AntiDetectionSettings: AntiDetectionSettings{
			RequestDelay: 500 * time.Millisecond, // 减少延迟以提高速度
			UserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			},
			Proxies:            []string{},
			EnableFormAutoFill: true, // 启用表单自动填充
			CookieFile:         "",   // Cookie文件路径（留空表示不使用）
			CookieString:       "",   // Cookie字符串（留空表示不使用）
			InsecureSkipVerify: false, // ✅ 默认验证HTTPS证书
		},
		DeduplicationSettings: DeduplicationSettings{
			SimilarityThreshold:           0.85, // 85%相似度阈值
			EnableDOMDeduplication:        true, // 启用DOM去重
			EnableURLPatternRecognition:   true, // 启用URL模式识别
			EnableSmartParamDedup:         true, // 启用智能参数值去重（v2.6.1）
			MaxParamValueVariantsPerGroup: 3,    // 每种特征最多爬取3个（v2.6.1）
			
			// 🆕 v4.5: URL模式+DOM相似度去重配置
			EnableURLPatternDOMDedup:     true,  // 启用URL模式+DOM去重（更精准）
			URLPatternDOMSampleCount:     3,     // 采样3次验证
			URLPatternDOMThreshold:       0.85,  // DOM相似度阈值85%
			
			// 业务感知过滤配置（v2.7 新增）
			EnableBusinessAwareFilter:        true,  // 启用业务感知过滤
			BusinessFilterMinScore:           30.0,  // 最低分数30
			BusinessFilterHighValueThreshold: 70.0,  // 高价值阈值70
			BusinessFilterMaxLowValue:        2,     // 低价值最多2个
			BusinessFilterMaxMidValue:        5,     // 中等价值最多5个
			BusinessFilterMaxHighValue:       20,    // 高价值最多20个
			BusinessFilterAdaptiveLearning:  true,   // 启用自适应学习
			
			// 智能参数验证配置（v2.8 新增）
			EnableParamValidation:      true,  // 启用参数验证
			ParamValidationSimilarity:  0.95,  // 95%相似度阈值
			ParamValidationMaxSimilar:  3,     // 连续3次相同响应就停止
			ParamValidationMinDiff:     50,    // 最小50字节差异
		},
		LogSettings: LogSettings{
			Level:       "INFO", // 默认INFO级别
			OutputFile:  "",     // 默认输出到控制台
			Format:      "json", // 默认JSON格式
			ShowMetrics: false,  // 默认不显示实时指标
		},
		
		// 🆕 v2.9 新增功能默认配置
		OutputSettings: OutputSettings{
			Format:     "text",    // 默认文本格式
			OutputFile: "",        // 默认输出到stdout
			JSONMode:   "line",    // 默认行分隔JSON (NDJSON)
			IncludeAll: false,     // 默认只输出核心字段
			Verbose:    false,     // 默认非详细模式
		},
		
		RateLimitSettings: RateLimitSettings{
			Enabled:           false, // 默认不启用速率限制
			RequestsPerSecond: 100,   // 默认100 req/s
			BurstSize:         10,    // 默认允许10个突发请求
			MinDelay:          0,     // 默认无最小延迟
			MaxDelay:          0,     // 默认无最大延迟
			Adaptive:          false, // 默认不启用自适应
			AdaptiveMinRate:   10,    // 自适应最小速率
			AdaptiveMaxRate:   200,   // 自适应最大速率
		},
		
		ExternalSourceSettings: ExternalSourceSettings{
			Enabled:              false, // 默认不启用外部数据源
			EnableWaybackMachine: false,
			EnableVirusTotal:     false,
			VirusTotalAPIKey:     "",
			EnableCommonCrawl:    false,
			MaxResultsPerSource:  1000, // 每个数据源最多1000个结果
			Timeout:              30,   // 30秒超时
		},
		
		ScopeSettings: ScopeSettings{
			Enabled:           true,   // ✅ 修复4: 默认启用Scope控制
			IncludeDomains:    []string{},
			ExcludeDomains:    []string{},
			IncludePaths:      []string{},
			ExcludePaths:      []string{},
			IncludeRegex:      "",
			ExcludeRegex:      "",
			IncludeExtensions: []string{},
			ExcludeExtensions: []string{
				// ✅ 修复6&7: JS已从排除列表移除,程序会自动处理
				// 静态资源:图片、样式、字体、文档等(JS已特殊处理,会被访问)
				"jpg", "jpeg", "png", "gif", "svg", "ico", "webp", "bmp",
				"css", "scss", "sass",
				"woff", "woff2", "ttf", "eot", "otf",
				"mp4", "mp3", "avi", "mov", "wmv", "flv",
				"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx",
				"zip", "rar", "tar", "gz", "7z",
			}, // ✅ 默认排除静态资源(JS除外,会被特殊处理)
			AllowSubdomains: false, // 默认不允许子域名
			StayInDomain:    true,  // 默认限制在同一域名内
			AllowHTTP:       true,  // 允许HTTP
			AllowHTTPS:      true,  // 允许HTTPS
		},
		
		PipelineSettings: PipelineSettings{
			Enabled:      false,  // 默认不启用管道模式
			EnableStdin:  false,
			EnableStdout: false,
			InputFormat:  "text", // 默认文本输入
			OutputFormat: "text", // 默认文本输出
			Quiet:        false,  // 默认不静默
		},
		
		// 🆕 v2.10: 敏感信息检测默认配置
		SensitiveDetectionSettings: SensitiveDetectionSettings{
			Enabled:              true,   // 默认启用敏感信息检测
			ScanResponseBody:     true,   // 扫描响应体
			ScanResponseHeaders:  true,   // 扫描响应头
			MinSeverity:          "LOW",  // 最低级别：显示所有
			EnableCustomPatterns: false,  // 默认不启用自定义模式
			CustomPatterns:       []CustomPattern{},
			SaveFullValue:        false,  // 只保存脱敏值（安全）
			OutputFile:           "",     // 默认不单独保存（包含在总报告中）
			RealTimeOutput:       true,   // 实时输出敏感信息发现
			ExcludeURLPatterns:   []string{}, // 默认不排除任何URL
			RulesFile:            "sensitive_rules.json", // 默认规则文件
		},
		
		// 🆕 v3.0: 黑名单默认配置
		BlacklistSettings: BlacklistSettings{
			Enabled:    true, // 默认启用黑名单
			Domains:    []string{"*.gov.cn", "*.edu.cn", "*.mil.cn"}, // 默认黑名单
			DomainPatterns: []string{},
			StrictMode: true, // 严格模式
		},
		
		// 🆕 v3.0: 批量扫描默认配置
		BatchScanSettings: BatchScanSettings{
			Enabled:               false,  // 默认不启用
			InputFile:             "targets.txt",
			Concurrency:           5,
			OutputDir:             "./batch_results",
			PerTargetTimeout:      3600,
			ContinueOnError:       true,
			SaveIndividualReports: true,
			SaveSummaryReport:     true,
		},
		
		// 🆕 v3.4: 调度策略默认配置
		SchedulingSettings: SchedulingSettings{
			Algorithm: "BFS", // 默认使用BFS（向下兼容）
			HybridConfig: HybridSchedulingConfig{
				EnableAdaptiveLearning: true,  // 启用自适应学习
				PriorityWeights: PriorityWeights{
					Depth:         3.0,  // 深度因子
					Internal:      2.0,  // 域内链接
					Params:        1.5,  // 参数
					Recent:        1.0,  // 新鲜度
					PathValue:     4.0,  // 路径价值
					BusinessValue: 0.5,  // 业务价值
				},
				MaxURLsPerLayer:    100,  // 每层最多100个URL
				HighValueThreshold: 80.0, // 高价值阈值80分
				LearningRate:       0.15, // 学习率15%
			},
			PerformanceConfig: PerformanceConfig{
				MaxConcurrentRequests: 20,   // 最大并发20
				RequestTimeout:        30,   // 超时30秒
				MaxRetry:              3,    // 最多重试3次
				EnableConnectionPool:  true, // 启用连接池
				MaxMemoryMB:           1024, // 最大内存1GB
				EnableDiskCache:       false, // 默认不启用磁盘缓存
			},
		},
		
		// 🆕 v3.4: 高级功能默认配置
		AdvancedSettings: AdvancedSettings{
			EnableSmartThrottling:        true,  // 启用智能限速
			EnableCDNOptimization:        true,  // 启用CDN优化
			EnableGraphQLDetection:       true,  // 启用GraphQL检测
			EnableWebSocketMonitoring:    false, // WebSocket监控（实验性，默认关闭）
			EnableAPIVersioningDetection: true,  // 启用API版本检测
		},
		
		// 🆕 v3.4: 输出增强默认配置
		OutputAdvanced: OutputAdvanced{
			SaveCrawlTimeline:         true,  // 保存爬取时间线
			SavePriorityDistribution:  true,  // 保存优先级分布
			SaveBusinessValueAnalysis: true,  // 保存业务价值分析
			EnableRealtimeDashboard:   false, // 实时仪表板（默认关闭）
			DashboardPort:             8080,  // 仪表板端口
		},
		
		// 🆕 v4.2: 统一URL过滤管理器默认配置
		FilterSettings: FilterSettings{
			Enabled:              true,      // 启用新的过滤管理器
			Preset:               "balanced", // 默认平衡模式
			Mode:                 "balanced",
			EnableCaching:        true,
			CacheSize:            10000,
			EnableEarlyStop:      true,
			EnableTrace:          false,
			TraceBufferSize:      100,
			VerboseLogging:       false,
			ExternalLinkAction:   "degrade", // 外部链接降级（记录但不爬取）
			StaticResourceAction: "degrade", // 静态资源降级
			MinBusinessScore:     30.0,
			HighValueThreshold:   70.0,
		},
	}
}

// Validate 验证配置（优化：添加配置验证）
func (c *Config) Validate() error {
	// 验证目标URL
	if c.TargetURL == "" {
		return fmt.Errorf("目标URL不能为空")
	}

	// 验证深度设置
	if c.DepthSettings.MaxDepth < 0 {
		return fmt.Errorf("最大深度不能为负数，当前值: %d", c.DepthSettings.MaxDepth)
	}

	if c.DepthSettings.MaxDepth > 20 {
		return fmt.Errorf("最大深度不能超过20层（防止过度爬取），当前值: %d", c.DepthSettings.MaxDepth)
	}

	// ✅ v3.4修复: 支持新的调度算法
	// 优先检查新的SchedulingSettings.Algorithm
	if c.SchedulingSettings.Algorithm != "" {
		validAlgorithms := map[string]bool{
			"BFS":            true,
			"DFS":            true,
			"PRIORITY_QUEUE": true,
			"HYBRID":         true,
		}
		if !validAlgorithms[strings.ToUpper(c.SchedulingSettings.Algorithm)] {
			return fmt.Errorf("调度算法必须是 BFS, DFS, PRIORITY_QUEUE 或 HYBRID，当前值: %s", c.SchedulingSettings.Algorithm)
		}
	} else if c.DepthSettings.SchedulingAlgorithm != "" {
		// 向下兼容旧配置
		if c.DepthSettings.SchedulingAlgorithm != "BFS" && c.DepthSettings.SchedulingAlgorithm != "DFS" {
			return fmt.Errorf("调度算法必须是 BFS 或 DFS，当前值: %s", c.DepthSettings.SchedulingAlgorithm)
		}
	}

	// 验证策略设置
	// 参数爆破相关验证已移除

	// 验证反爬设置
	if c.AntiDetectionSettings.RequestDelay < 0 {
		return fmt.Errorf("请求延迟不能为负数，当前值: %v", c.AntiDetectionSettings.RequestDelay)
	}

	if len(c.AntiDetectionSettings.UserAgents) == 0 {
		return fmt.Errorf("至少需要配置一个User-Agent")
	}

	// 验证去重设置
	if c.DeduplicationSettings.SimilarityThreshold < 0 || c.DeduplicationSettings.SimilarityThreshold > 1 {
		return fmt.Errorf("相似度阈值必须在0-1之间，当前值: %.2f", c.DeduplicationSettings.SimilarityThreshold)
	}

	return nil
}

// ValidateAndFix 验证并修复配置（自动修复一些常见问题）
func (c *Config) ValidateAndFix() error {
	// 修复深度
	if c.DepthSettings.MaxDepth < 0 {
		c.DepthSettings.MaxDepth = 1
	}
	if c.DepthSettings.MaxDepth > 20 {
		c.DepthSettings.MaxDepth = 20
	}

	// ✅ v3.4修复: 支持新的调度算法
	// 优先检查新的SchedulingSettings.Algorithm
	if c.SchedulingSettings.Algorithm != "" {
		validAlgorithms := map[string]bool{
			"BFS":            true,
			"DFS":            true,
			"PRIORITY_QUEUE": true,
			"HYBRID":         true,
		}
		upperAlgo := strings.ToUpper(c.SchedulingSettings.Algorithm)
		if !validAlgorithms[upperAlgo] {
			// 无效算法，修复为BFS
			c.SchedulingSettings.Algorithm = "BFS"
		}
	} else if c.DepthSettings.SchedulingAlgorithm != "" {
		// 向下兼容旧配置
		if c.DepthSettings.SchedulingAlgorithm != "BFS" && c.DepthSettings.SchedulingAlgorithm != "DFS" {
			c.DepthSettings.SchedulingAlgorithm = "BFS"
		}
	}

	// 修复相似度阈值
	if c.DeduplicationSettings.SimilarityThreshold < 0 {
		c.DeduplicationSettings.SimilarityThreshold = 0
	}
	if c.DeduplicationSettings.SimilarityThreshold > 1 {
		c.DeduplicationSettings.SimilarityThreshold = 1
	}

	// 修复User-Agent
	if len(c.AntiDetectionSettings.UserAgents) == 0 {
		c.AntiDetectionSettings.UserAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		}
	}

	// 再次验证
	return c.Validate()
}
