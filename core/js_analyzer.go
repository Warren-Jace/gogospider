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
				if !containsString(apis, api) {
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
				if !isCommonWord(param) && !containsString(params, param) {
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
				if isValidLink(link) && !containsString(links, link) {
					links = append(links, link)
				}
			}
		}
	}
	
	return links
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

// ExtractRelativeURLs 从跨域JS中提取相对路径URL，并拼接为完整URL（增强版）
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

// ExtractFromJSObjects 从JavaScript对象和配置中提取URL（Phase 3增强）
func (j *JSAnalyzer) ExtractFromJSObjects(jsContent string) []string {
	urls := make([]string, 0)
	seen := make(map[string]bool)
	
	// 提取JSON配置对象中的URL
	patterns := []string{
		// 1. 配置对象
		`config\s*[=:]\s*{[^}]*["']url["']\s*:\s*["']([^"']+)["']`,
		`settings\s*[=:]\s*{[^}]*["']endpoint["']\s*:\s*["']([^"']+)["']`,
		
		// 2. API配置
		`API_BASE\s*[=:]\s*["']([^"']+)["']`,
		`BASE_URL\s*[=:]\s*["']([^"']+)["']`,
		`ENDPOINT\s*[=:]\s*["']([^"']+)["']`,
		
		// 3. 路由配置
		`routes\s*[=:]\s*\{([^}]+)\}`,
		`path\s*:\s*["']([^"']+)["']`,
		
		// 4. 模板字符串中的URL
		"[`][^`]*(/[a-zA-Z0-9/_\\-]+)[^`]*[`]",
		
		// 5. 动态URL构建
		`['"](/\$\{[^}]+\}/[^'"]*)['"]`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsContent, -1)
		
		for _, match := range matches {
			if len(match) >= 2 {
				urlPath := match[len(match)-1]
				
				// 清理和验证
				urlPath = strings.TrimSpace(urlPath)
				if urlPath != "" && !seen[urlPath] && j.isValidPath(urlPath) {
					seen[urlPath] = true
					
					// 拼接完整URL
					if strings.HasPrefix(urlPath, "/") {
						scheme := "http://"
						if strings.Contains(j.targetDomain, "https") {
							scheme = "https://"
						}
						cleanDomain := strings.TrimPrefix(j.targetDomain, "http://")
						cleanDomain = strings.TrimPrefix(cleanDomain, "https://")
						fullURL := scheme + cleanDomain + urlPath
						urls = append(urls, fullURL)
					}
				}
			}
		}
	}
	
	return urls
}

// ExtractAjaxURLs 专门提取AJAX请求URL（Phase 3增强）
func (j *JSAnalyzer) ExtractAjaxURLs(jsContent string) []string {
	urls := make([]string, 0)
	seen := make(map[string]bool)
	
	// AJAX特定模式
	patterns := []string{
		// XMLHttpRequest
		`xhr\.open\s*\(\s*["'](GET|POST)["']\s*,\s*["']([^"']+)["']`,
		`\.send\s*\(\s*["']([^"']+)["']`,
		
		// jQuery AJAX
		`\$\.ajax\s*\(\s*{[^}]*url\s*:\s*["']([^"']+)["']`,
		`\$\.getJSON\s*\(\s*["']([^"']+)["']`,
		
		// Fetch API
		`fetch\s*\(\s*["']([^"']+)["']`,
		
		// Axios
		`axios\s*\.\s*(get|post|put|delete|patch)\s*\(\s*["']([^"']+)["']`,
		`axios\s*\(\s*{[^}]*url\s*:\s*["']([^"']+)["']`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsContent, -1)
		
		for _, match := range matches {
			if len(match) >= 2 {
				// 获取URL（可能在不同位置）
				urlPath := ""
				for i := len(match) - 1; i >= 1; i-- {
					if strings.Contains(match[i], "/") || strings.Contains(match[i], "http") {
						urlPath = match[i]
						break
					}
				}
				
				if urlPath != "" && !seen[urlPath] {
					seen[urlPath] = true
					
					// 处理相对路径
					if strings.HasPrefix(urlPath, "/") && j.targetDomain != "" {
						scheme := "http://"
						if strings.Contains(j.targetDomain, "https") {
							scheme = "https://"
						}
						cleanDomain := strings.TrimPrefix(j.targetDomain, "http://")
						cleanDomain = strings.TrimPrefix(cleanDomain, "https://")
						urlPath = scheme + cleanDomain + urlPath
					}
					
					urls = append(urls, urlPath)
				}
			}
		}
	}
	
	return urls
}

// AnalyzeRouterConfig 分析前端路由配置（Phase 3增强）
func (j *JSAnalyzer) AnalyzeRouterConfig(jsContent string) []string {
	urls := make([]string, 0)
	seen := make(map[string]bool)
	
	// 路由配置模式
	patterns := []string{
		// Vue Router
		`path\s*:\s*["']([^"']+)["']`,
		`route\s*:\s*["']([^"']+)["']`,
		
		// React Router
		`<Route\s+path\s*=\s*["']([^"']+)["']`,
		
		// Angular Router
		`{[^}]*path\s*:\s*["']([^"']+)["'][^}]*}`,
		
		// 通用路由数组
		`routes\s*[=:]\s*\[([\s\S]*?)\]`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsContent, -1)
		
		for _, match := range matches {
			if len(match) >= 2 {
				path := match[1]
				
				// 清理路由参数
				path = regexp.MustCompile(`:\w+`).ReplaceAllString(path, "1")
				path = regexp.MustCompile(`\*`).ReplaceAllString(path, "")
				
				if path != "" && !seen[path] {
					seen[path] = true
					
					// 确保以/开头
					if !strings.HasPrefix(path, "/") {
						path = "/" + path
					}
					
					// 拼接完整URL
					if j.targetDomain != "" {
						scheme := "http://"
						if strings.Contains(j.targetDomain, "https") {
							scheme = "https://"
						}
						cleanDomain := strings.TrimPrefix(j.targetDomain, "http://")
						cleanDomain = strings.TrimPrefix(cleanDomain, "https://")
						fullURL := scheme + cleanDomain + path
						urls = append(urls, fullURL)
					}
				}
			}
		}
	}
	
	return urls
}

// EnhancedAnalyze 增强的综合分析（Phase 3集成方法）
func (j *JSAnalyzer) EnhancedAnalyze(jsContent string) map[string][]string {
	result := make(map[string][]string)
	
	// 基础分析
	apis, params, links := j.Analyze(jsContent)
	result["basic_apis"] = apis
	result["basic_params"] = params
	result["basic_links"] = links
	
	// 相对URL提取
	result["relative_urls"] = j.ExtractRelativeURLs(jsContent)
	
	// JavaScript对象中的URL
	result["object_urls"] = j.ExtractFromJSObjects(jsContent)
	
	// AJAX URL
	result["ajax_urls"] = j.ExtractAjaxURLs(jsContent)
	
	// 路由配置
	result["router_urls"] = j.AnalyzeRouterConfig(jsContent)
	
	return result
}