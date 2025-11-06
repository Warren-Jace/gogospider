package core

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// StaticResourceFilter 静态资源过滤器
// 核心功能：
// 1. 图片、CSS、字体等 → 只记录不请求
// 2. JS文件 → 正常请求和分析
// 3. 参数化URL如 ?file=test.css → 不算静态资源，需要请求
type StaticResourceFilter struct {
	mutex sync.RWMutex
	
	// 配置
	excludeExtensions map[string]bool // 要过滤的扩展名
	jsExtensions      map[string]bool // JS扩展名（特殊处理）
	
	// 记录的静态资源
	recordedResources map[string]ResourceInfo
	
	// 统计
	stats StaticFilterStats
}

// ResourceInfo 资源信息
type ResourceInfo struct {
	URL          string
	ResourceType string // image/css/font/document等
	RecordTime   string
}

// StaticFilterStats 静态过滤统计
type StaticFilterStats struct {
	TotalChecked    int
	ImagesFiltered  int
	CSSFiltered     int
	FontsFiltered   int
	DocsFiltered    int
	ArchivesFiltered int
	JSAllowed       int  // JS文件放行数
	ParamURLAllowed int  // 参数化URL放行数
}

// NewStaticResourceFilter 创建静态资源过滤器
func NewStaticResourceFilter(excludeExts []string) *StaticResourceFilter {
	filter := &StaticResourceFilter{
		excludeExtensions: make(map[string]bool),
		jsExtensions:      make(map[string]bool),
		recordedResources: make(map[string]ResourceInfo),
		stats:             StaticFilterStats{},
	}
	
	// JS扩展名（不过滤）
	jsExts := []string{"js", "mjs", "jsx"}
	for _, ext := range jsExts {
		filter.jsExtensions[strings.ToLower(ext)] = true
	}
	
	// 其他静态资源扩展名（过滤）
	for _, ext := range excludeExts {
		extLower := strings.ToLower(strings.TrimPrefix(ext, "."))
		// 跳过JS扩展名
		if !filter.jsExtensions[extLower] {
			filter.excludeExtensions[extLower] = true
		}
	}
	
	return filter
}

// ShouldFilter 判断URL是否应该过滤
// 返回: (是否过滤, 资源类型, 原因)
func (f *StaticResourceFilter) ShouldFilter(rawURL string) (bool, string, string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	f.stats.TotalChecked++
	
	// 1. 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, "", "URL解析失败"
	}
	
	// 2. 提取文件路径
	urlPath := parsedURL.Path
	
	// 🔧 关键判断：如果URL有参数，可能是动态资源
	if parsedURL.RawQuery != "" {
		// 检查参数中是否有 file/filename/path 等关键词
		query := parsedURL.Query()
		dynamicKeys := []string{"file", "filename", "path", "resource", "download", "view", "src", "url"}
		
		for _, key := range dynamicKeys {
			if _, hasKey := query[key]; hasKey {
				// 这是动态资源URL，不过滤！
				f.stats.ParamURLAllowed++
				return false, "dynamic", fmt.Sprintf("参数化URL（%s参数），不过滤", key)
			}
		}
	}
	
	// 3. 提取扩展名
	lastDot := strings.LastIndex(urlPath, ".")
	if lastDot == -1 || lastDot == len(urlPath)-1 {
		return false, "", "无扩展名"
	}
	
	// 获取扩展名（可能包含参数）
	extPart := urlPath[lastDot+1:]
	// 去除查询参数影响
	if qIndex := strings.Index(extPart, "?"); qIndex != -1 {
		extPart = extPart[:qIndex]
	}
	extension := strings.ToLower(extPart)
	
	// 4. 检查是否为JS文件（不过滤）
	if f.jsExtensions[extension] {
		f.stats.JSAllowed++
		return false, "javascript", "JS文件，允许请求"
	}
	
	// 5. 检查是否为静态资源
	if !f.excludeExtensions[extension] {
		return false, "", "不是静态资源"
	}
	
	// 6. 确定资源类型
	resourceType := f.classifyResource(extension)
	
	// 7. 记录该静态资源
	f.recordedResources[rawURL] = ResourceInfo{
		URL:          rawURL,
		ResourceType: resourceType,
		RecordTime:   time.Now().Format("2006-01-02 15:04:05"),
	}
	
	// 8. 更新统计
	switch resourceType {
	case "image":
		f.stats.ImagesFiltered++
	case "css":
		f.stats.CSSFiltered++
	case "font":
		f.stats.FontsFiltered++
	case "document":
		f.stats.DocsFiltered++
	case "archive":
		f.stats.ArchivesFiltered++
	}
	
	// ✅ 过滤：只记录不请求
	return true, resourceType, fmt.Sprintf("静态资源(%s)，只记录不请求", resourceType)
}

// classifyResource 分类资源类型
func (f *StaticResourceFilter) classifyResource(ext string) string {
	imageExts := map[string]bool{
		"jpg": true, "jpeg": true, "png": true, "gif": true, 
		"svg": true, "ico": true, "webp": true, "bmp": true,
	}
	
	cssExts := map[string]bool{
		"css": true, "scss": true, "sass": true, "less": true,
	}
	
	fontExts := map[string]bool{
		"woff": true, "woff2": true, "ttf": true, "eot": true, "otf": true,
	}
	
	docExts := map[string]bool{
		"pdf": true, "doc": true, "docx": true, "xls": true, 
		"xlsx": true, "ppt": true, "pptx": true,
	}
	
	archiveExts := map[string]bool{
		"zip": true, "rar": true, "tar": true, "gz": true, "7z": true,
	}
	
	if imageExts[ext] {
		return "image"
	} else if cssExts[ext] {
		return "css"
	} else if fontExts[ext] {
		return "font"
	} else if docExts[ext] {
		return "document"
	} else if archiveExts[ext] {
		return "archive"
	}
	
	return "other"
}

// GetRecordedResources 获取记录的静态资源
func (f *StaticResourceFilter) GetRecordedResources() map[string]ResourceInfo {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	
	// 返回副本
	result := make(map[string]ResourceInfo, len(f.recordedResources))
	for k, v := range f.recordedResources {
		result[k] = v
	}
	return result
}

// GetStatistics 获取统计信息
func (f *StaticResourceFilter) GetStatistics() StaticFilterStats {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.stats
}

// PrintReport 打印报告
func (f *StaticResourceFilter) PrintReport() {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	
	fmt.Println("\n╔═══════════════════════════════════════╗")
	fmt.Println("║     静态资源过滤统计报告             ║")
	fmt.Println("╚═══════════════════════════════════════╝")
	fmt.Printf("  总检查数:        %d\n", f.stats.TotalChecked)
	fmt.Printf("  过滤图片:        %d\n", f.stats.ImagesFiltered)
	fmt.Printf("  过滤CSS:         %d\n", f.stats.CSSFiltered)
	fmt.Printf("  过滤字体:        %d\n", f.stats.FontsFiltered)
	fmt.Printf("  过滤文档:        %d\n", f.stats.DocsFiltered)
	fmt.Printf("  过滤压缩包:      %d\n", f.stats.ArchivesFiltered)
	fmt.Printf("  JS放行:          %d\n", f.stats.JSAllowed)
	fmt.Printf("  参数URL放行:     %d\n", f.stats.ParamURLAllowed)
	
	totalFiltered := f.stats.ImagesFiltered + f.stats.CSSFiltered + 
	                 f.stats.FontsFiltered + f.stats.DocsFiltered + 
	                 f.stats.ArchivesFiltered
	if f.stats.TotalChecked > 0 {
		fmt.Printf("  过滤率:          %.1f%%\n", 
			float64(totalFiltered)*100/float64(f.stats.TotalChecked))
	}
	fmt.Println("─────────────────────────────────────────")
}

