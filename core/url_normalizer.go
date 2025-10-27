package core

import (
	"net/url"
	"regexp"
	"strings"
)

// URLNormalizer 统一的URL规范化处理器
// 解决问题：
// 1. 协议相对URL处理（//example.com）
// 2. 相对路径URL处理（/path/to/resource）
// 3. URL去重和规范化
// 4. 协议变体生成（http/https）
type URLNormalizer struct {
	baseURL         *url.URL
	baseScheme      string
	protocolPattern *regexp.Regexp
}

// NewURLNormalizer 创建URL规范化处理器
func NewURLNormalizer(baseURL string) (*URLNormalizer, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	
	normalizer := &URLNormalizer{
		baseURL:    parsed,
		baseScheme: parsed.Scheme,
	}
	
	// 协议相对URL正则
	normalizer.protocolPattern = regexp.MustCompile(`^//[a-zA-Z0-9\-._~:/?#\[\]@!$&'()*+,;=%]+`)
	
	return normalizer, nil
}

// NormalizeURL 规范化单个URL
// 返回规范化后的URL列表（协议相对URL会返回http和https两个版本）
func (n *URLNormalizer) NormalizeURL(rawURL string) []string {
	trimmed := strings.TrimSpace(rawURL)
	
	// 空URL
	if trimmed == "" {
		return nil
	}
	
	// 1. 已经是完整URL
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return []string{trimmed}
	}
	
	// 2. 协议相对URL（//example.com/path）
	if strings.HasPrefix(trimmed, "//") {
		// 🔧 核心修复：生成http和https两个版本
		httpURL := "http:" + trimmed
		httpsURL := "https:" + trimmed
		
		// 优先使用baseURL的协议
		if n.baseScheme == "https" {
			return []string{httpsURL, httpURL}
		}
		return []string{httpURL, httpsURL}
	}
	
	// 3. 绝对路径（/path/to/resource）
	if strings.HasPrefix(trimmed, "/") {
		absoluteURL := n.baseURL.Scheme + "://" + n.baseURL.Host + trimmed
		return []string{absoluteURL}
	}
	
	// 4. 相对路径（path/to/resource）
	// 解析相对URL
	parsedURL, err := url.Parse(trimmed)
	if err != nil {
		return nil
	}
	
	// 使用ResolveReference解析
	absoluteURL := n.baseURL.ResolveReference(parsedURL)
	return []string{absoluteURL.String()}
}

// NormalizeBatch 批量规范化URL
func (n *URLNormalizer) NormalizeBatch(rawURLs []string) []string {
	seen := make(map[string]bool)
	results := make([]string, 0, len(rawURLs)*2) // 预留空间（考虑协议相对URL会变成2个）
	
	for _, rawURL := range rawURLs {
		normalized := n.NormalizeURL(rawURL)
		for _, u := range normalized {
			if !seen[u] {
				seen[u] = true
				results = append(results, u)
			}
		}
	}
	
	return results
}

// IsProtocolRelativeURL 判断是否为协议相对URL
func (n *URLNormalizer) IsProtocolRelativeURL(rawURL string) bool {
	return n.protocolPattern.MatchString(rawURL)
}

// GetProtocolVariants 获取URL的协议变体
// 对于http URL返回对应的https URL（反之亦然）
func (n *URLNormalizer) GetProtocolVariants(rawURL string) []string {
	if strings.HasPrefix(rawURL, "http://") {
		httpsURL := "https://" + strings.TrimPrefix(rawURL, "http://")
		return []string{rawURL, httpsURL}
	}
	
	if strings.HasPrefix(rawURL, "https://") {
		httpURL := "http://" + strings.TrimPrefix(rawURL, "https://")
		return []string{rawURL, httpURL}
	}
	
	return []string{rawURL}
}

// ResolveURL 统一的URL解析方法（兼容旧代码）
func (n *URLNormalizer) ResolveURL(relativeURL string) string {
	normalized := n.NormalizeURL(relativeURL)
	if len(normalized) > 0 {
		return normalized[0] // 返回第一个（主要的）
	}
	return ""
}

