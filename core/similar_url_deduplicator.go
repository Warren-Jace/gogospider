package core

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// SimilarURLDeduplicator 相似URL去重器
// 核心功能：
// 1. 参数值不同视为相似：?t=a&tt=b vs ?tt=s&t=l
// 2. 路径变量不同视为相似：/test-1/ vs /test-2/
// 3. 使用Hash算法识别相似URL
type SimilarURLDeduplicator struct {
	mutex sync.RWMutex
	
	// Hash → 首个URL映射
	hashToFirstURL map[string]string
	
	// Hash → 相似URL列表
	hashToSimilarURLs map[string][]string
	
	// 统计
	stats SimilarURLStats
}

// SimilarURLStats 相似URL统计
type SimilarURLStats struct {
	TotalURLs      int
	UniqueHashes   int
	SimilarURLs    int
	ParamSimilar   int // 参数相似
	PathSimilar    int // 路径相似
}

// NewSimilarURLDeduplicator 创建相似URL去重器
func NewSimilarURLDeduplicator() *SimilarURLDeduplicator {
	return &SimilarURLDeduplicator{
		hashToFirstURL:    make(map[string]string),
		hashToSimilarURLs: make(map[string][]string),
		stats:             SimilarURLStats{},
	}
}

// ShouldCrawl 判断URL是否应该爬取
// 返回: (是否爬取, 相似的URL, 原因)
func (d *SimilarURLDeduplicator) ShouldCrawl(rawURL string) (bool, string, string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.stats.TotalURLs++
	
	// 计算URL的结构Hash
	structHash, similarityType := d.calculateStructureHash(rawURL)
	
	// 检查是否已有相同Hash
	if firstURL, exists := d.hashToFirstURL[structHash]; exists {
		// 相似URL，跳过
		d.stats.SimilarURLs++
		
		// 记录相似URL
		d.hashToSimilarURLs[structHash] = append(
			d.hashToSimilarURLs[structHash],
			rawURL,
		)
		
		// 更新统计
		if similarityType == "param" {
			d.stats.ParamSimilar++
		} else if similarityType == "path" {
			d.stats.PathSimilar++
		}
		
		return false, firstURL, fmt.Sprintf(
			"相似URL（%s相似），Hash=%s，首个URL: %s",
			similarityType, structHash[:8], firstURL,
		)
	}
	
	// 新的URL结构，允许爬取
	d.hashToFirstURL[structHash] = rawURL
	d.hashToSimilarURLs[structHash] = []string{}
	d.stats.UniqueHashes++
	
	return true, "", fmt.Sprintf("新URL结构，Hash=%s", structHash[:8])
}

// calculateStructureHash 计算URL结构Hash
// 返回: (Hash值, 相似类型)
func (d *SimilarURLDeduplicator) calculateStructureHash(rawURL string) (string, string) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		// 解析失败，使用原URL的Hash
		return d.simpleHash(rawURL), "unknown"
	}
	
	// 1. 提取路径结构（数字替换为占位符）
	pathStructure := d.extractPathStructure(parsedURL.Path)
	
	// 2. 提取参数结构（只保留参数名，不保留值）
	paramStructure := d.extractParamStructure(parsedURL.Query())
	
	// 3. 构造结构字符串
	structure := fmt.Sprintf(
		"%s://%s%s%s",
		parsedURL.Scheme,
		parsedURL.Host,
		pathStructure,
		paramStructure,
	)
	
	// 4. 判断相似类型
	similarityType := "normal"
	if strings.Contains(pathStructure, "{num}") || 
	   strings.Contains(pathStructure, "{id}") {
		similarityType = "path"
	} else if paramStructure != "" {
		similarityType = "param"
	}
	
	// 5. 计算Hash
	hash := d.simpleHash(structure)
	
	return hash, similarityType
}

// extractPathStructure 提取路径结构
func (d *SimilarURLDeduplicator) extractPathStructure(path string) string {
	// 正则：匹配数字序列
	// /test-123/ → /test-{num}/
	// /user/456/profile → /user/{id}/profile
	
	reNumber := regexp.MustCompile(`\d+`)
	
	// 替换连续数字为 {num}
	structure := reNumber.ReplaceAllString(path, "{num}")
	
	// 特殊处理：识别常见模式
	// /test-1/ → /test-{id}/
	// /page_2/ → /page_{id}/
	patterns := []struct {
		regex   *regexp.Regexp
		replace string
	}{
		{regexp.MustCompile(`-\{num\}`), "-{id}"},
		{regexp.MustCompile(`_\{num\}`), "_{id}"},
		{regexp.MustCompile(`/\{num\}/`), "/{id}/"},
	}
	
	for _, p := range patterns {
		structure = p.regex.ReplaceAllString(structure, p.replace)
	}
	
	return structure
}

// extractParamStructure 提取参数结构
func (d *SimilarURLDeduplicator) extractParamStructure(query url.Values) string {
	if len(query) == 0 {
		return ""
	}
	
	// 提取参数名并排序
	paramNames := make([]string, 0, len(query))
	for paramName := range query {
		paramNames = append(paramNames, paramName)
	}
	sort.Strings(paramNames)
	
	// 构造参数结构：只保留参数名
	paramParts := make([]string, 0, len(paramNames))
	for _, paramName := range paramNames {
		paramParts = append(paramParts, paramName+"=")
	}
	
	return "?" + strings.Join(paramParts, "&")
}

// simpleHash 简单Hash计算（MD5）
func (d *SimilarURLDeduplicator) simpleHash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetSimilarURLs 获取指定Hash的相似URL列表
func (d *SimilarURLDeduplicator) GetSimilarURLs(structHash string) []string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	if urls, exists := d.hashToSimilarURLs[structHash]; exists {
		return urls
	}
	return []string{}
}

// GetAllSimilarGroups 获取所有相似URL组
func (d *SimilarURLDeduplicator) GetAllSimilarGroups() map[string][]string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	result := make(map[string][]string)
	for hash, urls := range d.hashToSimilarURLs {
		if len(urls) > 0 {
			// 包含首个URL
			group := []string{d.hashToFirstURL[hash]}
			group = append(group, urls...)
			result[hash] = group
		}
	}
	return result
}

// GetStatistics 获取统计信息
func (d *SimilarURLDeduplicator) GetStatistics() SimilarURLStats {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.stats
}

// PrintReport 打印报告
func (d *SimilarURLDeduplicator) PrintReport() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	fmt.Println("\n╔═══════════════════════════════════════╗")
	fmt.Println("║      相似URL去重统计报告             ║")
	fmt.Println("╚═══════════════════════════════════════╝")
	fmt.Printf("  总URL数:         %d\n", d.stats.TotalURLs)
	fmt.Printf("  唯一Hash数:      %d\n", d.stats.UniqueHashes)
	fmt.Printf("  相似URL数:       %d\n", d.stats.SimilarURLs)
	fmt.Printf("    - 参数相似:    %d\n", d.stats.ParamSimilar)
	fmt.Printf("    - 路径相似:    %d\n", d.stats.PathSimilar)
	
	if d.stats.TotalURLs > 0 {
		fmt.Printf("  去重率:          %.1f%%\n",
			float64(d.stats.SimilarURLs)*100/float64(d.stats.TotalURLs))
	}
	
	// 显示Top 5相似组
	groups := d.GetAllSimilarGroups()
	if len(groups) > 0 {
		fmt.Println("\n【Top 5 相似URL组】")
		
		// 按组大小排序
		type groupSize struct {
			hash string
			size int
		}
		var groupSizes []groupSize
		for hash, urls := range groups {
			groupSizes = append(groupSizes, groupSize{hash, len(urls)})
		}
		sort.Slice(groupSizes, func(i, j int) bool {
			return groupSizes[i].size > groupSizes[j].size
		})
		
		// 显示前5组
		displayCount := 5
		if len(groupSizes) < displayCount {
			displayCount = len(groupSizes)
		}
		
		for i := 0; i < displayCount; i++ {
			gs := groupSizes[i]
			urls := groups[gs.hash]
			fmt.Printf("\n  [%d] Hash=%s, 共%d个URL\n", 
				i+1, gs.hash[:8], gs.size)
			fmt.Printf("      首个: %s\n", urls[0])
			if len(urls) > 1 {
				fmt.Printf("      相似: %s\n", urls[1])
			}
			if len(urls) > 2 {
				fmt.Printf("      ... (还有%d个)\n", len(urls)-2)
			}
		}
	}
	
	fmt.Println("\n─────────────────────────────────────────")
}

