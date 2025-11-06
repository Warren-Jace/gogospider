package core

import (
	"fmt"
	"strings"
	"sync"
)

// DeduplicationCoordinator 去重协调器
// 统一管理多个去重器，避免冲突，提供统一的去重决策
type DeduplicationCoordinator struct {
	mutex sync.RWMutex

	// 各层去重器
	exactMatcher    map[string]bool              // 精确匹配去重（visitedURLs）
	urlNormalizer   *URLCanonicalizer            // URL规范化器
	// patternDOMDedup *URLPatternWithDOMDeduplicator // 🔧 v4.8: 已废弃，使用 similarURLDedup 和 domEmbeddingDedup 替代
	businessFilter  *BusinessAwareURLFilter      // 业务感知过滤器
	layeredDedup    *LayeredDeduplicator         // 分层去重器

	// 统计信息
	stats CoordinatorStats
}

// CoordinatorStats 协调器统计信息
type CoordinatorStats struct {
	TotalURLs           int // 总处理URL数
	NormalizedURLs      int // 规范化的URL数
	ExactDuplicates     int // 精确重复数
	PatternFiltered     int // 模式过滤数
	BusinessFiltered    int // 业务过滤数
	LayeredFiltered     int // 分层过滤数
	AllowedURLs         int // 允许爬取的URL数
	NormalizationErrors int // 规范化错误数
}

// Decision 去重决策
type Decision struct {
	Allow         bool    // 是否允许爬取
	Reason        string  // 决策原因
	NormalizedURL string  // 规范化后的URL
	Priority      float64 // 优先级（业务价值分数）
	NeedsDOMAnalysis bool // 是否需要DOM分析
}

// NewDeduplicationCoordinator 创建去重协调器
// 🔧 v4.8: 移除 patternDOMDedup 参数，使用新的去重方案
func NewDeduplicationCoordinator(
	urlNormalizer *URLCanonicalizer,
	businessFilter *BusinessAwareURLFilter,
	layeredDedup *LayeredDeduplicator,
) *DeduplicationCoordinator {
	return &DeduplicationCoordinator{
		exactMatcher:    make(map[string]bool),
		urlNormalizer:   urlNormalizer,
		// patternDOMDedup: nil, // 已废弃
		businessFilter:  businessFilter,
		layeredDedup:    layeredDedup,
		stats:           CoordinatorStats{},
	}
}

// ShouldCrawl 统一的去重决策入口
// 返回：决策结果、错误
func (dc *DeduplicationCoordinator) ShouldCrawl(rawURL string) (Decision, error) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	dc.stats.TotalURLs++

	// 🔧 阶段1：URL规范化（最高优先级）
	var normalizedURL string
	var err error
	if dc.urlNormalizer != nil {
		normalizedURL, err = dc.urlNormalizer.CanonicalizeURL(rawURL)
		if err != nil {
			// 规范化失败，保守处理：使用原URL
			normalizedURL = rawURL
			dc.stats.NormalizationErrors++
		} else {
			dc.stats.NormalizedURLs++
		}
	} else {
		normalizedURL = rawURL
	}

	// 🔧 阶段2：精确匹配去重（快速路径）
	if dc.exactMatcher[normalizedURL] {
		dc.stats.ExactDuplicates++
		return Decision{
			Allow:         false,
			Reason:        "精确匹配去重：URL已访问",
			NormalizedURL: normalizedURL,
			Priority:      0,
		}, nil
	}

	// 🔧 阶段3：URL模式+DOM验证（已废弃，使用新方案）
	// v4.8: 使用 similarURLDedup 和 domEmbeddingDedup 替代
	var needsDOMAnalysis bool = false

	// 🔧 阶段4：业务价值评估（暂时跳过，等待业务过滤器接口统一）
	var businessScore float64 = 50.0 // 默认中等价值
	// TODO: 集成SmartBusinessScorer后启用

	// 🔧 阶段5：分层去重（可选）
	if dc.layeredDedup != nil {
		// LayeredDeduplicator使用ShouldProcess方法
		shouldProcess, _, reason := dc.layeredDedup.ShouldProcess(normalizedURL, "GET")
		if !shouldProcess {
			dc.stats.LayeredFiltered++
			return Decision{
				Allow:         false,
				Reason:        fmt.Sprintf("分层去重: %s", reason),
				NormalizedURL: normalizedURL,
				Priority:      businessScore,
			}, nil
		}
	}

	// ✅ 通过所有检查，允许爬取
	dc.exactMatcher[normalizedURL] = true
	dc.stats.AllowedURLs++

	return Decision{
		Allow:            true,
		Reason:           "通过所有去重检查",
		NormalizedURL:    normalizedURL,
		Priority:         businessScore,
		NeedsDOMAnalysis: needsDOMAnalysis,
	}, nil
}

// RecordDOMSignature 记录DOM签名（爬取完成后调用）
// 🔧 v4.8: 已废弃，使用 domEmbeddingDedup 替代
func (dc *DeduplicationCoordinator) RecordDOMSignature(rawURL string, htmlContent string) error {
	// 功能已被新的 DOM Embedding 去重器替代
	return nil
}

// MarkVisited 标记URL为已访问（用于外部调用）
func (dc *DeduplicationCoordinator) MarkVisited(rawURL string) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	var normalizedURL string
	if dc.urlNormalizer != nil {
		var err error
		normalizedURL, err = dc.urlNormalizer.CanonicalizeURL(rawURL)
		if err != nil {
			normalizedURL = rawURL
		}
	} else {
		normalizedURL = rawURL
	}

	dc.exactMatcher[normalizedURL] = true
}

// GetStatistics 获取统计信息
func (dc *DeduplicationCoordinator) GetStatistics() CoordinatorStats {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	return dc.stats
}

// PrintReport 打印详细报告
func (dc *DeduplicationCoordinator) PrintReport() {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()

	fmt.Println("\n" + strings.Repeat("═", 80))
	fmt.Println("                    去重协调器统计报告")
	fmt.Println(strings.Repeat("═", 80))

	fmt.Printf("\n【总体统计】\n")
	fmt.Printf("  总处理URL数:        %d\n", dc.stats.TotalURLs)
	fmt.Printf("  成功规范化:         %d (%.1f%%)\n", dc.stats.NormalizedURLs,
		getPercentage(dc.stats.NormalizedURLs, dc.stats.TotalURLs))
	fmt.Printf("  规范化错误:         %d\n", dc.stats.NormalizationErrors)

	fmt.Printf("\n【过滤统计】\n")
	fmt.Printf("  精确匹配过滤:       %d (%.1f%%)\n", dc.stats.ExactDuplicates,
		getPercentage(dc.stats.ExactDuplicates, dc.stats.TotalURLs))
	fmt.Printf("  URL模式+DOM过滤:    %d (%.1f%%)\n", dc.stats.PatternFiltered,
		getPercentage(dc.stats.PatternFiltered, dc.stats.TotalURLs))
	fmt.Printf("  业务价值过滤:       %d (%.1f%%)\n", dc.stats.BusinessFiltered,
		getPercentage(dc.stats.BusinessFiltered, dc.stats.TotalURLs))
	fmt.Printf("  分层去重过滤:       %d (%.1f%%)\n", dc.stats.LayeredFiltered,
		getPercentage(dc.stats.LayeredFiltered, dc.stats.TotalURLs))

	totalFiltered := dc.stats.ExactDuplicates + dc.stats.PatternFiltered +
		dc.stats.BusinessFiltered + dc.stats.LayeredFiltered
	fmt.Printf("  总过滤数:           %d (%.1f%%)\n", totalFiltered,
		getPercentage(totalFiltered, dc.stats.TotalURLs))

	fmt.Printf("\n【爬取统计】\n")
	fmt.Printf("  允许爬取:           %d (%.1f%%)\n", dc.stats.AllowedURLs,
		getPercentage(dc.stats.AllowedURLs, dc.stats.TotalURLs))

	if dc.stats.TotalURLs > 0 {
		fmt.Printf("\n【效率指标】\n")
		fmt.Printf("  去重率:             %.1f%%\n",
			float64(totalFiltered)/float64(dc.stats.TotalURLs)*100)
		fmt.Printf("  请求节省:           %d 个\n", totalFiltered)
	}

	fmt.Println("\n" + strings.Repeat("═", 80))
}

// Reset 重置协调器
func (dc *DeduplicationCoordinator) Reset() {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	dc.exactMatcher = make(map[string]bool)
	dc.stats = CoordinatorStats{}
	
	// patternDOMDedup 已废弃
}

// getPercentage 计算百分比辅助函数
func getPercentage(part, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(part) * 100.0 / float64(total)
}

