package core

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// URLQualityFilter 高质量URL过滤器
// 从爬虫专家和算法专家的角度，实现多层过滤策略
type URLQualityFilter struct {
	// 层1：完全黑名单（明确的垃圾）
	jsKeywordBlacklist    []string
	cssPropertyBlacklist  []string
	mimeTypeBlacklist     []string
	
	// 层2：模式匹配黑名单
	codePatterns          []*regexp.Regexp
	encodingPatterns      []*regexp.Regexp
	symbolPatterns        []*regexp.Regexp
	
	// 层3：内容特征检查
	maxControlChars       float64 // 控制字符比例阈值
	maxEncodingRatio      float64 // 编码字符比例阈值
	minValidChars         int     // 最少有效字符数
	
	// 统计
	stats FilterStats
}

// FilterStats 过滤统计
type FilterStats struct {
	TotalChecked        int
	PassedURLs          int
	FilteredByKeyword   int
	FilteredByPattern   int
	FilteredByEncoding  int
	FilteredByControl   int
	FilteredByLength    int
	FilteredByStructure int
}

// NewURLQualityFilter 创建高质量URL过滤器
func NewURLQualityFilter() *URLQualityFilter {
	f := &URLQualityFilter{
		maxControlChars:  0.2,  // 20%控制字符视为异常
		maxEncodingRatio: 0.4,  // 40%编码字符视为异常
		minValidChars:    2,    // 至少2个有效字符
	}
	
	// ========================================
	// 层1：JavaScript关键字黑名单
	// ========================================
	f.jsKeywordBlacklist = []string{
		// JavaScript保留字
		"function", "return", "var", "let", "const", "if", "else",
		"for", "while", "do", "switch", "case", "break", "continue",
		"try", "catch", "finally", "throw", "new", "this", "super",
		"class", "extends", "static", "async", "await", "yield",
		
		// JavaScript全局对象和方法
		"window", "document", "console", "typeof", "instanceof",
		"undefined", "null", "true", "false", "NaN", "Infinity",
		
		// HTTP方法（单独出现时）
		"get", "post", "put", "delete", "patch", "head", "options",
		
		// 其他常见垃圾
		"prototype", "constructor", "arguments", "length", "push",
		"pop", "shift", "unshift", "splice", "slice", "concat",
		"join", "split", "map", "filter", "reduce", "forEach",
	}
	
	// ========================================
	// 层1：CSS属性黑名单
	// ========================================
	f.cssPropertyBlacklist = []string{
		"margin", "padding", "border", "color", "background",
		"width", "height", "display", "position", "top", "left",
		"right", "bottom", "flex", "grid", "font", "text",
		"rgba", "rgb", "hsl", "hsla", "auto", "none", "center",
		"inherit", "initial", "unset", "relative", "absolute",
		"fixed", "sticky", "hidden", "visible", "inline", "block",
	}
	
	// ========================================
	// 层1：MIME类型黑名单
	// ========================================
	f.mimeTypeBlacklist = []string{
		"application/json", "application/xml", "text/html",
		"text/plain", "image/png", "image/jpeg", "video/mp4",
	}
	
	// ========================================
	// 层2：代码模式黑名单
	// ========================================
	f.codePatterns = []*regexp.Regexp{
		// JavaScript代码特征
		regexp.MustCompile(`\bfunction\s*\(`),
		regexp.MustCompile(`=>\s*[{(]`),
		regexp.MustCompile(`\b(var|let|const)\s+\w+\s*=`),
		regexp.MustCompile(`===|!==`),
		regexp.MustCompile(`&&|\|\|`),
		regexp.MustCompile(`console\.(log|warn|error)`),
		regexp.MustCompile(`\.(push|pop|splice|concat|join|split)\(`),
		regexp.MustCompile(`\.(forEach|map|filter|reduce)\(`),
		
		// HTML标签
		regexp.MustCompile(`</?[a-zA-Z][a-zA-Z0-9]*[^>]*>`),
		
		// 括号表达式（代码片段）
		regexp.MustCompile(`^\([^)]*\)$`),
		regexp.MustCompile(`^\{[^}]*\}$`),
		regexp.MustCompile(`^\[[^\]]*\]$`),
		
		// 正则表达式字面量
		regexp.MustCompile(`^/[^/]+/[gimuy]*$`),
	}
	
	// ========================================
	// 层2：编码模式
	// ========================================
	f.encodingPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\\x[0-9A-Fa-f]{2}`),  // \xAB
		regexp.MustCompile(`\\u[0-9A-Fa-f]{4}`),  // \u1234
		regexp.MustCompile(`&#\d+;`),             // HTML实体
		regexp.MustCompile(`&[a-z]+;`),           // &amp; &lt; etc
	}
	
	// ========================================
	// 层2：纯符号模式
	// ========================================
	f.symbolPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^[#?&=\-_./:\\|~!@$%^*()+\[\]{}]+$`), // 纯符号
		regexp.MustCompile(`^\d+$`),                               // 纯数字
		regexp.MustCompile(`^#[0-9A-Fa-f]{3,8}$`),                // 颜色值
		regexp.MustCompile(`^[a-zA-Z]$`),                         // 单字母
	}
	
	return f
}

// IsHighQualityURL 判断是否为高质量URL
func (f *URLQualityFilter) IsHighQualityURL(rawURL string) (bool, string) {
	f.stats.TotalChecked++
	
	trimmed := strings.TrimSpace(rawURL)
	
	// ========================================
	// 快速检查：基本要求
	// ========================================
	
	// 1. 长度检查
	if len(trimmed) < f.minValidChars {
		f.stats.FilteredByLength++
		return false, "URL过短"
	}
	
	if len(trimmed) > 500 {
		f.stats.FilteredByLength++
		return false, "URL过长"
	}
	
	// ========================================
	// 层1：黑名单关键字检查（精确匹配）
	// ========================================
	
	lowerURL := strings.ToLower(trimmed)
	
	// JavaScript关键字（完全匹配）
	for _, keyword := range f.jsKeywordBlacklist {
		if lowerURL == keyword {
			f.stats.FilteredByKeyword++
			return false, "JavaScript关键字: " + keyword
		}
	}
	
	// CSS属性（完全匹配或带连字符）
	for _, prop := range f.cssPropertyBlacklist {
		if lowerURL == prop || strings.HasPrefix(lowerURL, prop+"-") {
			f.stats.FilteredByKeyword++
			return false, "CSS属性: " + prop
		}
	}
	
	// MIME类型（完全匹配）
	for _, mime := range f.mimeTypeBlacklist {
		if lowerURL == mime {
			f.stats.FilteredByKeyword++
			return false, "MIME类型: " + mime
		}
	}
	
	// ========================================
	// 层2：模式匹配检查
	// ========================================
	
	// 代码模式
	for _, pattern := range f.codePatterns {
		if pattern.MatchString(rawURL) {
			f.stats.FilteredByPattern++
			return false, "代码模式匹配"
		}
	}
	
	// 纯符号模式
	for _, pattern := range f.symbolPatterns {
		if pattern.MatchString(trimmed) {
			f.stats.FilteredByPattern++
			return false, "纯符号或无意义字符"
		}
	}
	
	// ========================================
	// 层3：编码字符检查
	// ========================================
	
	encodedCount := 0
	for _, pattern := range f.encodingPatterns {
		matches := pattern.FindAllString(rawURL, -1)
		encodedCount += len(matches)
	}
	
	// URL编码检查（%XX）
	percentCount := strings.Count(rawURL, "%")
	encodedCount += percentCount
	
	// 计算编码字符比例
	if len(rawURL) > 0 {
		encodingRatio := float64(encodedCount*3) / float64(len(rawURL))
		if encodingRatio > f.maxEncodingRatio {
			f.stats.FilteredByEncoding++
			return false, "编码字符过多"
		}
	}
	
	// ========================================
	// 层4：控制字符和不可打印字符检查
	// ========================================
	
	controlCount := 0
	validCharCount := 0
	
	for _, r := range rawURL {
		if unicode.IsControl(r) {
			controlCount++
		} else if unicode.IsPrint(r) {
			validCharCount++
		}
	}
	
	totalChars := len([]rune(rawURL))
	if totalChars > 0 {
		controlRatio := float64(controlCount) / float64(totalChars)
		if controlRatio > f.maxControlChars {
			f.stats.FilteredByControl++
			return false, "控制字符过多"
		}
	}
	
	if validCharCount < f.minValidChars {
		f.stats.FilteredByControl++
		return false, "有效字符太少"
	}
	
	// ========================================
	// 层5：结构合理性检查
	// ========================================
	
	// 检查是否包含明显的JSON/代码片段特征
	if strings.Contains(rawURL, "]}") || strings.Contains(rawURL, "[{") ||
	   strings.Contains(rawURL, "})") || strings.Contains(rawURL, "({") {
		f.stats.FilteredByStructure++
		return false, "包含代码结构特征"
	}
	
	// 检查是否包含多个连续的引号
	if strings.Contains(rawURL, `""`) || strings.Contains(rawURL, "''") ||
	   strings.Contains(rawURL, "``") {
		f.stats.FilteredByStructure++
		return false, "包含连续引号"
	}
	
	// ========================================
	// 通过所有检查
	// ========================================
	
	f.stats.PassedURLs++
	return true, ""
}

// FilterURLs 批量过滤URL
func (f *URLQualityFilter) FilterURLs(urls []string) []string {
	filtered := make([]string, 0, len(urls))
	
	for _, u := range urls {
		if valid, _ := f.IsHighQualityURL(u); valid {
			filtered = append(filtered, u)
		}
	}
	
	return filtered
}

// GetStatistics 获取过滤统计
func (f *URLQualityFilter) GetStatistics() FilterStats {
	return f.stats
}

// ResetStatistics 重置统计
func (f *URLQualityFilter) ResetStatistics() {
	f.stats = FilterStats{}
}

// PrintStatistics 打印统计信息
func (f *URLQualityFilter) PrintStatistics() {
	stats := f.stats
	total := stats.TotalChecked
	
	if total == 0 {
		return
	}
	
	passRate := float64(stats.PassedURLs) / float64(total) * 100
	filterRate := float64(total-stats.PassedURLs) / float64(total) * 100
	
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║          URL质量过滤器统计（多层算法）                        ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ 总检查: %-7d  |  通过: %-7d  |  过滤: %-7d     ║\n", 
		total, stats.PassedURLs, total-stats.PassedURLs)
	fmt.Printf("║ 通过率: %.1f%%       |  过滤率: %.1f%%                        ║\n", 
		passRate, filterRate)
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Println("║ 过滤详情:                                                     ║")
	fmt.Printf("║   层1-关键字黑名单:  %-7d                                  ║\n", stats.FilteredByKeyword)
	fmt.Printf("║   层2-模式匹配:      %-7d                                  ║\n", stats.FilteredByPattern)
	fmt.Printf("║   层3-编码异常:      %-7d                                  ║\n", stats.FilteredByEncoding)
	fmt.Printf("║   层4-控制字符:      %-7d                                  ║\n", stats.FilteredByControl)
	fmt.Printf("║   层5-结构异常:      %-7d                                  ║\n", stats.FilteredByStructure)
	fmt.Printf("║   其他-长度问题:     %-7d                                  ║\n", stats.FilteredByLength)
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

