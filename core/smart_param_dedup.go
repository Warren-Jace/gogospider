package core

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// SmartParamDeduplicator 智能参数去重器
// 用于解决参数字段相同但值不同导致的大量重复爬取问题
type SmartParamDeduplicator struct {
	mutex       sync.RWMutex
	patterns    map[string]*PatternInfo
	maxPerGroup int  // 每个特征组最多爬取数量（默认3）
	enabled     bool // 是否启用智能去重
}

// PatternInfo URL模式信息
type PatternInfo struct {
	Pattern     string         // URL模式（不含参数值）: "http://test.com?id"
	ValueGroups map[string]int // 特征分类 -> 已爬取数量
	AllValues   []string       // 所有发现的参数值（用于调试）
}

// 参数值类型常量
const (
	ValueTypeNumeric1_5    = "num_1_5"   // 1-5位数字: 1, 123
	ValueTypeNumeric6_10   = "num_6_10"  // 6-10位数字: 123456, 1234567890
	ValueTypeNumeric11_20  = "num_11_20" // 11-20位数字
	ValueTypeNumeric20Plus = "num_20+"   // 20+位数字

	ValueTypeAlpha1_5    = "alpha_1_5"  // 1-5位字母: admin, test
	ValueTypeAlpha6_10   = "alpha_6_10" // 6-10位字母
	ValueTypeAlpha11Plus = "alpha_11+"  // 11+位字母

	ValueTypeAlphaNum = "alphanum" // 字母数字混合: abc123

	ValueTypeUUID   = "uuid"   // UUID格式
	ValueTypeMD5    = "md5"    // MD5哈希
	ValueTypeSHA1   = "sha1"   // SHA1哈希
	ValueTypeSHA256 = "sha256" // SHA256哈希

	ValueTypeBase64        = "base64"    // Base64编码
	ValueTypeHex           = "hex"       // 十六进制
	ValueTypePathTraversal = "path_trav" // 路径穿越
	ValueTypeSpecial       = "special"   // 特殊字符
	ValueTypeEmpty         = "empty"     // 空值

	ValueTypeOther = "other" // 其他
)

// NewSmartParamDeduplicator 创建智能参数去重器
func NewSmartParamDeduplicator(maxPerGroup int, enabled bool) *SmartParamDeduplicator {
	if maxPerGroup <= 0 {
		maxPerGroup = 3 // 默认每组最多3个
	}

	return &SmartParamDeduplicator{
		patterns:    make(map[string]*PatternInfo),
		maxPerGroup: maxPerGroup,
		enabled:     enabled,
	}
}

// ShouldCrawl 判断URL是否应该爬取
func (d *SmartParamDeduplicator) ShouldCrawl(rawURL string) (bool, string) {
	if !d.enabled {
		return true, "智能去重未启用"
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 1. 提取URL模式和参数值
	pattern, paramValue, err := d.extractURLPattern(rawURL)
	if err != nil {
		return true, fmt.Sprintf("URL解析失败: %v，默认爬取", err)
	}

	// 如果没有参数，直接允许爬取
	if paramValue == "" {
		return true, "无参数URL，允许爬取"
	}

	// 2. 如果是新模式，初始化并允许爬取
	if _, exists := d.patterns[pattern]; !exists {
		d.patterns[pattern] = &PatternInfo{
			Pattern:     pattern,
			ValueGroups: make(map[string]int),
			AllValues:   make([]string, 0),
		}
		d.patterns[pattern].AllValues = append(d.patterns[pattern].AllValues, paramValue)
		d.patterns[pattern].ValueGroups[d.classifyParamValue(paramValue)] = 1
		return true, fmt.Sprintf("新URL模式 '%s'，允许爬取", pattern)
	}

	// 3. 对参数值进行分类
	valueClass := d.classifyParamValue(paramValue)

	// 4. 检查该特征组的爬取数量
	info := d.patterns[pattern]
	currentCount := info.ValueGroups[valueClass]

	if currentCount >= d.maxPerGroup {
		// 已达到限制，跳过
		return false, fmt.Sprintf("URL模式 '%s' 的特征组 '%s' 已达限制 (%d/%d)，跳过",
			pattern, valueClass, currentCount, d.maxPerGroup)
	}

	// 5. 允许爬取，更新计数
	info.ValueGroups[valueClass]++
	info.AllValues = append(info.AllValues, paramValue)

	return true, fmt.Sprintf("URL模式 '%s' 的特征组 '%s' 计数 %d/%d，允许爬取",
		pattern, valueClass, currentCount+1, d.maxPerGroup)
}

// extractURLPattern 提取URL模式（不含参数值）
func (d *SmartParamDeduplicator) extractURLPattern(rawURL string) (pattern string, paramValue string, err error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", "", err
	}

	// 基础模式: 协议 + 主机 + 路径
	pattern = parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path

	// 如果有查询参数
	if parsedURL.RawQuery != "" {
		queryParams := parsedURL.Query()

		// 对参数键排序
		var paramKeys []string
		for key := range queryParams {
			paramKeys = append(paramKeys, key)
		}
		sort.Strings(paramKeys)

		// 模式包含参数名（不含值）
		pattern += "?" + strings.Join(paramKeys, "&")

		// 提取参数值（用于分类）
		var values []string
		for _, key := range paramKeys {
			values = append(values, queryParams.Get(key))
		}
		paramValue = strings.Join(values, "|")
	}

	return pattern, paramValue, nil
}

// classifyParamValue 对参数值进行分类
func (d *SmartParamDeduplicator) classifyParamValue(value string) string {
	length := len(value)

	// 空值
	if length == 0 {
		return ValueTypeEmpty
	}

	// UUID格式: 550e8400-e29b-41d4-a716-446655440000
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if uuidPattern.MatchString(strings.ToLower(value)) {
		return ValueTypeUUID
	}

	// MD5哈希: 32位十六进制
	md5Pattern := regexp.MustCompile(`^[a-f0-9]{32}$`)
	if md5Pattern.MatchString(strings.ToLower(value)) {
		return ValueTypeMD5
	}

	// SHA1哈希: 40位十六进制
	sha1Pattern := regexp.MustCompile(`^[a-f0-9]{40}$`)
	if sha1Pattern.MatchString(strings.ToLower(value)) {
		return ValueTypeSHA1
	}

	// SHA256哈希: 64位十六进制
	sha256Pattern := regexp.MustCompile(`^[a-f0-9]{64}$`)
	if sha256Pattern.MatchString(strings.ToLower(value)) {
		return ValueTypeSHA256
	}

	// 路径穿越: ../ 或 ..\
	if strings.Contains(value, "../") || strings.Contains(value, "..\\") || strings.Contains(value, "..%2F") {
		return ValueTypePathTraversal
	}

	// Base64编码（简单判断：只包含Base64字符且长度>10）
	base64Pattern := regexp.MustCompile(`^[A-Za-z0-9+/]+=*$`)
	if base64Pattern.MatchString(value) && length > 10 {
		return ValueTypeBase64
	}

	// 十六进制（全是hex字符且长度>6）
	hexPattern := regexp.MustCompile(`^[0-9a-fA-F]+$`)
	if hexPattern.MatchString(value) && length > 6 {
		return ValueTypeHex
	}

	// 纯数字
	numericPattern := regexp.MustCompile(`^\d+$`)
	if numericPattern.MatchString(value) {
		if length <= 5 {
			return ValueTypeNumeric1_5
		} else if length <= 10 {
			return ValueTypeNumeric6_10
		} else if length <= 20 {
			return ValueTypeNumeric11_20
		} else {
			return ValueTypeNumeric20Plus
		}
	}

	// 纯字母
	alphaPattern := regexp.MustCompile(`^[a-zA-Z]+$`)
	if alphaPattern.MatchString(value) {
		if length <= 5 {
			return ValueTypeAlpha1_5
		} else if length <= 10 {
			return ValueTypeAlpha6_10
		} else {
			return ValueTypeAlpha11Plus
		}
	}

	// 字母数字混合
	alphanumPattern := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if alphanumPattern.MatchString(value) {
		return ValueTypeAlphaNum
	}

	// 包含特殊字符
	return ValueTypeSpecial
}

// GetStatistics 获取统计信息
func (d *SmartParamDeduplicator) GetStatistics() map[string]interface{} {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["enabled"] = d.enabled
	stats["max_per_group"] = d.maxPerGroup
	stats["total_patterns"] = len(d.patterns)

	patternStats := make([]map[string]interface{}, 0)
	for pattern, info := range d.patterns {
		pstat := make(map[string]interface{})
		pstat["pattern"] = pattern
		pstat["total_values"] = len(info.AllValues)
		pstat["value_groups"] = info.ValueGroups

		// 计算跳过的数量
		totalCrawled := 0
		for _, count := range info.ValueGroups {
			totalCrawled += count
		}
		pstat["crawled"] = totalCrawled
		pstat["skipped"] = len(info.AllValues) - totalCrawled

		patternStats = append(patternStats, pstat)
	}

	stats["patterns"] = patternStats

	return stats
}

// PrintStatistics 打印统计信息
func (d *SmartParamDeduplicator) PrintStatistics() {
	stats := d.GetStatistics()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("           智能参数去重统计")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("状态:               %v\n", stats["enabled"])
	fmt.Printf("每组最大数量:       %d\n", stats["max_per_group"])
	fmt.Printf("URL模式总数:        %d\n", stats["total_patterns"])
	fmt.Println(strings.Repeat("-", 60))

	if patterns, ok := stats["patterns"].([]map[string]interface{}); ok {
		for i, pstat := range patterns {
			fmt.Printf("\n模式 %d: %s\n", i+1, pstat["pattern"])
			fmt.Printf("  发现值总数: %d\n", pstat["total_values"])
			fmt.Printf("  实际爬取:   %d\n", pstat["crawled"])
			fmt.Printf("  智能跳过:   %d\n", pstat["skipped"])

			if groups, ok := pstat["value_groups"].(map[string]int); ok {
				fmt.Println("  特征组分布:")
				for group, count := range groups {
					fmt.Printf("    - %s: %d个\n", group, count)
				}
			}
		}
	}

	fmt.Println(strings.Repeat("=", 60))
}

// Reset 重置去重器
func (d *SmartParamDeduplicator) Reset() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.patterns = make(map[string]*PatternInfo)
}

// SetMaxPerGroup 设置每组最大数量
func (d *SmartParamDeduplicator) SetMaxPerGroup(max int) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if max > 0 {
		d.maxPerGroup = max
	}
}

// Enable 启用智能去重
func (d *SmartParamDeduplicator) Enable() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.enabled = true
}

// Disable 禁用智能去重
func (d *SmartParamDeduplicator) Disable() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.enabled = false
}

// IsEnabled 检查是否启用
func (d *SmartParamDeduplicator) IsEnabled() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.enabled
}
