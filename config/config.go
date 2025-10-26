package config

import (
	"fmt"
	"time"
)

// Config 爬虫配置结构体
type Config struct {
	// 目标URL
	TargetURL string

	// 爬取深度设置
	DepthSettings DepthSettings

	// 爬取策略设置
	StrategySettings StrategySettings

	// 反爬设置
	AntiDetectionSettings AntiDetectionSettings

	// 去重设置
	DeduplicationSettings DeduplicationSettings

	// 日志设置（v2.6 新增）
	LogSettings LogSettings
	
	// 🆕 v2.9 新增功能
	OutputSettings OutputSettings         // 输出设置
	RateLimitSettings RateLimitSettings   // 速率限制设置
	ExternalSourceSettings ExternalSourceSettings // 外部数据源设置
	ScopeSettings ScopeSettings           // Scope设置
	PipelineSettings PipelineSettings     // 管道模式设置
	
	// 🆕 敏感信息检测设置
	SensitiveDetectionSettings SensitiveDetectionSettings // 敏感信息检测设置
	
	// 🆕 v3.0 新增功能
	BlacklistSettings BlacklistSettings   // 黑名单设置
	BatchScanSettings BatchScanSettings   // 批量扫描设置
}

// DepthSettings 爬取深度设置
type DepthSettings struct {
	// 最大深度
	MaxDepth int

	// 是否深度爬取
	DeepCrawling bool

	// 调度算法 DFS/BFS
	SchedulingAlgorithm string
}

// StrategySettings 爬取策略设置
type StrategySettings struct {
	// 是否启用静态爬虫
	EnableStaticCrawler bool

	// 是否启用动态爬虫
	EnableDynamicCrawler bool

	// 是否启用JS分析
	EnableJSAnalysis bool

	// 是否启用API推测
	EnableAPIInference bool

	// 参数爆破功能已移除（专注于纯爬虫功能）
	// EnableParamFuzzing bool (已废弃)
	// ParamFuzzLimit int (已废弃)
	// EnablePOSTParamFuzzing bool (已废弃)
	// POSTParamFuzzLimit int (已废弃)

	// 域名范围限制
	DomainScope string
	
	// 🆕 v2.8 新增配置
	UsePriorityQueue     bool // 是否使用优先级队列模式（默认false，使用BFS）
	EnableCommonPathScan bool // 是否启用200个常见路径扫描（默认false，性能考虑）
}

// AntiDetectionSettings 反爬设置
type AntiDetectionSettings struct {
	// 请求间隔
	RequestDelay time.Duration

	// User-Agent列表
	UserAgents []string

	// 代理列表
	Proxies []string

	// 是否启用表单自动填充
	EnableFormAutoFill bool
}

// DeduplicationSettings 去重设置
type DeduplicationSettings struct {
	// 相似度阈值
	SimilarityThreshold float64

	// 是否启用DOM相似度去重
	EnableDOMDeduplication bool

	// 是否启用URL模式识别
	EnableURLPatternRecognition bool

	// 是否启用智能参数值去重（v2.6.1 新增）
	EnableSmartParamDedup bool

	// 每个参数值特征组最多爬取数量（v2.6.1 新增）
	MaxParamValueVariantsPerGroup int
	
	// 是否启用业务感知过滤（v2.7 新增）
	EnableBusinessAwareFilter bool
	
	// 业务感知过滤配置（v2.7 新增）
	BusinessFilterMinScore        float64 // 最低业务分数 (0-100)
	BusinessFilterHighValueThreshold float64 // 高价值URL阈值
	BusinessFilterMaxLowValue     int     // 低价值URL同模式最大数量
	BusinessFilterMaxMidValue     int     // 中等价值URL同模式最大数量
	BusinessFilterMaxHighValue    int     // 高价值URL同模式最大数量
	BusinessFilterAdaptiveLearning bool   // 是否启用自适应学习
	
	// 智能参数验证（v2.8 新增）
	EnableParamValidation      bool    // 是否启用参数验证
	ParamValidationSimilarity  float64 // 响应相似度阈值 (0-1)
	ParamValidationMaxSimilar  int     // 最大相同响应次数
	ParamValidationMinDiff     int     // 最小响应差异（字节）
}

// LogSettings 日志设置（v2.6 新增）
type LogSettings struct {
	// 日志级别: DEBUG, INFO, WARN, ERROR
	Level string

	// 日志文件路径，空表示 stdout
	OutputFile string

	// 日志格式: json, text
	Format string

	// 是否显示实时指标
	ShowMetrics bool
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
	Enabled bool
	
	// 包含的域名
	IncludeDomains []string
	
	// 排除的域名
	ExcludeDomains []string
	
	// 包含的路径模式
	IncludePaths []string
	
	// 排除的路径模式
	ExcludePaths []string
	
	// 包含的URL正则
	IncludeRegex string
	
	// 排除的URL正则
	ExcludeRegex string
	
	// 包含的文件扩展名
	IncludeExtensions []string
	
	// 排除的文件扩展名
	ExcludeExtensions []string
	
	// 允许子域名
	AllowSubdomains bool
	
	// 限制在同一域名内
	StayInDomain bool
	
	// 允许HTTP
	AllowHTTP bool
	
	// 允许HTTPS
	AllowHTTPS bool
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
	Enabled bool
	
	// 是否扫描HTTP响应体
	ScanResponseBody bool
	
	// 是否扫描HTTP响应头
	ScanResponseHeaders bool
	
	// 最低严重级别: LOW, MEDIUM, HIGH
	MinSeverity string
	
	// 是否启用自定义模式
	EnableCustomPatterns bool
	
	// 自定义检测模式列表（正则表达式）
	CustomPatterns []CustomPattern
	
	// 是否保存完整敏感值（默认false，只保存脱敏值）
	SaveFullValue bool
	
	// 敏感信息输出文件（为空则只在内存中保存）
	OutputFile string
	
	// 是否实时输出敏感信息发现
	RealTimeOutput bool
	
	// 排除的URL模式（不检测这些URL）
	ExcludeURLPatterns []string
	
	// 敏感信息规则文件路径
	RulesFile string
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
		},
		DeduplicationSettings: DeduplicationSettings{
			SimilarityThreshold:           0.85, // 85%相似度阈值
			EnableDOMDeduplication:        true, // 启用DOM去重
			EnableURLPatternRecognition:   true, // 启用URL模式识别
			EnableSmartParamDedup:         true, // 启用智能参数值去重（v2.6.1）
			MaxParamValueVariantsPerGroup: 3,    // 每种特征最多爬取3个（v2.6.1）
			
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
			Enabled:           false,   // 默认不启用Scope控制
			IncludeDomains:    []string{},
			ExcludeDomains:    []string{},
			IncludePaths:      []string{},
			ExcludePaths:      []string{},
			IncludeRegex:      "",
			ExcludeRegex:      "",
			IncludeExtensions: []string{},
			ExcludeExtensions: []string{
				"jpg", "jpeg", "png", "gif", "svg", "ico",
				"css", "js", "woff", "woff2", "ttf", "eot",
				"mp4", "mp3", "avi", "mov",
				"pdf", "doc", "docx", "xls", "xlsx",
				"zip", "rar", "tar", "gz",
			}, // 默认排除静态资源
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
			RulesFile:            "./sensitive_rules_config.json", // 默认规则文件
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

	if c.DepthSettings.SchedulingAlgorithm != "BFS" && c.DepthSettings.SchedulingAlgorithm != "DFS" {
		return fmt.Errorf("调度算法必须是 BFS 或 DFS，当前值: %s", c.DepthSettings.SchedulingAlgorithm)
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

	// 修复调度算法
	if c.DepthSettings.SchedulingAlgorithm != "BFS" && c.DepthSettings.SchedulingAlgorithm != "DFS" {
		c.DepthSettings.SchedulingAlgorithm = "BFS"
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
