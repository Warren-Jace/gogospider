package core

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// JSSpecialHandler JS文件特殊处理器
// 核心功能：
// 1. JS文件不受Scope限制（可跨域爬取CDN的JS）
// 2. 黑名单检查仍然生效
// 3. 自动路径拼接
type JSSpecialHandler struct {
	// 黑名单域名
	blacklistDomains map[string]bool
	
	// 目标基础URL（用于拼接）
	baseURL *url.URL
	
	// 统计
	stats JSHandlerStats
}

// JSHandlerStats JS处理统计
type JSHandlerStats struct {
	TotalJSFiles    int // 总JS文件数
	CrossDomainJS   int // 跨域JS数
	LocalJS         int // 本域JS数
	PathJoined      int // 路径拼接数
	BlacklistBlock  int // 黑名单拦截数
}

// NewJSSpecialHandler 创建JS特殊处理器
func NewJSSpecialHandler(targetURL string, blacklist []string) (*JSSpecialHandler, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("解析目标URL失败: %v", err)
	}
	
	handler := &JSSpecialHandler{
		blacklistDomains: make(map[string]bool),
		baseURL:          parsedURL,
		stats:            JSHandlerStats{},
	}
	
	// 初始化黑名单
	for _, domain := range blacklist {
		handler.blacklistDomains[strings.ToLower(domain)] = true
	}
	
	return handler, nil
}

// ShouldProcessJS 判断JS文件是否应该处理
// 返回: (是否处理, 处理后的URL, 原因)
func (h *JSSpecialHandler) ShouldProcessJS(rawURL string) (bool, string, string) {
	// 1. 判断是否为JS文件
	if !h.isJSFile(rawURL) {
		return false, "", "不是JS文件"
	}
	
	h.stats.TotalJSFiles++
	
	// 2. 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, "", fmt.Sprintf("URL解析失败: %v", err)
	}
	
	// 3. 黑名单检查（必需）
	if h.isBlacklisted(parsedURL.Host) {
		h.stats.BlacklistBlock++
		return false, "", fmt.Sprintf("黑名单域名: %s", parsedURL.Host)
	}
	
	// 4. 路径拼接处理
	finalURL := rawURL
	
	// 如果是相对路径或协议相对路径，进行拼接
	if strings.HasPrefix(rawURL, "//") {
		// 协议相对路径: //cdn.example.com/app.js
		finalURL = h.baseURL.Scheme + ":" + rawURL
		h.stats.PathJoined++
		
	} else if strings.HasPrefix(rawURL, "/") && !strings.HasPrefix(rawURL, "//") {
		// 绝对路径: /static/app.js
		finalURL = fmt.Sprintf("%s://%s%s", 
			h.baseURL.Scheme, 
			h.baseURL.Host, 
			rawURL)
		h.stats.PathJoined++
		h.stats.LocalJS++
		
	} else if !strings.HasPrefix(rawURL, "http://") && 
	          !strings.HasPrefix(rawURL, "https://") {
		// 相对路径: ../js/app.js 或 js/app.js
		basePath := path.Dir(h.baseURL.Path)
		if basePath == "" || basePath == "." {
			basePath = "/"
		}
		joinedPath := path.Join(basePath, rawURL)
		finalURL = fmt.Sprintf("%s://%s%s",
			h.baseURL.Scheme,
			h.baseURL.Host,
			joinedPath)
		h.stats.PathJoined++
		h.stats.LocalJS++
		
	} else {
		// 完整URL: http://cdn.example.com/app.js
		if parsedURL.Host != "" && parsedURL.Host != h.baseURL.Host {
			h.stats.CrossDomainJS++
		} else {
			h.stats.LocalJS++
		}
	}
	
	// ✅ JS文件不受Scope限制，直接允许
	return true, finalURL, "JS文件，允许跨域爬取"
}

// isJSFile 判断是否为JS文件
func (h *JSSpecialHandler) isJSFile(rawURL string) bool {
	lowerURL := strings.ToLower(rawURL)
	
	// 检查文件扩展名
	jsExtensions := []string{".js", ".mjs", ".jsx"}
	for _, ext := range jsExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return true
		}
		// 处理带参数的情况: app.js?v=123
		if strings.Contains(lowerURL, ext+"?") {
			return true
		}
	}
	
	// 检查MIME类型提示
	if strings.Contains(lowerURL, "javascript") {
		return true
	}
	
	return false
}

// isBlacklisted 检查域名是否在黑名单中
func (h *JSSpecialHandler) isBlacklisted(domain string) bool {
	if domain == "" {
		return false
	}
	
	domain = strings.ToLower(domain)
	
	// 精确匹配
	if h.blacklistDomains[domain] {
		return true
	}
	
	// 通配符匹配: *.example.com
	parts := strings.Split(domain, ".")
	for i := 0; i < len(parts); i++ {
		wildcard := "*." + strings.Join(parts[i:], ".")
		if h.blacklistDomains[wildcard] {
			return true
		}
	}
	
	return false
}

// GetStatistics 获取统计信息
func (h *JSSpecialHandler) GetStatistics() JSHandlerStats {
	return h.stats
}

// PrintReport 打印统计报告
func (h *JSSpecialHandler) PrintReport() {
	fmt.Println("\n╔═══════════════════════════════════════╗")
	fmt.Println("║      JS文件特殊处理统计报告          ║")
	fmt.Println("╚═══════════════════════════════════════╝")
	fmt.Printf("  总JS文件数:      %d\n", h.stats.TotalJSFiles)
	fmt.Printf("  本域JS:          %d\n", h.stats.LocalJS)
	fmt.Printf("  跨域JS:          %d\n", h.stats.CrossDomainJS)
	fmt.Printf("  路径拼接:        %d\n", h.stats.PathJoined)
	fmt.Printf("  黑名单拦截:      %d\n", h.stats.BlacklistBlock)
	fmt.Println("─────────────────────────────────────────")
}

