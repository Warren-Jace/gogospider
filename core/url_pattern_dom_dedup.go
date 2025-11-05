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
	"time"
)

// URLPatternWithDOMDeduplicator URL模式+DOM相似度去重器
// 核心思路：对于URL相似的请求，先访问N次计算DOM相似度，如果确认内容相似才跳过
type URLPatternWithDOMDeduplicator struct {
	mutex sync.RWMutex

	// URL模式分组（pattern -> 模式信息）
	patternGroups map[string]*PatternGroup

	// DOM相似度检测器
	domDetector *DOMSimilarityDetector

	// 配置
	sampleCount        int     // 采样次数（默认3次）
	domSimilarityThreshold float64 // DOM相似度阈值（默认0.85）

	// 统计信息
	stats URLPatternDOMStats
}

// PatternGroup URL模式分组
type PatternGroup struct {
	Pattern          string              // URL模式（不含参数值）
	SampleURLs       []string            // 采样的URL列表
	DOMSignatures    []*DOMSignature     // 采样页面的DOM签名
	HTMLContents     []string            // 采样页面的HTML内容
	SampleCount      int                 // 当前采样次数
	IsVerified       bool                // 是否已完成验证
	IsSimilar        bool                // 验证结果：是否相似
	AvgSimilarity    float64             // 平均相似度
	SkippedCount     int                 // 跳过的URL数量
	FirstSeenURL     string              // 第一次见到的URL
	VerificationTime string              // 验证完成时间
}

// URLPatternDOMStats 统计信息
type URLPatternDOMStats struct {
	TotalURLs          int // 总处理URL数
	UniquePatterns     int // 唯一模式数
	SamplingPatterns   int // 正在采样的模式数
	VerifiedPatterns   int // 已验证的模式数
	SimilarPatterns    int // 相似的模式数
	DifferentPatterns  int // 不同的模式数
	SkippedURLs        int // 跳过的URL数
	SampledURLs        int // 采样的URL数
}

// NewURLPatternWithDOMDeduplicator 创建URL模式+DOM去重器
func NewURLPatternWithDOMDeduplicator(sampleCount int, domThreshold float64) *URLPatternWithDOMDeduplicator {
	if sampleCount <= 0 {
		sampleCount = 3 // 默认采样3次
	}
	if domThreshold <= 0 || domThreshold > 1 {
		domThreshold = 0.85 // 默认相似度阈值85%
	}

	return &URLPatternWithDOMDeduplicator{
		patternGroups:          make(map[string]*PatternGroup),
		domDetector:            NewDOMSimilarityDetector(domThreshold),
		sampleCount:            sampleCount,
		domSimilarityThreshold: domThreshold,
		stats:                  URLPatternDOMStats{},
	}
}

// ShouldCrawl 判断是否应该爬取该URL（核心方法）
// 返回: (是否爬取, 原因, 是否需要DOM分析)
func (d *URLPatternWithDOMDeduplicator) ShouldCrawl(rawURL string) (bool, string, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.stats.TotalURLs++

	// 1. 提取URL模式
	pattern, err := d.extractURLPattern(rawURL)
	if err != nil {
		// 解析失败，保守处理：允许爬取
		return true, fmt.Sprintf("URL解析失败，保守允许: %v", err), false
	}

	// 2. 检查是否已有该模式的分组
	group, exists := d.patternGroups[pattern]
	if !exists {
		// 新模式，创建分组
		group = &PatternGroup{
			Pattern:       pattern,
			SampleURLs:    make([]string, 0, d.sampleCount),
			DOMSignatures: make([]*DOMSignature, 0, d.sampleCount),
			HTMLContents:  make([]string, 0, d.sampleCount),
			SampleCount:   0,
			IsVerified:    false,
			FirstSeenURL:  rawURL,
		}
		d.patternGroups[pattern] = group
		d.stats.UniquePatterns++
		d.stats.SamplingPatterns++
	}

	// 3. 如果已完成验证
	if group.IsVerified {
		if group.IsSimilar {
			// 验证结果：相似 -> 跳过
			group.SkippedCount++
			d.stats.SkippedURLs++
			return false, fmt.Sprintf("URL模式已验证为相似（平均相似度%.1f%%），跳过爬取", 
				group.AvgSimilarity*100), false
		} else {
			// 验证结果：不同 -> 允许爬取
			return true, "URL模式已验证为内容不同，允许爬取", false
		}
	}

	// 4. 采样阶段：还未达到采样次数
	if group.SampleCount < d.sampleCount {
		group.SampleURLs = append(group.SampleURLs, rawURL)
		group.SampleCount++
		d.stats.SampledURLs++
		
		return true, fmt.Sprintf("采样阶段 (%d/%d)，需要爬取并分析DOM", 
			group.SampleCount, d.sampleCount), true
	}

	// 5. 达到采样次数但还未验证（不应该出现这种情况）
	// 这种情况说明前面的URL爬取失败或未调用 RecordDOMSignature
	return false, "采样已完成但未验证，等待验证", false
}

// RecordDOMSignature 记录URL的DOM签名（爬取完成后调用）
func (d *URLPatternWithDOMDeduplicator) RecordDOMSignature(rawURL string, htmlContent string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 1. 提取URL模式
	pattern, err := d.extractURLPattern(rawURL)
	if err != nil {
		return fmt.Errorf("URL解析失败: %v", err)
	}

	// 2. 查找模式分组
	group, exists := d.patternGroups[pattern]
	if !exists {
		return fmt.Errorf("未找到模式分组: %s", pattern)
	}

	// 3. 如果已验证，不需要再记录
	if group.IsVerified {
		return nil
	}

	// 4. 提取DOM签名
	signature, err := d.domDetector.ExtractDOMSignature(rawURL, htmlContent)
	if err != nil {
		return fmt.Errorf("提取DOM签名失败: %v", err)
	}

	// 5. 保存DOM签名和HTML内容
	group.DOMSignatures = append(group.DOMSignatures, signature)
	group.HTMLContents = append(group.HTMLContents, htmlContent)

	// 6. 检查是否达到采样次数并完成验证
	if len(group.DOMSignatures) >= d.sampleCount {
		d.verifyPatternSimilarity(group)
	}

	return nil
}

// verifyPatternSimilarity 验证模式相似度（内部方法，已加锁）
func (d *URLPatternWithDOMDeduplicator) verifyPatternSimilarity(group *PatternGroup) {
	if group.IsVerified {
		return
	}

	if len(group.DOMSignatures) < d.sampleCount {
		return
	}

	// 计算所有采样页面之间的DOM相似度
	similarities := make([]float64, 0)

	// 两两比较，计算相似度
	for i := 0; i < len(group.DOMSignatures); i++ {
		for j := i + 1; j < len(group.DOMSignatures); j++ {
			similarity := d.calculateDOMSimilarity(
				group.DOMSignatures[i],
				group.DOMSignatures[j],
			)
			similarities = append(similarities, similarity)
		}
	}

	// 计算平均相似度
	totalSimilarity := 0.0
	for _, sim := range similarities {
		totalSimilarity += sim
	}
	avgSimilarity := totalSimilarity / float64(len(similarities))

	// 保存结果
	group.AvgSimilarity = avgSimilarity
	group.IsVerified = true
	group.VerificationTime = getCurrentTime()

	// 更新统计
	d.stats.SamplingPatterns--
	d.stats.VerifiedPatterns++

	// 判断是否相似
	if avgSimilarity >= d.domSimilarityThreshold {
		group.IsSimilar = true
		d.stats.SimilarPatterns++
	} else {
		group.IsSimilar = false
		d.stats.DifferentPatterns++
	}
}

// extractURLPattern 提取URL模式（不含参数值）
func (d *URLPatternWithDOMDeduplicator) extractURLPattern(rawURL string) (string, error) {
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
		// Fragment通常是客户端路由，保留
		pattern += "#" + parsedURL.Fragment
	}

	return pattern, nil
}

// calculateHash 计算字符串的MD5 hash
func (d *URLPatternWithDOMDeduplicator) calculateHash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetStatistics 获取统计信息
func (d *URLPatternWithDOMDeduplicator) GetStatistics() URLPatternDOMStats {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.stats
}

// GetPatternGroups 获取所有模式分组信息
func (d *URLPatternWithDOMDeduplicator) GetPatternGroups() []*PatternGroup {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	groups := make([]*PatternGroup, 0, len(d.patternGroups))
	for _, group := range d.patternGroups {
		groups = append(groups, group)
	}

	// 按跳过次数排序
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].SkippedCount > groups[j].SkippedCount
	})

	return groups
}

// PrintReport 打印详细报告
func (d *URLPatternWithDOMDeduplicator) PrintReport() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("            URL模式+DOM相似度去重报告")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("\n【总体统计】\n")
	fmt.Printf("  处理URL总数:       %d\n", d.stats.TotalURLs)
	fmt.Printf("  唯一URL模式:       %d\n", d.stats.UniquePatterns)
	fmt.Printf("  正在采样的模式:     %d\n", d.stats.SamplingPatterns)
	fmt.Printf("  已验证的模式:       %d\n", d.stats.VerifiedPatterns)
	fmt.Printf("    - 相似模式:       %d (%.1f%%)\n", d.stats.SimilarPatterns,
		getPercentage(d.stats.SimilarPatterns, d.stats.VerifiedPatterns))
	fmt.Printf("    - 不同模式:       %d (%.1f%%)\n", d.stats.DifferentPatterns,
		getPercentage(d.stats.DifferentPatterns, d.stats.VerifiedPatterns))
	fmt.Printf("  采样的URL数:       %d\n", d.stats.SampledURLs)
	fmt.Printf("  跳过的URL数:       %d\n", d.stats.SkippedURLs)

	if d.stats.TotalURLs > 0 {
		fmt.Printf("  去重率:           %.1f%%\n",
			float64(d.stats.SkippedURLs)/float64(d.stats.TotalURLs)*100)
	}

	// 显示相似模式详情
	fmt.Printf("\n【相似模式详情】（Top 10）\n")
	fmt.Println(strings.Repeat("-", 80))

	groups := d.GetPatternGroups()
	count := 0
	for _, group := range groups {
		if !group.IsVerified || !group.IsSimilar {
			continue
		}

		count++
		if count > 10 {
			break
		}

		fmt.Printf("\n%d. 模式: %s\n", count, group.Pattern)
		fmt.Printf("   平均DOM相似度: %.1f%%\n", group.AvgSimilarity*100)
		fmt.Printf("   采样URL数: %d\n", len(group.SampleURLs))
		fmt.Printf("   跳过URL数: %d\n", group.SkippedCount)
		fmt.Printf("   首次URL: %s\n", group.FirstSeenURL)
		if group.VerificationTime != "" {
			fmt.Printf("   验证时间: %s\n", group.VerificationTime)
		}
		
		// 显示采样URL示例
		if len(group.SampleURLs) > 0 {
			fmt.Printf("   采样示例:\n")
			for i, sampleURL := range group.SampleURLs {
				if i >= 3 { // 只显示前3个
					fmt.Printf("     ... (共%d个)\n", len(group.SampleURLs))
					break
				}
				fmt.Printf("     [%d] %s\n", i+1, sampleURL)
			}
		}
	}

	// 显示不同模式详情
	fmt.Printf("\n【内容不同的模式】（说明：这些URL模式相似但内容不同，都会保留）\n")
	fmt.Println(strings.Repeat("-", 80))

	differentCount := 0
	for _, group := range groups {
		if !group.IsVerified || group.IsSimilar {
			continue
		}

		differentCount++
		if differentCount > 5 {
			break
		}

		fmt.Printf("\n%d. 模式: %s\n", differentCount, group.Pattern)
		fmt.Printf("   平均DOM相似度: %.1f%% (低于阈值%.1f%%，内容确实不同)\n",
			group.AvgSimilarity*100, d.domSimilarityThreshold*100)
		fmt.Printf("   采样URL数: %d\n", len(group.SampleURLs))
		fmt.Printf("   首次URL: %s\n", group.FirstSeenURL)
	}

	if differentCount == 0 {
		fmt.Printf("\n  暂无内容不同的模式\n")
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

// Reset 重置去重器
func (d *URLPatternWithDOMDeduplicator) Reset() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.patternGroups = make(map[string]*PatternGroup)
	d.stats = URLPatternDOMStats{}
}

// GetSimilarityThreshold 获取相似度阈值
func (d *URLPatternWithDOMDeduplicator) GetSimilarityThreshold() float64 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.domSimilarityThreshold
}

// SetSimilarityThreshold 设置相似度阈值
func (d *URLPatternWithDOMDeduplicator) SetSimilarityThreshold(threshold float64) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if threshold > 0 && threshold <= 1 {
		d.domSimilarityThreshold = threshold
		d.domDetector.SetThreshold(threshold)
	}
}

// GetSampleCount 获取采样次数
func (d *URLPatternWithDOMDeduplicator) GetSampleCount() int {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.sampleCount
}

// SetSampleCount 设置采样次数
func (d *URLPatternWithDOMDeduplicator) SetSampleCount(count int) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if count > 0 {
		d.sampleCount = count
	}
}

// getPercentage 计算百分比
func getPercentage(part, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(part) / float64(total) * 100
}

// calculateDOMSimilarity 计算两个DOM签名的相似度
func (d *URLPatternWithDOMDeduplicator) calculateDOMSimilarity(sig1, sig2 *DOMSignature) float64 {
	similarities := make([]float64, 0)

	// 1. 结构哈希快速对比（完全相同）
	if sig1.StructHash == sig2.StructHash {
		return 1.0
	}

	// 2. SimHash相似度（汉明距离）
	simHashSim := d.calculateSimHashSimilarity(sig1.SimHash, sig2.SimHash)
	similarities = append(similarities, simHashSim)

	// 3. 结构特征相似度
	structFeatureSim := d.calculateStructFeatureSimilarity(sig1, sig2)
	similarities = append(similarities, structFeatureSim)

	// 4. 标签分布相似度（余弦相似度）
	tagDistSim := d.calculateTagDistributionSimilarity(sig1.TagCount, sig2.TagCount)
	similarities = append(similarities, tagDistSim)

	// 综合相似度（加权平均）
	totalSimilarity := 0.0
	for _, sim := range similarities {
		totalSimilarity += sim
	}
	
	if len(similarities) == 0 {
		return 0.0
	}
	
	return totalSimilarity / float64(len(similarities))
}

// calculateSimHashSimilarity 计算SimHash相似度（基于汉明距离）
func (d *URLPatternWithDOMDeduplicator) calculateSimHashSimilarity(hash1, hash2 uint64) float64 {
	// 计算汉明距离
	xor := hash1 ^ hash2
	distance := 0
	for xor != 0 {
		distance++
		xor &= xor - 1 // 清除最低位的1
	}

	// 转换为相似度（距离越小，相似度越高）
	maxDistance := 64.0
	similarity := 1.0 - (float64(distance) / maxDistance)
	return similarity
}

// calculateStructFeatureSimilarity 计算结构特征相似度
func (d *URLPatternWithDOMDeduplicator) calculateStructFeatureSimilarity(sig1, sig2 *DOMSignature) float64 {
	features := []struct {
		val1, val2 float64
		weight     float64
	}{
		{float64(sig1.Depth), float64(sig2.Depth), 0.2},
		{float64(sig1.NodeCount), float64(sig2.NodeCount), 0.2},
		{float64(sig1.LinkCount), float64(sig2.LinkCount), 0.2},
		{float64(sig1.FormCount), float64(sig2.FormCount), 0.2},
		{float64(sig1.InputCount), float64(sig2.InputCount), 0.2},
	}

	totalSimilarity := 0.0
	for _, f := range features {
		// 计算特征相似度（使用归一化差异）
		maxVal := f.val1
		if f.val2 > maxVal {
			maxVal = f.val2
		}
		if maxVal == 0 {
			totalSimilarity += f.weight // 两者都为0，完全相同
		} else {
			diff := f.val1 - f.val2
			if diff < 0 {
				diff = -diff
			}
			similarity := 1.0 - (diff / maxVal)
			totalSimilarity += similarity * f.weight
		}
	}

	return totalSimilarity
}

// calculateTagDistributionSimilarity 计算标签分布相似度（余弦相似度）
func (d *URLPatternWithDOMDeduplicator) calculateTagDistributionSimilarity(tags1, tags2 map[string]int) float64 {
	if len(tags1) == 0 || len(tags2) == 0 {
		return 0.0
	}

	// 获取所有标签
	allTags := make(map[string]bool)
	for tag := range tags1 {
		allTags[tag] = true
	}
	for tag := range tags2 {
		allTags[tag] = true
	}

	// 计算余弦相似度
	var dotProduct, norm1, norm2 float64
	for tag := range allTags {
		count1 := float64(tags1[tag])
		count2 := float64(tags2[tag])

		dotProduct += count1 * count2
		norm1 += count1 * count1
		norm2 += count2 * count2
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// getCurrentTime 获取当前时间字符串
func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

