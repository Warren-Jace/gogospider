package core

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// BusinessAwareURLFilter 业务感知的URL过滤器
// 通过多维度分析URL的业务价值，智能决策是否爬取
type BusinessAwareURLFilter struct {
	mutex sync.RWMutex
	
	// URL模式记录
	urlPatterns map[string]*URLBusinessPattern
	
	// 配置参数
	config BusinessFilterConfig
	
	// 学习统计
	stats FilterStatistics
}

// BusinessFilterConfig 业务过滤器配置
type BusinessFilterConfig struct {
	// 基础配置
	MinBusinessScore     float64 // 最低业务价值分数（0-100），低于此分数的URL会被过滤
	HighValueThreshold   float64 // 高价值URL阈值，高于此分数的URL总是保留
	
	// 同模式URL限制
	MaxSamePatternLowValue  int // 同一低价值模式最多爬取数量
	MaxSamePatternMidValue  int // 同一中等价值模式最多爬取数量
	MaxSamePatternHighValue int // 同一高价值模式最多爬取数量（或无限制）
	
	// 自适应学习
	EnableAdaptiveLearning bool    // 是否启用自适应学习
	LearningRate           float64 // 学习率（调整业务价值的速度）
	
	// 启用标志
	Enabled bool // 是否启用业务感知过滤
}

// URLBusinessPattern URL业务模式
type URLBusinessPattern struct {
	// 基本信息
	Pattern        string   // URL模式（路径+参数名）
	PathSegments   []string // 路径片段
	ParamNames     []string // 参数名列表
	
	// 业务分析
	BusinessType   string  // 业务类型（如：user_profile, order_detail, search等）
	BusinessScore  float64 // 业务价值分数（0-100）
	
	// 统计信息
	SeenCount      int                    // 发现次数
	CrawledCount   int                    // 实际爬取次数
	SkippedCount   int                    // 跳过次数
	ParamValues    map[string][]string    // 参数值样本
	ResponseCodes  map[int]int            // 响应状态码统计
	
	// 学习指标
	AvgResponseTime float64 // 平均响应时间
	HasNewLinks     int     // 发现新链接的次数
	HasForms        int     // 包含表单的次数
	HasAPIs         int     // 包含API的次数
	
	// 价值调整
	ValueAdjustment float64 // 基于学习的价值调整（-20到+20）
}

// FilterStatistics 过滤统计
type FilterStatistics struct {
	TotalURLs         int // 处理的URL总数
	AllowedURLs       int // 允许爬取的URL数
	FilteredURLs      int // 过滤掉的URL数
	HighValueURLs     int // 高价值URL数
	LowValueURLs      int // 低价值URL数
	AdaptiveAdjustments int // 自适应调整次数
}

// NewBusinessAwareURLFilter 创建业务感知URL过滤器
func NewBusinessAwareURLFilter(config BusinessFilterConfig) *BusinessAwareURLFilter {
	// 设置默认值
	if config.MinBusinessScore == 0 {
		config.MinBusinessScore = 30.0 // 默认最低分30分
	}
	if config.HighValueThreshold == 0 {
		config.HighValueThreshold = 70.0 // 默认高价值阈值70分
	}
	if config.MaxSamePatternLowValue == 0 {
		config.MaxSamePatternLowValue = 2 // 低价值最多2个
	}
	if config.MaxSamePatternMidValue == 0 {
		config.MaxSamePatternMidValue = 5 // 中等价值最多5个
	}
	if config.MaxSamePatternHighValue == 0 {
		config.MaxSamePatternHighValue = 20 // 高价值最多20个（或更多）
	}
	if config.LearningRate == 0 {
		config.LearningRate = 0.1 // 默认学习率10%
	}
	
	return &BusinessAwareURLFilter{
		urlPatterns: make(map[string]*URLBusinessPattern),
		config:      config,
		stats:       FilterStatistics{},
	}
}

// DefaultBusinessFilterConfig 返回默认配置
func DefaultBusinessFilterConfig() BusinessFilterConfig {
	return BusinessFilterConfig{
		MinBusinessScore:        30.0,
		HighValueThreshold:      70.0,
		MaxSamePatternLowValue:  2,
		MaxSamePatternMidValue:  5,
		MaxSamePatternHighValue: 20,
		EnableAdaptiveLearning:  true,
		LearningRate:            0.1,
		Enabled:                 true,
	}
}

// ShouldCrawlURL 判断URL是否应该爬取（主入口）
func (f *BusinessAwareURLFilter) ShouldCrawlURL(rawURL string) (bool, string, float64) {
	if !f.config.Enabled {
		return true, "业务感知过滤器未启用", 0.0
	}
	
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	f.stats.TotalURLs++
	
	// 1. 解析URL并提取业务模式
	pattern, err := f.extractBusinessPattern(rawURL)
	if err != nil {
		// 解析失败，保守起见允许爬取
		f.stats.AllowedURLs++
		return true, fmt.Sprintf("URL解析失败: %v", err), 0.0
	}
	
	// 2. 查找或创建模式记录
	patternKey := pattern.Pattern
	existing, exists := f.urlPatterns[patternKey]
	
	if !exists {
		// 新模式，计算初始业务价值
		f.urlPatterns[patternKey] = pattern
		pattern.SeenCount = 1
		pattern.CrawledCount = 1
		f.stats.AllowedURLs++
		
		if pattern.BusinessScore >= f.config.HighValueThreshold {
			f.stats.HighValueURLs++
		}
		
		return true, fmt.Sprintf("新URL模式，业务类型: %s，初始价值: %.1f", 
			pattern.BusinessType, pattern.BusinessScore), pattern.BusinessScore
	}
	
	// 3. 更新统计
	existing.SeenCount++
	
	// 4. 计算当前业务价值（包含学习调整）
	currentScore := existing.BusinessScore + existing.ValueAdjustment
	
	// 5. 高价值URL总是保留（即使超过限制）
	if currentScore >= f.config.HighValueThreshold {
		existing.CrawledCount++
		f.stats.AllowedURLs++
		f.stats.HighValueURLs++
		return true, fmt.Sprintf("高价值URL（分数: %.1f），总是保留", currentScore), currentScore
	}
	
	// 6. 检查是否超过同模式限制
	maxAllowed := f.getMaxAllowedByScore(currentScore)
	
	if existing.CrawledCount >= maxAllowed {
		existing.SkippedCount++
		f.stats.FilteredURLs++
		
		if currentScore < 50 {
			f.stats.LowValueURLs++
		}
		
		return false, fmt.Sprintf("同模式URL已达限制（已爬: %d/%d，业务价值: %.1f）", 
			existing.CrawledCount, maxAllowed, currentScore), currentScore
	}
	
	// 7. 检查最低分数要求
	if currentScore < f.config.MinBusinessScore {
		existing.SkippedCount++
		f.stats.FilteredURLs++
		f.stats.LowValueURLs++
		return false, fmt.Sprintf("业务价值过低（分数: %.1f < 最低要求: %.1f）", 
			currentScore, f.config.MinBusinessScore), currentScore
	}
	
	// 8. 允许爬取
	existing.CrawledCount++
	f.stats.AllowedURLs++
	
	// 记录参数值样本（用于后续分析）
	f.recordParamSamples(existing, rawURL)
	
	return true, fmt.Sprintf("允许爬取（业务价值: %.1f，已爬: %d/%d）", 
		currentScore, existing.CrawledCount, maxAllowed), currentScore
}

// extractBusinessPattern 提取URL的业务模式和价值
func (f *BusinessAwareURLFilter) extractBusinessPattern(rawURL string) (*URLBusinessPattern, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	
	pattern := &URLBusinessPattern{
		PathSegments:   strings.Split(strings.Trim(parsedURL.Path, "/"), "/"),
		ParamNames:     make([]string, 0),
		ParamValues:    make(map[string][]string),
		ResponseCodes:  make(map[int]int),
		SeenCount:      0,
		CrawledCount:   0,
		SkippedCount:   0,
		ValueAdjustment: 0.0,
	}
	
	// 提取参数名
	if parsedURL.RawQuery != "" {
		queryParams := parsedURL.Query()
		for paramName := range queryParams {
			pattern.ParamNames = append(pattern.ParamNames, paramName)
			pattern.ParamValues[paramName] = make([]string, 0)
		}
		sort.Strings(pattern.ParamNames)
	}
	
	// 生成模式字符串
	pathPattern := strings.Join(pattern.PathSegments, "/")
	if len(pattern.ParamNames) > 0 {
		pattern.Pattern = fmt.Sprintf("/%s?%s", pathPattern, strings.Join(pattern.ParamNames, "&"))
	} else {
		pattern.Pattern = fmt.Sprintf("/%s", pathPattern)
	}
	
	// 识别业务类型和计算价值
	pattern.BusinessType = f.identifyBusinessType(pattern)
	pattern.BusinessScore = f.calculateBusinessScore(pattern)
	
	return pattern, nil
}

// identifyBusinessType 识别URL的业务类型
func (f *BusinessAwareURLFilter) identifyBusinessType(pattern *URLBusinessPattern) string {
	pathStr := strings.ToLower(strings.Join(pattern.PathSegments, "/"))
	paramStr := strings.ToLower(strings.Join(pattern.ParamNames, " "))
	combined := pathStr + " " + paramStr
	
	// 定义业务类型特征（按优先级排序）
	businessTypes := []struct {
		name     string
		keywords []string
		priority int
	}{
		// 高价值业务类型
		{"admin_panel", []string{"admin", "管理", "backend", "console", "dashboard"}, 100},
		{"authentication", []string{"login", "登录", "auth", "signin", "signup", "register", "注册", "password", "密码"}, 95},
		{"api_endpoint", []string{"api/", "/v1/", "/v2/", "/rest/", "/graphql", "/json"}, 90},
		{"user_profile", []string{"user", "profile", "用户", "account", "账号", "member", "会员"}, 85},
		{"payment", []string{"pay", "payment", "支付", "order", "订单", "checkout", "cart", "购物车"}, 90},
		{"file_upload", []string{"upload", "上传", "file", "attachment", "附件"}, 85},
		
		// 中等价值业务类型
		{"search", []string{"search", "搜索", "query", "find", "lookup"}, 70},
		{"detail_page", []string{"detail", "详情", "show", "view", "item", "product"}, 65},
		{"list_page", []string{"list", "列表", "index", "catalog", "category", "分类"}, 60},
		{"form_page", []string{"form", "表单", "submit", "提交", "post"}, 65},
		{"comment", []string{"comment", "评论", "reply", "回复", "feedback", "反馈"}, 60},
		
		// 较低价值但可能重要
		{"pagination", []string{"page", "页", "offset", "limit", "pagesize"}, 40},
		{"filter", []string{"filter", "筛选", "sort", "排序", "category"}, 45},
		{"language", []string{"lang", "language", "语言", "locale", "i18n"}, 35},
		
		// 低价值
		{"static_resource", []string{"static", "assets", "resource", "public", ".css", ".js", ".jpg", ".png", ".ico"}, 10},
		{"analytics", []string{"track", "analytics", "统计", "beacon", "pixel", "ga"}, 20},
	}
	
	// 匹配业务类型
	for _, bt := range businessTypes {
		for _, keyword := range bt.keywords {
			if strings.Contains(combined, keyword) {
				return bt.name
			}
		}
	}
	
	// 根据参数数量和路径深度推断
	if len(pattern.ParamNames) == 0 && len(pattern.PathSegments) <= 2 {
		return "simple_page" // 简单页面
	}
	
	if len(pattern.ParamNames) >= 3 {
		return "complex_query" // 复杂查询
	}
	
	return "unknown" // 未知类型
}

// calculateBusinessScore 计算URL的业务价值分数（0-100）
func (f *BusinessAwareURLFilter) calculateBusinessScore(pattern *URLBusinessPattern) float64 {
	score := 50.0 // 基础分数
	
	// 1. 根据业务类型调整（最重要）
	typeScores := map[string]float64{
		"admin_panel":      95.0,
		"authentication":   90.0,
		"api_endpoint":     85.0,
		"payment":          90.0,
		"file_upload":      85.0,
		"user_profile":     80.0,
		"form_page":        75.0,
		"search":           70.0,
		"detail_page":      65.0,
		"list_page":        60.0,
		"comment":          60.0,
		"complex_query":    55.0,
		"filter":           45.0,
		"pagination":       40.0,
		"language":         35.0,
		"simple_page":      50.0,
		"analytics":        20.0,
		"static_resource":  10.0,
		"unknown":          50.0,
	}
	
	if typeScore, exists := typeScores[pattern.BusinessType]; exists {
		score = typeScore
	}
	
	// 2. 参数名价值加分（重要参数）
	valuableParams := map[string]float64{
		"id":       5.0,
		"uid":      8.0,
		"user_id":  8.0,
		"userid":   8.0,
		"token":    10.0,
		"key":      10.0,
		"api_key":  10.0,
		"secret":   10.0,
		"password": 10.0,
		"email":    7.0,
		"username": 7.0,
		"phone":    7.0,
		"action":   6.0,
		"cmd":      8.0,
		"command":  8.0,
		"method":   6.0,
		"callback": 5.0,
		"redirect": 7.0,
		"url":      7.0,
		"path":     7.0,
		"file":     8.0,
		"upload":   9.0,
	}
	
	paramBonus := 0.0
	for _, paramName := range pattern.ParamNames {
		paramLower := strings.ToLower(paramName)
		if bonus, exists := valuableParams[paramLower]; exists {
			paramBonus += bonus
		}
		
		// 参数名包含重要关键词
		if strings.Contains(paramLower, "admin") || strings.Contains(paramLower, "auth") {
			paramBonus += 8.0
		}
	}
	score += paramBonus
	
	// 3. 路径深度调整
	pathDepth := len(pattern.PathSegments)
	if pathDepth == 0 {
		score -= 5.0 // 根路径可能价值较低
	} else if pathDepth >= 3 && pathDepth <= 5 {
		score += 5.0 // 中等深度通常更有价值
	} else if pathDepth > 6 {
		score -= 3.0 // 过深可能是分页或次要页面
	}
	
	// 4. 参数数量调整
	paramCount := len(pattern.ParamNames)
	if paramCount == 1 {
		score += 3.0 // 单参数通常是重要标识
	} else if paramCount >= 2 && paramCount <= 4 {
		score += 5.0 // 适度参数通常是业务功能
	} else if paramCount > 8 {
		score -= 5.0 // 过多参数可能是过滤/排序，价值较低
	}
	
	// 5. REST风格路径加分
	if f.isRESTfulPath(pattern.PathSegments) {
		score += 8.0
	}
	
	// 6. 特殊模式识别
	pathStr := strings.ToLower(strings.Join(pattern.PathSegments, "/"))
	
	// CRUD操作
	if regexp.MustCompile(`(create|update|delete|edit|modify|remove)`).MatchString(pathStr) {
		score += 10.0
	}
	
	// 敏感操作
	if regexp.MustCompile(`(config|setting|permission|role|privilege)`).MatchString(pathStr) {
		score += 12.0
	}
	
	// 确保分数在0-100范围内
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// isRESTfulPath 判断是否为RESTful风格路径
func (f *BusinessAwareURLFilter) isRESTfulPath(segments []string) bool {
	if len(segments) < 2 {
		return false
	}
	
	// 典型RESTful模式：/api/users/123, /v1/products/456
	restPatterns := []string{"api", "v1", "v2", "v3", "rest"}
	for _, seg := range segments {
		for _, pattern := range restPatterns {
			if strings.Contains(strings.ToLower(seg), pattern) {
				return true
			}
		}
	}
	
	// 检查是否有资源+ID模式
	if len(segments) >= 2 {
		lastSeg := segments[len(segments)-1]
		// 最后一段是数字或UUID，倒数第二段是名词
		if regexp.MustCompile(`^\d+$`).MatchString(lastSeg) || 
		   regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}`).MatchString(lastSeg) {
			return true
		}
	}
	
	return false
}

// getMaxAllowedByScore 根据业务价值分数返回同模式最大爬取数
func (f *BusinessAwareURLFilter) getMaxAllowedByScore(score float64) int {
	if score >= f.config.HighValueThreshold {
		return f.config.MaxSamePatternHighValue
	} else if score >= 50.0 {
		return f.config.MaxSamePatternMidValue
	} else {
		return f.config.MaxSamePatternLowValue
	}
}

// recordParamSamples 记录参数值样本
func (f *BusinessAwareURLFilter) recordParamSamples(pattern *URLBusinessPattern, rawURL string) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return
	}
	
	queryParams := parsedURL.Query()
	for paramName, values := range queryParams {
		if _, exists := pattern.ParamValues[paramName]; !exists {
			pattern.ParamValues[paramName] = make([]string, 0)
		}
		
		// 最多保留10个样本
		if len(pattern.ParamValues[paramName]) < 10 {
			for _, value := range values {
				pattern.ParamValues[paramName] = append(pattern.ParamValues[paramName], value)
			}
		}
	}
}

// UpdateCrawlResult 根据爬取结果更新URL模式的价值（自适应学习）
func (f *BusinessAwareURLFilter) UpdateCrawlResult(rawURL string, statusCode int, 
	responseTime float64, hasNewLinks bool, hasNewForms bool, hasNewAPIs bool) {
	
	if !f.config.EnableAdaptiveLearning {
		return
	}
	
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	// 提取模式
	pattern, err := f.extractBusinessPattern(rawURL)
	if err != nil {
		return
	}
	
	existing, exists := f.urlPatterns[pattern.Pattern]
	if !exists {
		return
	}
	
	// 更新统计
	existing.ResponseCodes[statusCode]++
	
	// 更新平均响应时间
	totalTime := existing.AvgResponseTime * float64(existing.CrawledCount-1)
	existing.AvgResponseTime = (totalTime + responseTime) / float64(existing.CrawledCount)
	
	// 更新发现指标
	if hasNewLinks {
		existing.HasNewLinks++
	}
	if hasNewForms {
		existing.HasForms++
	}
	if hasNewAPIs {
		existing.HasAPIs++
	}
	
	// 自适应调整价值
	adjustment := 0.0
	
	// 成功率影响
	successCount := 0
	for code, count := range existing.ResponseCodes {
		if code >= 200 && code < 300 {
			successCount += count
		}
	}
	successRate := float64(successCount) / float64(existing.CrawledCount)
	
	if successRate < 0.5 {
		adjustment -= 10.0 // 大量失败，降低价值
	} else if successRate > 0.9 {
		adjustment += 5.0 // 高成功率，提升价值
	}
	
	// 发现新内容影响
	discoveryRate := float64(existing.HasNewLinks+existing.HasForms+existing.HasAPIs) / 
		float64(existing.CrawledCount)
	
	if discoveryRate > 0.5 {
		adjustment += 10.0 // 经常发现新内容，价值高
	} else if discoveryRate < 0.1 {
		adjustment -= 5.0 // 很少发现新内容，价值低
	}
	
	// 响应时间影响
	if existing.AvgResponseTime > 5000 { // 超过5秒
		adjustment -= 3.0 // 响应慢，降低优先级
	}
	
	// 应用调整（使用学习率平滑）
	targetAdjustment := adjustment
	existing.ValueAdjustment += (targetAdjustment - existing.ValueAdjustment) * f.config.LearningRate
	
	// 限制调整范围在-20到+20之间
	if existing.ValueAdjustment < -20 {
		existing.ValueAdjustment = -20
	}
	if existing.ValueAdjustment > 20 {
		existing.ValueAdjustment = 20
	}
	
	f.stats.AdaptiveAdjustments++
}

// GetStatistics 获取过滤统计信息
func (f *BusinessAwareURLFilter) GetStatistics() FilterStatistics {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.stats
}

// GetTopPatterns 获取Top业务模式（按爬取次数排序）
func (f *BusinessAwareURLFilter) GetTopPatterns(limit int) []*URLBusinessPattern {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	
	patterns := make([]*URLBusinessPattern, 0, len(f.urlPatterns))
	for _, pattern := range f.urlPatterns {
		patterns = append(patterns, pattern)
	}
	
	// 按爬取次数排序
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].CrawledCount > patterns[j].CrawledCount
	})
	
	if limit > 0 && limit < len(patterns) {
		return patterns[:limit]
	}
	return patterns
}

// PrintReport 打印详细报告
func (f *BusinessAwareURLFilter) PrintReport() {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("                    业务感知URL过滤器 - 详细报告")
	fmt.Println(strings.Repeat("=", 80))
	
	// 总体统计
	fmt.Printf("\n【总体统计】\n")
	fmt.Printf("  处理URL总数:       %d\n", f.stats.TotalURLs)
	fmt.Printf("  允许爬取:          %d (%.1f%%)\n", f.stats.AllowedURLs, 
		float64(f.stats.AllowedURLs)/float64(f.stats.TotalURLs)*100)
	fmt.Printf("  智能过滤:          %d (%.1f%%)\n", f.stats.FilteredURLs,
		float64(f.stats.FilteredURLs)/float64(f.stats.TotalURLs)*100)
	fmt.Printf("  高价值URL:         %d\n", f.stats.HighValueURLs)
	fmt.Printf("  低价值URL:         %d\n", f.stats.LowValueURLs)
	
	if f.config.EnableAdaptiveLearning {
		fmt.Printf("  自适应调整次数:    %d\n", f.stats.AdaptiveAdjustments)
	}
	
	// 配置信息
	fmt.Printf("\n【过滤配置】\n")
	fmt.Printf("  最低业务分数:      %.1f\n", f.config.MinBusinessScore)
	fmt.Printf("  高价值阈值:        %.1f\n", f.config.HighValueThreshold)
	fmt.Printf("  低价值限制:        %d 个/模式\n", f.config.MaxSamePatternLowValue)
	fmt.Printf("  中等价值限制:      %d 个/模式\n", f.config.MaxSamePatternMidValue)
	fmt.Printf("  高价值限制:        %d 个/模式\n", f.config.MaxSamePatternHighValue)
	fmt.Printf("  自适应学习:        %v\n", f.config.EnableAdaptiveLearning)
	
	// Top模式
	fmt.Printf("\n【Top 10 业务模式】（按爬取次数）\n")
	fmt.Println(strings.Repeat("-", 80))
	
	topPatterns := f.GetTopPatterns(10)
	for i, pattern := range topPatterns {
		currentScore := pattern.BusinessScore + pattern.ValueAdjustment
		
		fmt.Printf("\n%d. 模式: %s\n", i+1, pattern.Pattern)
		fmt.Printf("   业务类型:  %s\n", pattern.BusinessType)
		fmt.Printf("   业务价值:  %.1f", pattern.BusinessScore)
		if pattern.ValueAdjustment != 0 {
			fmt.Printf(" (调整: %+.1f → 当前: %.1f)", pattern.ValueAdjustment, currentScore)
		}
		fmt.Println()
		fmt.Printf("   发现次数:  %d\n", pattern.SeenCount)
		fmt.Printf("   爬取次数:  %d\n", pattern.CrawledCount)
		fmt.Printf("   跳过次数:  %d\n", pattern.SkippedCount)
		
		if pattern.CrawledCount > 0 {
			// 计算成功率
			successCount := 0
			totalRequests := 0
			for code, count := range pattern.ResponseCodes {
				totalRequests += count
				if code >= 200 && code < 300 {
					successCount += count
				}
			}
			if totalRequests > 0 {
				successRate := float64(successCount) / float64(totalRequests) * 100
				fmt.Printf("   成功率:    %.1f%% (%d/%d)\n", successRate, successCount, totalRequests)
			}
			
			// 发现指标
			if pattern.HasNewLinks > 0 || pattern.HasForms > 0 || pattern.HasAPIs > 0 {
				fmt.Printf("   发现内容:  ")
				discoveries := make([]string, 0)
				if pattern.HasNewLinks > 0 {
					discoveries = append(discoveries, fmt.Sprintf("%d次新链接", pattern.HasNewLinks))
				}
				if pattern.HasForms > 0 {
					discoveries = append(discoveries, fmt.Sprintf("%d次表单", pattern.HasForms))
				}
				if pattern.HasAPIs > 0 {
					discoveries = append(discoveries, fmt.Sprintf("%d次API", pattern.HasAPIs))
				}
				fmt.Println(strings.Join(discoveries, ", "))
			}
			
			// 平均响应时间
			if pattern.AvgResponseTime > 0 {
				fmt.Printf("   平均响应:  %.0fms\n", pattern.AvgResponseTime)
			}
		}
		
		// 参数值样本
		if len(pattern.ParamValues) > 0 {
			fmt.Printf("   参数样本:  ")
			sampleStrs := make([]string, 0)
			for paramName, values := range pattern.ParamValues {
				if len(values) > 0 {
					if len(values) <= 3 {
						sampleStrs = append(sampleStrs, 
							fmt.Sprintf("%s=[%s]", paramName, strings.Join(values, ",")))
					} else {
						sampleStrs = append(sampleStrs, 
							fmt.Sprintf("%s=[%s...] (%d个)", paramName, values[0], len(values)))
					}
				}
			}
			if len(sampleStrs) > 0 {
				fmt.Println(strings.Join(sampleStrs, ", "))
			}
		}
	}
	
	fmt.Println("\n" + strings.Repeat("=", 80))
}

// ResetStatistics 重置统计信息（但保留学习到的模式）
func (f *BusinessAwareURLFilter) ResetStatistics() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	f.stats = FilterStatistics{}
	
	// 重置每个模式的统计，但保留学习到的价值调整
	for _, pattern := range f.urlPatterns {
		pattern.SeenCount = 0
		pattern.CrawledCount = 0
		pattern.SkippedCount = 0
		// 保留 ValueAdjustment（学习结果）
	}
}

