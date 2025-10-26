package core

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// URLPatternDeduplicator 基于URL模式的去重器
// 核心思路：提取URL模式（不含参数值）+ 请求方式，计算hash去重
type URLPatternDeduplicator struct {
	mutex sync.RWMutex
	
	// 已处理的URL模式hash集合
	processedPatterns map[string]bool
	
	// 模式详情（用于调试和统计）
	patternDetails map[string]*PatternDetail
	
	// 统计信息
	stats DeduplicationStats
}

// PatternDetail 模式详情
type PatternDetail struct {
	Pattern       string   // URL模式（不含参数值）
	Method        string   // 请求方式（GET/POST等）
	Hash          string   // 模式hash
	FirstURL      string   // 第一个URL示例
	DuplicateURLs []string // 重复的URL列表（采样前5个）
	Count         int      // 重复次数
}

// DeduplicationStats 去重统计
type DeduplicationStats struct {
	TotalURLs       int // 总处理URL数
	UniquePatterns  int // 唯一模式数
	DuplicateURLs   int // 重复URL数
	SavedRequests   int // 节省的请求数
}

// NewURLPatternDeduplicator 创建URL模式去重器
func NewURLPatternDeduplicator() *URLPatternDeduplicator {
	return &URLPatternDeduplicator{
		processedPatterns: make(map[string]bool),
		patternDetails:    make(map[string]*PatternDetail),
		stats:             DeduplicationStats{},
	}
}

// ShouldProcess 判断URL是否应该处理（核心方法）
// 返回: (是否处理, 模式hash, 原因)
func (d *URLPatternDeduplicator) ShouldProcess(rawURL string, method string) (bool, string, string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.stats.TotalURLs++
	
	// 1. 提取URL模式（不含参数值）
	pattern, err := d.extractURLPattern(rawURL)
	if err != nil {
		// 解析失败，保守处理
		return true, "", fmt.Sprintf("URL解析失败: %v", err)
	}
	
	// 2. 构造完整模式：方法 + URL模式
	fullPattern := fmt.Sprintf("%s %s", strings.ToUpper(method), pattern)
	
	// 3. 计算hash
	patternHash := d.calculateHash(fullPattern)
	
	// 4. 检查是否已处理
	if d.processedPatterns[patternHash] {
		// 已处理，记录重复
		d.stats.DuplicateURLs++
		d.stats.SavedRequests++
		
		// 更新模式详情
		if detail, exists := d.patternDetails[patternHash]; exists {
			detail.Count++
			// 只保留前5个重复URL作为样本
			if len(detail.DuplicateURLs) < 5 {
				detail.DuplicateURLs = append(detail.DuplicateURLs, rawURL)
			}
		}
		
		return false, patternHash, fmt.Sprintf("重复模式: %s", fullPattern)
	}
	
	// 5. 新模式，标记为已处理
	d.processedPatterns[patternHash] = true
	d.stats.UniquePatterns++
	
	// 记录模式详情
	d.patternDetails[patternHash] = &PatternDetail{
		Pattern:       pattern,
		Method:        strings.ToUpper(method),
		Hash:          patternHash,
		FirstURL:      rawURL,
		DuplicateURLs: make([]string, 0),
		Count:         1,
	}
	
	return true, patternHash, fmt.Sprintf("新模式: %s", fullPattern)
}

// extractURLPattern 提取URL模式（不含参数值）
// 例如: http://test.com?a=123&b=456 -> http://test.com?a=&b=
func (d *URLPatternDeduplicator) extractURLPattern(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	
	// 基础部分：协议 + 主机 + 路径
	pattern := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	
	// 处理查询参数（只保留参数名，不保留参数值）
	if parsedURL.RawQuery != "" {
		queryParams := parsedURL.Query()
		
		// 提取参数名并排序（确保一致性）
		paramNames := make([]string, 0, len(queryParams))
		for paramName := range queryParams {
			paramNames = append(paramNames, paramName)
		}
		sort.Strings(paramNames)
		
		// 构造模式：参数名=（不含值）
		paramParts := make([]string, 0, len(paramNames))
		for _, paramName := range paramNames {
			paramParts = append(paramParts, paramName+"=")
		}
		
		if len(paramParts) > 0 {
			pattern += "?" + strings.Join(paramParts, "&")
		}
	}
	
	// 处理Fragment（如果有）
	if parsedURL.Fragment != "" {
		pattern += "#" + parsedURL.Fragment
	}
	
	return pattern, nil
}

// calculateHash 计算字符串的MD5 hash
func (d *URLPatternDeduplicator) calculateHash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetStatistics 获取统计信息
func (d *URLPatternDeduplicator) GetStatistics() DeduplicationStats {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.stats
}

// GetPatternDetails 获取模式详情
func (d *URLPatternDeduplicator) GetPatternDetails() []*PatternDetail {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	details := make([]*PatternDetail, 0, len(d.patternDetails))
	for _, detail := range d.patternDetails {
		details = append(details, detail)
	}
	
	// 按重复次数排序
	sort.Slice(details, func(i, j int) bool {
		return details[i].Count > details[j].Count
	})
	
	return details
}

// PrintReport 打印去重报告
func (d *URLPatternDeduplicator) PrintReport() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("                    URL模式去重报告")
	fmt.Println(strings.Repeat("=", 80))
	
	fmt.Printf("\n【总体统计】\n")
	fmt.Printf("  处理URL总数:    %d\n", d.stats.TotalURLs)
	fmt.Printf("  唯一模式数:     %d\n", d.stats.UniquePatterns)
	fmt.Printf("  重复URL数:      %d\n", d.stats.DuplicateURLs)
	fmt.Printf("  节省请求数:     %d\n", d.stats.SavedRequests)
	
	if d.stats.TotalURLs > 0 {
		deduplicationRate := float64(d.stats.DuplicateURLs) / float64(d.stats.TotalURLs) * 100
		fmt.Printf("  去重率:         %.1f%%\n", deduplicationRate)
	}
	
	// 显示Top重复模式
	fmt.Printf("\n【Top 10 重复模式】\n")
	fmt.Println(strings.Repeat("-", 80))
	
	details := d.GetPatternDetails()
	topCount := 10
	if len(details) < topCount {
		topCount = len(details)
	}
	
	for i := 0; i < topCount; i++ {
		detail := details[i]
		fmt.Printf("\n%d. %s %s\n", i+1, detail.Method, detail.Pattern)
		fmt.Printf("   重复次数: %d\n", detail.Count)
		fmt.Printf("   首次URL:  %s\n", detail.FirstURL)
		
		if len(detail.DuplicateURLs) > 0 {
			fmt.Printf("   重复示例: ")
			if len(detail.DuplicateURLs) <= 2 {
				fmt.Printf("%s\n", strings.Join(detail.DuplicateURLs, ", "))
			} else {
				fmt.Printf("%s, ... (共%d个)\n", 
					strings.Join(detail.DuplicateURLs[:2], ", "), 
					len(detail.DuplicateURLs))
			}
		}
	}
	
	fmt.Println("\n" + strings.Repeat("=", 80))
}

// Reset 重置去重器
func (d *URLPatternDeduplicator) Reset() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.processedPatterns = make(map[string]bool)
	d.patternDetails = make(map[string]*PatternDetail)
	d.stats = DeduplicationStats{}
}

// IsProcessed 检查URL模式是否已处理（不更新状态）
func (d *URLPatternDeduplicator) IsProcessed(rawURL string, method string) bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	pattern, err := d.extractURLPattern(rawURL)
	if err != nil {
		return false
	}
	
	fullPattern := fmt.Sprintf("%s %s", strings.ToUpper(method), pattern)
	patternHash := d.calculateHash(fullPattern)
	
	return d.processedPatterns[patternHash]
}

// GetPattern 获取URL的模式字符串（用于调试）
func (d *URLPatternDeduplicator) GetPattern(rawURL string, method string) string {
	pattern, err := d.extractURLPattern(rawURL)
	if err != nil {
		return ""
	}
	
	return fmt.Sprintf("%s %s", strings.ToUpper(method), pattern)
}

