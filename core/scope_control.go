package core

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ScopeConfig Scope配置（参考Katana设计）
type ScopeConfig struct {
	// 域名控制
	IncludeDomains []string // 包含的域名列表（支持通配符）
	ExcludeDomains []string // 排除的域名列表
	
	// 路径控制
	IncludePaths   []string // 包含的路径模式
	ExcludePaths   []string // 排除的路径模式
	
	// 正则控制
	IncludeRegex   string   // 包含的URL正则
	ExcludeRegex   string   // 排除的URL正则
	
	// 文件扩展名控制
	IncludeExtensions []string // 包含的文件扩展名
	ExcludeExtensions []string // 排除的文件扩展名
	
	// 参数控制
	IncludeParams  []string // 包含的参数名
	ExcludeParams  []string // 排除的参数名
	
	// 深度控制
	MaxDepth       int      // 最大深度
	StayInDomain   bool     // 是否限制在同一域名内
	
	// 协议控制
	AllowHTTP      bool     // 允许HTTP
	AllowHTTPS     bool     // 允许HTTPS
	
	// 其他
	AllowSubdomains bool    // 允许子域名
}

// ScopeController Scope控制器
type ScopeController struct {
	config ScopeConfig
	
	// 编译后的正则
	includeRegex *regexp.Regexp
	excludeRegex *regexp.Regexp
	
	// 缓存
	domainCache map[string]bool
	pathCache   map[string]bool
}

// NewScopeController 创建Scope控制器
func NewScopeController(config ScopeConfig) (*ScopeController, error) {
	sc := &ScopeController{
		config:      config,
		domainCache: make(map[string]bool),
		pathCache:   make(map[string]bool),
	}
	
	// 编译正则表达式
	var err error
	if config.IncludeRegex != "" {
		sc.includeRegex, err = regexp.Compile(config.IncludeRegex)
		if err != nil {
			return nil, fmt.Errorf("编译包含正则失败: %v", err)
		}
	}
	
	if config.ExcludeRegex != "" {
		sc.excludeRegex, err = regexp.Compile(config.ExcludeRegex)
		if err != nil {
			return nil, fmt.Errorf("编译排除正则失败: %v", err)
		}
	}
	
	return sc, nil
}

// IsInScope 检查URL是否在scope内
func (sc *ScopeController) IsInScope(rawURL string) bool {
	// 1. 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	
	// 2. 协议检查
	if !sc.checkProtocol(parsedURL.Scheme) {
		return false
	}
	
	// 3. 域名检查
	if !sc.checkDomain(parsedURL.Host) {
		return false
	}
	
	// 4. 路径检查
	if !sc.checkPath(parsedURL.Path) {
		return false
	}
	
	// 5. 扩展名检查
	if !sc.checkExtension(parsedURL.Path) {
		return false
	}
	
	// 6. 参数检查
	if !sc.checkParams(parsedURL.Query()) {
		return false
	}
	
	// 7. 正则检查
	if !sc.checkRegex(rawURL) {
		return false
	}
	
	return true
}

// checkProtocol 检查协议
func (sc *ScopeController) checkProtocol(scheme string) bool {
	scheme = strings.ToLower(scheme)
	
	if scheme == "http" && !sc.config.AllowHTTP {
		return false
	}
	
	if scheme == "https" && !sc.config.AllowHTTPS {
		return false
	}
	
	return true
}

// checkDomain 检查域名
func (sc *ScopeController) checkDomain(host string) bool {
	// 缓存检查
	if result, exists := sc.domainCache[host]; exists {
		return result
	}
	
	// 移除端口号
	hostWithoutPort := host
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		hostWithoutPort = parts[0]
	}
	
	// 1. 检查排除列表
	for _, excludeDomain := range sc.config.ExcludeDomains {
		if sc.matchDomain(hostWithoutPort, excludeDomain) {
			sc.domainCache[host] = false
			return false
		}
	}
	
	// 2. 如果有包含列表，必须匹配
	if len(sc.config.IncludeDomains) > 0 {
		matched := false
		for _, includeDomain := range sc.config.IncludeDomains {
			if sc.matchDomain(hostWithoutPort, includeDomain) {
				matched = true
				break
			}
		}
		
		sc.domainCache[host] = matched
		return matched
	}
	
	// 3. 没有包含列表，默认允许
	sc.domainCache[host] = true
	return true
}

// matchDomain 匹配域名（支持通配符）
func (sc *ScopeController) matchDomain(host, pattern string) bool {
	// 完全匹配
	if host == pattern {
		return true
	}
	
	// 通配符匹配: *.example.com
	if strings.HasPrefix(pattern, "*.") {
		baseDomain := pattern[2:]
		if host == baseDomain {
			return true
		}
		if strings.HasSuffix(host, "."+baseDomain) {
			return true
		}
	}
	
	// 子域名匹配
	if sc.config.AllowSubdomains {
		if strings.HasSuffix(host, "."+pattern) {
			return true
		}
	}
	
	return false
}

// checkPath 检查路径
func (sc *ScopeController) checkPath(path string) bool {
	// 缓存检查
	if result, exists := sc.pathCache[path]; exists {
		return result
	}
	
	// 1. 检查排除路径
	for _, excludePath := range sc.config.ExcludePaths {
		if sc.matchPath(path, excludePath) {
			sc.pathCache[path] = false
			return false
		}
	}
	
	// 2. 如果有包含路径，必须匹配
	if len(sc.config.IncludePaths) > 0 {
		matched := false
		for _, includePath := range sc.config.IncludePaths {
			if sc.matchPath(path, includePath) {
				matched = true
				break
			}
		}
		
		sc.pathCache[path] = matched
		return matched
	}
	
	// 3. 没有包含列表，默认允许
	sc.pathCache[path] = true
	return true
}

// matchPath 匹配路径（支持通配符）
func (sc *ScopeController) matchPath(path, pattern string) bool {
	// 精确匹配
	if path == pattern {
		return true
	}
	
	// 前缀匹配: /api/*
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	
	// 后缀匹配: *.php
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	
	return false
}

// checkExtension 检查文件扩展名
func (sc *ScopeController) checkExtension(path string) bool {
	// 提取扩展名
	ext := ""
	if idx := strings.LastIndex(path, "."); idx != -1 {
		ext = strings.ToLower(path[idx+1:])
	}
	
	// 没有扩展名
	if ext == "" {
		return true
	}
	
	// 🔧 v3.1: exclude_extensions不再阻止URL进入作用域
	// URL会被记录，但在请求前判断是否需要访问
	// JS/CSS文件始终需要访问（可能包含隐藏URL和敏感信息）
	
	// 1. 检查排除列表（但不返回false，只做标记）
	// 注释掉原来的逻辑
	// for _, excludeExt := range sc.config.ExcludeExtensions {
	// 	if ext == strings.ToLower(excludeExt) {
	// 		return false
	// 	}
	// }
	
	// 2. 如果有包含列表，必须匹配
	if len(sc.config.IncludeExtensions) > 0 {
		for _, includeExt := range sc.config.IncludeExtensions {
			if ext == strings.ToLower(includeExt) {
				return true
			}
		}
		return false
	}
	
	return true
}

// ShouldRequestURL 判断URL是否需要发起HTTP请求
// 返回值: (是否请求, 原因)
func (sc *ScopeController) ShouldRequestURL(urlStr string) (bool, string) {
	// 解析URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false, "URL解析失败"
	}
	
	path := parsedURL.Path
	
	// 提取扩展名
	ext := ""
	if idx := strings.LastIndex(path, "."); idx != -1 {
		ext = strings.ToLower(path[idx+1:])
	}
	
	// 没有扩展名，需要请求
	if ext == "" {
		return true, "无扩展名"
	}
	
	// ✅ 修复6: JS文件始终需要请求（可能包含隐藏URL、API端点、敏感信息）
	jsExtensions := []string{"js", "jsx", "mjs", "ts", "tsx"}
	for _, jsExt := range jsExtensions {
		if ext == jsExt {
			return true, "JS文件需要访问和分析"
		}
	}
	
	// ✅ 修复7: 静态资源（图片、CSS、字体等）只记录不请求
	staticExtensions := []string{
		"css", "scss", "sass", // 样式文件
		"jpg", "jpeg", "png", "gif", "svg", "ico", "webp", "bmp", // 图片
		"woff", "woff2", "ttf", "eot", "otf", // 字体
		"mp4", "mp3", "avi", "mov", "wmv", "flv", "webm", "ogg", "wav", // 音视频
		"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", // 文档
		"zip", "rar", "tar", "gz", "7z", // 压缩包
	}
	
	for _, staticExt := range staticExtensions {
		if ext == staticExt {
			return false, fmt.Sprintf("静态资源（%s）已记录,不请求", ext)
		}
	}
	
	// 检查是否在配置的排除列表中（但JS已特殊处理）
	for _, excludeExt := range sc.config.ExcludeExtensions {
		excludeExtLower := strings.ToLower(excludeExt)
		// 跳过JS（已在上方特殊处理）
		if excludeExtLower == "js" || excludeExtLower == "jsx" || excludeExtLower == "mjs" || 
		   excludeExtLower == "ts" || excludeExtLower == "tsx" {
			continue
		}
		if ext == excludeExtLower {
			return false, fmt.Sprintf("排除扩展名（%s）已记录,不请求", ext)
		}
	}
	
	// 默认需要请求
	return true, "默认需要请求"
}

// checkParams 检查URL参数
func (sc *ScopeController) checkParams(params url.Values) bool {
	// 如果没有参数限制，直接通过
	if len(sc.config.IncludeParams) == 0 && len(sc.config.ExcludeParams) == 0 {
		return true
	}
	
	// 如果URL没有参数
	if len(params) == 0 {
		return len(sc.config.IncludeParams) == 0
	}
	
	// 1. 检查排除参数
	for paramName := range params {
		for _, excludeParam := range sc.config.ExcludeParams {
			if paramName == excludeParam {
				return false
			}
		}
	}
	
	// 2. 如果有包含参数，必须至少匹配一个
	if len(sc.config.IncludeParams) > 0 {
		for paramName := range params {
			for _, includeParam := range sc.config.IncludeParams {
				if paramName == includeParam {
					return true
				}
			}
		}
		return false
	}
	
	return true
}

// checkRegex 检查正则表达式
func (sc *ScopeController) checkRegex(rawURL string) bool {
	// 1. 排除正则
	if sc.excludeRegex != nil {
		if sc.excludeRegex.MatchString(rawURL) {
			return false
		}
	}
	
	// 2. 包含正则
	if sc.includeRegex != nil {
		return sc.includeRegex.MatchString(rawURL)
	}
	
	return true
}

// FilterURLs 批量过滤URL
func (sc *ScopeController) FilterURLs(urls []string) []string {
	filtered := make([]string, 0)
	
	for _, url := range urls {
		if sc.IsInScope(url) {
			filtered = append(filtered, url)
		}
	}
	
	return filtered
}

// GetStatistics 获取统计信息
func (sc *ScopeController) GetStatistics() ScopeStatistics {
	return ScopeStatistics{
		DomainRules:    len(sc.config.IncludeDomains) + len(sc.config.ExcludeDomains),
		PathRules:      len(sc.config.IncludePaths) + len(sc.config.ExcludePaths),
		ExtensionRules: len(sc.config.IncludeExtensions) + len(sc.config.ExcludeExtensions),
		ParamRules:     len(sc.config.IncludeParams) + len(sc.config.ExcludeParams),
		HasRegexRules:  sc.includeRegex != nil || sc.excludeRegex != nil,
	}
}

// ScopeStatistics Scope统计信息
type ScopeStatistics struct {
	DomainRules    int
	PathRules      int
	ExtensionRules int
	ParamRules     int
	HasRegexRules  bool
}

// PrintConfiguration 打印配置
func (sc *ScopeController) PrintConfiguration() {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("          Scope配置")
	fmt.Println(strings.Repeat("=", 70))
	
	if len(sc.config.IncludeDomains) > 0 {
		fmt.Println("包含域名:")
		for _, domain := range sc.config.IncludeDomains {
			fmt.Printf("  ✓ %s\n", domain)
		}
	}
	
	if len(sc.config.ExcludeDomains) > 0 {
		fmt.Println("排除域名:")
		for _, domain := range sc.config.ExcludeDomains {
			fmt.Printf("  ✗ %s\n", domain)
		}
	}
	
	if len(sc.config.IncludePaths) > 0 {
		fmt.Println("包含路径:")
		for _, path := range sc.config.IncludePaths {
			fmt.Printf("  ✓ %s\n", path)
		}
	}
	
	if len(sc.config.ExcludePaths) > 0 {
		fmt.Println("排除路径:")
		for _, path := range sc.config.ExcludePaths {
			fmt.Printf("  ✗ %s\n", path)
		}
	}
	
	if sc.config.IncludeRegex != "" {
		fmt.Printf("包含正则: %s\n", sc.config.IncludeRegex)
	}
	
	if sc.config.ExcludeRegex != "" {
		fmt.Printf("排除正则: %s\n", sc.config.ExcludeRegex)
	}
	
	fmt.Printf("允许子域名: %v\n", sc.config.AllowSubdomains)
	fmt.Printf("限制同域名: %v\n", sc.config.StayInDomain)
	
	fmt.Println(strings.Repeat("=", 70))
}

