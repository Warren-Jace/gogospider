package core

import (
	"net/url"
	"regexp"
	"strings"
)

// URLTypeClassifier URL类型分类器
type URLTypeClassifier struct {
	// 正则表达式模式
	restfulPattern     *regexp.Regexp
	ajaxPattern        *regexp.Regexp
	fileParamPattern   *regexp.Regexp
	staticAssetPattern *regexp.Regexp
	
	// 文件参数关键字
	fileParamKeys []string
}

// NewURLTypeClassifier 创建URL类型分类器
func NewURLTypeClassifier() *URLTypeClassifier {
	return &URLTypeClassifier{
		// RESTful模式：路径中包含数字ID或UUID
		// 例如: /user/123/profile, /api/products/abc-def-123
		restfulPattern: regexp.MustCompile(`/[a-zA-Z_-]+/[0-9a-zA-Z_-]+/[a-zA-Z_-]+/?`),
		
		// AJAX/API模式
		// 例如: /ajax/, /api/, /v1/, /graphql, *.json, *.xml
		ajaxPattern: regexp.MustCompile(`(?i)/(ajax|api|v\d+|graphql|rest)/|\.json|\.xml|/rpc/`),
		
		// 文件参数模式
		// 例如: ?file=, ?path=, ?document=
		fileParamPattern: regexp.MustCompile(`(?i)[?&](file|path|document|doc|image|img|attachment|download)=`),
		
		// 静态资源模式
		staticAssetPattern: regexp.MustCompile(`\.(jpg|jpeg|png|gif|bmp|svg|webp|ico|css|js|woff|woff2|ttf|eot|mp4|mp3|avi|pdf|zip|rar|swf)$`),
		
		// 文件参数关键字
		fileParamKeys: []string{
			"file", "path", "document", "doc", "image", "img",
			"attachment", "download", "filename", "filepath",
		},
	}
}

// ClassifyURL 分类URL类型
func (c *URLTypeClassifier) ClassifyURL(rawURL string) URLType {
	// 优先级从高到低检测
	
	// 1. 检测静态资源（最高优先级，避免误判）
	if c.isStaticAsset(rawURL) {
		return URLTypeStaticAsset
	}
	
	// 2. 检测AJAX/API接口（第二优先级）
	if c.isAJAXAPI(rawURL) {
		return URLTypeAJAX
	}
	
	// 3. 检测文件参数URL（第三优先级）
	if c.isFileParamURL(rawURL) {
		return URLTypeFileParam
	}
	
	// 4. 检测RESTful路径（第四优先级）
	if c.isRESTfulURL(rawURL) {
		return URLTypeRESTful
	}
	
	// 5. 检测多参数URL
	if c.isMultiParamURL(rawURL) {
		return URLTypeMultiParam
	}
	
	// 6. 默认为普通URL
	return URLTypeNormal
}

// isStaticAsset 判断是否为静态资源
func (c *URLTypeClassifier) isStaticAsset(rawURL string) bool {
	return c.staticAssetPattern.MatchString(strings.ToLower(rawURL))
}

// isAJAXAPI 判断是否为AJAX/API接口
func (c *URLTypeClassifier) isAJAXAPI(rawURL string) bool {
	// 检查路径模式
	if c.ajaxPattern.MatchString(rawURL) {
		return true
	}
	
	// 检查特殊路径
	lowerURL := strings.ToLower(rawURL)
	ajaxKeywords := []string{
		"/ajax/", "/api/", "/rest/", "/graphql",
		"/v1/", "/v2/", "/v3/",
		".json", ".xml", ".api",
	}
	
	for _, keyword := range ajaxKeywords {
		if strings.Contains(lowerURL, keyword) {
			return true
		}
	}
	
	return false
}

// isFileParamURL 判断是否包含文件参数
func (c *URLTypeClassifier) isFileParamURL(rawURL string) bool {
	// 使用正则检测
	if c.fileParamPattern.MatchString(rawURL) {
		return true
	}
	
	// 解析URL参数
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	
	query := parsedURL.Query()
	for paramName := range query {
		paramLower := strings.ToLower(paramName)
		for _, fileKey := range c.fileParamKeys {
			if strings.Contains(paramLower, fileKey) {
				return true
			}
		}
	}
	
	return false
}

// isRESTfulURL 判断是否为RESTful风格URL
func (c *URLTypeClassifier) isRESTfulURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	
	path := parsedURL.Path
	
	// RESTful特征1：正则匹配
	if c.restfulPattern.MatchString(path) {
		return true
	}
	
	// RESTful特征2：路径分段分析
	// 例如: /Mod_Rewrite_Shop/BuyProduct-1/
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) >= 2 {
		// 检查是否包含资源名+ID的模式
		for i := 0; i < len(segments)-1; i++ {
			// 资源名（字母）+ ID（数字或字母数字混合）
			if c.isResourceName(segments[i]) && c.isResourceID(segments[i+1]) {
				return true
			}
		}
	}
	
	// RESTful特征3：路径中包含连字符的ID
	// 例如: /BuyProduct-1/, /Details/network-attached-storage-dlink/1/
	for _, segment := range segments {
		if strings.Contains(segment, "-") && c.containsNumberOrUUID(segment) {
			return true
		}
	}
	
	return false
}

// isMultiParamURL 判断是否为多参数URL
func (c *URLTypeClassifier) isMultiParamURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	
	query := parsedURL.Query()
	return len(query) >= 2
}

// ===== 辅助方法 =====

// isResourceName 判断是否为资源名称
func (c *URLTypeClassifier) isResourceName(segment string) bool {
	// 资源名通常是字母开头，可能包含下划线
	if len(segment) < 2 {
		return false
	}
	
	// 首字母必须是字母
	first := segment[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')) {
		return false
	}
	
	// 主要由字母和下划线组成
	alphaCount := 0
	for _, ch := range segment {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			alphaCount++
		}
	}
	
	return float64(alphaCount)/float64(len(segment)) > 0.6
}

// isResourceID 判断是否为资源ID
func (c *URLTypeClassifier) isResourceID(segment string) bool {
	if len(segment) == 0 {
		return false
	}
	
	// 纯数字
	if c.isNumeric(segment) {
		return true
	}
	
	// UUID格式
	if c.isUUID(segment) {
		return true
	}
	
	// 包含数字的混合格式（如: abc123, user-123）
	if c.containsNumberOrUUID(segment) {
		return true
	}
	
	return false
}

// isNumeric 判断是否为纯数字
func (c *URLTypeClassifier) isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// isUUID 判断是否为UUID格式
func (c *URLTypeClassifier) isUUID(s string) bool {
	// UUID格式: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	uuidPattern := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidPattern.MatchString(s)
}

// containsNumberOrUUID 判断是否包含数字或UUID
func (c *URLTypeClassifier) containsNumberOrUUID(s string) bool {
	// 检查是否包含数字
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			return true
		}
	}
	
	// 检查UUID格式的片段
	if strings.Contains(s, "-") && len(s) > 8 {
		parts := strings.Split(s, "-")
		for _, part := range parts {
			if c.isNumeric(part) || c.isHexString(part) {
				return true
			}
		}
	}
	
	return false
}

// isHexString 判断是否为十六进制字符串
func (c *URLTypeClassifier) isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

// GetURLTypeString 获取URL类型的字符串表示
func GetURLTypeString(urlType URLType) string {
	switch urlType {
	case URLTypeRESTful:
		return "RESTful"
	case URLTypeAJAX:
		return "AJAX/API"
	case URLTypeFileParam:
		return "FileParam"
	case URLTypeMultiParam:
		return "MultiParam"
	case URLTypeStaticAsset:
		return "StaticAsset"
	default:
		return "Normal"
	}
}

