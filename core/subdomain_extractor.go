package core

import (
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// SubdomainExtractor 子域名提取器
type SubdomainExtractor struct {
	// 主域名
	mainDomain string
	
	// 提取的子域名集合（去重）
	subdomains map[string]bool
	mutex      sync.RWMutex
	
	// 正则表达式模式
	patterns []*regexp.Regexp
}

// NewSubdomainExtractor 创建子域名提取器
func NewSubdomainExtractor(targetURL string) *SubdomainExtractor {
	se := &SubdomainExtractor{
		subdomains: make(map[string]bool),
	}
	
	// 解析主域名
	if parsedURL, err := url.Parse(targetURL); err == nil {
		se.mainDomain = se.extractMainDomain(parsedURL.Host)
	}
	
	// 初始化正则表达式
	se.initializePatterns()
	
	return se
}

// initializePatterns 初始化子域名匹配模式
func (se *SubdomainExtractor) initializePatterns() {
	se.patterns = []*regexp.Regexp{
		// 匹配标准URL格式
		regexp.MustCompile(`https?://([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}`),
		
		// 匹配双斜杠格式 //subdomain.example.com
		regexp.MustCompile(`//([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}`),
		
		// 匹配引号中的域名
		regexp.MustCompile(`["']https?://([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}["']`),
		
		// 匹配JS变量赋值中的域名
		regexp.MustCompile(`domain\s*[:=]\s*["']([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}["']`),
		regexp.MustCompile(`host\s*[:=]\s*["']([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}["']`),
		
		// 匹配API端点配置
		regexp.MustCompile(`api[_-]?(?:url|endpoint|host)\s*[:=]\s*["']([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}["']`),
		
		// 匹配location.href等
		regexp.MustCompile(`location\.href\s*=\s*["']https?://([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}`),
		
		// 匹配window.location
		regexp.MustCompile(`window\.location\s*=\s*["']https?://([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}`),
		
		// 匹配a标签href
		regexp.MustCompile(`href\s*=\s*["']https?://([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}`),
		
		// 匹配src属性
		regexp.MustCompile(`src\s*=\s*["']https?://([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}`),
		
		// 匹配CSS中的url()
		regexp.MustCompile(`url\(["']?https?://([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}`),
		
		// 匹配注释中的域名
		regexp.MustCompile(`//\s*https?://([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}`),
		
		// 匹配纯域名格式（不含协议）
		regexp.MustCompile(`\b([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}\b`),
	}
}

// ExtractFromHTML 从HTML内容提取子域名
func (se *SubdomainExtractor) ExtractFromHTML(htmlContent string) []string {
	discovered := make([]string, 0)
	
	// 使用所有模式提取
	for _, pattern := range se.patterns {
		matches := pattern.FindAllString(htmlContent, -1)
		for _, match := range matches {
			// 清理匹配结果
			domain := se.cleanDomain(match)
			if domain != "" && se.isSubdomain(domain) {
				if se.addSubdomain(domain) {
					discovered = append(discovered, domain)
				}
			}
		}
	}
	
	return discovered
}

// ExtractFromJS 从JavaScript内容提取子域名
func (se *SubdomainExtractor) ExtractFromJS(jsContent string) []string {
	// JS文件通常包含更多的域名配置
	return se.ExtractFromHTML(jsContent)
}

// ExtractFromCSS 从CSS内容提取子域名
func (se *SubdomainExtractor) ExtractFromCSS(cssContent string) []string {
	discovered := make([]string, 0)
	
	// CSS主要通过url()引用资源
	urlPattern := regexp.MustCompile(`url\(["']?(https?://)?([^"')]+)["']?\)`)
	matches := urlPattern.FindAllStringSubmatch(cssContent, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			urlStr := match[2]
			// 尝试解析域名
			if parsedURL, err := url.Parse("http://" + urlStr); err == nil {
				domain := parsedURL.Host
				if domain != "" && se.isSubdomain(domain) {
					if se.addSubdomain(domain) {
						discovered = append(discovered, domain)
					}
				}
			}
		}
	}
	
	return discovered
}

// ExtractFromURL 从URL中提取子域名
func (se *SubdomainExtractor) ExtractFromURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	
	domain := parsedURL.Host
	// 移除端口
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	
	if domain != "" && se.isSubdomain(domain) {
		if se.addSubdomain(domain) {
			return domain
		}
	}
	
	return ""
}

// cleanDomain 清理域名字符串
func (se *SubdomainExtractor) cleanDomain(match string) string {
	// 移除协议前缀
	domain := strings.TrimPrefix(match, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "//")
	
	// 移除引号
	domain = strings.Trim(domain, `"'`)
	
	// 移除路径部分
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	
	// 移除端口
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	
	// 移除多余的空白字符
	domain = strings.TrimSpace(domain)
	
	// 转换为小写
	domain = strings.ToLower(domain)
	
	return domain
}

// extractMainDomain 提取主域名（去除子域名部分）
func (se *SubdomainExtractor) extractMainDomain(host string) string {
	// 移除端口
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		// 返回最后两部分作为主域名 (example.com)
		// 注意：这个简化实现不处理 .co.uk 等特殊TLD
		return strings.Join(parts[len(parts)-2:], ".")
	}
	
	return host
}

// isSubdomain 判断是否为目标域名的子域名
func (se *SubdomainExtractor) isSubdomain(domain string) bool {
	if se.mainDomain == "" {
		return false
	}
	
	// 必须以主域名结尾
	if !strings.HasSuffix(domain, se.mainDomain) {
		return false
	}
	
	// 不能是主域名本身（除非它就是子域名）
	if domain == se.mainDomain {
		// 检查是否是www等常见子域名
		return false
	}
	
	// 必须是有效的域名格式
	if !se.isValidDomain(domain) {
		return false
	}
	
	return true
}

// isValidDomain 验证域名格式
func (se *SubdomainExtractor) isValidDomain(domain string) bool {
	// 基本格式验证
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	
	// 不能包含空格
	if strings.Contains(domain, " ") {
		return false
	}
	
	// 必须包含至少一个点
	if !strings.Contains(domain, ".") {
		return false
	}
	
	// 域名标签验证
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		
		// 标签不能以连字符开头或结尾
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}
	
	return true
}

// addSubdomain 添加子域名（带去重）
func (se *SubdomainExtractor) addSubdomain(subdomain string) bool {
	se.mutex.Lock()
	defer se.mutex.Unlock()
	
	if se.subdomains[subdomain] {
		return false // 已存在
	}
	
	se.subdomains[subdomain] = true
	return true
}

// GetAllSubdomains 获取所有发现的子域名
func (se *SubdomainExtractor) GetAllSubdomains() []string {
	se.mutex.RLock()
	defer se.mutex.RUnlock()
	
	subdomains := make([]string, 0, len(se.subdomains))
	for subdomain := range se.subdomains {
		subdomains = append(subdomains, subdomain)
	}
	
	return subdomains
}

// GetSubdomainCount 获取子域名数量
func (se *SubdomainExtractor) GetSubdomainCount() int {
	se.mutex.RLock()
	defer se.mutex.RUnlock()
	
	return len(se.subdomains)
}

// GetMainDomain 获取主域名
func (se *SubdomainExtractor) GetMainDomain() string {
	return se.mainDomain
}

// Clear 清除所有子域名记录
func (se *SubdomainExtractor) Clear() {
	se.mutex.Lock()
	defer se.mutex.Unlock()
	
	se.subdomains = make(map[string]bool)
}

// GetSubdomainsByLevel 按层级分类子域名
func (se *SubdomainExtractor) GetSubdomainsByLevel() map[int][]string {
	se.mutex.RLock()
	defer se.mutex.RUnlock()
	
	byLevel := make(map[int][]string)
	
	for subdomain := range se.subdomains {
		// 计算子域名层级
		level := strings.Count(subdomain, ".") - strings.Count(se.mainDomain, ".")
		byLevel[level] = append(byLevel[level], subdomain)
	}
	
	return byLevel
}

// GetStatistics 获取统计信息
func (se *SubdomainExtractor) GetStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["main_domain"] = se.mainDomain
	stats["total_subdomains"] = se.GetSubdomainCount()
	stats["subdomains_by_level"] = se.GetSubdomainsByLevel()
	
	return stats
}

// ExportSubdomains 导出子域名列表（格式化）
func (se *SubdomainExtractor) ExportSubdomains() []string {
	subdomains := se.GetAllSubdomains()
	
	// 按字母顺序排序
	sortedSubdomains := make([]string, len(subdomains))
	copy(sortedSubdomains, subdomains)
	
	// 简单的冒泡排序
	for i := 0; i < len(sortedSubdomains); i++ {
		for j := i + 1; j < len(sortedSubdomains); j++ {
			if sortedSubdomains[i] > sortedSubdomains[j] {
				sortedSubdomains[i], sortedSubdomains[j] = sortedSubdomains[j], sortedSubdomains[i]
			}
		}
	}
	
	return sortedSubdomains
}

