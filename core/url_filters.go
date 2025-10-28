package core

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// ============================================================================
// 1. 基础格式验证过滤器（优先级：10）
// ============================================================================

// BasicFormatFilter 基础格式验证过滤器
// 过滤明显无效的URL：空URL、无效协议、格式错误等
type BasicFormatFilter struct {
	enabled bool
	mu      sync.RWMutex
	
	// 统计
	totalChecked  int64
	totalRejected int64
	
	// 编译的正则
	invalidSchemePattern *regexp.Regexp
}

func NewBasicFormatFilter() *BasicFormatFilter {
	return &BasicFormatFilter{
		enabled:              true,
		invalidSchemePattern: regexp.MustCompile(`^(javascript|data|blob|about|vbscript|file):`),
	}
}

func (f *BasicFormatFilter) Name() string { return "BasicFormat" }
func (f *BasicFormatFilter) Priority() int { return 10 }

func (f *BasicFormatFilter) Filter(rawURL string, ctx *FilterContext) FilterResult {
	f.mu.Lock()
	f.totalChecked++
	f.mu.Unlock()
	
	// 1. 空URL
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		f.mu.Lock()
		f.totalRejected++
		f.mu.Unlock()
		return FilterResult{
			Allowed: false,
			Action:  FilterReject,
			Reason:  "空URL",
		}
	}
	
	// 2. 无效协议
	if f.invalidSchemePattern.MatchString(strings.ToLower(trimmed)) {
		f.mu.Lock()
		f.totalRejected++
		f.mu.Unlock()
		return FilterResult{
			Allowed: false,
			Action:  FilterReject,
			Reason:  "无效的URL协议（javascript/data/blob等）",
		}
	}
	
	// 3. URL解析失败
	if ctx.ParsedURL == nil {
		_, err := url.Parse(rawURL)
		if err != nil {
			f.mu.Lock()
			f.totalRejected++
			f.mu.Unlock()
			return FilterResult{
				Allowed: false,
				Action:  FilterReject,
				Reason:  fmt.Sprintf("URL解析失败: %v", err),
			}
		}
	}
	
	// 4. 长度检查
	if len(rawURL) > 2048 {
		f.mu.Lock()
		f.totalRejected++
		f.mu.Unlock()
		return FilterResult{
			Allowed: false,
			Action:  FilterReject,
			Reason:  "URL过长（超过2048字符）",
		}
	}
	
	return FilterResult{
		Allowed: true,
		Action:  FilterAllow,
		Reason:  "基础格式检查通过",
	}
}

func (f *BasicFormatFilter) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return map[string]interface{}{
		"total_checked":  f.totalChecked,
		"total_rejected": f.totalRejected,
	}
}

func (f *BasicFormatFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.totalChecked = 0
	f.totalRejected = 0
}

func (f *BasicFormatFilter) SetEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = enabled
}

func (f *BasicFormatFilter) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.enabled
}

// ============================================================================
// 2. 黑名单过滤器（优先级：20）
// ============================================================================

// BlacklistFilter 黑名单过滤器
// 快速过滤已知的垃圾URL模式
type BlacklistFilter struct {
	enabled bool
	mu      sync.RWMutex
	
	// 统计
	totalChecked  int64
	totalRejected int64
	
	// 黑名单
	jsKeywords       map[string]bool
	cssProperties    map[string]bool
	forbiddenPatterns []*regexp.Regexp
}

func NewBlacklistFilter() *BlacklistFilter {
	f := &BlacklistFilter{
		enabled:       true,
		jsKeywords:    make(map[string]bool),
		cssProperties: make(map[string]bool),
	}
	
	// JavaScript关键字黑名单
	jsKeywords := []string{
		"function", "return", "var", "let", "const",
		"true", "false", "null", "undefined",
		"typeof", "instanceof", "arguments",
		"window", "document", "console",
	}
	for _, kw := range jsKeywords {
		f.jsKeywords[kw] = true
	}
	
	// CSS属性黑名单
	cssProps := []string{
		"margin", "padding", "border", "color",
		"width", "height", "display", "position",
		"rgba", "rgb", "flex", "grid",
	}
	for _, prop := range cssProps {
		f.cssProperties[prop] = true
	}
	
	// 禁止模式
	f.forbiddenPatterns = []*regexp.Regexp{
		regexp.MustCompile(`</?[a-zA-Z][^>]*>`),            // HTML标签
		regexp.MustCompile(`#[0-9A-Fa-f]{3,8}$`),          // 颜色值
		regexp.MustCompile(`^\d+$`),                        // 纯数字
		regexp.MustCompile(`^[#?&=\-_./:\\]+$`),           // 纯符号
		regexp.MustCompile(`function\s*\(`),                // JS函数
		regexp.MustCompile(`=>`),                           // 箭头函数
		regexp.MustCompile(`===|!==`),                      // JS运算符
	}
	
	return f
}

func (f *BlacklistFilter) Name() string { return "Blacklist" }
func (f *BlacklistFilter) Priority() int { return 20 }

func (f *BlacklistFilter) Filter(rawURL string, ctx *FilterContext) FilterResult {
	f.mu.Lock()
	f.totalChecked++
	f.mu.Unlock()
	
	lowerURL := strings.ToLower(strings.TrimSpace(rawURL))
	
	// 1. JavaScript关键字
	if f.jsKeywords[lowerURL] {
		f.mu.Lock()
		f.totalRejected++
		f.mu.Unlock()
		return FilterResult{
			Allowed: false,
			Action:  FilterReject,
			Reason:  "JavaScript关键字",
		}
	}
	
	// 2. CSS属性
	if f.cssProperties[lowerURL] {
		f.mu.Lock()
		f.totalRejected++
		f.mu.Unlock()
		return FilterResult{
			Allowed: false,
			Action:  FilterReject,
			Reason:  "CSS属性",
		}
	}
	
	// 3. 禁止模式
	for _, pattern := range f.forbiddenPatterns {
		if pattern.MatchString(rawURL) {
			f.mu.Lock()
			f.totalRejected++
			f.mu.Unlock()
			return FilterResult{
				Allowed: false,
				Action:  FilterReject,
				Reason:  "匹配禁止模式",
			}
		}
	}
	
	return FilterResult{
		Allowed: true,
		Action:  FilterAllow,
		Reason:  "黑名单检查通过",
	}
}

func (f *BlacklistFilter) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return map[string]interface{}{
		"total_checked":  f.totalChecked,
		"total_rejected": f.totalRejected,
	}
}

func (f *BlacklistFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.totalChecked = 0
	f.totalRejected = 0
}

func (f *BlacklistFilter) SetEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = enabled
}

func (f *BlacklistFilter) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.enabled
}

// ============================================================================
// 3. 域名作用域过滤器（优先级：30）
// ============================================================================

// ScopeFilter 域名作用域过滤器
// 检查URL是否在允许的作用域内
type ScopeFilter struct {
	enabled bool
	mu      sync.RWMutex
	
	// 统计
	totalChecked  int64
	totalRejected int64
	totalDegraded int64
	
	// 配置
	targetDomain    string
	allowSubdomains bool
	allowHTTP       bool
	allowHTTPS      bool
	
	// 外部链接处理策略
	externalLinkAction FilterAction // Allow/Reject/Degrade
}

// ScopeFilterConfig 作用域过滤器配置
type ScopeFilterConfig struct {
	TargetDomain       string
	AllowSubdomains    bool
	AllowHTTP          bool
	AllowHTTPS         bool
	ExternalLinkAction FilterAction // 外部链接的处理方式
}

func NewScopeFilter(config ScopeFilterConfig) *ScopeFilter {
	// 默认值
	if config.ExternalLinkAction == 0 {
		config.ExternalLinkAction = FilterDegrade // 外部链接默认降级（记录但不爬取）
	}
	
	return &ScopeFilter{
		enabled:            true,
		targetDomain:       config.TargetDomain,
		allowSubdomains:    config.AllowSubdomains,
		allowHTTP:          config.AllowHTTP,
		allowHTTPS:         config.AllowHTTPS,
		externalLinkAction: config.ExternalLinkAction,
	}
}

func (f *ScopeFilter) Name() string { return "Scope" }
func (f *ScopeFilter) Priority() int { return 30 }

func (f *ScopeFilter) Filter(rawURL string, ctx *FilterContext) FilterResult {
	f.mu.Lock()
	f.totalChecked++
	f.mu.Unlock()
	
	parsedURL := ctx.ParsedURL
	if parsedURL == nil {
		var err error
		parsedURL, err = url.Parse(rawURL)
		if err != nil {
			return FilterResult{
				Allowed: true,
				Action:  FilterAllow,
				Reason:  "URL解析失败，跳过作用域检查",
			}
		}
	}
	
	// 1. 协议检查
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme == "http" && !f.allowHTTP {
		f.mu.Lock()
		f.totalRejected++
		f.mu.Unlock()
		return FilterResult{
			Allowed: false,
			Action:  FilterReject,
			Reason:  "不允许HTTP协议",
		}
	}
	if scheme == "https" && !f.allowHTTPS {
		f.mu.Lock()
		f.totalRejected++
		f.mu.Unlock()
		return FilterResult{
			Allowed: false,
			Action:  FilterReject,
			Reason:  "不允许HTTPS协议",
		}
	}
	
	// 2. 域名检查
	urlHost := parsedURL.Hostname()
	if urlHost == "" {
		// 相对URL，认为是域内的
		return FilterResult{
			Allowed: true,
			Action:  FilterAllow,
			Reason:  "相对URL（域内）",
		}
	}
	
	// 完全匹配
	if urlHost == f.targetDomain {
		return FilterResult{
			Allowed: true,
			Action:  FilterAllow,
			Reason:  "目标域名",
			Score:   100,
		}
	}
	
	// 子域名匹配
	if f.allowSubdomains && strings.HasSuffix(urlHost, "."+f.targetDomain) {
		return FilterResult{
			Allowed: true,
			Action:  FilterAllow,
			Reason:  "子域名",
			Score:   90,
		}
	}
	
	// 外部域名
	f.mu.Lock()
	if f.externalLinkAction == FilterDegrade {
		f.totalDegraded++
	} else if f.externalLinkAction == FilterReject {
		f.totalRejected++
	}
	f.mu.Unlock()
	
	return FilterResult{
		Allowed: f.externalLinkAction != FilterReject,
		Action:  f.externalLinkAction,
		Reason:  fmt.Sprintf("外部域名: %s", urlHost),
		Score:   0,
	}
}

func (f *ScopeFilter) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return map[string]interface{}{
		"total_checked":  f.totalChecked,
		"total_rejected": f.totalRejected,
		"total_degraded": f.totalDegraded,
	}
}

func (f *ScopeFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.totalChecked = 0
	f.totalRejected = 0
	f.totalDegraded = 0
}

func (f *ScopeFilter) SetEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = enabled
}

func (f *ScopeFilter) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.enabled
}

// ============================================================================
// 4. URL类型分类过滤器（优先级：40）
// ============================================================================

// TypeClassifierFilter URL类型分类过滤器
// 识别URL类型（API/静态资源/RESTful/普通页面），并根据策略处理
type TypeClassifierFilter struct {
	enabled bool
	mu      sync.RWMutex
	
	// 统计
	totalChecked  int64
	totalRejected int64
	totalDegraded int64
	
	// 按类型统计
	typeStats map[string]int64
	
	// 配置
	staticResourceAction FilterAction // 静态资源处理方式（Degrade/Reject）
	jsFileAction         FilterAction // JS文件处理方式（Allow/Degrade）
	cssFileAction        FilterAction // CSS文件处理方式
	
	// 扩展名映射
	staticExtensions map[string]bool
	jsExtensions     map[string]bool
	cssExtensions    map[string]bool
}

// TypeClassifierConfig 类型分类器配置
type TypeClassifierConfig struct {
	StaticResourceAction FilterAction // 静态资源如何处理
	JSFileAction         FilterAction // JS文件如何处理
	CSSFileAction        FilterAction // CSS文件如何处理
}

func NewTypeClassifierFilter(config TypeClassifierConfig) *TypeClassifierFilter {
	// 默认策略
	if config.StaticResourceAction == 0 {
		config.StaticResourceAction = FilterDegrade // 静态资源降级（记录不爬取）
	}
	if config.JSFileAction == 0 {
		config.JSFileAction = FilterAllow // JS文件允许（需要分析）
	}
	if config.CSSFileAction == 0 {
		config.CSSFileAction = FilterDegrade // CSS文件降级
	}
	
	f := &TypeClassifierFilter{
		enabled:              true,
		typeStats:            make(map[string]int64),
		staticResourceAction: config.StaticResourceAction,
		jsFileAction:         config.JSFileAction,
		cssFileAction:        config.CSSFileAction,
		staticExtensions:     make(map[string]bool),
		jsExtensions:         make(map[string]bool),
		cssExtensions:        make(map[string]bool),
	}
	
	// 静态资源扩展名
	staticExts := []string{
		"jpg", "jpeg", "png", "gif", "svg", "ico", "webp", "bmp", // 图片
		"woff", "woff2", "ttf", "eot", "otf", // 字体
		"mp4", "mp3", "avi", "mov", "wmv", "flv", "webm", "ogg", "wav", // 音视频
		"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", // 文档
		"zip", "rar", "tar", "gz", "7z", // 压缩包
	}
	for _, ext := range staticExts {
		f.staticExtensions[ext] = true
	}
	
	// JS扩展名
	jsExts := []string{"js", "jsx", "mjs", "ts", "tsx"}
	for _, ext := range jsExts {
		f.jsExtensions[ext] = true
	}
	
	// CSS扩展名
	cssExts := []string{"css", "scss", "sass", "less"}
	for _, ext := range cssExts {
		f.cssExtensions[ext] = true
	}
	
	return f
}

func (f *TypeClassifierFilter) Name() string { return "TypeClassifier" }
func (f *TypeClassifierFilter) Priority() int { return 40 }

func (f *TypeClassifierFilter) Filter(rawURL string, ctx *FilterContext) FilterResult {
	f.mu.Lock()
	f.totalChecked++
	f.mu.Unlock()
	
	parsedURL := ctx.ParsedURL
	if parsedURL == nil {
		var err error
		parsedURL, err = url.Parse(rawURL)
		if err != nil {
			return FilterResult{
				Allowed: true,
				Action:  FilterAllow,
				Reason:  "URL解析失败，跳过类型检查",
			}
		}
	}
	
	// 提取扩展名
	path := parsedURL.Path
	ext := ""
	if idx := strings.LastIndex(path, "."); idx != -1 && idx < len(path)-1 {
		ext = strings.ToLower(path[idx+1:])
	}
	
	// 没有扩展名，认为是普通页面
	if ext == "" {
		f.recordType("page")
		return FilterResult{
			Allowed: true,
			Action:  FilterAllow,
			Reason:  "普通页面（无扩展名）",
			Score:   80,
			Metadata: map[string]interface{}{
				"url_type": "page",
			},
		}
	}
	
	// JS文件
	if f.jsExtensions[ext] {
		f.recordType("js")
		action := f.jsFileAction
		if action == FilterDegrade {
			f.mu.Lock()
			f.totalDegraded++
			f.mu.Unlock()
		}
		return FilterResult{
			Allowed: action != FilterReject,
			Action:  action,
			Reason:  "JavaScript文件",
			Score:   85,
			Metadata: map[string]interface{}{
				"url_type": "js",
			},
		}
	}
	
	// CSS文件
	if f.cssExtensions[ext] {
		f.recordType("css")
		action := f.cssFileAction
		if action == FilterDegrade {
			f.mu.Lock()
			f.totalDegraded++
			f.mu.Unlock()
		}
		return FilterResult{
			Allowed: action != FilterReject,
			Action:  action,
			Reason:  "CSS文件",
			Score:   50,
			Metadata: map[string]interface{}{
				"url_type": "css",
			},
		}
	}
	
	// 静态资源
	if f.staticExtensions[ext] {
		f.recordType("static")
		action := f.staticResourceAction
		if action == FilterDegrade {
			f.mu.Lock()
			f.totalDegraded++
			f.mu.Unlock()
		} else if action == FilterReject {
			f.mu.Lock()
			f.totalRejected++
			f.mu.Unlock()
		}
		return FilterResult{
			Allowed: action != FilterReject,
			Action:  action,
			Reason:  fmt.Sprintf("静态资源（.%s）", ext),
			Score:   20,
			Metadata: map[string]interface{}{
				"url_type":   "static",
				"static_ext": ext,
			},
		}
	}
	
	// 其他动态文件
	dynamicExts := map[string]bool{
		"php": true, "asp": true, "aspx": true, "jsp": true,
		"do": true, "action": true, "htm": true, "html": true,
	}
	if dynamicExts[ext] {
		f.recordType("dynamic")
		return FilterResult{
			Allowed: true,
			Action:  FilterAllow,
			Reason:  fmt.Sprintf("动态页面（.%s）", ext),
			Score:   90,
			Metadata: map[string]interface{}{
				"url_type": "dynamic",
			},
		}
	}
	
	// 未知类型，保守允许
	f.recordType("unknown")
	return FilterResult{
		Allowed: true,
		Action:  FilterAllow,
		Reason:  fmt.Sprintf("未知类型（.%s）", ext),
		Score:   70,
		Metadata: map[string]interface{}{
			"url_type": "unknown",
		},
	}
}

func (f *TypeClassifierFilter) recordType(urlType string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.typeStats[urlType]++
}

func (f *TypeClassifierFilter) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return map[string]interface{}{
		"total_checked":  f.totalChecked,
		"total_rejected": f.totalRejected,
		"total_degraded": f.totalDegraded,
		"type_stats":     f.typeStats,
	}
}

func (f *TypeClassifierFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.totalChecked = 0
	f.totalRejected = 0
	f.totalDegraded = 0
	f.typeStats = make(map[string]int64)
}

func (f *TypeClassifierFilter) SetEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = enabled
}

func (f *TypeClassifierFilter) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.enabled
}

// ============================================================================
// 5. 业务价值评估过滤器（优先级：50）
// ============================================================================

// BusinessValueFilter 业务价值评估过滤器
// 评估URL的业务价值，过滤低价值URL
type BusinessValueFilter struct {
	enabled bool
	mu      sync.RWMutex
	
	// 统计
	totalChecked    int64
	totalRejected   int64
	highValueCount  int64
	lowValueCount   int64
	
	// 配置
	minScore      float64 // 最低分数要求
	highThreshold float64 // 高价值阈值
}

func NewBusinessValueFilter(minScore, highThreshold float64) *BusinessValueFilter {
	if minScore == 0 {
		minScore = 30.0
	}
	if highThreshold == 0 {
		highThreshold = 70.0
	}
	
	return &BusinessValueFilter{
		enabled:       true,
		minScore:      minScore,
		highThreshold: highThreshold,
	}
}

func (f *BusinessValueFilter) Name() string { return "BusinessValue" }
func (f *BusinessValueFilter) Priority() int { return 50 }

func (f *BusinessValueFilter) Filter(rawURL string, ctx *FilterContext) FilterResult {
	f.mu.Lock()
	f.totalChecked++
	f.mu.Unlock()
	
	// 计算业务价值分数
	score := f.calculateScore(rawURL, ctx)
	
	// 高价值URL
	if score >= f.highThreshold {
		f.mu.Lock()
		f.highValueCount++
		f.mu.Unlock()
		return FilterResult{
			Allowed: true,
			Action:  FilterAllow,
			Reason:  fmt.Sprintf("高价值URL（分数: %.1f）", score),
			Score:   score,
		}
	}
	
	// 低价值URL
	if score < f.minScore {
		f.mu.Lock()
		f.totalRejected++
		f.lowValueCount++
		f.mu.Unlock()
		return FilterResult{
			Allowed: false,
			Action:  FilterReject,
			Reason:  fmt.Sprintf("低价值URL（分数: %.1f < %.1f）", score, f.minScore),
			Score:   score,
		}
	}
	
	// 中等价值URL
	return FilterResult{
		Allowed: true,
		Action:  FilterAllow,
		Reason:  fmt.Sprintf("中等价值URL（分数: %.1f）", score),
		Score:   score,
	}
}

// calculateScore 计算业务价值分数
func (f *BusinessValueFilter) calculateScore(rawURL string, ctx *FilterContext) float64 {
	score := 50.0 // 基础分数
	
	lowerURL := strings.ToLower(rawURL)
	
	// 高价值关键词
	highValueKeywords := map[string]float64{
		"admin":    20.0,
		"api":      15.0,
		"login":    15.0,
		"auth":     15.0,
		"user":     10.0,
		"account":  10.0,
		"payment":  20.0,
		"order":    15.0,
		"config":   15.0,
		"setting":  12.0,
		"upload":   15.0,
		"edit":     10.0,
		"create":   10.0,
		"delete":   12.0,
	}
	
	for keyword, bonus := range highValueKeywords {
		if strings.Contains(lowerURL, keyword) {
			score += bonus
		}
	}
	
	// 参数数量加分
	if strings.Contains(rawURL, "?") {
		paramCount := strings.Count(rawURL, "=")
		if paramCount == 1 {
			score += 5.0
		} else if paramCount >= 2 && paramCount <= 4 {
			score += 10.0
		}
	}
	
	// RESTful风格加分
	if regexp.MustCompile(`/\d+/?$`).MatchString(rawURL) {
		score += 10.0
	}
	
	// 低价值模式减分
	lowValuePatterns := []string{
		"track", "analytics", "beacon", "pixel",
		"ads", "advertisement",
	}
	for _, pattern := range lowValuePatterns {
		if strings.Contains(lowerURL, pattern) {
			score -= 15.0
		}
	}
	
	// 限制分数范围
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

func (f *BusinessValueFilter) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return map[string]interface{}{
		"total_checked":    f.totalChecked,
		"total_rejected":   f.totalRejected,
		"high_value_count": f.highValueCount,
		"low_value_count":  f.lowValueCount,
	}
}

func (f *BusinessValueFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.totalChecked = 0
	f.totalRejected = 0
	f.highValueCount = 0
	f.lowValueCount = 0
}

func (f *BusinessValueFilter) SetEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = enabled
}

func (f *BusinessValueFilter) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.enabled
}

