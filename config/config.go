package config

import (
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

// NewDefaultConfig 创建默认配置
func NewDefaultConfig() *Config {
	return &Config{
		DepthSettings: DepthSettings{
			MaxDepth:            3,
			DeepCrawling:        false,
			SchedulingAlgorithm: "DFS",
		},
		StrategySettings: StrategySettings{
			EnableStaticCrawler:    true,
			EnableDynamicCrawler:   true,
			EnableJSAnalysis:       true,
			EnableAPIInference:     true,
			DomainScope:            "",
		},
		AntiDetectionSettings: AntiDetectionSettings{
			RequestDelay:       1 * time.Second,
			UserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			},
			Proxies:                []string{},
			EnableFormAutoFill:     true,
		},
		DeduplicationSettings: DeduplicationSettings{
			SimilarityThreshold:         0.85,
			EnableDOMDeduplication:      true,
			EnableURLPatternRecognition: true,
		},
	}
}