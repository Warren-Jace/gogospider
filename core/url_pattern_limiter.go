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

// URLPatternLimiter URL模式限流器
// 限制相同URL模式的爬取数量，避免资源浪费
type URLPatternLimiter struct {
	mutex sync.RWMutex
	
	// URL模式到爬取次数的映射
	patternCounts map[string]int
	
	// URL模式的详细信息
	patternInfo map[string]*PatternLimitInfo
	
	// 配置
	config PatternLimiterConfig
	
	// 统计
	stats LimiterStats
}

// PatternLimitInfo 模式限流信息
type PatternLimitInfo struct {
	Pattern      string   // URL模式（例如：/product.php?id=）
	CrawledURLs  []string // 已爬取的URL列表（采样）
	SkippedCount int      // 跳过的URL数量
	FirstURL     string   // 第一个URL
	Hash         string   // 模式hash
}

// PatternLimiterConfig 模式限流器配置
type PatternLimiterConfig struct {
	// 相同模式URL的最大爬取数量（0表示不限制）
	MaxURLsPerPattern int
	
	// 权重策略：不同类型的URL有不同的限制
	WeightedLimits map[string]int
	
	// 是否启用智能模式（根据URL重要性动态调整）
	EnableSmartMode bool
	
	// 采样数量（保留前N个URL作为样本）
	SampleSize int
}

// LimiterStats 限流器统计
type LimiterStats struct {
	TotalURLs       int // 总检查URL数
	UniquePatterns  int // 唯一模式数
	LimitedURLs     int // 被限流的URL数
	AllowedURLs     int // 允许爬取的URL数
}

// NewURLPatternLimiter 创建URL模式限流器
func NewURLPatternLimiter(config PatternLimiterConfig) *URLPatternLimiter {
	// 设置默认值
	if config.MaxURLsPerPattern == 0 {
		config.MaxURLsPerPattern = 3 // 默认每个模式最多爬3个
	}
	if config.SampleSize == 0 {
		config.SampleSize = 5 // 默认保留5个样本
	}
	
	// 默认权重策略
	if config.WeightedLimits == nil {
		config.WeightedLimits = map[string]int{
			"api":      5,  // API端点可以多爬几个
			"form":     5,  // 表单相关
			"image":    2,  // 图片只爬2个
			"static":   1,  // 静态资源只爬1个
			"normal":   3,  // 普通页面3个
		}
	}
	
	return &URLPatternLimiter{
		patternCounts: make(map[string]int),
		patternInfo:   make(map[string]*PatternLimitInfo),
		config:        config,
		stats:         LimiterStats{},
	}
}

// ShouldCrawl 判断URL是否应该爬取
// 返回：(是否允许爬取, 原因, 模式信息)
func (l *URLPatternLimiter) ShouldCrawl(rawURL string) (bool, string, *PatternLimitInfo) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	l.stats.TotalURLs++
	
	// 提取URL模式
	pattern, urlType := l.extractURLPattern(rawURL)
	if pattern == "" {
		// 无法提取模式，允许爬取
		l.stats.AllowedURLs++
		return true, "无法提取URL模式，允许爬取", nil
	}
	
	// 计算模式hash
	patternHash := l.calculateHash(pattern)
	
	// 获取或创建模式信息
	info, exists := l.patternInfo[patternHash]
	if !exists {
		info = &PatternLimitInfo{
			Pattern:      pattern,
			CrawledURLs:  make([]string, 0, l.config.SampleSize),
			SkippedCount: 0,
			FirstURL:     rawURL,
			Hash:         patternHash,
		}
		l.patternInfo[patternHash] = info
		l.stats.UniquePatterns++
	}
	
	// 获取当前计数
	currentCount := l.patternCounts[patternHash]
	
	// 确定限制数量（根据URL类型）
	limit := l.getLimit(urlType)
	
	// 判断是否超过限制
	if currentCount >= limit {
		// 超过限制，拒绝爬取
		info.SkippedCount++
		l.stats.LimitedURLs++
		
		reason := fmt.Sprintf("URL模式限流 - 已爬取%d个相同模式URL（限制:%d），模式: %s", 
			currentCount, limit, pattern)
		
		return false, reason, info
	}
	
	// 允许爬取
	l.patternCounts[patternHash]++
	
	// 添加到采样列表（只保留前N个）
	if len(info.CrawledURLs) < l.config.SampleSize {
		info.CrawledURLs = append(info.CrawledURLs, rawURL)
	}
	
	l.stats.AllowedURLs++
	
	reason := fmt.Sprintf("允许爬取 - 第%d个该模式URL（限制:%d）", currentCount+1, limit)
	return true, reason, info
}

// extractURLPattern 提取URL模式（去除参数值）
// 返回：(URL模式, URL类型)
func (l *URLPatternLimiter) extractURLPattern(rawURL string) (string, string) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", "unknown"
	}
	
	// 构造基础模式：scheme + host + path
	pattern := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	
	// 如果有查询参数，提取参数名（不含值）
	if parsedURL.RawQuery != "" {
		queryParams := parsedURL.Query()
		
		// 对参数名排序
		var paramNames []string
		for name := range queryParams {
			paramNames = append(paramNames, name)
		}
		sort.Strings(paramNames)
		
		// 构建参数模式（只保留参数名）
		if len(paramNames) > 0 {
			pattern += "?" + strings.Join(paramNames, "&") + "="
		}
	}
	
	// 判断URL类型
	urlType := l.classifyURLType(rawURL, parsedURL)
	
	return pattern, urlType
}

// classifyURLType 分类URL类型
func (l *URLPatternLimiter) classifyURLType(rawURL string, parsedURL *url.URL) string {
	lowerPath := strings.ToLower(parsedURL.Path)
	
	// 静态资源
	staticExts := []string{
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico", ".webp", ".bmp",
		".css", ".scss", ".sass", ".less",
		".js", ".ts", ".jsx", ".tsx",
		".woff", ".woff2", ".ttf", ".eot", ".otf",
		".mp4", ".mp3", ".avi", ".mov", ".wmv", ".flv", ".webm",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".zip", ".rar", ".tar", ".gz", ".7z",
	}
	
	for _, ext := range staticExts {
		if strings.HasSuffix(lowerPath, ext) {
			if strings.Contains(ext, "jpg") || strings.Contains(ext, "png") || 
			   strings.Contains(ext, "gif") || strings.Contains(ext, "svg") {
				return "image"
			}
			return "static"
		}
	}
	
	// API端点
	if strings.Contains(lowerPath, "api") || strings.Contains(lowerPath, "ajax") ||
	   strings.Contains(lowerPath, "/v1/") || strings.Contains(lowerPath, "/v2/") {
		return "api"
	}
	
	// 表单相关
	if strings.Contains(lowerPath, "login") || strings.Contains(lowerPath, "register") ||
	   strings.Contains(lowerPath, "signup") || strings.Contains(lowerPath, "form") ||
	   strings.Contains(lowerPath, "submit") {
		return "form"
	}
	
	return "normal"
}

// getLimit 获取限制数量（根据URL类型）
func (l *URLPatternLimiter) getLimit(urlType string) int {
	if limit, exists := l.config.WeightedLimits[urlType]; exists {
		return limit
	}
	return l.config.MaxURLsPerPattern
}

// calculateHash 计算字符串的hash
func (l *URLPatternLimiter) calculateHash(s string) string {
	hasher := md5.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetStats 获取统计信息
func (l *URLPatternLimiter) GetStats() LimiterStats {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.stats
}

// PrintReport 打印限流报告
func (l *URLPatternLimiter) PrintReport() {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 URL模式限流报告")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("🔍 总检查URL数: %d\n", l.stats.TotalURLs)
	fmt.Printf("📦 唯一模式数: %d\n", l.stats.UniquePatterns)
	fmt.Printf("✅ 允许爬取: %d\n", l.stats.AllowedURLs)
	fmt.Printf("🚫 限流拒绝: %d\n", l.stats.LimitedURLs)
	
	if l.stats.TotalURLs > 0 {
		limitRate := float64(l.stats.LimitedURLs) / float64(l.stats.TotalURLs) * 100
		fmt.Printf("📈 限流率: %.1f%%\n", limitRate)
	}
	
	fmt.Println("\n【模式详情（按跳过数量排序）】")
	
	// 收集所有模式信息
	type patternStat struct {
		Pattern      string
		CrawledCount int
		SkippedCount int
		FirstURL     string
		Samples      []string
	}
	
	var patterns []patternStat
	for hash, info := range l.patternInfo {
		count := l.patternCounts[hash]
		patterns = append(patterns, patternStat{
			Pattern:      info.Pattern,
			CrawledCount: count,
			SkippedCount: info.SkippedCount,
			FirstURL:     info.FirstURL,
			Samples:      info.CrawledURLs,
		})
	}
	
	// 按跳过数量排序
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].SkippedCount > patterns[j].SkippedCount
	})
	
	// 只显示前10个
	displayCount := 10
	if len(patterns) < displayCount {
		displayCount = len(patterns)
	}
	
	for i := 0; i < displayCount; i++ {
		p := patterns[i]
		fmt.Printf("\n%d. 模式: %s\n", i+1, p.Pattern)
		fmt.Printf("   爬取: %d个 | 跳过: %d个\n", p.CrawledCount, p.SkippedCount)
		fmt.Printf("   首个URL: %s\n", p.FirstURL)
		
		if len(p.Samples) > 0 {
			fmt.Printf("   采样:\n")
			for j, sample := range p.Samples {
				if j < 3 { // 只显示前3个
					fmt.Printf("     • %s\n", sample)
				}
			}
		}
	}
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

