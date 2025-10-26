package core

import (
	"net/url"
	"strings"
)

// ResourceType 资源类型
type ResourceType string

const (
	// 需要请求和分析的资源
	ResourceTypePage       ResourceType = "page"        // 页面（需要爬取）
	ResourceTypeJavaScript ResourceType = "javascript"  // JS文件（需要下载分析）
	ResourceTypeCSS        ResourceType = "css"         // CSS文件（需要下载分析）
	
	// 只收集不请求的静态资源
	ResourceTypeImage      ResourceType = "image"       // 图片
	ResourceTypeVideo      ResourceType = "video"       // 视频
	ResourceTypeAudio      ResourceType = "audio"       // 音频
	ResourceTypeFont       ResourceType = "font"        // 字体
	ResourceTypeDocument   ResourceType = "document"    // 文档（PDF/Word等）
	ResourceTypeArchive    ResourceType = "archive"     // 压缩包
	ResourceTypeOther      ResourceType = "other"       // 其他静态资源
	
	// 特殊类型
	ResourceTypeExternal   ResourceType = "external"    // 域外URL（只收集）
	ResourceTypeAPI        ResourceType = "api"         // API端点（需要测试）
)

// ResourceClassifier 资源分类器 - 区分需要请求和只收集的资源
type ResourceClassifier struct {
	targetDomain string // 目标域名
}

// NewResourceClassifier 创建资源分类器
func NewResourceClassifier(targetDomain string) *ResourceClassifier {
	return &ResourceClassifier{
		targetDomain: targetDomain,
	}
}

// ClassifyURL 分类URL，判断是否需要请求
func (r *ResourceClassifier) ClassifyURL(urlStr string) (ResourceType, bool) {
	// 解析URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ResourceTypeOther, false
	}
	
	// 检查是否为外部域名
	if parsedURL.Host != "" && !r.isSameDomain(parsedURL.Host) {
		return ResourceTypeExternal, false // 域外URL，不请求
	}
	
	// 获取路径和查询参数
	path := strings.ToLower(parsedURL.Path)
	
	// 1. JavaScript文件 - 需要下载和分析
	if r.isJavaScript(path) {
		return ResourceTypeJavaScript, true // 需要请求
	}
	
	// 2. CSS文件 - 需要下载和分析
	if r.isCSS(path) {
		return ResourceTypeCSS, true // 需要请求
	}
	
	// 3. API端点 - 需要测试
	if r.isAPI(path) {
		return ResourceTypeAPI, true // 需要请求
	}
	
	// 4. 图片 - 只收集
	if r.isImage(path) {
		return ResourceTypeImage, false // 不请求
	}
	
	// 5. 视频 - 只收集
	if r.isVideo(path) {
		return ResourceTypeVideo, false // 不请求
	}
	
	// 6. 音频 - 只收集
	if r.isAudio(path) {
		return ResourceTypeAudio, false // 不请求
	}
	
	// 7. 字体 - 只收集
	if r.isFont(path) {
		return ResourceTypeFont, false // 不请求
	}
	
	// 8. 文档 - 只收集
	if r.isDocument(path) {
		return ResourceTypeDocument, false // 不请求
	}
	
	// 9. 压缩包 - 只收集
	if r.isArchive(path) {
		return ResourceTypeArchive, false // 不请求
	}
	
	// 10. 其他静态资源 - 只收集
	if r.isOtherStatic(path) {
		return ResourceTypeOther, false // 不请求
	}
	
	// 默认：页面类型，需要爬取
	return ResourceTypePage, true
}

// isSameDomain 检查是否为同一域名
func (r *ResourceClassifier) isSameDomain(host string) bool {
	if host == "" {
		return true // 相对路径视为同域名
	}
	
	// 清理域名
	cleanTarget := strings.TrimPrefix(r.targetDomain, "http://")
	cleanTarget = strings.TrimPrefix(cleanTarget, "https://")
	cleanTarget = strings.Split(cleanTarget, ":")[0] // 去除端口
	
	cleanHost := strings.Split(host, ":")[0] // 去除端口
	
	// 精确匹配
	if cleanHost == cleanTarget {
		return true
	}
	
	// 检查子域名
	if strings.HasSuffix(cleanHost, "."+cleanTarget) {
		return true
	}
	
	return false
}

// isJavaScript 判断是否为JavaScript文件
func (r *ResourceClassifier) isJavaScript(path string) bool {
	jsExtensions := []string{".js", ".mjs", ".jsx"}
	for _, ext := range jsExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// isCSS 判断是否为CSS文件
func (r *ResourceClassifier) isCSS(path string) bool {
	cssExtensions := []string{".css", ".scss", ".sass", ".less"}
	for _, ext := range cssExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// isAPI 判断是否为API端点
func (r *ResourceClassifier) isAPI(path string) bool {
	apiKeywords := []string{"/api/", "/v1/", "/v2/", "/v3/", "/rest/", "/graphql", "/ajax/"}
	for _, keyword := range apiKeywords {
		if strings.Contains(path, keyword) {
			return true
		}
	}
	return false
}

// isImage 判断是否为图片
func (r *ResourceClassifier) isImage(path string) bool {
	imageExtensions := []string{
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", 
		".ico", ".bmp", ".tif", ".tiff", ".avif",
	}
	for _, ext := range imageExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// isVideo 判断是否为视频
func (r *ResourceClassifier) isVideo(path string) bool {
	videoExtensions := []string{
		".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv",
		".webm", ".m4v", ".mpg", ".mpeg", ".3gp",
	}
	for _, ext := range videoExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// isAudio 判断是否为音频
func (r *ResourceClassifier) isAudio(path string) bool {
	audioExtensions := []string{
		".mp3", ".wav", ".ogg", ".m4a", ".aac", ".flac",
		".wma", ".opus", ".aiff",
	}
	for _, ext := range audioExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// isFont 判断是否为字体
func (r *ResourceClassifier) isFont(path string) bool {
	fontExtensions := []string{
		".woff", ".woff2", ".ttf", ".eot", ".otf",
	}
	for _, ext := range fontExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// isDocument 判断是否为文档
func (r *ResourceClassifier) isDocument(path string) bool {
	docExtensions := []string{
		".pdf", ".doc", ".docx", ".xls", ".xlsx", 
		".ppt", ".pptx", ".txt", ".csv", ".rtf",
	}
	for _, ext := range docExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// isArchive 判断是否为压缩包
func (r *ResourceClassifier) isArchive(path string) bool {
	archiveExtensions := []string{
		".zip", ".rar", ".tar", ".gz", ".bz2", ".7z",
		".tgz", ".tar.gz", ".tar.bz2",
	}
	for _, ext := range archiveExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// isOtherStatic 判断是否为其他静态资源
func (r *ResourceClassifier) isOtherStatic(path string) bool {
	staticExtensions := []string{
		".swf", ".apk", ".dmg", ".exe", ".msi",
		".deb", ".rpm", ".iso",
	}
	for _, ext := range staticExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// ShouldCrawl 判断URL是否应该被爬取
func (r *ResourceClassifier) ShouldCrawl(urlStr string) bool {
	_, shouldRequest := r.ClassifyURL(urlStr)
	return shouldRequest
}

// GetResourceTypeString 获取资源类型的字符串描述
func (r *ResourceClassifier) GetResourceTypeString(resType ResourceType) string {
	descriptions := map[ResourceType]string{
		ResourceTypePage:       "页面",
		ResourceTypeJavaScript: "JavaScript",
		ResourceTypeCSS:        "CSS",
		ResourceTypeImage:      "图片",
		ResourceTypeVideo:      "视频",
		ResourceTypeAudio:      "音频",
		ResourceTypeFont:       "字体",
		ResourceTypeDocument:   "文档",
		ResourceTypeArchive:    "压缩包",
		ResourceTypeOther:      "其他静态资源",
		ResourceTypeExternal:   "外部链接",
		ResourceTypeAPI:        "API端点",
	}
	
	if desc, ok := descriptions[resType]; ok {
		return desc
	}
	return "未知类型"
}

// ClassifyURLs 批量分类URL
func (r *ResourceClassifier) ClassifyURLs(urls []string) map[ResourceType][]string {
	classified := make(map[ResourceType][]string)
	
	for _, urlStr := range urls {
		resType, _ := r.ClassifyURL(urlStr)
		classified[resType] = append(classified[resType], urlStr)
	}
	
	return classified
}

// GetStatistics 获取分类统计
func (r *ResourceClassifier) GetStatistics(classified map[ResourceType][]string) map[string]interface{} {
	stats := make(map[string]interface{})
	
	// 需要请求的资源
	needRequest := 0
	needRequest += len(classified[ResourceTypePage])
	needRequest += len(classified[ResourceTypeJavaScript])
	needRequest += len(classified[ResourceTypeCSS])
	needRequest += len(classified[ResourceTypeAPI])
	
	// 只收集的资源
	onlyCollect := 0
	onlyCollect += len(classified[ResourceTypeImage])
	onlyCollect += len(classified[ResourceTypeVideo])
	onlyCollect += len(classified[ResourceTypeAudio])
	onlyCollect += len(classified[ResourceTypeFont])
	onlyCollect += len(classified[ResourceTypeDocument])
	onlyCollect += len(classified[ResourceTypeArchive])
	onlyCollect += len(classified[ResourceTypeOther])
	onlyCollect += len(classified[ResourceTypeExternal])
	
	stats["need_request"] = needRequest
	stats["only_collect"] = onlyCollect
	stats["total"] = needRequest + onlyCollect
	
	// 详细统计
	for resType, urls := range classified {
		key := string(resType) + "_count"
		stats[key] = len(urls)
	}
	
	return stats
}

