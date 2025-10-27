package core

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// LayeredDeduplicator 分层去重器 - 针对不同类型URL使用不同策略
type LayeredDeduplicator struct {
	mutex sync.RWMutex
	
	// 不同层级的去重存储
	restfulURLs      map[string]bool            // RESTful路径: 保留所有变体
	ajaxAPIs         map[string]bool            // AJAX接口: 每个端点都保留
	fileParamURLs    map[string]*FileParamGroup // 文件参数: 保留编码差异
	normalURLs       map[string]bool            // 普通URL: 标准去重
	staticAssets     map[string]bool            // 静态资源: 保留（用于JS/CSS分析）
	postRequests     map[string]*POSTRequestInfo // POST请求: 完整去重
	
	// 统计信息
	stats LayeredDeduplicationStats
	
	// URL类型分类器
	classifier *URLTypeClassifier
}

// FileParamGroup 文件参数分组
type FileParamGroup struct {
	Pattern        string   // URL模式
	NormalEncoded  bool     // 是否有正常编码的
	URLEncoded     bool     // 是否有URL编码的
	PathTraversal  bool     // 是否有路径穿越的
	Samples        []string // 样本URL
}

// POSTRequestInfo POST请求信息
type POSTRequestInfo struct {
	URL        string
	Method     string
	Parameters map[string]string
	Hash       string
	Count      int
}

// LayeredDeduplicationStats 分层去重统计
type LayeredDeduplicationStats struct {
	TotalURLs           int
	RESTfulURLs         int
	AJAXAPIs            int
	FileParamURLs       int
	NormalURLs          int
	POSTRequests        int
	DuplicatePOSTs      int
	ParameterVariations int
	SavedRequests       int
}

// URLType URL类型枚举
type URLType int

const (
	URLTypeRESTful     URLType = iota // RESTful风格路径
	URLTypeAJAX                       // AJAX/API接口
	URLTypeFileParam                  // 包含文件参数
	URLTypeMultiParam                 // 多参数URL
	URLTypeStaticAsset                // 静态资源
	URLTypeNormal                     // 普通URL
)

// NewLayeredDeduplicator 创建分层去重器
func NewLayeredDeduplicator() *LayeredDeduplicator {
	return &LayeredDeduplicator{
		restfulURLs:   make(map[string]bool),
		ajaxAPIs:      make(map[string]bool),
		fileParamURLs: make(map[string]*FileParamGroup),
		normalURLs:    make(map[string]bool),
		staticAssets:  make(map[string]bool),
		postRequests:  make(map[string]*POSTRequestInfo),
		stats:         LayeredDeduplicationStats{},
		classifier:    NewURLTypeClassifier(),
	}
}

// ShouldProcess 判断URL是否应该处理（核心方法）
// 返回: (是否处理, URL类型, 原因)
func (d *LayeredDeduplicator) ShouldProcess(rawURL string, method string) (bool, URLType, string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.stats.TotalURLs++
	
	// 1. 识别URL类型
	urlType := d.classifier.ClassifyURL(rawURL)
	
	// 2. 根据类型使用不同的去重策略
	switch urlType {
	case URLTypeRESTful:
		return d.processRESTfulURL(rawURL)
		
	case URLTypeAJAX:
		return d.processAJAXURL(rawURL)
		
	case URLTypeFileParam:
		return d.processFileParamURL(rawURL)
		
	case URLTypeMultiParam:
		return d.processMultiParamURL(rawURL)
		
	case URLTypeStaticAsset:
		return d.processStaticAsset(rawURL)
		
	default:
		return d.processNormalURL(rawURL)
	}
}

// processRESTfulURL 处理RESTful风格URL - 保留所有路径变体
func (d *LayeredDeduplicator) processRESTfulURL(rawURL string) (bool, URLType, string) {
	// RESTful URL不做路径归一化，保留所有变体
	if d.restfulURLs[rawURL] {
		d.stats.SavedRequests++
		return false, URLTypeRESTful, "RESTful URL已存在"
	}
	
	d.restfulURLs[rawURL] = true
	d.stats.RESTfulURLs++
	return true, URLTypeRESTful, "新的RESTful URL，保留所有路径变体"
}

// processAJAXURL 处理AJAX/API接口 - 每个端点都保留
func (d *LayeredDeduplicator) processAJAXURL(rawURL string) (bool, URLType, string) {
	// AJAX接口不合并，每个端点都重要
	if d.ajaxAPIs[rawURL] {
		d.stats.SavedRequests++
		return false, URLTypeAJAX, "AJAX API已存在"
	}
	
	d.ajaxAPIs[rawURL] = true
	d.stats.AJAXAPIs++
	return true, URLTypeAJAX, "新的AJAX API端点"
}

// processFileParamURL 处理文件参数URL - 保留编码差异
func (d *LayeredDeduplicator) processFileParamURL(rawURL string) (bool, URLType, string) {
	// 提取基础模式（不含参数值）
	pattern := d.extractFileParamPattern(rawURL)
	
	// 检测编码类型
	encodingType := d.detectEncodingType(rawURL)
	
	// 检查是否已有该模式
	group, exists := d.fileParamURLs[pattern]
	if !exists {
		// 新模式，创建分组
		group = &FileParamGroup{
			Pattern: pattern,
			Samples: []string{},
		}
		d.fileParamURLs[pattern] = group
	}
	
	// 检查是否已有该编码类型
	shouldKeep := false
	reason := ""
	
	switch encodingType {
	case "normal":
		if !group.NormalEncoded {
			group.NormalEncoded = true
			shouldKeep = true
			reason = "保留正常编码样本"
		}
	case "urlencoded":
		if !group.URLEncoded {
			group.URLEncoded = true
			shouldKeep = true
			reason = "保留URL编码样本（可能触发不同解析逻辑）"
		}
	case "pathtraversal":
		if !group.PathTraversal {
			group.PathTraversal = true
			shouldKeep = true
			reason = "保留路径穿越样本（安全测试关键）"
		}
	}
	
	if shouldKeep {
		group.Samples = append(group.Samples, rawURL)
		d.stats.FileParamURLs++
		d.stats.ParameterVariations++
		return true, URLTypeFileParam, reason
	}
	
	d.stats.SavedRequests++
	return false, URLTypeFileParam, fmt.Sprintf("该编码类型已存在: %s", encodingType)
}

// processMultiParamURL 处理多参数URL - 保留参数组合差异
func (d *LayeredDeduplicator) processMultiParamURL(rawURL string) (bool, URLType, string) {
	// 提取参数结构（保留参数名和数量）
	paramStructure := d.extractParamStructure(rawURL)
	
	if d.normalURLs[paramStructure] {
		d.stats.SavedRequests++
		return false, URLTypeMultiParam, "相同参数结构已存在"
	}
	
	d.normalURLs[paramStructure] = true
	d.stats.NormalURLs++
	return true, URLTypeMultiParam, "新的参数组合，保留"
}

// processStaticAsset 处理静态资源
func (d *LayeredDeduplicator) processStaticAsset(rawURL string) (bool, URLType, string) {
	// 🔧 修复：静态资源也要存储，用于后续分析（JS/CSS可能包含API端点）
	if d.staticAssets[rawURL] {
		d.stats.SavedRequests++
		return false, URLTypeStaticAsset, "静态资源已存在"
	}
	
	d.staticAssets[rawURL] = true
	d.stats.NormalURLs++ // 计入普通URL统计
	return true, URLTypeStaticAsset, "静态资源，记录并保留（用于分析）"
}

// processNormalURL 处理普通URL - 标准去重
func (d *LayeredDeduplicator) processNormalURL(rawURL string) (bool, URLType, string) {
	// 标准URL模式去重
	pattern := d.extractURLPattern(rawURL)
	
	if d.normalURLs[pattern] {
		d.stats.SavedRequests++
		return false, URLTypeNormal, "URL模式已存在"
	}
	
	d.normalURLs[pattern] = true
	d.stats.NormalURLs++
	return true, URLTypeNormal, "新的URL模式"
}

// ProcessPOSTRequest 处理POST请求 - 完整去重
func (d *LayeredDeduplicator) ProcessPOSTRequest(postReq POSTRequest) (bool, string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	// 计算POST请求的唯一hash
	hash := d.calculatePOSTHash(postReq)
	
	if info, exists := d.postRequests[hash]; exists {
		// 已存在，更新计数
		info.Count++
		d.stats.DuplicatePOSTs++
		return false, fmt.Sprintf("POST请求重复（已出现%d次）", info.Count)
	}
	
	// 新POST请求
	d.postRequests[hash] = &POSTRequestInfo{
		URL:        postReq.URL,
		Method:     postReq.Method,
		Parameters: postReq.Parameters,
		Hash:       hash,
		Count:      1,
	}
	d.stats.POSTRequests++
	return true, "新的POST请求"
}

// ===== 辅助方法 =====

// extractFileParamPattern 提取文件参数模式
func (d *LayeredDeduplicator) extractFileParamPattern(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	
	// 基础部分
	pattern := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	
	// 只保留参数名
	if parsedURL.RawQuery != "" {
		query := parsedURL.Query()
		paramNames := make([]string, 0, len(query))
		for paramName := range query {
			paramNames = append(paramNames, paramName)
		}
		sort.Strings(paramNames)
		
		if len(paramNames) > 0 {
			pattern += "?" + strings.Join(paramNames, "&")
		}
	}
	
	return pattern
}

// detectEncodingType 检测编码类型
func (d *LayeredDeduplicator) detectEncodingType(rawURL string) string {
	// 检测路径穿越
	if strings.Contains(rawURL, "../") || strings.Contains(rawURL, "..\\") {
		return "pathtraversal"
	}
	
	// 检测URL编码
	if strings.Contains(rawURL, "%2F") || strings.Contains(rawURL, "%5C") || 
	   strings.Contains(rawURL, "%2E") {
		return "urlencoded"
	}
	
	return "normal"
}

// extractParamStructure 提取参数结构
func (d *LayeredDeduplicator) extractParamStructure(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	
	base := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	
	if parsedURL.RawQuery != "" {
		query := parsedURL.Query()
		paramNames := make([]string, 0, len(query))
		for paramName := range query {
			paramNames = append(paramNames, paramName+"=")
		}
		sort.Strings(paramNames)
		
		if len(paramNames) > 0 {
			base += "?" + strings.Join(paramNames, "&")
		}
	}
	
	return base
}

// extractURLPattern 提取URL模式（标准方法）
func (d *LayeredDeduplicator) extractURLPattern(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	
	pattern := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	
	if parsedURL.RawQuery != "" {
		query := parsedURL.Query()
		paramNames := make([]string, 0, len(query))
		for paramName := range query {
			paramNames = append(paramNames, paramName)
		}
		sort.Strings(paramNames)
		
		paramParts := make([]string, 0, len(paramNames))
		for _, paramName := range paramNames {
			paramParts = append(paramParts, paramName+"=")
		}
		
		if len(paramParts) > 0 {
			pattern += "?" + strings.Join(paramParts, "&")
		}
	}
	
	return pattern
}

// calculatePOSTHash 计算POST请求的hash
func (d *LayeredDeduplicator) calculatePOSTHash(postReq POSTRequest) string {
	// 构建唯一标识：URL + Method + 排序后的参数
	paramKeys := make([]string, 0, len(postReq.Parameters))
	for key := range postReq.Parameters {
		paramKeys = append(paramKeys, key)
	}
	sort.Strings(paramKeys)
	
	paramPairs := make([]string, 0, len(paramKeys))
	for _, key := range paramKeys {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", key, postReq.Parameters[key]))
	}
	
	identifier := fmt.Sprintf("%s|%s|%s", 
		postReq.URL, 
		postReq.Method,
		strings.Join(paramPairs, "&"))
	
	// 计算MD5
	hasher := md5.New()
	hasher.Write([]byte(identifier))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetStatistics 获取统计信息
func (d *LayeredDeduplicator) GetStatistics() LayeredDeduplicationStats {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.stats
}

// GetUniquePOSTRequests 获取去重后的POST请求列表
func (d *LayeredDeduplicator) GetUniquePOSTRequests() []POSTRequestInfo {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	unique := make([]POSTRequestInfo, 0, len(d.postRequests))
	for _, info := range d.postRequests {
		unique = append(unique, *info)
	}
	
	return unique
}

// PrintStatistics 打印统计信息
func (d *LayeredDeduplicator) PrintStatistics() {
	stats := d.GetStatistics()
	
	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("  分层去重统计")
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("总URL数量: %d\n", stats.TotalURLs)
	fmt.Printf("  - RESTful路径: %d (保留所有变体)\n", stats.RESTfulURLs)
	fmt.Printf("  - AJAX/API接口: %d (每个端点独立)\n", stats.AJAXAPIs)
	fmt.Printf("  - 文件参数URL: %d (保留编码差异)\n", stats.FileParamURLs)
	fmt.Printf("  - 多参数URL: %d\n", stats.NormalURLs)
	fmt.Printf("  - POST请求: %d (去重后)\n", stats.POSTRequests)
	fmt.Printf("  - POST重复: %d 次\n", stats.DuplicatePOSTs)
	fmt.Printf("参数变体: %d\n", stats.ParameterVariations)
	fmt.Printf("节省请求: %d 个\n", stats.SavedRequests)
	
	if stats.TotalURLs > 0 {
		actualDeduped := stats.SavedRequests
		effectiveRate := float64(actualDeduped) / float64(stats.TotalURLs) * 100
		fmt.Printf("有效去重率: %.1f%%\n", effectiveRate)
	}
	
	fmt.Println(strings.Repeat("═", 60))
}

