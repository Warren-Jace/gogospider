package core

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// SmartURLValidator 智能URL验证器 v2.0 - 基于黑名单机制
// 核心理念：只过滤明确的垃圾URL，宁可多爬不要漏爬
type SmartURLValidator struct {
	// 编译后的正则表达式
	htmlTagPattern        *regexp.Regexp
	jsCodePattern         *regexp.Regexp
	urlEncodingPattern    *regexp.Regexp
	pureSymbolPattern     *regexp.Regexp
	invalidSchemePattern  *regexp.Regexp
	
	// 配置项
	maxURLLength          int     // 最大URL长度
	encodingThreshold     float64 // URL编码字符阈值（超过此比例认为异常）
	minPathLength         int     // 最小路径长度（太短可能是无意义的）
	
	// 统计信息
	filteredByJSCode      int
	filteredByHTMLTag     int
	filteredBySymbol      int
	filteredByEncoding    int
	filteredByScheme      int
	filteredByLength      int
	filteredByInvalid     int
	totalChecked          int
	totalPassed           int
}

// NewSmartURLValidator 创建智能URL验证器
func NewSmartURLValidator() *SmartURLValidator {
	v := &SmartURLValidator{
		maxURLLength:      500,  // 默认最大500字符
		encodingThreshold: 0.4,  // 超过40%是编码字符认为异常
		minPathLength:     0,    // 不限制最小长度（保留所有可能有效的）
	}
	
	// 编译正则表达式
	
	// 1. HTML标签匹配
	v.htmlTagPattern = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	
	// 2. JavaScript代码特征（函数、运算符、语句）
	v.jsCodePattern = regexp.MustCompile(`(?i)(\bfunction\s*\(|=>\s*{|\bvar\s+\w+\s*=|\blet\s+\w+\s*=|\bconst\s+\w+\s*=|===|!==|\)\s*{|/\*|\*/|console\.log|window\.|document\.|return\s+\w+)`)
	
	// 3. URL编码字符
	v.urlEncodingPattern = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)
	
	// 4. 纯符号URL（单个或少量符号）
	v.pureSymbolPattern = regexp.MustCompile(`^[#?&=\-_./:\\]*$`)
	
	// 5. 无效的URL scheme
	v.invalidSchemePattern = regexp.MustCompile(`^(javascript|data|blob|about|vbscript|file):`)
	
	return v
}

// IsValidBusinessURL 判断是否为有效的业务URL
// 返回：是否有效, 过滤原因
func (v *SmartURLValidator) IsValidBusinessURL(rawURL string) (bool, string) {
	v.totalChecked++
	
	// ========================================
	// 阶段1: 基本格式检查
	// ========================================
	
	// 1.1 空URL或纯空格
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		v.filteredByInvalid++
		return false, "空URL"
	}
	
	// ========================================
	// 🔧 v3.6.3: 黑名单快速检查（提前拦截垃圾数据）
	// ========================================
	
	// 1.1a JavaScript关键字黑名单
	jsKeywords := []string{
		"get", "set", "post", "put", "delete", "patch",
		"function", "return", "var", "let", "const",
		"true", "false", "null", "undefined",
		"typeof", "instanceof", "arguments",
		"this", "super", "new", "class",
	}
	lowerURL := strings.ToLower(trimmed)
	for _, keyword := range jsKeywords {
		if lowerURL == keyword {
			v.filteredByJSCode++
			return false, "JavaScript关键字"
		}
	}
	
	// 1.1b CSS属性黑名单
	cssProperties := []string{
		"margin", "padding", "border", "color",
		"width", "height", "display", "position",
		"rgba", "rgb", "hsl", "flex", "grid",
		"font", "background", "text", "align",
		"auto", "none", "center", "left", "right",
	}
	for _, prop := range cssProperties {
		if lowerURL == prop || strings.HasPrefix(lowerURL, prop+"-") {
			v.filteredByHTMLTag++
			return false, "CSS属性"
		}
	}
	
	// 1.1c 单字符
	if len(trimmed) == 1 {
		v.filteredBySymbol++
		return false, "单字符"
	}
	
	// 1.1d 纯数字
	if matched, _ := regexp.MatchString(`^\d+$`, trimmed); matched {
		v.filteredBySymbol++
		return false, "纯数字"
	}
	
	// 1.1e 颜色值（十六进制）
	if matched, _ := regexp.MatchString(`^#[0-9A-Fa-f]{3,8}$`, trimmed); matched {
		v.filteredBySymbol++
		return false, "颜色值"
	}
	
	// ========================================
	// 黑名单检查结束
	// ========================================
	
	// 1.2 长度检查（防止恶意超长URL）
	if len(rawURL) > v.maxURLLength {
		v.filteredByLength++
		return false, "URL过长"
	}
	
	// 1.3 纯符号URL
	if v.pureSymbolPattern.MatchString(trimmed) {
		v.filteredBySymbol++
		return false, "纯符号URL"
	}
	
	// 1.4 无效的URL scheme
	if v.invalidSchemePattern.MatchString(strings.ToLower(trimmed)) {
		v.filteredByScheme++
		return false, "无效的URL协议"
	}
	
	// ========================================
	// 阶段2: 解析URL
	// ========================================
	
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		v.filteredByInvalid++
		return false, "URL解析失败"
	}
	
	path := parsedURL.Path
	if path == "" && parsedURL.RawQuery == "" && parsedURL.Fragment == "" {
		// 只有域名，没有路径、参数和锚点，可能是根URL
		if parsedURL.Host != "" {
			v.totalPassed++
			return true, ""
		}
		v.filteredByInvalid++
		return false, "无路径信息"
	}
	
	// ========================================
	// 阶段3: 过滤JavaScript代码
	// ========================================
	
	// 3.1 检查是否包含JavaScript代码特征
	if v.jsCodePattern.MatchString(rawURL) {
		v.filteredByJSCode++
		return false, "包含JavaScript代码"
	}
	
	// 3.2 检查是否有多个连续的JavaScript运算符
	if strings.Contains(rawURL, "===") || strings.Contains(rawURL, "!==") || 
	   strings.Contains(rawURL, "&&") || strings.Contains(rawURL, "||") {
		v.filteredByJSCode++
		return false, "包含JavaScript运算符"
	}
	
	// ========================================
	// 阶段4: 过滤HTML标签
	// ========================================
	
	if v.htmlTagPattern.MatchString(rawURL) {
		v.filteredByHTMLTag++
		return false, "包含HTML标签"
	}
	
	// ========================================
	// 阶段5: 检查URL编码异常
	// ========================================
	
	// 统计URL编码字符的比例
	encodedMatches := v.urlEncodingPattern.FindAllString(rawURL, -1)
	encodedCount := len(encodedMatches)
	totalChars := len(rawURL)
	
	if totalChars > 0 {
		encodingRatio := float64(encodedCount*3) / float64(totalChars) // 每个%XX占3个字符
		if encodingRatio > v.encodingThreshold {
			v.filteredByEncoding++
			return false, "URL编码字符过多"
		}
	}
	
	// ========================================
	// 阶段6: 特殊字符检查（宽松）
	// ========================================
	
	// 6.1 检查是否包含明显的代码注释
	if strings.Contains(rawURL, "//") && !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		// 排除正常的http://和https://
		afterScheme := rawURL
		if idx := strings.Index(rawURL, "://"); idx != -1 {
			afterScheme = rawURL[idx+3:]
		}
		if strings.Contains(afterScheme, "//") {
			v.filteredByJSCode++
			return false, "包含注释符号"
		}
	}
	
	// 6.2 检查是否包含多个连续的特殊字符（可能是代码片段）
	specialChars := []string{"{{", "}}", "[[", "]]", "<%", "%>", "<?", "?>"}
	for _, sc := range specialChars {
		if strings.Contains(rawURL, sc) {
			v.filteredBySymbol++
			return false, "包含模板语法或特殊符号"
		}
	}
	
	// ========================================
	// 阶段7: 路径合理性检查（非常宽松）
	// ========================================
	
	// 7.1 检查路径是否全是不可打印字符
	if path != "" {
		hasValidChar := false
		for _, r := range path {
			if unicode.IsPrint(r) && r != '/' {
				hasValidChar = true
				break
			}
		}
		if !hasValidChar {
			v.filteredByInvalid++
			return false, "路径无有效字符"
		}
	}
	
	// 7.2 检查是否为明显的MIME类型（但要精确判断，不是包含）
	// 只过滤路径本身就是MIME类型的情况，如 "/application/json"
	if path != "" {
		cleanPath := strings.Trim(path, "/")
		segments := strings.Split(cleanPath, "/")
		
		// 只有当第一段完全是MIME类型前缀时才过滤
		if len(segments) > 0 {
			firstSeg := segments[0]
			pureMimeTypes := []string{
				"application", "text", "image", "video", "audio", "font", "multipart",
			}
			isPureMime := false
			for _, mime := range pureMimeTypes {
				if firstSeg == mime && len(segments) > 1 {
					// 第二段也是MIME类型子类的情况，如 /application/json
					secondSeg := segments[1]
					if isMIMESubtype(secondSeg) {
						isPureMime = true
						break
					}
				}
			}
			if isPureMime {
				v.filteredByInvalid++
				return false, "路径为MIME类型"
			}
		}
	}
	
	// ========================================
	// 通过所有检查，认为是有效URL
	// ========================================
	
	v.totalPassed++
	return true, ""
}

// isMIMESubtype 判断是否为MIME子类型
func isMIMESubtype(segment string) bool {
	mimeSubtypes := map[string]bool{
		"json":                  true,
		"xml":                   true,
		"html":                  true,
		"plain":                 true,
		"javascript":            true,
		"css":                   true,
		"pdf":                   true,
		"octet-stream":          true,
		"x-www-form-urlencoded": true,
		"jpeg":                  true,
		"png":                   true,
		"gif":                   true,
		"svg":                   true,
		"mpeg":                  true,
		"mp4":                   true,
	}
	return mimeSubtypes[segment]
}

// FilterURLs 批量过滤URL列表
func (v *SmartURLValidator) FilterURLs(urls []string) []string {
	filtered := make([]string, 0, len(urls))
	
	for _, u := range urls {
		if valid, _ := v.IsValidBusinessURL(u); valid {
			filtered = append(filtered, u)
		}
	}
	
	return filtered
}

// GetStatistics 获取过滤统计信息
func (v *SmartURLValidator) GetStatistics() map[string]int {
	return map[string]int{
		"total_checked":          v.totalChecked,
		"total_passed":           v.totalPassed,
		"filtered_by_js_code":    v.filteredByJSCode,
		"filtered_by_html_tag":   v.filteredByHTMLTag,
		"filtered_by_symbol":     v.filteredBySymbol,
		"filtered_by_encoding":   v.filteredByEncoding,
		"filtered_by_scheme":     v.filteredByScheme,
		"filtered_by_length":     v.filteredByLength,
		"filtered_by_invalid":    v.filteredByInvalid,
	}
}

// PrintStatistics 打印统计信息
func (v *SmartURLValidator) PrintStatistics() {
	stats := v.GetStatistics()
	total := stats["total_checked"]
	passed := stats["total_passed"]
	filtered := total - passed
	
	if total == 0 {
		return
	}
	
	passRate := float64(passed) / float64(total) * 100
	
	fmt.Println("\n╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              智能URL过滤器统计 (v2.0 黑名单机制)            ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ 总检查数: %-6d  |  通过: %-6d  |  过滤: %-6d      ║\n", total, passed, filtered)
	fmt.Printf("║ 通过率: %.1f%%                                                  ║\n", passRate)
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("║ 过滤原因分布:                                                ║")
	fmt.Printf("║   · JavaScript代码:  %-6d                                  ║\n", stats["filtered_by_js_code"])
	fmt.Printf("║   · HTML标签:        %-6d                                  ║\n", stats["filtered_by_html_tag"])
	fmt.Printf("║   · 纯符号/特殊符号: %-6d                                  ║\n", stats["filtered_by_symbol"])
	fmt.Printf("║   · URL编码异常:     %-6d                                  ║\n", stats["filtered_by_encoding"])
	fmt.Printf("║   · 无效协议:        %-6d                                  ║\n", stats["filtered_by_scheme"])
	fmt.Printf("║   · URL过长:         %-6d                                  ║\n", stats["filtered_by_length"])
	fmt.Printf("║   · 其他无效:        %-6d                                  ║\n", stats["filtered_by_invalid"])
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
}

// ResetStatistics 重置统计信息
func (v *SmartURLValidator) ResetStatistics() {
	v.filteredByJSCode = 0
	v.filteredByHTMLTag = 0
	v.filteredBySymbol = 0
	v.filteredByEncoding = 0
	v.filteredByScheme = 0
	v.filteredByLength = 0
	v.filteredByInvalid = 0
	v.totalChecked = 0
	v.totalPassed = 0
}

// SetEncodingThreshold 设置URL编码阈值
func (v *SmartURLValidator) SetEncodingThreshold(threshold float64) {
	if threshold > 0 && threshold <= 1.0 {
		v.encodingThreshold = threshold
	}
}

// SetMaxURLLength 设置最大URL长度
func (v *SmartURLValidator) SetMaxURLLength(length int) {
	if length > 0 {
		v.maxURLLength = length
	}
}

// =========================================================================
// 兼容层 - 提供与旧版URLValidator相同的接口
// =========================================================================

// SmartURLValidatorCompat 兼容适配器 - 提供与旧版相同的接口
type SmartURLValidatorCompat struct {
	*SmartURLValidator
}

// NewSmartURLValidatorCompat 创建兼容的验证器
func NewSmartURLValidatorCompat() *SmartURLValidatorCompat {
	return &SmartURLValidatorCompat{
		SmartURLValidator: NewSmartURLValidator(),
	}
}

// IsValidBusinessURL 兼容方法 - 只返回bool（与旧版接口一致）
// 内部使用新的验证逻辑，但接口保持不变
func (v *SmartURLValidatorCompat) IsValidBusinessURL(rawURL string) bool {
	valid, _ := v.SmartURLValidator.IsValidBusinessURL(rawURL)
	return valid
}

// FilterURLs 批量过滤URL列表（兼容接口）
func (v *SmartURLValidatorCompat) FilterURLs(urls []string) []string {
	return v.SmartURLValidator.FilterURLs(urls)
}

