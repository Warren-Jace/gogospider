package core

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

// SmartBusinessScorer 智能业务评分器（改进版）
// 🔧 修复：添加上下文判断和智能识别，区分真实价值
type SmartBusinessScorer struct {
	// 路径关键字权重（改进版）
	pathKeywords map[string]int
	
	// 参数特征权重（改进版）
	paramKeywords map[string]int
	
	// 低价值模式（新增：识别图片显示等低价值功能）
	lowValuePatterns []*regexp.Regexp
	
	// 高价值模式（新增：识别真实的业务逻辑）
	highValuePatterns []*regexp.Regexp
}

// NewSmartBusinessScorer 创建智能业务评分器
func NewSmartBusinessScorer() *SmartBusinessScorer {
	scorer := &SmartBusinessScorer{
		pathKeywords:      make(map[string]int),
		paramKeywords:     make(map[string]int),
		lowValuePatterns:  make([]*regexp.Regexp, 0),
		highValuePatterns: make([]*regexp.Regexp, 0),
	}
	
	// 🔧 改进：路径关键字权重（更细致的分类）
	scorer.pathKeywords = map[string]int{
		// 高价值（40-50分）
		"/admin":    50,
		"/manage":   45,
		"/backend":  45,
		"/console":  40,
		
		// 中高价值（30-40分）
		"/api":      35,
		"/upload":   35,
		"/login":    30,
		"/register": 30,
		"/user":     30,
		
		// 中等价值（20-30分）
		"/search":   25,
		"/list":     20,
		"/category": 20,
		"/product":  20,
		
		// 低价值（10-20分）- 展示类
		"/show":     10,
		"/display":  10,
		"/view":     10,
		"/image":    5,
		"/img":      5,
		"/pic":      5,
		"/photo":    5,
		"/thumb":    5,
	}
	
	// 🔧 改进：参数特征权重（区分真实风险）
	scorer.paramKeywords = map[string]int{
		// 高风险参数（25-30分）
		"cmd":      30,
		"exec":     30,
		"eval":     30,
		"system":   30,
		
		// 中高风险参数（20-25分）
		"sql":      25,
		"query":    20,
		"action":   20,
		
		// 文件操作参数（15-25分，取决于上下文）
		"upload":   25,
		"download": 20,
		"path":     20,
		"dir":      20,
		
		// 普通参数（10-15分）
		"id":       15,
		"user":     15,
		"page":     10,
		"limit":    10,
		
		// 🆕 低价值参数（5-10分）- 展示类
		"show":     5,
		"display":  5,
		"view":     5,
	}
	
	// 🆕 低价值模式识别
	scorer.lowValuePatterns = []*regexp.Regexp{
		// 图片显示脚本
		regexp.MustCompile(`(?i)/(show|display|view|get)(image|img|pic|photo|thumb)`),
		regexp.MustCompile(`(?i)/image\.(php|jsp|asp)`),
		regexp.MustCompile(`(?i)\?.*file=.*\.(jpg|jpeg|png|gif|bmp|webp|svg)`),
		
		// 静态资源代理
		regexp.MustCompile(`(?i)/proxy\.(php|jsp).*\.(css|js|jpg|png)`),
		regexp.MustCompile(`(?i)/static/`),
		regexp.MustCompile(`(?i)/assets/`),
		
		// 缩略图生成
		regexp.MustCompile(`(?i)/(thumb|thumbnail|resize)`),
		regexp.MustCompile(`(?i)\?(w|h|width|height|size)=\d+`),
	}
	
	// 🆕 高价值模式识别
	scorer.highValuePatterns = []*regexp.Regexp{
		// 文件上传
		regexp.MustCompile(`(?i)/(upload|uploader|file_upload)`),
		
		// 数据库操作
		regexp.MustCompile(`(?i)/(delete|update|insert|modify)`),
		
		// 用户管理
		regexp.MustCompile(`(?i)/(user|account|profile)/(edit|delete|update)`),
		
		// API端点
		regexp.MustCompile(`(?i)/api/v\d+/`),
		regexp.MustCompile(`(?i)\.(json|xml)(\?|$)`),
		
		// 搜索和查询
		regexp.MustCompile(`(?i)/(search|query|find).*\?`),
	}
	
	return scorer
}

// CalculateBusinessScore 计算业务价值分数（智能版本）
// 返回：0-100分
func (sbs *SmartBusinessScorer) CalculateBusinessScore(rawURL string) float64 {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return 50.0 // 解析失败，返回默认中等分数
	}
	
	score := 0.0
	
	// 🔧 步骤1：检查低价值模式（优先级最高）
	urlStr := strings.ToLower(rawURL)
	for _, pattern := range sbs.lowValuePatterns {
		if pattern.MatchString(urlStr) {
			// 匹配低价值模式，直接返回低分
			return 15.0 // 固定低分
		}
	}
	
	// 🔧 步骤2：检查高价值模式
	isHighValue := false
	for _, pattern := range sbs.highValuePatterns {
		if pattern.MatchString(urlStr) {
			isHighValue = true
			score += 30.0 // 高价值模式加分
			break
		}
	}
	
	// 🔧 步骤3：路径评分
	pathLower := strings.ToLower(parsedURL.Path)
	pathScore := 0
	for keyword, weight := range sbs.pathKeywords {
		if strings.Contains(pathLower, keyword) {
			pathScore += weight
		}
	}
	score += float64(pathScore)
	
	// 🔧 步骤4：文件扩展名评分
	ext := strings.ToLower(path.Ext(parsedURL.Path))
	switch ext {
	case ".php", ".jsp", ".asp", ".aspx":
		score += 20 // 动态脚本
	case ".do", ".action":
		score += 15 // 框架端点
	case ".json", ".xml":
		score += 25 // API响应
	case ".html", ".htm":
		score += 10 // 静态页面
	case ".jpg", ".jpeg", ".png", ".gif", ".css", ".js":
		score -= 20 // 静态资源（减分）
	}
	
	// 🔧 步骤5：参数评分（智能上下文判断）
	query := parsedURL.Query()
	paramScore := 0
	
	for param := range query {
		paramLower := strings.ToLower(param)
		
		// 检查是否为文件参数
		if paramLower == "file" || paramLower == "filename" || paramLower == "path" {
			// 🆕 上下文判断：区分图片显示和文件操作
			if sbs.isImageDisplayContext(parsedURL) {
				// 图片显示上下文：低价值
				paramScore += 5
			} else {
				// 文件操作上下文：高价值
				paramScore += 20
			}
		} else if weight, exists := sbs.paramKeywords[paramLower]; exists {
			paramScore += weight
		}
	}
	score += float64(paramScore)
	
	// 🔧 步骤6：组合特征加权
	// 如果有多个参数，说明功能更复杂
	if len(query) > 3 {
		score += 10
	}
	
	// 如果是高价值模式且有参数，额外加分
	if isHighValue && len(query) > 0 {
		score += 15
	}
	
	// 🔧 步骤7：规范化到0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// isImageDisplayContext 判断是否为图片显示上下文
func (sbs *SmartBusinessScorer) isImageDisplayContext(parsedURL *url.URL) bool {
	pathLower := strings.ToLower(parsedURL.Path)
	
	// 路径包含图片显示关键字
	imageKeywords := []string{"showimage", "displayimage", "getimage", "viewimage", 
		"image.php", "img.php", "picture.php", "photo.php"}
	for _, keyword := range imageKeywords {
		if strings.Contains(pathLower, keyword) {
			return true
		}
	}
	
	// 参数值指向图片文件
	query := parsedURL.Query()
	for param, values := range query {
		paramLower := strings.ToLower(param)
		if paramLower == "file" || paramLower == "path" || paramLower == "src" {
			for _, val := range values {
				valLower := strings.ToLower(val)
				// 检查是否为图片扩展名
				if strings.HasSuffix(valLower, ".jpg") || 
					strings.HasSuffix(valLower, ".jpeg") ||
					strings.HasSuffix(valLower, ".png") ||
					strings.HasSuffix(valLower, ".gif") ||
					strings.HasSuffix(valLower, ".bmp") ||
					strings.HasSuffix(valLower, ".webp") ||
					strings.Contains(valLower, "/pictures/") ||
					strings.Contains(valLower, "/images/") ||
					strings.Contains(valLower, "/photos/") {
					return true
				}
			}
		}
	}
	
	return false
}

// ClassifyURL 对URL进行分类
// 返回：类别名称、业务分数
func (sbs *SmartBusinessScorer) ClassifyURL(rawURL string) (string, float64) {
	score := sbs.CalculateBusinessScore(rawURL)
	
	var category string
	switch {
	case score >= 70:
		category = "高价值"
	case score >= 40:
		category = "中等价值"
	case score >= 20:
		category = "低价值"
	default:
		category = "极低价值"
	}
	
	return category, score
}

// GetRecommendedLimit 根据业务分数推荐爬取限制
func (sbs *SmartBusinessScorer) GetRecommendedLimit(score float64) int {
	switch {
	case score >= 70:
		return 20 // 高价值，允许更多
	case score >= 40:
		return 10 // 中等价值
	case score >= 20:
		return 5  // 低价值
	default:
		return 2  // 极低价值，最多2个
	}
}

