package core

import (
	"net/url"
	"regexp"
	"strings"
)

// ScopeMode 作用域模式
type ScopeMode string

const (
	ScopeDomain       ScopeMode = "dn"   // 域名模式（example.com）
	ScopeFQDN         ScopeMode = "fqdn" // 完全限定域名（www.example.com）
	ScopeRDN          ScopeMode = "rdn"  // 根域名（包含子域名）
	ScopeCustom       ScopeMode = "custom" // 自定义模式
)

// AdvancedScope 高级作用域控制器
type AdvancedScope struct {
	// 基础配置
	mode           ScopeMode
	targetDomain   string
	allowedDomains []string
	
	// 正则表达式作用域
	includeRegexes []*regexp.Regexp
	excludeRegexes []*regexp.Regexp
	
	// 路径过滤
	includePaths []string
	excludePaths []string
	
	// 扩展名过滤
	includeExtensions []string
	excludeExtensions []string
	
	// 参数过滤
	excludeParams []string
	
	// 高级选项
	allowQueryStrings bool
	allowFragments    bool
	maxPathDepth      int
	
	// 统计
	checkedCount  int
	allowedCount  int
	blockedCount  int
}

// NewAdvancedScope 创建高级作用域控制器
func NewAdvancedScope(targetDomain string) *AdvancedScope {
	return &AdvancedScope{
		mode:              ScopeRDN, // 默认根域名模式
		targetDomain:      targetDomain,
		allowedDomains:    []string{targetDomain},
		includeRegexes:    make([]*regexp.Regexp, 0),
		excludeRegexes:    make([]*regexp.Regexp, 0),
		includePaths:      make([]string, 0),
		excludePaths:      make([]string, 0),
		includeExtensions: make([]string, 0),
		excludeExtensions: make([]string, 0),
		excludeParams:     make([]string, 0),
		allowQueryStrings: true,
		allowFragments:    true,
		maxPathDepth:      0, // 0表示无限制
	}
}

// SetMode 设置作用域模式
func (as *AdvancedScope) SetMode(mode ScopeMode) {
	as.mode = mode
}

// AddAllowedDomain 添加允许的域名
func (as *AdvancedScope) AddAllowedDomain(domain string) {
	as.allowedDomains = append(as.allowedDomains, domain)
}

// AddIncludeRegex 添加包含正则表达式
func (as *AdvancedScope) AddIncludeRegex(pattern string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	as.includeRegexes = append(as.includeRegexes, regex)
	return nil
}

// AddExcludeRegex 添加排除正则表达式
func (as *AdvancedScope) AddExcludeRegex(pattern string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	as.excludeRegexes = append(as.excludeRegexes, regex)
	return nil
}

// AddIncludePath 添加包含路径
func (as *AdvancedScope) AddIncludePath(path string) {
	as.includePaths = append(as.includePaths, path)
}

// AddExcludePath 添加排除路径
func (as *AdvancedScope) AddExcludePath(path string) {
	as.excludePaths = append(as.excludePaths, path)
}

// AddIncludeExtension 添加包含的扩展名
func (as *AdvancedScope) AddIncludeExtension(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	as.includeExtensions = append(as.includeExtensions, ext)
}

// AddExcludeExtension 添加排除的扩展名
func (as *AdvancedScope) AddExcludeExtension(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	as.excludeExtensions = append(as.excludeExtensions, ext)
}

// AddExcludeParam 添加排除的参数
func (as *AdvancedScope) AddExcludeParam(param string) {
	as.excludeParams = append(as.excludeParams, param)
}

// SetAllowQueryStrings 设置是否允许查询字符串
func (as *AdvancedScope) SetAllowQueryStrings(allow bool) {
	as.allowQueryStrings = allow
}

// SetAllowFragments 设置是否允许URL片段
func (as *AdvancedScope) SetAllowFragments(allow bool) {
	as.allowFragments = allow
}

// SetMaxPathDepth 设置最大路径深度
func (as *AdvancedScope) SetMaxPathDepth(depth int) {
	as.maxPathDepth = depth
}

// InScope 判断URL是否在作用域内
func (as *AdvancedScope) InScope(rawURL string) (bool, string) {
	as.checkedCount++
	
	// 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		as.blockedCount++
		return false, "URL解析失败"
	}
	
	// 1. 检查域名
	if !as.checkDomain(parsedURL.Host) {
		as.blockedCount++
		return false, "域名不在范围内"
	}
	
	// 2. 检查路径深度
	if !as.checkPathDepth(parsedURL.Path) {
		as.blockedCount++
		return false, "路径深度超限"
	}
	
	// 3. 检查扩展名
	if !as.checkExtension(parsedURL.Path) {
		as.blockedCount++
		return false, "扩展名被过滤"
	}
	
	// 4. 检查包含路径
	if len(as.includePaths) > 0 && !as.checkIncludePath(parsedURL.Path) {
		as.blockedCount++
		return false, "路径不在包含列表"
	}
	
	// 5. 检查排除路径
	if as.checkExcludePath(parsedURL.Path) {
		as.blockedCount++
		return false, "路径在排除列表"
	}
	
	// 6. 检查包含正则
	if len(as.includeRegexes) > 0 && !as.checkIncludeRegex(rawURL) {
		as.blockedCount++
		return false, "不匹配包含正则"
	}
	
	// 7. 检查排除正则
	if as.checkExcludeRegex(rawURL) {
		as.blockedCount++
		return false, "匹配排除正则"
	}
	
	// 8. 检查查询字符串
	if !as.allowQueryStrings && parsedURL.RawQuery != "" {
		as.blockedCount++
		return false, "不允许查询字符串"
	}
	
	// 9. 检查URL片段
	if !as.allowFragments && parsedURL.Fragment != "" {
		as.blockedCount++
		return false, "不允许URL片段"
	}
	
	// 10. 检查参数过滤
	if as.checkExcludeParams(parsedURL) {
		as.blockedCount++
		return false, "包含被排除的参数"
	}
	
	as.allowedCount++
	return true, "通过所有检查"
}

// checkDomain 检查域名
func (as *AdvancedScope) checkDomain(host string) bool {
	if host == "" {
		return true // 相对URL
	}
	
	// 去除端口号
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	
	switch as.mode {
	case ScopeDomain:
		// 精确匹配域名
		for _, allowed := range as.allowedDomains {
			if host == allowed {
				return true
			}
		}
		
	case ScopeFQDN:
		// 完全限定域名匹配
		for _, allowed := range as.allowedDomains {
			if host == allowed {
				return true
			}
		}
		
	case ScopeRDN:
		// 根域名匹配（包含子域名）
		for _, allowed := range as.allowedDomains {
			if host == allowed || strings.HasSuffix(host, "."+allowed) {
				return true
			}
		}
		
	case ScopeCustom:
		// 自定义模式（由正则表达式控制）
		return true
	}
	
	return false
}

// checkPathDepth 检查路径深度
func (as *AdvancedScope) checkPathDepth(path string) bool {
	if as.maxPathDepth == 0 {
		return true
	}
	
	// 计算路径深度
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	// 过滤空部分
	actualDepth := 0
	for _, part := range parts {
		if part != "" {
			actualDepth++
		}
	}
	
	return actualDepth <= as.maxPathDepth
}

// checkExtension 检查扩展名
func (as *AdvancedScope) checkExtension(path string) bool {
	// 获取扩展名
	ext := ""
	if idx := strings.LastIndex(path, "."); idx != -1 {
		ext = path[idx:]
	}
	
	if ext == "" {
		return true // 无扩展名默认允许
	}
	
	ext = strings.ToLower(ext)
	
	// 检查排除扩展名
	for _, excludeExt := range as.excludeExtensions {
		if ext == strings.ToLower(excludeExt) {
			return false
		}
	}
	
	// 如果有包含扩展名列表，检查是否在列表中
	if len(as.includeExtensions) > 0 {
		for _, includeExt := range as.includeExtensions {
			if ext == strings.ToLower(includeExt) {
				return true
			}
		}
		return false // 不在包含列表中
	}
	
	return true
}

// checkIncludePath 检查包含路径
func (as *AdvancedScope) checkIncludePath(path string) bool {
	for _, includePath := range as.includePaths {
		if strings.HasPrefix(path, includePath) {
			return true
		}
	}
	return false
}

// checkExcludePath 检查排除路径
func (as *AdvancedScope) checkExcludePath(path string) bool {
	for _, excludePath := range as.excludePaths {
		if strings.Contains(path, excludePath) {
			return true
		}
	}
	return false
}

// checkIncludeRegex 检查包含正则
func (as *AdvancedScope) checkIncludeRegex(rawURL string) bool {
	for _, regex := range as.includeRegexes {
		if regex.MatchString(rawURL) {
			return true
		}
	}
	return false
}

// checkExcludeRegex 检查排除正则
func (as *AdvancedScope) checkExcludeRegex(rawURL string) bool {
	for _, regex := range as.excludeRegexes {
		if regex.MatchString(rawURL) {
			return true
		}
	}
	return false
}

// checkExcludeParams 检查排除的参数
func (as *AdvancedScope) checkExcludeParams(parsedURL *url.URL) bool {
	if len(as.excludeParams) == 0 {
		return false
	}
	
	query := parsedURL.Query()
	for param := range query {
		for _, excludeParam := range as.excludeParams {
			if strings.Contains(strings.ToLower(param), strings.ToLower(excludeParam)) {
				return true
			}
		}
	}
	
	return false
}

// GetStatistics 获取统计信息
func (as *AdvancedScope) GetStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["checked_count"] = as.checkedCount
	stats["allowed_count"] = as.allowedCount
	stats["blocked_count"] = as.blockedCount
	
	if as.checkedCount > 0 {
		stats["block_rate"] = float64(as.blockedCount) / float64(as.checkedCount) * 100
	} else {
		stats["block_rate"] = 0.0
	}
	
	stats["mode"] = string(as.mode)
	stats["allowed_domains"] = as.allowedDomains
	stats["include_regex_count"] = len(as.includeRegexes)
	stats["exclude_regex_count"] = len(as.excludeRegexes)
	stats["include_path_count"] = len(as.includePaths)
	stats["exclude_path_count"] = len(as.excludePaths)
	stats["exclude_extension_count"] = len(as.excludeExtensions)
	
	return stats
}

// Reset 重置统计
func (as *AdvancedScope) Reset() {
	as.checkedCount = 0
	as.allowedCount = 0
	as.blockedCount = 0
}

// Clone 克隆作用域控制器
func (as *AdvancedScope) Clone() *AdvancedScope {
	newScope := &AdvancedScope{
		mode:              as.mode,
		targetDomain:      as.targetDomain,
		allowedDomains:    make([]string, len(as.allowedDomains)),
		includeRegexes:    make([]*regexp.Regexp, len(as.includeRegexes)),
		excludeRegexes:    make([]*regexp.Regexp, len(as.excludeRegexes)),
		includePaths:      make([]string, len(as.includePaths)),
		excludePaths:      make([]string, len(as.excludePaths)),
		includeExtensions: make([]string, len(as.includeExtensions)),
		excludeExtensions: make([]string, len(as.excludeExtensions)),
		excludeParams:     make([]string, len(as.excludeParams)),
		allowQueryStrings: as.allowQueryStrings,
		allowFragments:    as.allowFragments,
		maxPathDepth:      as.maxPathDepth,
	}
	
	copy(newScope.allowedDomains, as.allowedDomains)
	copy(newScope.includeRegexes, as.includeRegexes)
	copy(newScope.excludeRegexes, as.excludeRegexes)
	copy(newScope.includePaths, as.includePaths)
	copy(newScope.excludePaths, as.excludePaths)
	copy(newScope.includeExtensions, as.includeExtensions)
	copy(newScope.excludeExtensions, as.excludeExtensions)
	copy(newScope.excludeParams, as.excludeParams)
	
	return newScope
}

// PresetAPIScope 预设：API测试作用域
func (as *AdvancedScope) PresetAPIScope() {
	as.AddIncludePath("/api/")
	as.AddIncludePath("/v1/")
	as.AddIncludePath("/v2/")
	as.AddExcludeExtension(".jpg")
	as.AddExcludeExtension(".png")
	as.AddExcludeExtension(".gif")
	as.AddExcludeExtension(".css")
	as.AddExcludeExtension(".js")
}

// PresetAdminScope 预设：管理后台作用域
func (as *AdvancedScope) PresetAdminScope() {
	as.AddIncludePath("/admin/")
	as.AddIncludePath("/manage/")
	as.AddIncludePath("/backend/")
	as.AddExcludePath("/logout")
	as.AddExcludePath("/signout")
}

// PresetStaticFilterScope 预设：过滤静态资源
func (as *AdvancedScope) PresetStaticFilterScope() {
	staticExts := []string{
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico", ".webp",
		".css", ".less", ".sass", ".scss",
		".woff", ".woff2", ".ttf", ".eot", ".otf",
		".mp4", ".mp3", ".avi", ".mov",
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
		".zip", ".rar", ".tar", ".gz",
	}
	
	for _, ext := range staticExts {
		as.AddExcludeExtension(ext)
	}
}

