package core

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// URLInfo URL信息
type URLInfo struct {
	URL       string // 原始URL
	Index     int    // 序号（在results中的索引，从1开始）
	IsCrawled bool   // 是否真正被爬取过（不是仅仅被发现）
}

// DuplicateHandler 去重处理器
type DuplicateHandler struct {
	// 🔧 修复：添加互斥锁保护并发访问
	mutex sync.RWMutex
	
	// 已处理URL的哈希集合
	processedURLs map[string]bool
	
	// 🆕 v4.7: 哈希到URL信息的映射（包含原始URL、序号和爬取状态）
	hashToInfo map[string]*URLInfo
	
	// 已处理内容的哈希集合
	processedContent map[string]bool
	
	// 相似度阈值
	similarityThreshold float64
	
	// 🆕 调试统计信息
	totalChecks   int64
	duplicateHits int64
	enableDebug   bool
}

// NewDuplicateHandler 创建去重处理器实例
func NewDuplicateHandler(threshold float64) *DuplicateHandler {
	d := &DuplicateHandler{
		processedURLs:       make(map[string]bool),
		hashToInfo:          make(map[string]*URLInfo),
		processedContent:    make(map[string]bool),
		similarityThreshold: threshold,
		enableDebug:         true, // 启用调试模式
	}
	fmt.Printf("🔧 [去重器] 创建新实例 (地址: %p)\n", d)
	return d
}

// IsDuplicateURL 检查URL是否重复
func (d *DuplicateHandler) IsDuplicateURL(rawURL string) bool {
	isDup, _ := d.IsDuplicateURLWithOriginal(rawURL)
	return isDup
}

// MarkURLAsStarted 标记URL开始爬取（在OnRequest时调用）
func (d *DuplicateHandler) MarkURLAsStarted(rawURL string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	// 解析URL并计算hash
	parsedURL, err := url.Parse(rawURL)
	var hash string
	if err != nil {
		hash = d.calculateMD5(rawURL)
	} else {
		urlKey := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
		if parsedURL.RawQuery != "" {
			queryParams := parsedURL.Query()
			var paramKeys []string
			for key := range queryParams {
				paramKeys = append(paramKeys, key)
			}
			sort.Strings(paramKeys)
			
			var queryParts []string
			for _, key := range paramKeys {
				for _, value := range queryParams[key] {
					queryParts = append(queryParts, key+"="+value)
				}
			}
			if len(queryParts) > 0 {
				urlKey += "?" + strings.Join(queryParts, "&")
			}
		}
		hash = d.calculateMD5(urlKey)
	}
	
	// 如果URL不存在，添加到去重器（标记为开始爬取）
	if _, exists := d.processedURLs[hash]; !exists {
		d.processedURLs[hash] = true
		d.hashToInfo[hash] = &URLInfo{
			URL:       rawURL,
			Index:     0,
			IsCrawled: false,
		}
	}
}

// 🆕 v4.7: IsDuplicateURLWithOriginal 检查URL是否重复，并返回重复的URL信息
// 返回: (是否重复, 重复的URL信息)
// 注意：此方法只检查，不添加！
func (d *DuplicateHandler) IsDuplicateURLWithOriginal(rawURL string) (bool, *URLInfo) {
	// 🆕 统计检查次数
	atomic.AddInt64(&d.totalChecks, 1)
	
	// 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		// 如果无法解析URL，则使用原始去重逻辑
		hash := d.calculateMD5(rawURL)
		
		// 🔧 修复：加锁保护并发访问
		d.mutex.Lock()
		defer d.mutex.Unlock()
		
		if _, exists := d.processedURLs[hash]; exists {
			atomic.AddInt64(&d.duplicateHits, 1)
			if d.enableDebug {
				fmt.Printf("❌ [去重] 跳过重复URL: %s\n", rawURL)
			}
			info := d.hashToInfo[hash]
			return true, info
		}
		// 🔧 v4.7: 不在检查时添加，只检查不添加
		return false, nil
	}
	
	// 构造用于去重检查的URL键值
	// 包含协议、主机和路径，但不包含查询参数
	urlKey := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	
	// 如果有查询参数，则将其包含在键值中
	if parsedURL.RawQuery != "" {
		// 解析查询参数
		queryParams := parsedURL.Query()
		
		// 对查询参数进行排序以确保一致性
		var paramKeys []string
		for key := range queryParams {
			paramKeys = append(paramKeys, key)
		}
		sort.Strings(paramKeys)
		
		// 构建排序后的查询字符串
		var queryParts []string
		for _, key := range paramKeys {
			for _, value := range queryParams[key] {
				queryParts = append(queryParts, key+"="+value)
			}
		}
		
		if len(queryParts) > 0 {
			urlKey += "?" + strings.Join(queryParts, "&")
		}
	}
	
	// 计算URL键值的MD5哈希
	hash := d.calculateMD5(urlKey)
	
	// 🔧 修复：加锁保护并发访问
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	// 检查是否已处理过
	if _, exists := d.processedURLs[hash]; exists {
		atomic.AddInt64(&d.duplicateHits, 1)
		info := d.hashToInfo[hash]
		if d.enableDebug && strings.Contains(rawURL, "showimage.php") {
			// 只打印showimage.php的重复信息，避免日志过多
			fmt.Printf("❌ [去重] 跳过重复URL: %s\n    → 规范化: %s\n    → hash: %s\n    → 原始URL: %s (序号:%d, 已爬取:%v)\n", 
				rawURL, urlKey, hash[:8], info.URL, info.Index, info.IsCrawled)
		}
		return true, info
	}
	
	// 🔧 v4.7: 不在检查时添加，只检查不添加
	return false, nil
}

// IsDuplicateContent 检查内容是否重复
func (d *DuplicateHandler) IsDuplicateContent(content string) bool {
	// 计算内容的MD5哈希
	hash := d.calculateMD5(content)
	
	// 🔧 修复：加锁保护并发访问
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	// 检查是否已处理过
	if _, exists := d.processedContent[hash]; exists {
		return true
	}
	
	// 添加到已处理集合
	d.processedContent[hash] = true
	return false
}

// IsSimilarContent 基于相似度检查内容是否相似
func (d *DuplicateHandler) IsSimilarContent(content1, content2 string) bool {
	similarity := d.calculateSimilarity(content1, content2)
	return similarity >= d.similarityThreshold
}

// calculateMD5 计算字符串的MD5哈希值
func (d *DuplicateHandler) calculateMD5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

// calculateSimilarity 计算两个字符串的相似度（使用余弦相似度简化版）
func (d *DuplicateHandler) calculateSimilarity(text1, text2 string) float64 {
	// 转换为小写并分割为词汇
	words1 := strings.Fields(strings.ToLower(text1))
	words2 := strings.Fields(strings.ToLower(text2))
	
	// 创建词汇频率映射
	freq1 := make(map[string]int)
	freq2 := make(map[string]int)
	
	for _, word := range words1 {
		// 简单清理词汇（移除标点符号）
		cleanWord := d.cleanWord(word)
		if cleanWord != "" {
			freq1[cleanWord]++
		}
	}
	
	for _, word := range words2 {
		// 简单清理词汇（移除标点符号）
		cleanWord := d.cleanWord(word)
		if cleanWord != "" {
			freq2[cleanWord]++
		}
	}
	
	// 计算点积
	dotProduct := 0.0
	for word, freq := range freq1 {
		if freq2[word] > 0 {
			dotProduct += float64(freq * freq2[word])
		}
	}
	
	// 计算向量的模
	magnitude1 := 0.0
	magnitude2 := 0.0
	
	for _, freq := range freq1 {
		magnitude1 += float64(freq * freq)
	}
	
	for _, freq := range freq2 {
		magnitude2 += float64(freq * freq)
	}
	
	// 计算余弦相似度
	if magnitude1 == 0 || magnitude2 == 0 {
		return 0.0
	}
	
	similarity := dotProduct / (math.Sqrt(magnitude1) * math.Sqrt(magnitude2))
	return similarity
}

// cleanWord 清理词汇，移除标点符号
func (d *DuplicateHandler) cleanWord(word string) string {
	// 移除常见的标点符号
	cleaned := strings.Trim(word, ".,;:!?()[]{}\"'`-")
	return strings.ToLower(cleaned)
}

// IsSimilarDOM 基于DOM结构检查相似性
func (d *DuplicateHandler) IsSimilarDOM(dom1, dom2 string) bool {
	// 提取DOM结构特征
	features1 := d.extractDOMFeatures(dom1)
	features2 := d.extractDOMFeatures(dom2)
	
	// 计算特征相似度
	similarity := d.calculateFeatureSimilarity(features1, features2)
	return similarity >= d.similarityThreshold
}

// extractDOMFeatures 提取DOM结构特征
func (d *DuplicateHandler) extractDOMFeatures(dom string) map[string]int {
	features := make(map[string]int)
	
	// 简化的DOM特征提取
	// 实际应用中可以使用HTML解析器提取更精确的特征
	
	// 统计标签类型
	tagPatterns := []string{"<div", "<span", "<a", "<img", "<form", "<input", "<button"}
	
	for _, pattern := range tagPatterns {
		count := strings.Count(dom, pattern)
		if count > 0 {
			features[pattern] = count
		}
	}
	
	// 统计类名和ID
	// 这里简化处理
	classCount := strings.Count(dom, "class=")
	idCount := strings.Count(dom, "id=")
	
	if classCount > 0 {
		features["class"] = classCount
	}
	
	if idCount > 0 {
		features["id"] = idCount
	}
	
	return features
}

// calculateFeatureSimilarity 计算特征相似度
func (d *DuplicateHandler) calculateFeatureSimilarity(features1, features2 map[string]int) float64 {
	// 计算点积
	dotProduct := 0.0
	for feature, freq := range features1 {
		if features2[feature] > 0 {
			dotProduct += float64(freq * features2[feature])
		}
	}
	
	// 计算向量的模
	magnitude1 := 0.0
	magnitude2 := 0.0
	
	for _, freq := range features1 {
		magnitude1 += float64(freq * freq)
	}
	
	for _, freq := range features2 {
		magnitude2 += float64(freq * freq)
	}
	
	// 计算余弦相似度
	if magnitude1 == 0 || magnitude2 == 0 {
		return 0.0
	}
	
	similarity := dotProduct / (math.Sqrt(magnitude1) * math.Sqrt(magnitude2))
	return similarity
}

// 🆕 v4.7: UpdateURLInfo 更新URL信息（标记为已爬取并设置序号）
func (d *DuplicateHandler) UpdateURLInfo(rawURL string, index int, isCrawled bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	// 解析URL并计算hash（与IsDuplicateURLWithOriginal中的逻辑一致）
	parsedURL, err := url.Parse(rawURL)
	var hash string
	if err != nil {
		hash = d.calculateMD5(rawURL)
	} else {
		urlKey := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
		if parsedURL.RawQuery != "" {
			queryParams := parsedURL.Query()
			var paramKeys []string
			for key := range queryParams {
				paramKeys = append(paramKeys, key)
			}
			sort.Strings(paramKeys)
			
			var queryParts []string
			for _, key := range paramKeys {
				for _, value := range queryParams[key] {
					queryParts = append(queryParts, key+"="+value)
				}
			}
			if len(queryParts) > 0 {
				urlKey += "?" + strings.Join(queryParts, "&")
			}
		}
		hash = d.calculateMD5(urlKey)
	}
	
	// 更新URL信息
	if info, exists := d.hashToInfo[hash]; exists {
		// 🔧 v4.7: 只更新序号和状态（如果提供）
		// 如果Index > 0，则更新序号
		if index > 0 {
			info.Index = index
		}
		// IsCrawled只能从false变成true，不能反向（一旦爬取成功就永远是true）
		if isCrawled {
			info.IsCrawled = true
		}
	}
}

// ClearProcessed 清空已处理记录
func (d *DuplicateHandler) ClearProcessed() {
	// 🔧 修复：加锁保护并发访问
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.processedURLs = make(map[string]bool)
	d.hashToInfo = make(map[string]*URLInfo)
	d.processedContent = make(map[string]bool)
}

// PrintStats 打印统计信息
func (d *DuplicateHandler) PrintStats() {
	totalChecks := atomic.LoadInt64(&d.totalChecks)
	duplicateHits := atomic.LoadInt64(&d.duplicateHits)
	
	if totalChecks == 0 {
		fmt.Println("\n⚠️  [去重器统计] 未进行任何去重检查！")
		return
	}
	
	dupRate := float64(duplicateHits) / float64(totalChecks) * 100
	
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 去重器统计信息")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("🔍 总检查次数: %d\n", totalChecks)
	fmt.Printf("❌ 重复命中: %d\n", duplicateHits)
	fmt.Printf("✅ 新URL: %d\n", totalChecks-duplicateHits)
	fmt.Printf("📈 去重率: %.1f%%\n", dupRate)
	
	d.mutex.RLock()
	urlCount := len(d.processedURLs)
	d.mutex.RUnlock()
	
	fmt.Printf("💾 已存储URL数: %d\n", urlCount)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}