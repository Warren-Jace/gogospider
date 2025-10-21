package core

import (
	"crypto/md5"
	"encoding/hex"
	"math"
	"net/url"
	"sort"
	"strings"
)

// DuplicateHandler 去重处理器
type DuplicateHandler struct {
	// 已处理URL的哈希集合
	processedURLs map[string]bool
	
	// 已处理内容的哈希集合
	processedContent map[string]bool
	
	// 相似度阈值
	similarityThreshold float64
}

// NewDuplicateHandler 创建去重处理器实例
func NewDuplicateHandler(threshold float64) *DuplicateHandler {
	return &DuplicateHandler{
		processedURLs:      make(map[string]bool),
		processedContent:   make(map[string]bool),
		similarityThreshold: threshold,
	}
}

// IsDuplicateURL 检查URL是否重复
func (d *DuplicateHandler) IsDuplicateURL(rawURL string) bool {
	// 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		// 如果无法解析URL，则使用原始去重逻辑
		hash := d.calculateMD5(rawURL)
		if _, exists := d.processedURLs[hash]; exists {
			return true
		}
		d.processedURLs[hash] = true
		return false
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
	
	// 检查是否已处理过
	if _, exists := d.processedURLs[hash]; exists {
		return true
	}
	
	// 添加到已处理集合
	d.processedURLs[hash] = true
	return false
}

// IsDuplicateContent 检查内容是否重复
func (d *DuplicateHandler) IsDuplicateContent(content string) bool {
	// 计算内容的MD5哈希
	hash := d.calculateMD5(content)
	
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

// ClearProcessed 清空已处理记录
func (d *DuplicateHandler) ClearProcessed() {
	d.processedURLs = make(map[string]bool)
	d.processedContent = make(map[string]bool)
}