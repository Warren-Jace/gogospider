package core

import (
	"strings"
	"sync"
)

// MemoryOptimizer 内存优化器
// 🔧 修复：清理不必要的HTML内容，优化大型网站的内存占用
type MemoryOptimizer struct {
	mutex sync.RWMutex
	
	// 配置
	keepHTMLContent      bool // 是否保留HTML内容
	maxHTMLLength        int  // 最大HTML长度（超过则截断）
	keepHTMLSummaryLength int  // 保留的HTML摘要长度
	
	// 统计
	totalResults         int   // 总结果数
	originalSize         int64 // 原始大小
	optimizedSize        int64 // 优化后大小
	cleanedCount         int   // 清理的结果数
}

// NewMemoryOptimizer 创建内存优化器
func NewMemoryOptimizer() *MemoryOptimizer {
	return &MemoryOptimizer{
		keepHTMLContent:       false, // 默认不保留完整HTML
		maxHTMLLength:         10240, // 10KB
		keepHTMLSummaryLength: 500,   // 保留500字符的摘要
		totalResults:          0,
		originalSize:          0,
		optimizedSize:         0,
		cleanedCount:          0,
	}
}

// SetKeepHTMLContent 设置是否保留HTML内容
func (mo *MemoryOptimizer) SetKeepHTMLContent(keep bool) {
	mo.mutex.Lock()
	defer mo.mutex.Unlock()
	mo.keepHTMLContent = keep
}

// SetMaxHTMLLength 设置最大HTML长度
func (mo *MemoryOptimizer) SetMaxHTMLLength(length int) {
	mo.mutex.Lock()
	defer mo.mutex.Unlock()
	if length > 0 {
		mo.maxHTMLLength = length
	}
}

// OptimizeResult 优化单个结果的内存占用
func (mo *MemoryOptimizer) OptimizeResult(result *Result) {
	if result == nil {
		return
	}
	
	mo.mutex.Lock()
	defer mo.mutex.Unlock()
	
	mo.totalResults++
	originalSize := len(result.HTMLContent)
	mo.originalSize += int64(originalSize)
	
	// 如果不保留HTML内容
	if !mo.keepHTMLContent {
		// 提取摘要
		summary := mo.extractHTMLSummary(result.HTMLContent)
		result.HTMLContent = summary
		mo.cleanedCount++
	} else if originalSize > mo.maxHTMLLength {
		// 如果HTML过大，截断
		result.HTMLContent = result.HTMLContent[:mo.maxHTMLLength] + "... [截断]"
		mo.cleanedCount++
	}
	
	// 统计优化后的大小
	mo.optimizedSize += int64(len(result.HTMLContent))
	
	// 清理重复的链接（去重）
	result.Links = mo.deduplicateStringSlice(result.Links)
	result.Assets = mo.deduplicateStringSlice(result.Assets)
	result.APIs = mo.deduplicateStringSlice(result.APIs)
}

// OptimizeResults 批量优化结果
func (mo *MemoryOptimizer) OptimizeResults(results []*Result) {
	for _, result := range results {
		mo.OptimizeResult(result)
	}
}

// extractHTMLSummary 提取HTML摘要
// 保留关键信息：title、meta、部分body内容
func (mo *MemoryOptimizer) extractHTMLSummary(htmlContent string) string {
	if len(htmlContent) == 0 {
		return ""
	}
	
	summary := ""
	
	// 提取 <title>
	if title := extractBetween(htmlContent, "<title>", "</title>"); title != "" {
		summary += "[Title] " + title + "\n"
	}
	
	// 提取关键 meta 标签
	metas := []string{
		"description",
		"keywords",
		"author",
		"robots",
	}
	for _, meta := range metas {
		if content := extractMetaContent(htmlContent, meta); content != "" {
			summary += "[Meta-" + meta + "] " + content + "\n"
		}
	}
	
	// 提取body的开头部分
	if bodyStart := strings.Index(strings.ToLower(htmlContent), "<body"); bodyStart != -1 {
		bodyContent := htmlContent[bodyStart:]
		if len(bodyContent) > mo.keepHTMLSummaryLength {
			bodyContent = bodyContent[:mo.keepHTMLSummaryLength]
		}
		// 移除标签，只保留文本
		bodyText := removeHTMLTags(bodyContent)
		if bodyText != "" {
			summary += "[Body Preview] " + bodyText + "\n"
		}
	}
	
	// 如果摘要为空，保留前N个字符
	if summary == "" {
		maxLen := mo.keepHTMLSummaryLength
		if len(htmlContent) < maxLen {
			maxLen = len(htmlContent)
		}
		summary = "[Raw] " + htmlContent[:maxLen]
	}
	
	return summary
}

// deduplicateStringSlice 字符串切片去重
func (mo *MemoryOptimizer) deduplicateStringSlice(slice []string) []string {
	if len(slice) == 0 {
		return slice
	}
	
	seen := make(map[string]bool, len(slice))
	result := make([]string, 0, len(slice))
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// GetStatistics 获取优化统计
func (mo *MemoryOptimizer) GetStatistics() map[string]interface{} {
	mo.mutex.RLock()
	defer mo.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_results"] = mo.totalResults
	stats["cleaned_count"] = mo.cleanedCount
	stats["original_size_bytes"] = mo.originalSize
	stats["optimized_size_bytes"] = mo.optimizedSize
	
	if mo.originalSize > 0 {
		savedBytes := mo.originalSize - mo.optimizedSize
		savedRatio := float64(savedBytes) / float64(mo.originalSize)
		stats["saved_bytes"] = savedBytes
		stats["saved_ratio"] = savedRatio
		stats["saved_percent"] = savedRatio * 100
		stats["original_size_mb"] = float64(mo.originalSize) / 1024 / 1024
		stats["optimized_size_mb"] = float64(mo.optimizedSize) / 1024 / 1024
		stats["saved_size_mb"] = float64(savedBytes) / 1024 / 1024
	}
	
	return stats
}

// PrintReport 打印优化报告
func (mo *MemoryOptimizer) PrintReport() {
	mo.mutex.RLock()
	defer mo.mutex.RUnlock()
	
	if mo.totalResults == 0 {
		return
	}
	
	stats := mo.GetStatistics()
	
	println()
	println("═══════════════════════════════════════════════════════════")
	println("📊 内存优化报告")
	println("═══════════════════════════════════════════════════════════")
	
	println("【处理统计】")
	println("  总结果数:", mo.totalResults)
	println("  优化的结果数:", mo.cleanedCount)
	
	if mo.originalSize > 0 {
		println("\n【内存使用】")
		print("  原始大小: ")
		print(stats["original_size_mb"].(float64))
		println(" MB")
		
		print("  优化后大小: ")
		print(stats["optimized_size_mb"].(float64))
		println(" MB")
		
		print("  节省大小: ")
		print(stats["saved_size_mb"].(float64))
		println(" MB")
		
		print("  节省比例: ")
		print(stats["saved_percent"].(float64))
		println("%")
	}
	
	println("\n【优化策略】")
	if mo.keepHTMLContent {
		println("  HTML内容: 保留（截断至", mo.maxHTMLLength, "字符）")
	} else {
		println("  HTML内容: 仅保留摘要（", mo.keepHTMLSummaryLength, "字符）")
	}
	
	println("═══════════════════════════════════════════════════════════")
}

// Reset 重置统计
func (mo *MemoryOptimizer) Reset() {
	mo.mutex.Lock()
	defer mo.mutex.Unlock()
	
	mo.totalResults = 0
	mo.originalSize = 0
	mo.optimizedSize = 0
	mo.cleanedCount = 0
}

// 辅助函数

// extractBetween 提取两个标记之间的内容
func extractBetween(content, start, end string) string {
	startIdx := strings.Index(strings.ToLower(content), strings.ToLower(start))
	if startIdx == -1 {
		return ""
	}
	startIdx += len(start)
	
	endIdx := strings.Index(strings.ToLower(content[startIdx:]), strings.ToLower(end))
	if endIdx == -1 {
		return ""
	}
	
	return strings.TrimSpace(content[startIdx : startIdx+endIdx])
}

// extractMetaContent 提取meta标签的content
func extractMetaContent(htmlContent, metaName string) string {
	// 简单的正则匹配替代（避免引入regexp包开销）
	lowerHTML := strings.ToLower(htmlContent)
	searchStr := `name="` + strings.ToLower(metaName) + `"`
	
	idx := strings.Index(lowerHTML, searchStr)
	if idx == -1 {
		searchStr = `name='` + strings.ToLower(metaName) + `'`
		idx = strings.Index(lowerHTML, searchStr)
	}
	
	if idx == -1 {
		return ""
	}
	
	// 查找content属性
	contentIdx := strings.Index(lowerHTML[idx:], `content="`)
	if contentIdx == -1 {
		contentIdx = strings.Index(lowerHTML[idx:], `content='`)
	}
	
	if contentIdx == -1 {
		return ""
	}
	
	contentStart := idx + contentIdx + 9 // 跳过 content="
	quote := htmlContent[contentStart-1]
	
	contentEnd := strings.IndexByte(htmlContent[contentStart:], quote)
	if contentEnd == -1 {
		return ""
	}
	
	return strings.TrimSpace(htmlContent[contentStart : contentStart+contentEnd])
}

// removeHTMLTags 移除HTML标签，只保留文本
func removeHTMLTags(htmlContent string) string {
	var result strings.Builder
	inTag := false
	
	for _, ch := range htmlContent {
		if ch == '<' {
			inTag = true
		} else if ch == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(ch)
		}
	}
	
	text := result.String()
	// 清理多余空白
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

