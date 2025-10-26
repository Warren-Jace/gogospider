package core

import (
	"net/url"
	"regexp"
	"strings"
)

// URLValidator URL验证器 - 过滤无效和垃圾URL
type URLValidator struct {
	// MIME类型列表
	mimeTypes map[string]bool
	// JavaScript关键字和常见变量名
	jsKeywords map[string]bool
	// 编译后的正则表达式
	encodedJSPattern      *regexp.Regexp
	htmlTagPattern        *regexp.Regexp
	tooShortPathPattern   *regexp.Regexp
	onlyNumbersPattern    *regexp.Regexp
	specialCharsPattern   *regexp.Regexp
}

// NewURLValidator 创建URL验证器
func NewURLValidator() *URLValidator {
	v := &URLValidator{
		mimeTypes:  make(map[string]bool),
		jsKeywords: make(map[string]bool),
	}
	
	// 初始化MIME类型列表
	v.initMIMETypes()
	
	// 初始化JavaScript关键字
	v.initJSKeywords()
	
	// 编译正则表达式
	v.encodedJSPattern = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)      // URL编码
	v.htmlTagPattern = regexp.MustCompile(`</?\w+>`)                 // HTML标签
	v.tooShortPathPattern = regexp.MustCompile(`^/[a-zA-Z]{1,2}$`)  // 太短的路径 /a /ab
	v.onlyNumbersPattern = regexp.MustCompile(`^/\d+$`)             // 纯数字路径 /123
	v.specialCharsPattern = regexp.MustCompile(`[{}\[\]<>()%]`)     // 特殊字符
	
	return v
}

// initMIMETypes 初始化MIME类型列表
func (v *URLValidator) initMIMETypes() {
	// 常见MIME类型前缀
	mimePrefix := []string{
		"application/",
		"text/",
		"image/",
		"video/",
		"audio/",
		"font/",
		"multipart/",
	}
	
	for _, prefix := range mimePrefix {
		v.mimeTypes[prefix] = true
	}
	
	// 特定MIME类型
	specificTypes := []string{
		"vnd.ms-excel",
		"vnd.ms-office",
		"vnd.openxmlformats",
		"json",
		"xml",
		"html",
		"plain",
		"javascript",
		"x-www-form-urlencoded",
	}
	
	for _, t := range specificTypes {
		v.mimeTypes[t] = true
	}
}

// initJSKeywords 初始化JavaScript关键字
func (v *URLValidator) initJSKeywords() {
	keywords := []string{
		// JavaScript关键字
		"function", "var", "let", "const", "if", "else", "for", "while",
		"return", "break", "continue", "switch", "case", "default",
		"try", "catch", "finally", "throw", "new", "this", "typeof",
		"instanceof", "void", "delete", "in", "with",
		
		// 常见JavaScript对象和方法
		"Math", "Date", "Array", "Object", "String", "Number", "Boolean",
		"RegExp", "Error", "JSON", "Promise", "Symbol",
		"console", "window", "document", "navigator", "location",
		"each", "map", "filter", "reduce", "forEach", "some", "every",
		"find", "findIndex", "includes", "indexOf", "slice", "splice",
		"push", "pop", "shift", "unshift", "concat", "join",
		"match", "replace", "search", "split", "substring", "trim",
		
		// 常见库和框架
		"jQuery", "React", "Vue", "Angular", "lodash", "axios",
		"CodeMirror", "TreeNode", "Workbook", "Book",
		
		// 常见变量名和函数名
		"data", "item", "index", "key", "value", "result", "response",
		"request", "options", "config", "params", "args", "props",
		"state", "context", "callback", "handler", "listener",
		"true", "false", "null", "undefined", "NaN", "Infinity",
		
		// HTML/CSS相关
		"div", "span", "a", "b", "i", "p", "h", "ul", "li", "table",
		"tr", "td", "th", "form", "input", "button", "select", "option",
		"block", "inline", "none", "flex", "grid", "absolute", "relative",
		"fixed", "static", "sticky",
		
		// 协议和编码
		"http", "https", "ftp", "mailto", "tel", "javascript",
		"data", "blob", "base64",
		
		// 方法名模式
		"get", "set", "add", "remove", "del", "update", "create",
		"init", "load", "save", "open", "close", "start", "stop",
		"show", "hide", "toggle", "enable", "disable",
		
		// 路径相关
		"path", "route", "url", "href", "src", "link",
		
		// 其他常见单词
		"can", "has", "is", "will", "should", "must",
		"and", "or", "not", "but", "with", "from", "to",
	}
	
	for _, k := range keywords {
		v.jsKeywords[strings.ToLower(k)] = true
	}
}

// IsValidBusinessURL 判断是否为有效的业务URL
func (v *URLValidator) IsValidBusinessURL(rawURL string) bool {
	// 1. 基本格式检查
	if rawURL == "" || rawURL == "/" {
		return false
	}
	
	// 2. 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	
	path := parsedURL.Path
	if path == "" {
		path = "/"
	}
	
	// 3. 检查是否包含URL编码的JavaScript代码
	// 如果URL编码字符超过30%，很可能是编码的代码
	encodedCount := v.encodedJSPattern.FindAllString(rawURL, -1)
	if len(encodedCount) > len(rawURL)/10 { // 超过10%是编码字符
		return false
	}
	
	// 4. 检查是否包含HTML标签
	if v.htmlTagPattern.MatchString(rawURL) {
		return false
	}
	
	// 5. 检查路径是否为MIME类型
	if v.isMIMEType(path) {
		return false
	}
	
	// 6. 检查是否为JavaScript关键字或常见变量名
	if v.isJSKeyword(path) {
		return false
	}
	
	// 7. 检查路径是否太短（单字符或双字符）
	if v.tooShortPathPattern.MatchString(path) {
		return false
	}
	
	// 8. 检查是否为纯数字路径（除非是常见的ID格式）
	if v.onlyNumbersPattern.MatchString(path) && len(path) < 4 {
		return false
	}
	
	// 9. 检查是否包含过多特殊字符（可能是代码片段）
	specialCount := len(v.specialCharsPattern.FindAllString(path, -1))
	if specialCount > 3 {
		return false
	}
	
	// 10. 检查路径长度
	if len(path) > 200 {
		return false
	}
	
	// 11. 检查是否包含明显的代码模式
	if v.containsCodePattern(rawURL) {
		return false
	}
	
	// 12. 检查是否为有意义的路径
	if !v.hasMeaningfulPath(path) {
		return false
	}
	
	return true
}

// isMIMEType 检查路径是否为MIME类型
func (v *URLValidator) isMIMEType(path string) bool {
	// 移除开头的/
	cleanPath := strings.TrimPrefix(path, "/")
	
	// 检查是否匹配MIME类型模式
	for prefix := range v.mimeTypes {
		if strings.HasPrefix(cleanPath, prefix) {
			return true
		}
		if strings.Contains(cleanPath, prefix) {
			return true
		}
	}
	
	// 检查是否包含MIME类型特征
	if strings.Contains(cleanPath, "vnd.") {
		return true
	}
	
	return false
}

// isJSKeyword 检查路径是否为JavaScript关键字
func (v *URLValidator) isJSKeyword(path string) bool {
	// 移除开头的/和结尾的/
	cleanPath := strings.Trim(path, "/")
	
	// 转换为小写进行检查
	cleanPath = strings.ToLower(cleanPath)
	
	// 检查完整路径
	if v.jsKeywords[cleanPath] {
		return true
	}
	
	// 检查路径的最后一段
	segments := strings.Split(cleanPath, "/")
	if len(segments) > 0 {
		lastSegment := segments[len(segments)-1]
		if v.jsKeywords[lastSegment] {
			return true
		}
	}
	
	return false
}

// containsCodePattern 检查是否包含代码模式
func (v *URLValidator) containsCodePattern(rawURL string) bool {
	codePatterns := []string{
		// JavaScript函数调用模式
		`function\s*\(`,
		`\)\s*{`,
		`=\s*function`,
		`=>`,
		
		// 变量赋值模式
		`var\s+\w+\s*=`,
		`let\s+\w+\s*=`,
		`const\s+\w+\s*=`,
		
		// 比较运算符
		`===`, `!==`, `==`, `!=`,
		
		// 注释模式
		`//\s*\w+`,
		`/\*`, `\*/`,
		
		// 其他代码特征
		`.concat\(`, `.replace\(`, `.slice\(`,
	}
	
	for _, pattern := range codePatterns {
		matched, _ := regexp.MatchString(pattern, rawURL)
		if matched {
			return true
		}
	}
	
	return false
}

// hasMeaningfulPath 检查是否有有意义的路径
func (v *URLValidator) hasMeaningfulPath(path string) bool {
	// 移除开头和结尾的/
	cleanPath := strings.Trim(path, "/")
	
	// 空路径认为是有效的（首页）
	if cleanPath == "" {
		return true
	}
	
	// 路径至少要有3个字符（除了特殊情况）
	if len(cleanPath) < 3 {
		// 允许一些常见的短路径
		commonShortPaths := map[string]bool{
			"ui": true, "id": true, "no": true,
			"en": true, "zh": true, "cn": true,
			"v1": true, "v2": true, "v3": true,
		}
		
		if !commonShortPaths[strings.ToLower(cleanPath)] {
			return false
		}
	}
	
	// 检查是否包含常见的业务路径关键词
	businessKeywords := []string{
		"api", "admin", "user", "login", "logout", "register",
		"account", "profile", "setting", "config", "management",
		"list", "detail", "edit", "create", "update", "delete",
		"search", "query", "export", "import", "download", "upload",
		"home", "index", "main", "dashboard", "workbench",
		"page", "view", "portal", "center",
	}
	
	pathLower := strings.ToLower(cleanPath)
	for _, keyword := range businessKeywords {
		if strings.Contains(pathLower, keyword) {
			return true
		}
	}
	
	// 检查是否包含文件扩展名（有意义的资源）
	meaningfulExts := []string{
		".php", ".asp", ".aspx", ".jsp", ".do", ".action",
		".html", ".htm", ".shtml",
		".json", ".xml",
	}
	
	for _, ext := range meaningfulExts {
		if strings.HasSuffix(pathLower, ext) {
			return true
		}
	}
	
	// 如果路径包含多个段，认为是有意义的
	segments := strings.Split(cleanPath, "/")
	if len(segments) >= 2 {
		return true
	}
	
	return false
}

// FilterURLs 批量过滤URL列表
func (v *URLValidator) FilterURLs(urls []string) []string {
	filtered := make([]string, 0, len(urls))
	
	for _, u := range urls {
		if v.IsValidBusinessURL(u) {
			filtered = append(filtered, u)
		}
	}
	
	return filtered
}

