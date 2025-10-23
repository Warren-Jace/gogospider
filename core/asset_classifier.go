package core

import (
	"path/filepath"
	"strings"
)

// AssetType 资源类型
type AssetType string

const (
	AssetTypeImage     AssetType = "IMAGE"     // 图片
	AssetTypeScript    AssetType = "SCRIPT"    // 脚本
	AssetTypeStyle     AssetType = "STYLE"     // 样式
	AssetTypeFont      AssetType = "FONT"      // 字体
	AssetTypeComponent AssetType = "COMPONENT" // 前端组件
	AssetTypeOther     AssetType = "OTHER"     // 其他
	AssetTypeUnknown   AssetType = "UNKNOWN"   // 未知
)

// AssetClassifier 静态资源分类器
type AssetClassifier struct {
	imageExts     []string
	scriptExts    []string
	styleExts     []string
	fontExts      []string
	componentExts []string
	otherExts     []string
}

// NewAssetClassifier 创建资源分类器
func NewAssetClassifier() *AssetClassifier {
	return &AssetClassifier{
		imageExts: []string{
			".png", ".jpg", ".jpeg", ".gif", ".svg", ".bmp", ".webp", ".ico",
			".tiff", ".tif", ".psd", ".raw", ".heif", ".heic", ".avif",
		},
		scriptExts: []string{
			".js", ".ts", ".mjs", ".cjs", ".jsx", ".tsx",
		},
		styleExts: []string{
			".css", ".scss", ".less", ".sass", ".styl",
		},
		fontExts: []string{
			".woff", ".woff2", ".ttf", ".eot", ".otf", ".fon", ".fnt",
		},
		componentExts: []string{
			".vue", ".svelte", ".jsx", ".tsx",
		},
		otherExts: []string{
			".xml", ".json", ".pdf", ".doc", ".docx", ".xls", ".xlsx",
			".mp3", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm",
			".zip", ".rar", ".tar", ".gz", ".7z", ".bz2",
			".swf", ".flv",
		},
	}
}

// ClassifyAsset 分类单个资源
func (ac *AssetClassifier) ClassifyAsset(url string) AssetType {
	// 提取文件扩展名
	ext := strings.ToLower(filepath.Ext(url))
	
	// 处理查询参数（例如：image.php?id=1）
	// 如果URL包含查询参数，先去除
	if idx := strings.Index(url, "?"); idx != -1 {
		urlWithoutQuery := url[:idx]
		ext = strings.ToLower(filepath.Ext(urlWithoutQuery))
	}
	
	// 空扩展名的情况，尝试从URL路径判断
	if ext == "" {
		return ac.classifyByPath(url)
	}
	
	if ac.contains(ac.imageExts, ext) {
		return AssetTypeImage
	}
	if ac.contains(ac.scriptExts, ext) {
		return AssetTypeScript
	}
	if ac.contains(ac.styleExts, ext) {
		return AssetTypeStyle
	}
	if ac.contains(ac.fontExts, ext) {
		return AssetTypeFont
	}
	if ac.contains(ac.componentExts, ext) {
		return AssetTypeComponent
	}
	if ac.contains(ac.otherExts, ext) {
		return AssetTypeOther
	}
	
	return AssetTypeUnknown
}

// classifyByPath 根据URL路径特征分类（无扩展名时使用）
func (ac *AssetClassifier) classifyByPath(url string) AssetType {
	urlLower := strings.ToLower(url)
	
	// 检查路径中是否包含特定关键词
	if strings.Contains(urlLower, "/images/") || strings.Contains(urlLower, "/img/") || 
	   strings.Contains(urlLower, "/pictures/") || strings.Contains(urlLower, "/pics/") {
		return AssetTypeImage
	}
	
	if strings.Contains(urlLower, "/js/") || strings.Contains(urlLower, "/javascript/") ||
	   strings.Contains(urlLower, "/scripts/") {
		return AssetTypeScript
	}
	
	if strings.Contains(urlLower, "/css/") || strings.Contains(urlLower, "/styles/") ||
	   strings.Contains(urlLower, "/stylesheets/") {
		return AssetTypeStyle
	}
	
	if strings.Contains(urlLower, "/fonts/") || strings.Contains(urlLower, "/webfonts/") {
		return AssetTypeFont
	}
	
	return AssetTypeUnknown
}

// contains 检查slice中是否包含指定元素
func (ac *AssetClassifier) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ClassifyAssets 批量分类资源
func (ac *AssetClassifier) ClassifyAssets(urls []string) map[AssetType][]string {
	result := make(map[AssetType][]string)
	result[AssetTypeImage] = make([]string, 0)
	result[AssetTypeScript] = make([]string, 0)
	result[AssetTypeStyle] = make([]string, 0)
	result[AssetTypeFont] = make([]string, 0)
	result[AssetTypeComponent] = make([]string, 0)
	result[AssetTypeOther] = make([]string, 0)
	result[AssetTypeUnknown] = make([]string, 0)
	
	// 用于去重
	seen := make(map[string]bool)
	
	for _, url := range urls {
		// 去重
		if seen[url] {
			continue
		}
		seen[url] = true
		
		category := ac.ClassifyAsset(url)
		result[category] = append(result[category], url)
	}
	
	return result
}

// GetAssetStats 获取资源统计信息
func (ac *AssetClassifier) GetAssetStats(classifiedAssets map[AssetType][]string) map[string]int {
	stats := make(map[string]int)
	
	stats["images"] = len(classifiedAssets[AssetTypeImage])
	stats["scripts"] = len(classifiedAssets[AssetTypeScript])
	stats["styles"] = len(classifiedAssets[AssetTypeStyle])
	stats["fonts"] = len(classifiedAssets[AssetTypeFont])
	stats["components"] = len(classifiedAssets[AssetTypeComponent])
	stats["others"] = len(classifiedAssets[AssetTypeOther])
	stats["unknown"] = len(classifiedAssets[AssetTypeUnknown])
	
	// 计算总数
	total := 0
	for _, count := range stats {
		total += count
	}
	stats["total"] = total
	
	return stats
}

