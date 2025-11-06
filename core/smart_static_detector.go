package core

import (
	"bytes"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// SmartStaticDetector 智能静态资源检测器
// 🔧 修复：支持扩展名、Content-Type、魔数多重检测
type SmartStaticDetector struct {
	// 静态扩展名列表
	staticExtensions map[string]bool
	
	// 动态图片URL模式（如showimage.php）
	dynamicImagePatterns []*regexp.Regexp
}

// NewSmartStaticDetector 创建智能静态资源检测器
func NewSmartStaticDetector() *SmartStaticDetector {
	detector := &SmartStaticDetector{
		staticExtensions:     make(map[string]bool),
		dynamicImagePatterns: make([]*regexp.Regexp, 0),
	}
	
	// 初始化静态扩展名
	staticExts := []string{
		// 图片
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico",
		".tiff", ".tif", ".psd", ".raw", ".heif", ".heic",
		
		// 视频
		".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".webm", ".m4v",
		
		// 音频
		".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac", ".wma",
		
		// 字体
		".woff", ".woff2", ".ttf", ".eot", ".otf",
		
		// 样式
		".css", ".scss", ".sass", ".less",
		
		// 文档
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		
		// 压缩包
		".zip", ".rar", ".7z", ".tar", ".gz", ".bz2",
		
		// 其他
		".map", // source map
	}
	
	for _, ext := range staticExts {
		detector.staticExtensions[ext] = true
	}
	
	// 动态图片URL模式
	detector.dynamicImagePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)/(show|display|get|view)(image|img|pic|photo|thumb)`),
		regexp.MustCompile(`(?i)/image\.(php|jsp|asp|aspx)`),
		regexp.MustCompile(`(?i)/(thumb|thumbnail|resize)\.(php|jsp|asp)`),
	}
	
	return detector
}

// IsStatic 综合判断URL是否为静态资源
// 参数：
//   - urlStr: URL字符串
//   - contentType: Content-Type响应头（可选，爬取后才有）
//   - content: 响应内容（可选，用于魔数检测）
// 返回：是否为静态资源、资源类型
func (ssd *SmartStaticDetector) IsStatic(urlStr string, contentType string, content []byte) (bool, string) {
	// 1. 🔧 扩展名检测（最快）
	if isStatic, resType := ssd.isStaticByExtension(urlStr); isStatic {
		return true, resType
	}
	
	// 2. 🔧 Content-Type检测（如果有响应头）
	if contentType != "" {
		if isStatic, resType := ssd.isStaticByContentType(contentType); isStatic {
			return true, resType
		}
	}
	
	// 3. 🔧 魔数检测（如果有内容）
	if len(content) > 0 {
		if isStatic, resType := ssd.isStaticByMagicBytes(content); isStatic {
			return true, resType
		}
	}
	
	// 4. 🔧 动态图片URL模式检测
	if ssd.isDynamicImageURL(urlStr) {
		// 虽然是PHP等动态脚本，但实际返回图片
		return true, "dynamic_image"
	}
	
	// 不是静态资源
	return false, ""
}

// isStaticByExtension 通过扩展名判断
func (ssd *SmartStaticDetector) isStaticByExtension(urlStr string) (bool, string) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false, ""
	}
	
	ext := strings.ToLower(path.Ext(parsedURL.Path))
	if ext == "" {
		return false, ""
	}
	
	if ssd.staticExtensions[ext] {
		// 确定资源类型
		resType := ssd.classifyExtension(ext)
		return true, resType
	}
	
	return false, ""
}

// isStaticByContentType 通过Content-Type判断
func (ssd *SmartStaticDetector) isStaticByContentType(contentType string) (bool, string) {
	contentType = strings.ToLower(contentType)
	
	// 提取主类型（去除charset等参数）
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)
	
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return true, "image"
	case strings.HasPrefix(contentType, "video/"):
		return true, "video"
	case strings.HasPrefix(contentType, "audio/"):
		return true, "audio"
	case strings.HasPrefix(contentType, "font/"):
		return true, "font"
	case contentType == "text/css":
		return true, "css"
	case contentType == "application/pdf":
		return true, "document"
	case strings.Contains(contentType, "zip") || 
		 strings.Contains(contentType, "compressed"):
		return true, "archive"
	}
	
	return false, ""
}

// isStaticByMagicBytes 通过文件魔数（文件头）判断
func (ssd *SmartStaticDetector) isStaticByMagicBytes(content []byte) (bool, string) {
	if len(content) < 8 {
		return false, ""
	}
	
	// 图片魔数
	imageMagics := map[string]string{
		"\xFF\xD8\xFF":                           "image", // JPEG
		"\x89PNG\r\n\x1a\n":                      "image", // PNG
		"GIF87a":                                 "image", // GIF87a
		"GIF89a":                                 "image", // GIF89a
		"BM":                                     "image", // BMP
		"RIFF":                                   "image", // WEBP (需要进一步检查)
		"\x00\x00\x01\x00":                       "image", // ICO
	}
	
	for magic, resType := range imageMagics {
		if bytes.HasPrefix(content, []byte(magic)) {
			return true, resType
		}
	}
	
	// WEBP特殊检测（RIFF...WEBP）
	if bytes.HasPrefix(content, []byte("RIFF")) && len(content) >= 12 {
		if bytes.Equal(content[8:12], []byte("WEBP")) {
			return true, "image"
		}
	}
	
	// 视频魔数
	videoMagics := []string{
		"\x00\x00\x00\x18ftypmp42", // MP4
		"\x00\x00\x00\x20ftypisom", // MP4
		"FLV",                       // FLV
	}
	
	for _, magic := range videoMagics {
		if bytes.HasPrefix(content, []byte(magic)) {
			return true, "video"
		}
	}
	
	// PDF
	if bytes.HasPrefix(content, []byte("%PDF")) {
		return true, "document"
	}
	
	// ZIP/压缩包
	if bytes.HasPrefix(content, []byte("PK\x03\x04")) {
		return true, "archive"
	}
	
	return false, ""
}

// isDynamicImageURL 检测动态图片URL模式
func (ssd *SmartStaticDetector) isDynamicImageURL(urlStr string) bool {
	urlLower := strings.ToLower(urlStr)
	
	// 检查URL模式
	for _, pattern := range ssd.dynamicImagePatterns {
		if pattern.MatchString(urlLower) {
			return true
		}
	}
	
	// 检查参数值是否指向图片
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	
	query := parsedURL.Query()
	for param, values := range query {
		paramLower := strings.ToLower(param)
		if paramLower == "file" || paramLower == "path" || 
		   paramLower == "img" || paramLower == "image" {
			for _, val := range values {
				valLower := strings.ToLower(val)
				// 检查参数值是否为图片路径
				if strings.HasSuffix(valLower, ".jpg") ||
					strings.HasSuffix(valLower, ".jpeg") ||
					strings.HasSuffix(valLower, ".png") ||
					strings.HasSuffix(valLower, ".gif") ||
					strings.HasSuffix(valLower, ".webp") ||
					strings.Contains(valLower, "/pictures/") ||
					strings.Contains(valLower, "/images/") {
					return true
				}
			}
		}
	}
	
	return false
}

// classifyExtension 根据扩展名分类
func (ssd *SmartStaticDetector) classifyExtension(ext string) string {
	ext = strings.ToLower(ext)
	
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico":
		return "image"
	case ".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".webm":
		return "video"
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac":
		return "audio"
	case ".woff", ".woff2", ".ttf", ".eot", ".otf":
		return "font"
	case ".css", ".scss", ".sass", ".less":
		return "css"
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		return "document"
	case ".zip", ".rar", ".7z", ".tar", ".gz", ".bz2":
		return "archive"
	case ".map":
		return "sourcemap"
	default:
		return "static"
	}
}

// ShouldCrawl 判断是否应该爬取（基于静态检测）
// 返回：是否应该爬取、原因
func (ssd *SmartStaticDetector) ShouldCrawl(urlStr string, contentType string, content []byte) (bool, string) {
	isStatic, resType := ssd.IsStatic(urlStr, contentType, content)
	
	if isStatic {
		return false, "静态资源：" + resType
	}
	
	return true, "动态资源"
}

// AddStaticExtension 添加自定义静态扩展名
func (ssd *SmartStaticDetector) AddStaticExtension(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ssd.staticExtensions[strings.ToLower(ext)] = true
}

// RemoveStaticExtension 移除静态扩展名（如.js需要分析）
func (ssd *SmartStaticDetector) RemoveStaticExtension(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	delete(ssd.staticExtensions, strings.ToLower(ext))
}

