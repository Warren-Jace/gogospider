package core

import (
	"regexp"
	"strings"
)

// JSAnalyzer JS分析器
type JSAnalyzer struct {
	targetDomain string // 目标域名，用于拼接相对路径
}

// NewJSAnalyzer 创建JS分析器实例
func NewJSAnalyzer() *JSAnalyzer {
	return &JSAnalyzer{}
}

// SetTargetDomain 设置目标域名
func (j *JSAnalyzer) SetTargetDomain(domain string) {
	j.targetDomain = domain
}

// Analyze 分析JavaScript内容，提取API端点、参数和隐藏链接
func (j *JSAnalyzer) Analyze(jsContent string) ([]string, []string, []string) {
	apis := make([]string, 0)
	params := make([]string, 0)
	links := make([]string, 0)
	
	// 提取API端点
	apis = j.extractAPIs(jsContent)
	
	// 提取参数
	params = j.extractParams(jsContent)
	
	// 提取隐藏链接
	links = j.extractLinks(jsContent)
	
	return apis, params, links
}

// extractAPIs 从JavaScript中提取API端点
func (j *JSAnalyzer) extractAPIs(jsContent string) []string {
	apis := make([]string, 0)
	
	// 定义API端点的正则表达式模式
	apiPatterns := []string{
		`['"](/api/[^'"]*)['"]`,
		`['"](/v\d+/[^'"]*)['"]`,
		`['"](/AJAX/[^'"]*)['"]`,  // 添加AJAX路径支持
		`['"](/hpp/[^'"]*)['"]`,   // 添加HPP路径支持
		`(https?://[^\s'"]*/api/[^\s'"]*)`,
		`(https?://[^\s'"]*/v\d+/[^\s'"]*)`,
		`(https?://[^\s'"]*/AJAX/[^\s'"]*)`,  // 添加AJAX完整URL支持
		`(https?://[^\s'"]*/hpp/[^\s'"]*)`,   // 添加HPP完整URL支持
		`['"](api/[^\s'"]*)['"]`,
		`['"](AJAX/[^\s'"]*)['"]`,  // 添加相对AJAX路径支持
		`['"](hpp/[^\s'"]*)['"]`,   // 添加相对HPP路径支持
	}
	
	for _, pattern := range apiPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsContent, -1)
		
		for _, match := range matches {
			if len(match) > 1 {
				api := strings.Trim(match[1], `"'`)
				if !contains(apis, api) {
					apis = append(apis, api)
				}
			}
		}
	}
	
	return apis
}

// extractParams 从JavaScript中提取参数
func (j *JSAnalyzer) extractParams(jsContent string) []string {
	params := make([]string, 0)
	
	// 查找对象属性和变量赋值中的参数
	paramPatterns := []string{
		`['"]([^'"]*)['"]\s*:\s*['"][^'"]*['"]`,
		`var\s+(\w+)\s*=\s*['"][^'"]*['"]`,
		`let\s+(\w+)\s*=\s*['"][^'"]*['"]`,
		`const\s+(\w+)\s*=\s*['"][^'"]*['"]`,
	}
	
	for _, pattern := range paramPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsContent, -1)
		
		for _, match := range matches {
			if len(match) > 1 {
				param := match[1]
				// 过滤掉一些常见的非参数词汇
				if !isCommonWord(param) && !contains(params, param) {
					params = append(params, param)
				}
			}
		}
	}
	
	return params
}

// extractLinks 从JavaScript中提取隐藏链接
func (j *JSAnalyzer) extractLinks(jsContent string) []string {
	links := make([]string, 0)
	
	// 查找可能的链接
	linkPatterns := []string{
		`(https?://[^\s'"]*)`,
		`['"](/[^'"]*\.[^'"]*)['"]`,
		`['"](/[^'"]*)['"]`,  // 添加通用路径匹配
	}
	
	for _, pattern := range linkPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsContent, -1)
		
		for _, match := range matches {
			if len(match) > 1 {
				link := strings.Trim(match[1], `"'`)
				// 验证是否为有效链接
				if isValidLink(link) && !contains(links, link) {
					links = append(links, link)
				}
			}
		}
	}
	
	return links
}

// contains 检查slice中是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isCommonWord 检查是否为常见词汇（非参数）
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"function": true, "var": true, "let": true, "const": true,
		"if": true, "else": true, "for": true, "while": true,
		"return": true, "break": true, "continue": true,
		"true": true, "false": true, "null": true, "undefined": true,
		"this": true, "new": true, "typeof": true, "instanceof": true,
	}
	
	_, exists := commonWords[strings.ToLower(word)]
	return exists
}

// isValidLink 简单验证链接是否有效
func isValidLink(link string) bool {
	// 过滤掉太短的字符串
	if len(link) < 3 {
		return false
	}
	
	// 过滤掉明显不是链接的内容
	invalidPatterns := []string{" ", "\t", "\n", ":", ";", "{", "}", "(", ")", "[", "]"}
	for _, pattern := range invalidPatterns {
		if strings.Contains(link, pattern) {
			return false
		}
	}
	
	return true
}

// ExtractRelativeURLs 从跨域JS中提取相对路径URL，并拼接为完整URL
func (j *JSAnalyzer) ExtractRelativeURLs(jsContent string) []string {
	if j.targetDomain == "" {
		return []string{}
	}
	
	urls := make([]string, 0)
	seenPaths := make(map[string]bool)
	
	// 匹配模式：相对路径（以/开头）
	patterns := []string{
		// 1. fetch/axios等API调用
		`fetch\s*\(\s*['"](/[^'"\s?#]+)`,
		`axios\.(get|post|put|delete|patch)\s*\(\s*['"](/[^'"\s?#]+)`,
		`\$\.ajax\s*\(\s*{[^}]*url\s*:\s*['"](/[^'"\s?#]+)`,
		`\$\.(get|post)\s*\(\s*['"](/[^'"\s?#]+)`,
		
		// 2. window.location/href相关
		`window\.location\s*=\s*['"](/[^'"\s?#]+)`,
		`window\.location\.href\s*=\s*['"](/[^'"\s?#]+)`,
		`location\.href\s*=\s*['"](/[^'"\s?#]+)`,
		`href\s*:\s*['"](/[^'"\s?#]+)`,
		
		// 3. 导航/路由相关
		`router\.(push|replace)\s*\(\s*['"](/[^'"\s?#]+)`,
		`navigate\s*\(\s*['"](/[^'"\s?#]+)`,
		`redirect\s*\(\s*['"](/[^'"\s?#]+)`,
		`path\s*:\s*['"](/[^'"\s?#]+)`,
		
		// 4. API端点定义
		`['"](/api/[^'"\s?#]+)['"]`,
		`['"](/v\d+/[^'"\s?#]+)['"]`,
		`['"](/admin/[^'"\s?#]+)['"]`,
		`['"](/user/[^'"\s?#]+)['"]`,
		`['"](/login[^'"\s?#]*)['"]`,
		`['"](/logout[^'"\s?#]*)['"]`,
		`['"](/register[^'"\s?#]*)['"]`,
		
		// 5. 资源路径
		`src\s*:\s*['"](/[^'"\s?#]+)`,
		`url\s*:\s*['"](/[^'"\s?#]+)`,
		`endpoint\s*:\s*['"](/[^'"\s?#]+)`,
		`baseURL\s*:\s*['"](/[^'"\s?#]+)`,
		
		// 6. 通用引号包含的路径
		`['"](/[a-zA-Z0-9_\-/]{3,})['"]`,
	}
	
	// 提取URL路径
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsContent, -1)
		
		for _, match := range matches {
			if len(match) >= 2 {
				// 获取路径（最后一个捕获组）
				path := match[len(match)-1]
				
				// 过滤无效路径
				if !j.isValidPath(path) {
					continue
				}
				
				// 去重
				if seenPaths[path] {
					continue
				}
				seenPaths[path] = true
				
				// 拼接完整URL
				scheme := "http://"
				if strings.Contains(j.targetDomain, "https") {
					scheme = "https://"
				}
				
				// 清理域名（去除可能的协议前缀）
				cleanDomain := strings.TrimPrefix(j.targetDomain, "http://")
				cleanDomain = strings.TrimPrefix(cleanDomain, "https://")
				
				fullURL := scheme + cleanDomain + path
				urls = append(urls, fullURL)
			}
		}
	}
	
	return urls
}

// isValidPath 判断路径是否有效
func (j *JSAnalyzer) isValidPath(path string) bool {
	// 必须以/开头
	if !strings.HasPrefix(path, "/") {
		return false
	}
	
	// 长度检查
	if len(path) < 2 || len(path) > 200 {
		return false
	}
	
	// 过滤明显的静态资源（我们只要页面和API）
	staticExts := []string{
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico", ".webp", ".bmp",
		".css", ".less", ".sass", ".scss",
		".woff", ".woff2", ".ttf", ".eot", ".otf",
		".mp4", ".mp3", ".avi", ".mov", ".wmv",
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
		".zip", ".rar", ".tar", ".gz",
	}
	
	pathLower := strings.ToLower(path)
	for _, ext := range staticExts {
		if strings.HasSuffix(pathLower, ext) {
			return false
		}
	}
	
	// 过滤特殊字符过多的路径（可能是数据而非URL）
	specialChars := strings.Count(path, "{") + strings.Count(path, "}") + 
	                strings.Count(path, "[") + strings.Count(path, "]") +
	                strings.Count(path, "<") + strings.Count(path, ">")
	if specialChars > 2 {
		return false
	}
	
	// 过滤纯数字路径（如 /123456）
	if matched, _ := regexp.MatchString(`^/\d+$`, path); matched {
		return false
	}
	
	return true
}

// AnalyzeExternalJS 分析外部JS文件（专用于跨域JS分析）
func (j *JSAnalyzer) AnalyzeExternalJS(jsContent string, sourceURL string) map[string]interface{} {
	result := make(map[string]interface{})
	
	// 提取相对路径URL
	relativeURLs := j.ExtractRelativeURLs(jsContent)
	result["urls"] = relativeURLs
	result["url_count"] = len(relativeURLs)
	result["source"] = sourceURL
	
	// 额外分析：API端点、参数等
	apis, params, links := j.Analyze(jsContent)
	result["apis"] = apis
	result["params"] = params
	result["links"] = links
	
	// 统计信息
	result["total_findings"] = len(relativeURLs) + len(apis) + len(links)
	
	return result
}