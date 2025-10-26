package core

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// URLStructureDeduplicator URL结构化去重器
// 不仅去除参数值重复，还识别路径中的变量（数字、ID等）
type URLStructureDeduplicator struct {
	structurePatterns map[string]*StructurePattern // 结构模式 -> 模式详情
	seenURLs          map[string]bool              // 已见过的完整URL
}

// StructurePattern 结构化模式
type StructurePattern struct {
	Pattern      string   // 归一化的URL模式
	FirstURL     string   // 第一个发现的URL（作为代表）
	SimilarURLs  []string // 相似的URL列表（采样前10个）
	Count        int      // 发现的相似URL数量
}

// NewURLStructureDeduplicator 创建结构化去重器
func NewURLStructureDeduplicator() *URLStructureDeduplicator {
	return &URLStructureDeduplicator{
		structurePatterns: make(map[string]*StructurePattern),
		seenURLs:          make(map[string]bool),
	}
}

// NormalizeURL 归一化URL，提取结构模式
// 例如:
//   http://test.com/product-123/buy → http://test.com/product-{num}/buy
//   http://test.com/user/456       → http://test.com/user/{num}
//   http://test.com/item?id=789    → http://test.com/item?id=
func (d *URLStructureDeduplicator) NormalizeURL(rawURL string) (string, error) {
	// 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	
	// 1. 归一化路径（识别路径中的数字、UUID等变量）
	normalizedPath := d.normalizePath(parsedURL.Path)
	
	// 2. 归一化查询参数（只保留参数名，清空参数值）
	normalizedQuery := d.normalizeQuery(parsedURL.Query())
	
	// 3. 构建归一化URL
	normalizedURL := parsedURL.Scheme + "://" + parsedURL.Host + normalizedPath
	if normalizedQuery != "" {
		normalizedURL += "?" + normalizedQuery
	}
	
	// 4. 处理Fragment（通常忽略，但如果有特殊需求可以保留）
	// if parsedURL.Fragment != "" {
	//     normalizedURL += "#" + parsedURL.Fragment
	// }
	
	return normalizedURL, nil
}

// normalizePath 归一化路径，识别变量部分
// 规则：
//   - 纯数字路径段 → {num}
//   - 包含数字的路径段（如 product-123, item_456） → 保留前缀，数字部分替换为 {num}
//   - UUID格式 → {uuid}
//   - 其他保持不变
func (d *URLStructureDeduplicator) normalizePath(path string) string {
	if path == "" || path == "/" {
		return path
	}
	
	// 分割路径
	segments := strings.Split(strings.Trim(path, "/"), "/")
	normalizedSegments := make([]string, 0, len(segments))
	
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		
		normalizedSegment := d.normalizePathSegment(segment)
		normalizedSegments = append(normalizedSegments, normalizedSegment)
	}
	
	// 重建路径
	normalizedPath := "/" + strings.Join(normalizedSegments, "/")
	
	// 保留末尾斜杠（如果原路径有的话）
	if strings.HasSuffix(path, "/") && !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}
	
	return normalizedPath
}

// normalizePathSegment 归一化单个路径段
func (d *URLStructureDeduplicator) normalizePathSegment(segment string) string {
	// 1. 检查是否为纯数字
	if regexp.MustCompile(`^\d+$`).MatchString(segment) {
		return "{num}"
	}
	
	// 2. 检查是否为UUID格式（8-4-4-4-12）
	uuidPattern := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	if uuidPattern.MatchString(strings.ToLower(segment)) {
		return "{uuid}"
	}
	
	// 3. 检查是否为较长的十六进制字符串（可能是hash或token）
	if regexp.MustCompile(`^[a-f0-9]{16,}$`).MatchString(strings.ToLower(segment)) {
		return "{hash}"
	}
	
	// 4. 检查是否包含数字（如 product-123, item_456, BuyProduct-2）
	// 匹配模式：prefix + separator + number + optional_suffix
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// 匹配: product-123, item_456, RateProduct-2.html
		{regexp.MustCompile(`^(.+?)([-_])(\d+)(\.[\w]+)?$`), "$1$2{num}$4"},
		
		// 匹配: product123 (无分隔符的情况)
		{regexp.MustCompile(`^([a-zA-Z]+)(\d+)$`), "$1{num}"},
		
		// 匹配: 123product (数字在前的情况，较少见)
		{regexp.MustCompile(`^(\d+)([a-zA-Z]+)$`), "{num}$2"},
	}
	
	for _, pattern := range patterns {
		if pattern.regex.MatchString(segment) {
			return pattern.regex.ReplaceAllString(segment, pattern.replacement)
		}
	}
	
	// 5. 其他情况保持不变
	return segment
}

// normalizeQuery 归一化查询参数（只保留参数名，清空参数值）
func (d *URLStructureDeduplicator) normalizeQuery(query url.Values) string {
	if len(query) == 0 {
		return ""
	}
	
	// 提取参数名并排序（保证一致性）
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
	
	return strings.Join(paramParts, "&")
}

// AddURL 添加URL并进行结构化去重
// 返回: (是否为新结构, 结构化模式)
func (d *URLStructureDeduplicator) AddURL(rawURL string) (bool, string) {
	// 检查是否已经见过这个完整URL
	if d.seenURLs[rawURL] {
		return false, ""
	}
	d.seenURLs[rawURL] = true
	
	// 归一化URL
	pattern, err := d.NormalizeURL(rawURL)
	if err != nil {
		// 解析失败，作为新URL处理
		pattern = rawURL
	}
	
	// 检查结构模式是否已存在
	if existing, exists := d.structurePatterns[pattern]; exists {
		// 已存在相同结构，更新统计
		existing.Count++
		
		// 采样保留前10个相似URL
		if len(existing.SimilarURLs) < 10 {
			existing.SimilarURLs = append(existing.SimilarURLs, rawURL)
		}
		
		return false, pattern
	}
	
	// 新结构模式
	d.structurePatterns[pattern] = &StructurePattern{
		Pattern:     pattern,
		FirstURL:    rawURL,
		SimilarURLs: []string{},
		Count:       1,
	}
	
	return true, pattern
}

// AddURLs 批量添加URL
func (d *URLStructureDeduplicator) AddURLs(urls []string) {
	for _, url := range urls {
		d.AddURL(url)
	}
}

// GetUniqueStructures 获取所有唯一的结构化URL（去重后）
// 返回每个结构的代表性URL
func (d *URLStructureDeduplicator) GetUniqueStructures() []string {
	urls := make([]string, 0, len(d.structurePatterns))
	
	for _, pattern := range d.structurePatterns {
		// 返回第一个发现的URL作为代表
		urls = append(urls, pattern.FirstURL)
	}
	
	// 排序
	sort.Strings(urls)
	
	return urls
}

// GetStructurePatterns 获取所有结构化模式（用于调试和统计）
func (d *URLStructureDeduplicator) GetStructurePatterns() []string {
	patterns := make([]string, 0, len(d.structurePatterns))
	
	for pattern := range d.structurePatterns {
		patterns = append(patterns, pattern)
	}
	
	// 排序
	sort.Strings(patterns)
	
	return patterns
}

// GetPatternDetails 获取模式详情列表
func (d *URLStructureDeduplicator) GetPatternDetails() []*StructurePattern {
	details := make([]*StructurePattern, 0, len(d.structurePatterns))
	
	for _, pattern := range d.structurePatterns {
		details = append(details, pattern)
	}
	
	// 按发现数量排序（多的在前）
	sort.Slice(details, func(i, j int) bool {
		return details[i].Count > details[j].Count
	})
	
	return details
}

// GetStatistics 获取统计信息
func (d *URLStructureDeduplicator) GetStatistics() map[string]int {
	stats := make(map[string]int)
	
	totalURLs := len(d.seenURLs)
	uniqueStructures := len(d.structurePatterns)
	
	stats["total_urls"] = totalURLs
	stats["unique_structures"] = uniqueStructures
	stats["duplicate_urls"] = totalURLs - uniqueStructures
	
	// 计算平均每个结构有多少个相似URL
	if uniqueStructures > 0 {
		stats["avg_similar_per_structure"] = totalURLs / uniqueStructures
	} else {
		stats["avg_similar_per_structure"] = 0
	}
	
	return stats
}

// PrintReport 打印结构化去重报告
func (d *URLStructureDeduplicator) PrintReport() {
	stats := d.GetStatistics()
	
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("                   URL结构化去重报告")
	fmt.Println(strings.Repeat("=", 80))
	
	fmt.Printf("\n【总体统计】\n")
	fmt.Printf("  总URL数:          %d 个\n", stats["total_urls"])
	fmt.Printf("  唯一结构数:       %d 个\n", stats["unique_structures"])
	fmt.Printf("  去重的URL数:      %d 个\n", stats["duplicate_urls"])
	
	if stats["total_urls"] > 0 {
		reductionPercent := float64(stats["duplicate_urls"]) / float64(stats["total_urls"]) * 100
		fmt.Printf("  去重率:           %.1f%%\n", reductionPercent)
		fmt.Printf("  平均相似度:       %.1f 个URL/结构\n", 
			float64(stats["total_urls"])/float64(stats["unique_structures"]))
	}
	
	// 显示Top重复结构
	details := d.GetPatternDetails()
	if len(details) > 0 {
		fmt.Printf("\n【Top 10 高频结构】\n")
		fmt.Println(strings.Repeat("-", 80))
		
		topCount := 10
		if len(details) < topCount {
			topCount = len(details)
		}
		
		for i := 0; i < topCount; i++ {
			pattern := details[i]
			fmt.Printf("\n%d. 结构模式: %s\n", i+1, pattern.Pattern)
			fmt.Printf("   相似URL数: %d 个\n", pattern.Count)
			fmt.Printf("   代表URL:   %s\n", pattern.FirstURL)
			
			// 显示一些相似URL示例
			if len(pattern.SimilarURLs) > 0 {
				fmt.Printf("   相似示例:  ")
				exampleCount := 3
				if len(pattern.SimilarURLs) < exampleCount {
					exampleCount = len(pattern.SimilarURLs)
				}
				
				examples := make([]string, 0, exampleCount)
				for j := 0; j < exampleCount; j++ {
					examples = append(examples, pattern.SimilarURLs[j])
				}
				
				fmt.Printf("%s", strings.Join(examples, ", "))
				
				if len(pattern.SimilarURLs) > exampleCount {
					fmt.Printf(", ... (还有%d个)", len(pattern.SimilarURLs)-exampleCount)
				}
				fmt.Println()
			}
		}
	}
	
	fmt.Println("\n" + strings.Repeat("=", 80))
}

// PrintDetailedReport 打印详细报告（带更多相似URL示例）
func (d *URLStructureDeduplicator) PrintDetailedReport(topN int) {
	details := d.GetPatternDetails()
	
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("               URL结构化去重详细报告")
	fmt.Println(strings.Repeat("=", 80))
	
	displayCount := topN
	if len(details) < topN {
		displayCount = len(details)
	}
	
	for i := 0; i < displayCount; i++ {
		pattern := details[i]
		
		fmt.Printf("\n[%d] 结构模式: %s\n", i+1, pattern.Pattern)
		fmt.Printf("    相似URL数: %d 个\n", pattern.Count)
		fmt.Printf("    代表URL:   %s\n", pattern.FirstURL)
		
		if len(pattern.SimilarURLs) > 0 {
			fmt.Println("\n    相似URL列表:")
			for j, similarURL := range pattern.SimilarURLs {
				fmt.Printf("      [%d] %s\n", j+1, similarURL)
			}
			
			if pattern.Count > len(pattern.SimilarURLs)+1 {
				fmt.Printf("      ... 还有 %d 个相似URL\n", 
					pattern.Count-len(pattern.SimilarURLs)-1)
			}
		}
	}
	
	if len(details) > displayCount {
		fmt.Printf("\n... 还有 %d 个结构模式\n", len(details)-displayCount)
	}
	
	fmt.Println("\n" + strings.Repeat("=", 80))
}

