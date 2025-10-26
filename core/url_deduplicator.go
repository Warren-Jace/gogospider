package core

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// URLDeduplicator URL去重器（忽略参数值）
type URLDeduplicator struct {
	urlPatterns map[string][]string // URL模式 -> 具体URL列表
}

// NewURLDeduplicator 创建URL去重器
func NewURLDeduplicator() *URLDeduplicator {
	return &URLDeduplicator{
		urlPatterns: make(map[string][]string),
	}
}

// GetURLPattern 获取URL模式（忽略参数值）
// 例如: http://example.com/user?id=123&name=abc
//   → http://example.com/user?id=&name=
func (d *URLDeduplicator) GetURLPattern(rawURL string) string {
	// 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		// 解析失败，返回原URL
		return rawURL
	}
	
	// 构建基础URL（协议+域名+路径）
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	
	// 处理查询参数（保留参数名，清空参数值）
	if parsedURL.RawQuery != "" {
		query := parsedURL.Query()
		
		// 提取所有参数名并排序（保证一致性）
		paramNames := make([]string, 0, len(query))
		for paramName := range query {
			paramNames = append(paramNames, paramName)
		}
		sort.Strings(paramNames)
		
		// 构建参数模式（参数名=）
		paramParts := make([]string, 0, len(paramNames))
		for _, paramName := range paramNames {
			paramParts = append(paramParts, paramName+"=")
		}
		
		if len(paramParts) > 0 {
			baseURL += "?" + strings.Join(paramParts, "&")
		}
	}
	
	// 处理锚点（通常忽略）
	// 如果需要保留锚点，可以加上：
	// if parsedURL.Fragment != "" {
	//     baseURL += "#" + parsedURL.Fragment
	// }
	
	return baseURL
}

// AddURL 添加URL
func (d *URLDeduplicator) AddURL(rawURL string) {
	pattern := d.GetURLPattern(rawURL)
	
	// 检查是否已存在
	if urls, exists := d.urlPatterns[pattern]; exists {
		// 检查具体URL是否已存在
		for _, existingURL := range urls {
			if existingURL == rawURL {
				return // 已存在，不重复添加
			}
		}
		// 添加新的具体URL
		d.urlPatterns[pattern] = append(urls, rawURL)
	} else {
		// 新模式
		d.urlPatterns[pattern] = []string{rawURL}
	}
}

// AddURLs 批量添加URL
func (d *URLDeduplicator) AddURLs(urls []string) {
	for _, url := range urls {
		d.AddURL(url)
	}
}

// GetUniquePatterns 获取所有唯一的URL模式（去重后）
func (d *URLDeduplicator) GetUniquePatterns() []string {
	patterns := make([]string, 0, len(d.urlPatterns))
	for pattern := range d.urlPatterns {
		patterns = append(patterns, pattern)
	}
	
	// 排序，便于查看和比较
	sort.Strings(patterns)
	
	return patterns
}

// GetAllURLs 获取所有URL（包含所有参数值变体）
func (d *URLDeduplicator) GetAllURLs() []string {
	allURLs := make([]string, 0)
	
	for _, urls := range d.urlPatterns {
		allURLs = append(allURLs, urls...)
	}
	
	// 排序
	sort.Strings(allURLs)
	
	return allURLs
}

// GetURLsByPattern 获取指定模式的所有URL
func (d *URLDeduplicator) GetURLsByPattern(pattern string) []string {
	if urls, exists := d.urlPatterns[pattern]; exists {
		return urls
	}
	return []string{}
}

// GetStatistics 获取统计信息
func (d *URLDeduplicator) GetStatistics() map[string]int {
	stats := make(map[string]int)
	
	totalURLs := 0
	for _, urls := range d.urlPatterns {
		totalURLs += len(urls)
	}
	
	stats["unique_patterns"] = len(d.urlPatterns)
	stats["total_urls"] = totalURLs
	
	// 计算平均每个模式有多少个变体
	if len(d.urlPatterns) > 0 {
		stats["avg_variants_per_pattern"] = totalURLs / len(d.urlPatterns)
	} else {
		stats["avg_variants_per_pattern"] = 0
	}
	
	return stats
}

// PrintReport 打印去重报告
func (d *URLDeduplicator) PrintReport() {
	stats := d.GetStatistics()
	
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("         URL去重统计报告")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  唯一URL模式: %d 个\n", stats["unique_patterns"])
	fmt.Printf("  URL总数:     %d 个\n", stats["total_urls"])
	fmt.Printf("  平均变体数:  %d 个/模式\n", stats["avg_variants_per_pattern"])
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	// 展示去重效果
	if stats["total_urls"] > stats["unique_patterns"] {
		reduction := stats["total_urls"] - stats["unique_patterns"]
		reductionPercent := float64(reduction) / float64(stats["total_urls"]) * 100
		fmt.Printf("  去重效果: 减少 %d 个URL (%.1f%%)\n", reduction, reductionPercent)
	}
	fmt.Println()
}

// GetDetailedReport 获取详细报告（显示每个模式的变体数量）
func (d *URLDeduplicator) GetDetailedReport() []PatternVariant {
	report := make([]PatternVariant, 0, len(d.urlPatterns))
	
	for pattern, urls := range d.urlPatterns {
		report = append(report, PatternVariant{
			Pattern:      pattern,
			VariantCount: len(urls),
			Variants:     urls,
		})
	}
	
	// 按变体数量排序（多的在前）
	sort.Slice(report, func(i, j int) bool {
		return report[i].VariantCount > report[j].VariantCount
	})
	
	return report
}

// PatternVariant URL模式变体信息
type PatternVariant struct {
	Pattern      string   // URL模式
	VariantCount int      // 变体数量
	Variants     []string // 所有变体URL
}

// PrintDetailedReport 打印详细报告（显示前N个最多变体的模式）
func (d *URLDeduplicator) PrintDetailedReport(topN int) {
	report := d.GetDetailedReport()
	
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("       URL模式变体详细报告")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	displayCount := topN
	if len(report) < topN {
		displayCount = len(report)
	}
	
	for i := 0; i < displayCount; i++ {
		pv := report[i]
		fmt.Printf("\n[%d] 模式: %s\n", i+1, pv.Pattern)
		fmt.Printf("    变体数: %d 个\n", pv.VariantCount)
		
		// 显示前3个变体作为示例
		exampleCount := 3
		if len(pv.Variants) < exampleCount {
			exampleCount = len(pv.Variants)
		}
		
		if exampleCount > 0 {
			fmt.Println("    示例:")
			for j := 0; j < exampleCount; j++ {
				fmt.Printf("      - %s\n", pv.Variants[j])
			}
			
			if len(pv.Variants) > exampleCount {
				fmt.Printf("      ... 还有 %d 个变体\n", len(pv.Variants)-exampleCount)
			}
		}
	}
	
	if len(report) > displayCount {
		fmt.Printf("\n... 还有 %d 个模式\n", len(report)-displayCount)
	}
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// FilterByDomain 按域名过滤URL（只保留指定域名的URL）
func (d *URLDeduplicator) FilterByDomain(domain string) *URLDeduplicator {
	filtered := NewURLDeduplicator()
	
	for pattern, urls := range d.urlPatterns {
		// 检查模式是否属于该域名
		if strings.Contains(pattern, domain) {
			filtered.urlPatterns[pattern] = urls
		}
	}
	
	return filtered
}

