package core

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// DOMEmbeddingDeduplicator DOM Embedding去重器
// 核心算法：
// 1. 遍历DOM节点
// 2. hash(节点内容) × 节点深度 × 权重
// 3. 求余展开到固定维度向量
// 4. 余弦相似度计算
type DOMEmbeddingDeduplicator struct {
	mutex sync.RWMutex
	
	// URL → Embedding映射
	urlEmbeddings map[string]*DOMEmbedding
	
	// 配置
	config EmbeddingConfig
	
	// 统计
	stats EmbeddingStats
}

// EmbeddingConfig Embedding配置
type EmbeddingConfig struct {
	Dimensions          int     // 向量维度（默认256）
	SimilarityThreshold float64 // 相似度阈值（默认0.85）
	DepthWeight         float64 // 深度权重（默认1.5）
	TagWeight           float64 // 标签权重（默认1.0）
}

// DOMEmbedding DOM向量表示
type DOMEmbedding struct {
	URL       string
	Vector    []float64 // embedding向量
	NodeCount int       // 节点数量
	Depth     int       // DOM深度
	Tags      []string  // 标签序列
}

// EmbeddingStats Embedding统计
type EmbeddingStats struct {
	TotalPages      int
	SimilarPages    int
	AvgSimilarity   float64
}

// NewDOMEmbeddingDeduplicator 创建DOM Embedding去重器
func NewDOMEmbeddingDeduplicator(dimensions int, threshold float64) *DOMEmbeddingDeduplicator {
	if dimensions <= 0 {
		dimensions = 256 // 默认256维
	}
	if threshold <= 0 || threshold > 1 {
		threshold = 0.85 // 默认85%
	}
	
	return &DOMEmbeddingDeduplicator{
		urlEmbeddings: make(map[string]*DOMEmbedding),
		config: EmbeddingConfig{
			Dimensions:          dimensions,
			SimilarityThreshold: threshold,
			DepthWeight:         1.5, // 深度权重
			TagWeight:           1.0, // 标签权重
		},
		stats: EmbeddingStats{},
	}
}

// CheckSimilarity 检查页面相似度
// 返回: (是否相似, 相似的URL, 相似度)
func (d *DOMEmbeddingDeduplicator) CheckSimilarity(rawURL string, htmlContent string) (bool, string, float64) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.stats.TotalPages++
	
	// 1. 提取DOM Embedding
	embedding, err := d.extractEmbedding(rawURL, htmlContent)
	if err != nil {
		return false, "", 0.0
	}
	
	// 2. 与已存储的页面对比
	maxSimilarity := 0.0
	var mostSimilarURL string
	
	for url, storedEmbedding := range d.urlEmbeddings {
		similarity := d.calculateCosineSimilarity(
			embedding.Vector,
			storedEmbedding.Vector,
		)
		
		if similarity > maxSimilarity {
			maxSimilarity = similarity
			mostSimilarURL = url
		}
	}
	
	// 3. 判断是否超过阈值
	if maxSimilarity >= d.config.SimilarityThreshold {
		d.stats.SimilarPages++
		d.stats.AvgSimilarity = (d.stats.AvgSimilarity*float64(d.stats.SimilarPages-1) + maxSimilarity) / float64(d.stats.SimilarPages)
		
		return true, mostSimilarURL, maxSimilarity
	}
	
	// 4. 新页面，存储embedding
	d.urlEmbeddings[rawURL] = embedding
	
	return false, "", 0.0
}

// extractEmbedding 提取DOM Embedding
func (d *DOMEmbeddingDeduplicator) extractEmbedding(rawURL string, htmlContent string) (*DOMEmbedding, error) {
	// 解析HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %v", err)
	}
	
	// 初始化embedding向量
	embedding := &DOMEmbedding{
		URL:    rawURL,
		Vector: make([]float64, d.config.Dimensions),
		Tags:   make([]string, 0),
	}
	
	// 遍历DOM树
	d.traverseDOM(doc.Selection, embedding, 0)
	
	// 归一化向量
	d.normalizeVector(embedding.Vector)
	
	return embedding, nil
}

// traverseDOM 遍历DOM树生成embedding
func (d *DOMEmbeddingDeduplicator) traverseDOM(sel *goquery.Selection, embedding *DOMEmbedding, depth int) {
	sel.Each(func(i int, s *goquery.Selection) {
		// 获取标签名
		tagName := strings.ToLower(goquery.NodeName(s))
		
		// 更新统计
		embedding.NodeCount++
		embedding.Tags = append(embedding.Tags, tagName)
		if depth > embedding.Depth {
			embedding.Depth = depth
		}
		
		// 计算节点内容的hash
		nodeContent := d.getNodeContent(s, tagName)
		hash := d.hashString(nodeContent)
		
		// 🔥 核心算法：hash × 深度 × 权重
		depthWeight := math.Pow(d.config.DepthWeight, float64(depth))
		tagWeight := d.getTagWeight(tagName)
		weightedHash := float64(hash) * depthWeight * tagWeight
		
		// 求余展开到向量维度
		index := int(weightedHash) % d.config.Dimensions
		if index < 0 {
			index = -index
		}
		
		// 累加到向量
		embedding.Vector[index] += 1.0
		
		// 递归处理子节点
		s.Children().Each(func(j int, child *goquery.Selection) {
			d.traverseDOM(child, embedding, depth+1)
		})
	})
}

// getNodeContent 获取节点内容用于hash
func (d *DOMEmbeddingDeduplicator) getNodeContent(s *goquery.Selection, tagName string) string {
	// 组合标签名和关键属性
	var parts []string
	parts = append(parts, tagName)
	
	// 添加重要属性
	importantAttrs := []string{"id", "class", "name", "type", "href", "src"}
	for _, attr := range importantAttrs {
		if val, exists := s.Attr(attr); exists && val != "" {
			parts = append(parts, attr+"="+val)
		}
	}
	
	// 添加部分文本内容（前50字符）
	text := strings.TrimSpace(s.Text())
	if len(text) > 50 {
		text = text[:50]
	}
	if text != "" {
		parts = append(parts, "text="+text)
	}
	
	return strings.Join(parts, "|")
}

// getTagWeight 获取标签权重
func (d *DOMEmbeddingDeduplicator) getTagWeight(tagName string) float64 {
	// 重要标签赋予更高权重
	weights := map[string]float64{
		"title":  2.0,
		"h1":     1.8,
		"h2":     1.6,
		"h3":     1.4,
		"form":   1.5,
		"input":  1.3,
		"button": 1.3,
		"a":      1.2,
		"div":    1.0,
		"span":   1.0,
		"p":      1.0,
	}
	
	if weight, exists := weights[tagName]; exists {
		return weight * d.config.TagWeight
	}
	
	return d.config.TagWeight
}

// hashString 字符串Hash（FNV-1a算法）
func (d *DOMEmbeddingDeduplicator) hashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// normalizeVector 归一化向量（L2范数）
func (d *DOMEmbeddingDeduplicator) normalizeVector(vector []float64) {
	// 计算L2范数
	var sumSquares float64
	for _, val := range vector {
		sumSquares += val * val
	}
	norm := math.Sqrt(sumSquares)
	
	if norm == 0 {
		return
	}
	
	// 归一化
	for i := range vector {
		vector[i] /= norm
	}
}

// calculateCosineSimilarity 计算余弦相似度
func (d *DOMEmbeddingDeduplicator) calculateCosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 0.0
	}
	
	// 计算点积
	var dotProduct float64
	for i := range vec1 {
		dotProduct += vec1[i] * vec2[i]
	}
	
	// 由于向量已归一化，点积即为余弦相似度
	return dotProduct
}

// GetStatistics 获取统计信息
func (d *DOMEmbeddingDeduplicator) GetStatistics() EmbeddingStats {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.stats
}

// PrintReport 打印报告
func (d *DOMEmbeddingDeduplicator) PrintReport() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	fmt.Println("\n╔═══════════════════════════════════════╗")
	fmt.Println("║    DOM Embedding去重统计报告         ║")
	fmt.Println("╚═══════════════════════════════════════╝")
	fmt.Printf("  总页面数:        %d\n", d.stats.TotalPages)
	fmt.Printf("  相似页面数:      %d\n", d.stats.SimilarPages)
	
	if d.stats.TotalPages > 0 {
		fmt.Printf("  去重率:          %.1f%%\n",
			float64(d.stats.SimilarPages)*100/float64(d.stats.TotalPages))
	}
	
	if d.stats.SimilarPages > 0 {
		fmt.Printf("  平均相似度:      %.1f%%\n", d.stats.AvgSimilarity*100)
	}
	
	fmt.Printf("  向量维度:        %d\n", d.config.Dimensions)
	fmt.Printf("  相似度阈值:      %.1f%%\n", d.config.SimilarityThreshold*100)
	fmt.Println("─────────────────────────────────────────")
}

// GetEmbedding 获取指定URL的embedding（用于调试）
func (d *DOMEmbeddingDeduplicator) GetEmbedding(url string) *DOMEmbedding {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	if emb, exists := d.urlEmbeddings[url]; exists {
		return emb
	}
	return nil
}

