package core

import (
	"crypto/md5"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// DOMSimilarityDetector DOM相似度检测器
type DOMSimilarityDetector struct {
	// 存储页面DOM结构特征
	pageSignatures map[string]*DOMSignature
	
	// 相似页面记录
	similarPages []SimilarPageRecord
	
	// 配置
	threshold        float64 // 相似度阈值（0-1之间）
	useStructHash    bool    // 是否使用结构哈希
	useTagSequence   bool    // 是否使用标签序列
	useSimHash       bool    // 是否使用SimHash
	
	mutex sync.RWMutex
}

// DOMSignature DOM结构签名
type DOMSignature struct {
	URL            string
	StructHash     string              // 结构哈希
	TagSequence    []string            // 标签序列
	TagCount       map[string]int      // 标签计数
	SimHash        uint64              // SimHash值
	Depth          int                 // DOM深度
	NodeCount      int                 // 节点总数
	LinkCount      int                 // 链接数量
	FormCount      int                 // 表单数量
	InputCount     int                 // 输入框数量
	StructFeatures map[string]float64  // 结构特征向量
}

// SimilarPageRecord 相似页面记录
type SimilarPageRecord struct {
	URL            string  // 当前页面URL
	SimilarToURL   string  // 相似的页面URL
	Similarity     float64 // 相似度分数（0-1）
	Reason         string  // 相似原因
	SkipCrawl      bool    // 是否跳过爬取
}

// NewDOMSimilarityDetector 创建DOM相似度检测器
func NewDOMSimilarityDetector(threshold float64) *DOMSimilarityDetector {
	if threshold <= 0 || threshold > 1 {
		threshold = 0.85 // 默认阈值85%
	}
	
	return &DOMSimilarityDetector{
		pageSignatures: make(map[string]*DOMSignature),
		similarPages:   make([]SimilarPageRecord, 0),
		threshold:      threshold,
		useStructHash:  true,
		useTagSequence: true,
		useSimHash:     true,
	}
}

// ExtractDOMSignature 提取DOM结构签名
func (dsd *DOMSimilarityDetector) ExtractDOMSignature(url string, htmlContent string) (*DOMSignature, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %v", err)
	}
	
	signature := &DOMSignature{
		URL:            url,
		TagCount:       make(map[string]int),
		TagSequence:    make([]string, 0),
		StructFeatures: make(map[string]float64),
	}
	
	// 1. 提取标签序列和统计
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		tagName := strings.ToLower(goquery.NodeName(s))
		signature.TagSequence = append(signature.TagSequence, tagName)
		signature.TagCount[tagName]++
		signature.NodeCount++
	})
	
	// 2. 统计特殊元素
	signature.LinkCount = doc.Find("a").Length()
	signature.FormCount = doc.Find("form").Length()
	signature.InputCount = doc.Find("input").Length()
	
	// 3. 计算DOM深度
	signature.Depth = dsd.calculateDOMDepth(doc.Selection)
	
	// 4. 生成结构哈希
	signature.StructHash = dsd.generateStructHash(signature)
	
	// 5. 计算SimHash
	signature.SimHash = dsd.calculateSimHash(signature.TagSequence)
	
	// 6. 生成结构特征向量
	signature.StructFeatures = dsd.generateStructFeatures(signature)
	
	return signature, nil
}

// CheckSimilarity 检查页面相似度
func (dsd *DOMSimilarityDetector) CheckSimilarity(url string, htmlContent string) (bool, *SimilarPageRecord) {
	dsd.mutex.Lock()
	defer dsd.mutex.Unlock()
	
	// 提取当前页面的DOM签名
	currentSig, err := dsd.ExtractDOMSignature(url, htmlContent)
	if err != nil {
		// 解析失败，允许爬取
		return false, nil
	}
	
	// 与已存储的页面进行对比
	for _, storedSig := range dsd.pageSignatures {
		similarity, reason := dsd.compareDOMSignatures(currentSig, storedSig)
		
		if similarity >= dsd.threshold {
			// 相似度超过阈值，判定为相似页面
			record := &SimilarPageRecord{
				URL:          url,
				SimilarToURL: storedSig.URL,
				Similarity:   similarity,
				Reason:       reason,
				SkipCrawl:    true,
			}
			
			dsd.similarPages = append(dsd.similarPages, *record)
			return true, record
		}
	}
	
	// 未找到相似页面，存储当前签名
	dsd.pageSignatures[url] = currentSig
	return false, nil
}

// compareDOMSignatures 比较两个DOM签名的相似度
func (dsd *DOMSimilarityDetector) compareDOMSignatures(sig1, sig2 *DOMSignature) (float64, string) {
	similarities := make([]float64, 0)
	reasons := make([]string, 0)
	
	// 1. 结构哈希快速对比（完全相同）
	if dsd.useStructHash && sig1.StructHash == sig2.StructHash {
		return 1.0, "结构哈希完全匹配"
	}
	
	// 2. 标签序列相似度
	if dsd.useTagSequence {
		tagSeqSim := dsd.calculateTagSequenceSimilarity(sig1.TagSequence, sig2.TagSequence)
		similarities = append(similarities, tagSeqSim)
		if tagSeqSim > 0.9 {
			reasons = append(reasons, fmt.Sprintf("标签序列相似度%.1f%%", tagSeqSim*100))
		}
	}
	
	// 3. SimHash相似度（汉明距离）
	if dsd.useSimHash {
		simHashSim := dsd.calculateSimHashSimilarity(sig1.SimHash, sig2.SimHash)
		similarities = append(similarities, simHashSim)
		if simHashSim > 0.9 {
			reasons = append(reasons, fmt.Sprintf("SimHash相似度%.1f%%", simHashSim*100))
		}
	}
	
	// 4. 结构特征相似度
	structFeatureSim := dsd.calculateStructFeatureSimilarity(sig1, sig2)
	similarities = append(similarities, structFeatureSim)
	if structFeatureSim > 0.9 {
		reasons = append(reasons, fmt.Sprintf("结构特征相似度%.1f%%", structFeatureSim*100))
	}
	
	// 5. 标签分布相似度（余弦相似度）
	tagDistSim := dsd.calculateTagDistributionSimilarity(sig1.TagCount, sig2.TagCount)
	similarities = append(similarities, tagDistSim)
	if tagDistSim > 0.9 {
		reasons = append(reasons, fmt.Sprintf("标签分布相似度%.1f%%", tagDistSim*100))
	}
	
	// 综合相似度（加权平均）
	totalSimilarity := 0.0
	for _, sim := range similarities {
		totalSimilarity += sim
	}
	avgSimilarity := totalSimilarity / float64(len(similarities))
	
	reason := strings.Join(reasons, ", ")
	if reason == "" {
		reason = fmt.Sprintf("综合相似度%.1f%%", avgSimilarity*100)
	}
	
	return avgSimilarity, reason
}

// calculateTagSequenceSimilarity 计算标签序列相似度（使用最长公共子序列）
func (dsd *DOMSimilarityDetector) calculateTagSequenceSimilarity(seq1, seq2 []string) float64 {
	if len(seq1) == 0 || len(seq2) == 0 {
		return 0.0
	}
	
	// 使用简化的编辑距离算法
	// 为了性能，如果序列太长，只比较前N个元素
	maxLen := 200
	if len(seq1) > maxLen {
		seq1 = seq1[:maxLen]
	}
	if len(seq2) > maxLen {
		seq2 = seq2[:maxLen]
	}
	
	lcs := dsd.longestCommonSubsequence(seq1, seq2)
	maxLen1 := len(seq1)
	maxLen2 := len(seq2)
	maxLength := maxLen1
	if maxLen2 > maxLength {
		maxLength = maxLen2
	}
	
	return float64(lcs) / float64(maxLength)
}

// longestCommonSubsequence 最长公共子序列长度
func (dsd *DOMSimilarityDetector) longestCommonSubsequence(seq1, seq2 []string) int {
	m, n := len(seq1), len(seq2)
	if m == 0 || n == 0 {
		return 0
	}
	
	// 使用动态规划
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if seq1[i-1] == seq2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}
	
	return dp[m][n]
}

// calculateSimHash 计算SimHash值
func (dsd *DOMSimilarityDetector) calculateSimHash(tagSequence []string) uint64 {
	// SimHash算法
	hashBits := 64
	v := make([]int, hashBits)
	
	for _, tag := range tagSequence {
		hash := dsd.hashString(tag)
		for i := 0; i < hashBits; i++ {
			if (hash & (1 << uint(i))) != 0 {
				v[i]++
			} else {
				v[i]--
			}
		}
	}
	
	var simhash uint64
	for i := 0; i < hashBits; i++ {
		if v[i] > 0 {
			simhash |= (1 << uint(i))
		}
	}
	
	return simhash
}

// calculateSimHashSimilarity 计算SimHash相似度（基于汉明距离）
func (dsd *DOMSimilarityDetector) calculateSimHashSimilarity(hash1, hash2 uint64) float64 {
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

// calculateTagDistributionSimilarity 计算标签分布相似度（余弦相似度）
func (dsd *DOMSimilarityDetector) calculateTagDistributionSimilarity(tags1, tags2 map[string]int) float64 {
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

// calculateStructFeatureSimilarity 计算结构特征相似度
func (dsd *DOMSimilarityDetector) calculateStructFeatureSimilarity(sig1, sig2 *DOMSignature) float64 {
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
		maxVal := math.Max(f.val1, f.val2)
		if maxVal == 0 {
			totalSimilarity += f.weight // 两者都为0，完全相同
		} else {
			diff := math.Abs(f.val1 - f.val2)
			similarity := 1.0 - (diff / maxVal)
			totalSimilarity += similarity * f.weight
		}
	}
	
	return totalSimilarity
}

// generateStructHash 生成结构哈希
func (dsd *DOMSimilarityDetector) generateStructHash(sig *DOMSignature) string {
	// 使用关键结构特征生成哈希
	features := fmt.Sprintf("depth:%d,nodes:%d,links:%d,forms:%d,inputs:%d",
		sig.Depth, sig.NodeCount, sig.LinkCount, sig.FormCount, sig.InputCount)
	
	// 添加标签分布（按字母顺序排序）
	tags := make([]string, 0, len(sig.TagCount))
	for tag := range sig.TagCount {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	
	for _, tag := range tags {
		count := sig.TagCount[tag]
		features += fmt.Sprintf(",%s:%d", tag, count)
	}
	
	return fmt.Sprintf("%x", md5.Sum([]byte(features)))
}

// generateStructFeatures 生成结构特征向量
func (dsd *DOMSimilarityDetector) generateStructFeatures(sig *DOMSignature) map[string]float64 {
	features := make(map[string]float64)
	
	// 基础特征
	features["depth"] = float64(sig.Depth)
	features["node_count"] = float64(sig.NodeCount)
	features["link_count"] = float64(sig.LinkCount)
	features["form_count"] = float64(sig.FormCount)
	features["input_count"] = float64(sig.InputCount)
	
	// 比例特征
	if sig.NodeCount > 0 {
		features["link_ratio"] = float64(sig.LinkCount) / float64(sig.NodeCount)
		features["form_ratio"] = float64(sig.FormCount) / float64(sig.NodeCount)
		features["input_ratio"] = float64(sig.InputCount) / float64(sig.NodeCount)
	}
	
	// 标签多样性
	features["tag_diversity"] = float64(len(sig.TagCount))
	
	return features
}

// calculateDOMDepth 计算DOM深度
func (dsd *DOMSimilarityDetector) calculateDOMDepth(s *goquery.Selection) int {
	maxDepth := 0
	
	var traverse func(*goquery.Selection, int)
	traverse = func(sel *goquery.Selection, depth int) {
		if depth > maxDepth {
			maxDepth = depth
		}
		
		sel.Children().Each(func(i int, child *goquery.Selection) {
			traverse(child, depth+1)
		})
	}
	
	traverse(s, 0)
	return maxDepth
}

// hashString 对字符串进行哈希
func (dsd *DOMSimilarityDetector) hashString(s string) uint64 {
	hash := uint64(5381)
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint64(c)
	}
	return hash
}

// GetSimilarPages 获取所有相似页面记录
func (dsd *DOMSimilarityDetector) GetSimilarPages() []SimilarPageRecord {
	dsd.mutex.RLock()
	defer dsd.mutex.RUnlock()
	
	records := make([]SimilarPageRecord, len(dsd.similarPages))
	copy(records, dsd.similarPages)
	return records
}

// GetStatistics 获取统计信息
func (dsd *DOMSimilarityDetector) GetStatistics() map[string]interface{} {
	dsd.mutex.RLock()
	defer dsd.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_pages_analyzed"] = len(dsd.pageSignatures)
	stats["similar_pages_found"] = len(dsd.similarPages)
	stats["similarity_threshold"] = dsd.threshold
	
	// 计算节省的爬取次数
	skippedCount := 0
	for _, record := range dsd.similarPages {
		if record.SkipCrawl {
			skippedCount++
		}
	}
	stats["crawls_skipped"] = skippedCount
	
	// 相似度分布
	if len(dsd.similarPages) > 0 {
		totalSimilarity := 0.0
		minSim := 1.0
		maxSim := 0.0
		
		for _, record := range dsd.similarPages {
			totalSimilarity += record.Similarity
			if record.Similarity < minSim {
				minSim = record.Similarity
			}
			if record.Similarity > maxSim {
				maxSim = record.Similarity
			}
		}
		
		stats["avg_similarity"] = totalSimilarity / float64(len(dsd.similarPages))
		stats["min_similarity"] = minSim
		stats["max_similarity"] = maxSim
	}
	
	return stats
}

// GenerateReport 生成相似页面报告
func (dsd *DOMSimilarityDetector) GenerateReport() string {
	dsd.mutex.RLock()
	defer dsd.mutex.RUnlock()
	
	if len(dsd.similarPages) == 0 {
		return "未发现相似页面"
	}
	
	var report strings.Builder
	
	report.WriteString(fmt.Sprintf("=== DOM相似度检测报告 ===\n\n"))
	report.WriteString(fmt.Sprintf("相似度阈值: %.1f%%\n", dsd.threshold*100))
	report.WriteString(fmt.Sprintf("分析页面总数: %d\n", len(dsd.pageSignatures)))
	report.WriteString(fmt.Sprintf("发现相似页面: %d\n", len(dsd.similarPages)))
	
	skippedCount := 0
	for _, record := range dsd.similarPages {
		if record.SkipCrawl {
			skippedCount++
		}
	}
	report.WriteString(fmt.Sprintf("跳过爬取次数: %d\n", skippedCount))
	report.WriteString(fmt.Sprintf("效率提升: %.1f%%\n\n", 
		float64(skippedCount)/float64(len(dsd.pageSignatures)+skippedCount)*100))
	
	// 列出相似页面
	report.WriteString("【相似页面列表】\n")
	for i, record := range dsd.similarPages {
		if i >= 20 { // 只显示前20个
			report.WriteString(fmt.Sprintf("... 还有 %d 个相似页面\n", len(dsd.similarPages)-20))
			break
		}
		report.WriteString(fmt.Sprintf("\n[%d] 相似度: %.1f%%\n", i+1, record.Similarity*100))
		report.WriteString(fmt.Sprintf("    页面: %s\n", record.URL))
		report.WriteString(fmt.Sprintf("    相似于: %s\n", record.SimilarToURL))
		report.WriteString(fmt.Sprintf("    原因: %s\n", record.Reason))
		if record.SkipCrawl {
			report.WriteString(fmt.Sprintf("    操作: ✓ 已跳过爬取\n"))
		}
	}
	
	return report.String()
}

// SetThreshold 设置相似度阈值
func (dsd *DOMSimilarityDetector) SetThreshold(threshold float64) {
	dsd.mutex.Lock()
	defer dsd.mutex.Unlock()
	
	if threshold > 0 && threshold <= 1 {
		dsd.threshold = threshold
	}
}

// GetThreshold 获取相似度阈值
func (dsd *DOMSimilarityDetector) GetThreshold() float64 {
	dsd.mutex.RLock()
	defer dsd.mutex.RUnlock()
	return dsd.threshold
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

