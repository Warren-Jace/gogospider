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
	EnableCommonPathScan bool // 是否启用200个常见路径扫描（默认true）
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

// NewDefaultConfig 创建默认配置（优化版 - 超越crawlergo）
func NewDefaultConfig() *Config {
	return &Config{
		DepthSettings: DepthSettings{
			MaxDepth:            5,     // 增加到5层深度
			DeepCrawling:        true,  // 启用深度爬取
			SchedulingAlgorithm: "BFS", // 广度优先，确保覆盖全面
		},
		StrategySettings: StrategySettings{
			EnableStaticCrawler:  true, // 启用静态爬虫
			EnableDynamicCrawler: true, // 启用动态爬虫（已优化）
			EnableJSAnalysis:     true, // 启用JS分析
			EnableAPIInference:   true, // 启用API推测
			DomainScope:          "",   // 默认不限制
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
