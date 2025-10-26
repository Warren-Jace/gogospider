package core

import (
	"regexp"
	"strings"
)

// CSSAnalyzer CSS分析器 - 从CSS内容中提取URL
type CSSAnalyzer struct {
	targetDomain string // 目标域名，用于拼接相对路径
}

// NewCSSAnalyzer 创建CSS分析器实例
func NewCSSAnalyzer() *CSSAnalyzer {
	return &CSSAnalyzer{}
}

// SetTargetDomain 设置目标域名
func (c *CSSAnalyzer) SetTargetDomain(domain string) {
	c.targetDomain = domain
}

// ExtractURLs 从CSS内容中提取所有URL
func (c *CSSAnalyzer) ExtractURLs(cssContent string) []string {
	urls := make([]string, 0)
	seen := make(map[string]bool)
	
	// 定义CSS中URL的正则表达式模式
	patterns := []string{
		// 1. url() 函数 - 最常见的形式
		`url\s*\(\s*['"]?([^'")]+)['"]?\s*\)`,
		
		// 2. @import 规则
		`@import\s+['"]([^'"]+)['"]`,
		`@import\s+url\s*\(\s*['"]?([^'")]+)['"]?\s*\)`,
		
		// 3. @font-face src
		`src\s*:\s*url\s*\(\s*['"]?([^'")]+)['"]?\s*\)`,
		
		// 4. image-set() 函数（现代CSS）
		`image-set\s*\(\s*url\s*\(\s*['"]?([^'")]+)['"]?\s*\)`,
	}
	
	// 提取URL
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(cssContent, -1)
		
		for _, match := range matches {
			if len(match) >= 2 {
				url := match[len(match)-1] // 获取最后一个捕获组
				url = strings.TrimSpace(url)
				
				// 过滤无效URL
				if !c.isValidURL(url) {
					continue
				}
				
				// 去重
				if seen[url] {
					continue
				}
				seen[url] = true
				
				urls = append(urls, url)
			}
		}
	}
	
	return urls
}

// isValidURL 判断URL是否有效
func (c *CSSAnalyzer) isValidURL(url string) bool {
	// 过滤空字符串
	if url == "" {
		return false
	}
	
	// 过滤data: URI（内联数据）
	if strings.HasPrefix(url, "data:") {
		return false
	}
	
	// 过滤javascript: URI
	if strings.HasPrefix(url, "javascript:") {
		return false
	}
	
	// 过滤过短的URL
	if len(url) < 2 {
		return false
	}
	
	return true
}

// AnalyzeCSS 综合分析CSS内容
func (c *CSSAnalyzer) AnalyzeCSS(cssContent string) map[string]interface{} {
	result := make(map[string]interface{})
	
	// 提取所有URL
	urls := c.ExtractURLs(cssContent)
	result["urls"] = urls
	result["url_count"] = len(urls)
	
	// 按类型分类
	images := make([]string, 0)
	fonts := make([]string, 0)
	stylesheets := make([]string, 0)
	others := make([]string, 0)
	
	for _, url := range urls {
		urlLower := strings.ToLower(url)
		
		// 图片资源
		if strings.HasSuffix(urlLower, ".jpg") || 
		   strings.HasSuffix(urlLower, ".jpeg") ||
		   strings.HasSuffix(urlLower, ".png") ||
		   strings.HasSuffix(urlLower, ".gif") ||
		   strings.HasSuffix(urlLower, ".svg") ||
		   strings.HasSuffix(urlLower, ".webp") ||
		   strings.HasSuffix(urlLower, ".ico") {
			images = append(images, url)
		} else if strings.HasSuffix(urlLower, ".woff") ||
		          strings.HasSuffix(urlLower, ".woff2") ||
		          strings.HasSuffix(urlLower, ".ttf") ||
		          strings.HasSuffix(urlLower, ".eot") ||
		          strings.HasSuffix(urlLower, ".otf") {
			// 字体资源
			fonts = append(fonts, url)
		} else if strings.HasSuffix(urlLower, ".css") {
			// CSS文件
			stylesheets = append(stylesheets, url)
		} else {
			// 其他资源
			others = append(others, url)
		}
	}
	
	result["images"] = images
	result["fonts"] = fonts
	result["stylesheets"] = stylesheets
	result["others"] = others
	
	result["image_count"] = len(images)
	result["font_count"] = len(fonts)
	result["stylesheet_count"] = len(stylesheets)
	result["other_count"] = len(others)
	
	return result
}

// ExtractImports 专门提取@import导入的CSS文件
func (c *CSSAnalyzer) ExtractImports(cssContent string) []string {
	imports := make([]string, 0)
	seen := make(map[string]bool)
	
	patterns := []string{
		`@import\s+['"]([^'"]+\.css)['"]`,
		`@import\s+url\s*\(\s*['"]?([^'")]+\.css)['"]?\s*\)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(cssContent, -1)
		
		for _, match := range matches {
			if len(match) >= 2 {
				importURL := strings.TrimSpace(match[1])
				if importURL != "" && !seen[importURL] {
					seen[importURL] = true
					imports = append(imports, importURL)
				}
			}
		}
	}
	
	return imports
}

// ExtractFontFaces 专门提取@font-face中的字体URL
func (c *CSSAnalyzer) ExtractFontFaces(cssContent string) []string {
	fonts := make([]string, 0)
	seen := make(map[string]bool)
	
	// 匹配@font-face块
	fontFacePattern := regexp.MustCompile(`@font-face\s*\{([^}]+)\}`)
	fontFaceMatches := fontFacePattern.FindAllStringSubmatch(cssContent, -1)
	
	for _, match := range fontFaceMatches {
		if len(match) >= 2 {
			fontFaceBlock := match[1]
			
			// 从@font-face块中提取src
			srcPattern := regexp.MustCompile(`src\s*:\s*url\s*\(\s*['"]?([^'")]+)['"]?\s*\)`)
			srcMatches := srcPattern.FindAllStringSubmatch(fontFaceBlock, -1)
			
			for _, srcMatch := range srcMatches {
				if len(srcMatch) >= 2 {
					fontURL := strings.TrimSpace(srcMatch[1])
					if fontURL != "" && !seen[fontURL] {
						seen[fontURL] = true
						fonts = append(fonts, fontURL)
					}
				}
			}
		}
	}
	
	return fonts
}

// GetStatistics 获取统计信息
func (c *CSSAnalyzer) GetStatistics() map[string]int {
	return map[string]int{
		"css_analyzer_version": 1,
	}
}

