package core

import (
	"net/url"
	"regexp"
	"strings"
)

// URLExtractorFix 修复版URL提取器 - 解决垃圾数据问题
// 问题根源：过于宽松的正则表达式匹配了大量非URL的字符串
// 修复策略：多层验证 + 严格的上下文匹配
type URLExtractorFix struct {
	// 编译后的正则表达式（避免重复编译）
	completeURLPattern  *regexp.Regexp
	apiPathPattern      *regexp.Regexp
	filePathPattern     *regexp.Regexp
	
	// 黑名单模式
	jsCodePattern       *regexp.Regexp
	htmlTagPattern      *regexp.Regexp
	cssPropertyPattern  *regexp.Regexp
	symbolOnlyPattern   *regexp.Regexp
	
	// 统计
	totalExtracted      int
	filteredGarbage     int
	validURLs          int
}

// NewURLExtractorFix 创建修复版URL提取器
func NewURLExtractorFix() *URLExtractorFix {
	e := &URLExtractorFix{}
	
	// 编译正则表达式
	e.completeURLPattern = regexp.MustCompile(`(https?://[a-zA-Z0-9\-._~:/?#\[\]@!$&'()*+,;=%]+)`)
	e.apiPathPattern = regexp.MustCompile(`^/(api|v\d+|admin|user|login|logout|auth|AJAX)/[a-zA-Z0-9_\-/]+`)
	e.filePathPattern = regexp.MustCompile(`^/[a-zA-Z0-9_\-/]+\.(php|jsp|asp|aspx|do|action|html|htm)`)
	
	// 黑名单模式
	e.jsCodePattern = regexp.MustCompile(`(function\s*\(|=>\s*{|\bvar\s+|\blet\s+|\bconst\s+|===|!==|console\.|\.concat\(|\.split\(|\.join\(|\.forEach\(|\.map\()`)
	e.htmlTagPattern = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	e.cssPropertyPattern = regexp.MustCompile(`^(margin|padding|border|color|width|height|font|display|position|flex|grid|rgba|rgb|hsl)`)
	e.symbolOnlyPattern = regexp.MustCompile(`^[#?&=\-_./:+*%!@$^|~\\<>{}[\]()]+$`)
	
	return e
}

// ExtractFromJSCode 从JavaScript代码中提取URL（严格模式）
func (e *URLExtractorFix) ExtractFromJSCode(jsCode string) []string {
	urls := make([]string, 0)
	seen := make(map[string]bool)
	
	// =======================================================
	// 策略1：提取完整HTTP/HTTPS URL（最可靠）
	// =======================================================
	completeMatches := e.completeURLPattern.FindAllString(jsCode, -1)
	for _, url := range completeMatches {
		url = strings.Trim(url, `"'` + "`")
		if e.isValidURL(url) && !seen[url] {
			seen[url] = true
			urls = append(urls, url)
		}
	}
	
	// =======================================================
	// 策略2：从明确的上下文中提取相对路径
	// =======================================================
	contextPatterns := []struct {
		pattern string
		desc    string
	}{
		// AJAX请求
		{`fetch\s*\(\s*['"]([^'"]+)['"]`, "Fetch API"},
		{`\$\.ajax\s*\(\s*\{[^}]*url\s*:\s*['"]([^'"]+)['"]`, "jQuery.ajax"},
		{`\$\.(get|post)\s*\(\s*['"]([^'"]+)['"]`, "jQuery.get/post"},
		{`axios\.(get|post|put|delete|patch)\s*\(\s*['"]([^'"]+)['"]`, "Axios"},
		{`xhr\.open\s*\(\s*['"](?:GET|POST)['"],\s*['"]([^'"]+)['"]`, "XMLHttpRequest"},
		
		// 导航
		{`window\.location\s*=\s*['"]([^'"]+)['"]`, "window.location"},
		{`location\.href\s*=\s*['"]([^'"]+)['"]`, "location.href"},
		{`window\.open\s*\(\s*['"]([^'"]+)['"]`, "window.open"},
		{`navigate\s*\(\s*['"]([^'"]+)['"]`, "navigate()"},
		{`redirect\s*\(\s*['"]([^'"]+)['"]`, "redirect()"},
		
		// API配置（严格匹配）
		{`apiUrl\s*[:=]\s*['"]([^'"]+)['"]`, "apiUrl配置"},
		{`baseURL\s*[:=]\s*['"]([^'"]+)['"]`, "baseURL配置"},
		{`endpoint\s*[:=]\s*['"]([^'"]+)['"]`, "endpoint配置"},
		
		// API端点（带引号）
		{`['"]/(api/[a-zA-Z0-9_\-/]+)['"]`, "API路径"},
		{`['"]/(v\d+/[a-zA-Z0-9_\-/]+)['"]`, "版本化API"},
		{`['"]/(admin/[a-zA-Z0-9_\-/]+)['"]`, "管理路径"},
	}
	
	for _, cp := range contextPatterns {
		re := regexp.MustCompile(cp.pattern)
		matches := re.FindAllStringSubmatch(jsCode, -1)
		
		for _, match := range matches {
			if len(match) >= 2 {
				url := match[len(match)-1]
				if e.isValidURL(url) && !seen[url] {
					seen[url] = true
					urls = append(urls, url)
				}
			}
		}
	}
	
	e.totalExtracted = len(seen)
	e.validURLs = len(urls)
	
	return urls
}

// isValidURL 严格验证URL是否有效
func (e *URLExtractorFix) isValidURL(rawURL string) bool {
	e.totalExtracted++
	
	// ========================================
	// 第1层：基本格式检查
	// ========================================
	
	trimmed := strings.TrimSpace(rawURL)
	
	// 1.1 空或太短/太长
	if len(trimmed) < 2 || len(trimmed) > 500 {
		e.filteredGarbage++
		return false
	}
	
	// 1.2 纯符号
	if e.symbolOnlyPattern.MatchString(trimmed) {
		e.filteredGarbage++
		return false
	}
	
	// ========================================
	// 第2层：黑名单过滤
	// ========================================
	
	// 2.1 JavaScript代码特征
	if e.jsCodePattern.MatchString(trimmed) {
		e.filteredGarbage++
		return false
	}
	
	// 2.2 HTML标签
	if e.htmlTagPattern.MatchString(trimmed) {
		e.filteredGarbage++
		return false
	}
	
	// 2.3 CSS属性
	if e.cssPropertyPattern.MatchString(strings.ToLower(trimmed)) {
		e.filteredGarbage++
		return false
	}
	
	// 2.4 JavaScript运算符
	if strings.Contains(trimmed, "===") || 
	   strings.Contains(trimmed, "!==") ||
	   strings.Contains(trimmed, "&&") || 
	   strings.Contains(trimmed, "||") {
		e.filteredGarbage++
		return false
	}
	
	// 2.5 Unicode/Hex编码字符串（如 \u0100, \xAB）
	if strings.Contains(trimmed, "\\u") || strings.Contains(trimmed, "\\x") {
		e.filteredGarbage++
		return false
	}
	
	// 2.6 HTML实体
	if strings.Contains(trimmed, "&amp;") || 
	   strings.Contains(trimmed, "&lt;") ||
	   strings.Contains(trimmed, "&gt;") ||
	   strings.Contains(trimmed, "&#") {
		e.filteredGarbage++
		return false
	}
	
	// ========================================
	// 第3层：URL格式验证
	// ========================================
	
	// 3.1 完整URL检查
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			e.filteredGarbage++
			return false
		}
		
		// 必须有有效的域名
		if parsed.Host == "" {
			e.filteredGarbage++
			return false
		}
		
		// 域名必须包含至少一个点（x.com）或者是localhost
		if !strings.Contains(parsed.Host, ".") && parsed.Host != "localhost" {
			e.filteredGarbage++
			return false
		}
		
		return true
	}
	
	// 3.2 协议相对URL（//example.com/path）
	if strings.HasPrefix(trimmed, "//") {
		// 移除//后解析
		testURL := "http:" + trimmed
		parsed, err := url.Parse(testURL)
		if err != nil || parsed.Host == "" {
			e.filteredGarbage++
			return false
		}
		return true
	}
	
	// 3.3 相对路径URL（/path/to/resource）
	if strings.HasPrefix(trimmed, "/") {
		// 相对路径必须满足以下条件之一：
		// a) 是API路径（/api/, /v1/, /admin/等）
		if e.apiPathPattern.MatchString(trimmed) {
			return true
		}
		
		// b) 是文件路径（*.php, *.jsp等）
		if e.filePathPattern.MatchString(trimmed) {
			return true
		}
		
		// c) 至少有2层路径（/a/b）
		parts := strings.Split(strings.TrimPrefix(trimmed, "/"), "/")
		if len(parts) >= 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
			return true
		}
		
		// d) 拒绝其他单层相对路径
		e.filteredGarbage++
		return false
	}
	
	// ========================================
	// 第4层：其他无效情况
	// ========================================
	
	// 不符合任何URL格式
	e.filteredGarbage++
	return false
}

// GetStatistics 获取统计信息
func (e *URLExtractorFix) GetStatistics() map[string]int {
	return map[string]int{
		"total_extracted":  e.totalExtracted,
		"filtered_garbage": e.filteredGarbage,
		"valid_urls":      e.validURLs,
	}
}

