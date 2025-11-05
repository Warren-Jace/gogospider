package core

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// SensitiveInfo 敏感信息
type SensitiveInfo struct {
	Type       string // 类型
	Value      string // 值（脱敏后）
	FullValue  string // 完整值
	Location   string // 位置
	Severity   string // 严重程度: HIGH/MEDIUM/LOW
	SourceURL  string // 来源URL
	LineNumber int    // 行号
}

// SensitiveInfoDetector 敏感信息检测器
type SensitiveInfoDetector struct {
	patterns      map[string]*SensitivePattern
	findings      []*SensitiveInfo
	totalScanned  int
	totalFindings int
}

// SensitivePattern 敏感信息模式
type SensitivePattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Severity    string
	Mask        bool   // 是否需要脱敏
	Description string // 规则描述
}

// RuleConfig 外部规则配置文件结构
type RuleConfig struct {
	Rules map[string]RulePattern `json:"rules"`
}

// RulePattern 外部规则模式
type RulePattern struct {
	Pattern     string `json:"pattern"`
	Severity    string `json:"severity"`
	Mask        bool   `json:"mask"`
	Description string `json:"description"`
}

// NewSensitiveInfoDetector 创建敏感信息检测器
func NewSensitiveInfoDetector() *SensitiveInfoDetector {
	sid := &SensitiveInfoDetector{
		patterns: make(map[string]*SensitivePattern),
		findings: make([]*SensitiveInfo, 0),
	}
	
	sid.initializePatterns()
	
	return sid
}

// initializePatterns 初始化检测模式
// 🔧 v3.1: 移除所有内置规则，完全依赖外部配置文件
// 如果用户不提供规则文件，检测器将为空（不会进行任何检测）
func (sid *SensitiveInfoDetector) initializePatterns() {
	// 所有规则通过外部JSON文件加载
	// 使用 LoadRulesFromFile() 或 MergeRulesFromFile() 方法加载规则
	fmt.Println("[敏感信息] 等待加载外部规则文件...")
}

// addPattern 添加检测模式
func (sid *SensitiveInfoDetector) addPattern(name string, pattern *regexp.Regexp, severity string, mask bool) {
	sid.patterns[name] = &SensitivePattern{
		Name:     name,
		Pattern:  pattern,
		Severity: severity,
		Mask:     mask,
	}
}

// LoadRulesFromFile 从外部JSON文件加载规则
func (sid *SensitiveInfoDetector) LoadRulesFromFile(filename string) error {
	// 读取文件
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取规则文件失败: %v", err)
	}
	
	// 🔧 修复: 使用map[string]interface{}来处理混合类型的JSON
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("解析规则文件失败: %v", err)
	}
	
	// 获取rules字段
	rulesInterface, ok := rawConfig["rules"]
	if !ok {
		return fmt.Errorf("规则文件中未找到'rules'字段")
	}
	
	rulesMap, ok := rulesInterface.(map[string]interface{})
	if !ok {
		return fmt.Errorf("'rules'字段格式不正确")
	}
	
	// 清空现有规则
	sid.patterns = make(map[string]*SensitivePattern)
	
	// 加载新规则
	loadedCount := 0
	for name, ruleInterface := range rulesMap {
		// 跳过注释字段（以_开头或_comment开头）
		if strings.HasPrefix(name, "_comment") || strings.HasPrefix(name, "_") {
			continue
		}
		
		// 检查是否为字符串类型（注释）
		if _, ok := ruleInterface.(string); ok {
			continue
		}
		
		// 转换为RulePattern
		ruleMap, ok := ruleInterface.(map[string]interface{})
		if !ok {
			fmt.Printf("警告: 规则 '%s' 格式不正确，跳过\n", name)
			continue
		}
		
		// 提取规则字段
		pattern, _ := ruleMap["pattern"].(string)
		severity, _ := ruleMap["severity"].(string)
		mask, _ := ruleMap["mask"].(bool)
		description, _ := ruleMap["description"].(string)
		
		if pattern == "" {
			fmt.Printf("警告: 规则 '%s' 缺少pattern字段，跳过\n", name)
			continue
		}
		
		// 编译正则表达式
		regex, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Printf("警告: 规则 '%s' 的正则表达式编译失败: %v\n", name, err)
			continue
		}
		
		// 添加到检测器
		sid.patterns[name] = &SensitivePattern{
			Name:        name,
			Pattern:     regex,
			Severity:    severity,
			Mask:        mask,
			Description: description,
		}
		loadedCount++
	}
	
	fmt.Printf("[敏感规则] 从 %s 加载了 %d 条规则\n", filename, loadedCount)
	return nil
}

// MergeRulesFromFile 从外部JSON文件合并规则（不清空现有规则）
func (sid *SensitiveInfoDetector) MergeRulesFromFile(filename string) error {
	// 读取文件
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取规则文件失败: %v", err)
	}
	
	// 🔧 修复: 使用map[string]interface{}来处理混合类型的JSON
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("解析规则文件失败: %v", err)
	}
	
	// 获取rules字段
	rulesInterface, ok := rawConfig["rules"]
	if !ok {
		return fmt.Errorf("规则文件中未找到'rules'字段")
	}
	
	rulesMap, ok := rulesInterface.(map[string]interface{})
	if !ok {
		return fmt.Errorf("'rules'字段格式不正确")
	}
	
	// 合并规则
	loadedCount := 0
	for name, ruleInterface := range rulesMap {
		// 跳过注释字段
		if strings.HasPrefix(name, "_comment") || strings.HasPrefix(name, "_") {
			continue
		}
		
		// 检查是否为字符串类型（注释）
		if _, ok := ruleInterface.(string); ok {
			continue
		}
		
		// 转换为RulePattern
		ruleMap, ok := ruleInterface.(map[string]interface{})
		if !ok {
			fmt.Printf("警告: 规则 '%s' 格式不正确，跳过\n", name)
			continue
		}
		
		// 提取规则字段
		pattern, _ := ruleMap["pattern"].(string)
		severity, _ := ruleMap["severity"].(string)
		mask, _ := ruleMap["mask"].(bool)
		description, _ := ruleMap["description"].(string)
		
		if pattern == "" {
			fmt.Printf("警告: 规则 '%s' 缺少pattern字段，跳过\n", name)
			continue
		}
		
		// 编译正则表达式
		regex, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Printf("警告: 规则 '%s' 的正则表达式编译失败: %v\n", name, err)
			continue
		}
		
		// 添加到检测器（会覆盖同名规则）
		sid.patterns[name] = &SensitivePattern{
			Name:        name,
			Pattern:     regex,
			Severity:    severity,
			Mask:        mask,
			Description: description,
		}
		loadedCount++
	}
	
	fmt.Printf("[敏感规则] 从 %s 合并了 %d 条规则，当前共 %d 条规则\n", filename, loadedCount, len(sid.patterns))
	return nil
}

// Scan 扫描内容
func (sid *SensitiveInfoDetector) Scan(content string, sourceURL string) []*SensitiveInfo {
	sid.totalScanned++
	findings := make([]*SensitiveInfo, 0)
	
	// 分行处理，记录行号
	lines := strings.Split(content, "\n")
	
	for lineNum, line := range lines {
		for _, pattern := range sid.patterns {
			matches := pattern.Pattern.FindAllStringSubmatch(line, -1)
			
			for _, match := range matches {
				if len(match) > 0 {
					// 🔧 修复: 始终使用match[0]（完整匹配）作为敏感信息的完整值
					// 如果规则需要提取特定部分，应该在规则设计时使用非捕获组(?:...)
					fullValue := match[0]
					
					// 脱敏处理
					displayValue := fullValue
					if pattern.Mask {
						displayValue = sid.maskValue(fullValue)
					}
					
					info := &SensitiveInfo{
						Type:       pattern.Name,
						Value:      displayValue,  // 脱敏后的值
						FullValue:  fullValue,     // 完整的原始值
						Location:   fmt.Sprintf("Line %d", lineNum+1),
						Severity:   pattern.Severity,
						SourceURL:  sourceURL,
						LineNumber: lineNum + 1,
					}
					
					findings = append(findings, info)
					sid.totalFindings++
				}
			}
		}
	}
	
	// 保存到总findings
	sid.findings = append(sid.findings, findings...)
	
	return findings
}

// ScanResponse 扫描HTTP响应
func (sid *SensitiveInfoDetector) ScanResponse(content string, headers map[string][]string, sourceURL string) []*SensitiveInfo {
	allFindings := make([]*SensitiveInfo, 0)
	
	// 扫描响应体
	bodyFindings := sid.Scan(content, sourceURL)
	allFindings = append(allFindings, bodyFindings...)
	
	// 扫描响应头
	for headerName, headerValues := range headers {
		for _, headerValue := range headerValues {
			headerContent := headerName + ": " + headerValue
			headerFindings := sid.Scan(headerContent, sourceURL+" (Header)")
			allFindings = append(allFindings, headerFindings...)
		}
	}
	
	return allFindings
}

// maskValue 脱敏处理
func (sid *SensitiveInfoDetector) maskValue(value string) string {
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	
	// 显示前4位和后4位
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

// GetFindings 获取所有发现
func (sid *SensitiveInfoDetector) GetFindings() []*SensitiveInfo {
	return sid.findings
}

// GetFindingsByType 按类型获取发现
func (sid *SensitiveInfoDetector) GetFindingsByType(infoType string) []*SensitiveInfo {
	findings := make([]*SensitiveInfo, 0)
	
	for _, finding := range sid.findings {
		if finding.Type == infoType {
			findings = append(findings, finding)
		}
	}
	
	return findings
}

// GetFindingsBySeverity 按严重程度获取发现
func (sid *SensitiveInfoDetector) GetFindingsBySeverity(severity string) []*SensitiveInfo {
	findings := make([]*SensitiveInfo, 0)
	
	for _, finding := range sid.findings {
		if finding.Severity == severity {
			findings = append(findings, finding)
		}
	}
	
	return findings
}

// GetStatistics 获取统计信息
func (sid *SensitiveInfoDetector) GetStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["total_scanned"] = sid.totalScanned
	stats["total_findings"] = sid.totalFindings
	
	// 按严重程度统计
	highCount := len(sid.GetFindingsBySeverity("HIGH"))
	mediumCount := len(sid.GetFindingsBySeverity("MEDIUM"))
	lowCount := len(sid.GetFindingsBySeverity("LOW"))
	
	stats["high_severity"] = highCount
	stats["medium_severity"] = mediumCount
	stats["low_severity"] = lowCount
	
	// 按类型统计
	typeCount := make(map[string]int)
	for _, finding := range sid.findings {
		typeCount[finding.Type]++
	}
	stats["findings_by_type"] = typeCount
	
	return stats
}

// GenerateReport 生成报告
func (sid *SensitiveInfoDetector) GenerateReport() string {
	if len(sid.findings) == 0 {
		return "未发现敏感信息泄露"
	}
	
	var report strings.Builder
	
	report.WriteString("=== 敏感信息泄露检测报告 ===\n\n")
	
	// 高危发现
	highFindings := sid.GetFindingsBySeverity("HIGH")
	if len(highFindings) > 0 {
		report.WriteString(fmt.Sprintf("【高危】发现 %d 处高危敏感信息\n", len(highFindings)))
		for i, finding := range highFindings {
			if i >= 10 {
				report.WriteString(fmt.Sprintf("  ... 还有 %d 处高危发现\n", len(highFindings)-10))
				break
			}
			report.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, finding.Type))
			report.WriteString(fmt.Sprintf("      值: %s\n", finding.Value))
			report.WriteString(fmt.Sprintf("      位置: %s (%s)\n", finding.SourceURL, finding.Location))
		}
		report.WriteString("\n")
	}
	
	// 中危发现
	mediumFindings := sid.GetFindingsBySeverity("MEDIUM")
	if len(mediumFindings) > 0 {
		report.WriteString(fmt.Sprintf("【中危】发现 %d 处中危敏感信息\n", len(mediumFindings)))
		for i, finding := range mediumFindings {
			if i >= 5 {
				report.WriteString(fmt.Sprintf("  ... 还有 %d 处中危发现\n", len(mediumFindings)-5))
				break
			}
			report.WriteString(fmt.Sprintf("  [%d] %s: %s\n", i+1, finding.Type, finding.Value))
		}
		report.WriteString("\n")
	}
	
	// 低危发现（只显示数量）
	lowFindings := sid.GetFindingsBySeverity("LOW")
	if len(lowFindings) > 0 {
		report.WriteString(fmt.Sprintf("【低危】发现 %d 处低危敏感信息\n", len(lowFindings)))
		
		// 按类型统计
		typeCount := make(map[string]int)
		for _, finding := range lowFindings {
			typeCount[finding.Type]++
		}
		
		for infoType, count := range typeCount {
			report.WriteString(fmt.Sprintf("  - %s: %d个\n", infoType, count))
		}
	}
	
	return report.String()
}

// GetSummary 获取摘要
func (sid *SensitiveInfoDetector) GetSummary() string {
	highCount := len(sid.GetFindingsBySeverity("HIGH"))
	mediumCount := len(sid.GetFindingsBySeverity("MEDIUM"))
	lowCount := len(sid.GetFindingsBySeverity("LOW"))
	
	if sid.totalFindings == 0 {
		return "✅ 未发现敏感信息泄露"
	}
	
	return fmt.Sprintf("⚠️  发现 %d 处敏感信息 (高危:%d, 中危:%d, 低危:%d)", 
		sid.totalFindings, highCount, mediumCount, lowCount)
}

// Clear 清空发现记录
func (sid *SensitiveInfoDetector) Clear() {
	sid.findings = make([]*SensitiveInfo, 0)
	sid.totalScanned = 0
	sid.totalFindings = 0
}

// AddCustomPattern 添加自定义检测模式
func (sid *SensitiveInfoDetector) AddCustomPattern(name string, pattern string, severity string, mask bool) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	
	sid.addPattern(name, regex, severity, mask)
	return nil
}

// ExportFindings 导出发现（用于外部处理）
func (sid *SensitiveInfoDetector) ExportFindings() []map[string]interface{} {
	exports := make([]map[string]interface{}, 0)
	
	for _, finding := range sid.findings {
		export := make(map[string]interface{})
		export["type"] = finding.Type
		export["value"] = finding.Value          // 脱敏后的值
		export["full_value"] = finding.FullValue // 完整值
		export["location"] = finding.Location
		export["severity"] = finding.Severity
		export["source_url"] = finding.SourceURL
		export["line_number"] = finding.LineNumber
		
		exports = append(exports, export)
	}
	
	return exports
}

