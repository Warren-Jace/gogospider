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
	
	// 是否启用参数爆破（对无参数URL进行参数枚举）
	EnableParamFuzzing bool
	
	// 参数爆破限制（每个URL最多生成多少个爆破变体，0表示不限制）
	ParamFuzzLimit int
	
	// 是否启用POST参数爆破（对无参数表单进行POST参数枚举）
	EnablePOSTParamFuzzing bool
	
	// POST参数爆破限制（每个表单最多生成多少个POST爆破变体，0表示不限制）
	POSTParamFuzzLimit int
	
	// 域名范围限制
	DomainScope string
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
			EnableStaticCrawler:      true,  // 启用静态爬虫
			EnableDynamicCrawler:     true,  // 启用动态爬虫（已优化）
			EnableJSAnalysis:         true,  // 启用JS分析
			EnableAPIInference:       true,  // 启用API推测
			EnableParamFuzzing:       true,  // 启用GET参数爆破（新增）
			ParamFuzzLimit:           100,   // 每个URL最多生成100个爆破变体（避免过多）
			EnablePOSTParamFuzzing:   true,  // 启用POST参数爆破（新增）
			POSTParamFuzzLimit:       50,    // 每个表单最多生成50个POST爆破变体
			DomainScope:              "",    // 默认不限制
		},
		AntiDetectionSettings: AntiDetectionSettings{
			RequestDelay:       500 * time.Millisecond, // 减少延迟以提高速度
			UserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			},
			Proxies:                []string{},
			EnableFormAutoFill:     true, // 启用表单自动填充
		},
		DeduplicationSettings: DeduplicationSettings{
			SimilarityThreshold:         0.85,  // 85%相似度阈值
			EnableDOMDeduplication:      true,  // 启用DOM去重
			EnableURLPatternRecognition: true,  // 启用URL模式识别
		},
		LogSettings: LogSettings{
			Level:       "INFO",  // 默认INFO级别
			OutputFile:  "",      // 默认输出到控制台
			Format:      "json",  // 默认JSON格式
			ShowMetrics: false,   // 默认不显示实时指标
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
	if c.StrategySettings.ParamFuzzLimit < 0 {
		return fmt.Errorf("参数爆破限制不能为负数，当前值: %d", c.StrategySettings.ParamFuzzLimit)
	}
	
	if c.StrategySettings.POSTParamFuzzLimit < 0 {
		return fmt.Errorf("POST参数爆破限制不能为负数，当前值: %d", c.StrategySettings.POSTParamFuzzLimit)
	}
	
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