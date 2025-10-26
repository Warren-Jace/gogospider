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
func (sid *SensitiveInfoDetector) initializePatterns() {
	// === AWS相关 ===
	sid.addPattern("AWS Access Key", 
		regexp.MustCompile(`(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}`),
		"HIGH", true)
	
	sid.addPattern("AWS Secret Key",
		regexp.MustCompile(`(?i)aws(.{0,20})?(?-i)['\"][0-9a-zA-Z/+]{40}['\"]`),
		"HIGH", true)
	
	sid.addPattern("AWS S3 Bucket",
		regexp.MustCompile(`[a-z0-9.-]+\.s3\.amazonaws\.com`),
		"MEDIUM", false)
	
	sid.addPattern("AWS S3 Bucket URL",
		regexp.MustCompile(`s3://[a-z0-9.-]+`),
		"MEDIUM", false)
	
	// === Google Cloud ===
	sid.addPattern("Google API Key",
		regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
		"HIGH", true)
	
	sid.addPattern("Google OAuth",
		regexp.MustCompile(`[0-9]+-[0-9A-Za-z_]{32}\.apps\.googleusercontent\.com`),
		"HIGH", true)
	
	// === 阿里云 ===
	sid.addPattern("阿里云AccessKey",
		regexp.MustCompile(`(?i)(aliyun|alibaba)(.{0,20})?(?-i)['\"]?[A-Z0-9]{16,24}['\"]?`),
		"HIGH", true)
	
	sid.addPattern("阿里云OSS",
		regexp.MustCompile(`[a-z0-9.-]+\.oss-[a-z0-9-]+\.aliyuncs\.com`),
		"MEDIUM", false)
	
	// === 腾讯云 ===
	sid.addPattern("腾讯云SecretId",
		regexp.MustCompile(`(?i)(tencent|qcloud)(.{0,20})?(?-i)['\"]?[A-Z0-9]{32,40}['\"]?`),
		"HIGH", true)
	
	sid.addPattern("腾讯云COS",
		regexp.MustCompile(`[a-z0-9.-]+\.cos\.[a-z0-9-]+\.myqcloud\.com`),
		"MEDIUM", false)
	
	// === API Keys ===
	sid.addPattern("Generic API Key",
		regexp.MustCompile(`(?i)(api[_-]?key|apikey|api[_-]?secret)['"]?\s*[:=]\s*['"]?([a-zA-Z0-9_\-]{20,})`),
		"HIGH", true)
	
	sid.addPattern("Authorization Token",
		regexp.MustCompile(`(?i)(authorization|auth[_-]?token)['"]?\s*[:=]\s*['"]?([a-zA-Z0-9_\-\.]{20,})`),
		"MEDIUM", true)
	
	// === 密钥文件 ===
	sid.addPattern("Private Key (RSA)",
		regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE KEY-----`),
		"HIGH", true)
	
	sid.addPattern("Private Key (EC)",
		regexp.MustCompile(`-----BEGIN EC PRIVATE KEY-----`),
		"HIGH", true)
	
	sid.addPattern("PGP Private Key",
		regexp.MustCompile(`-----BEGIN PGP PRIVATE KEY BLOCK-----`),
		"HIGH", true)
	
	// === 证书 ===
	sid.addPattern("Certificate",
		regexp.MustCompile(`-----BEGIN CERTIFICATE-----`),
		"LOW", false)
	
	// === JWT Token ===
	sid.addPattern("JWT Token",
		regexp.MustCompile(`eyJ[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*`),
		"MEDIUM", true)
	
	// === 数据库连接 ===
	sid.addPattern("MySQL Connection String",
		regexp.MustCompile(`(?i)(mysql|mariadb)://[^:]+:[^@]+@[^/]+`),
		"HIGH", true)
	
	sid.addPattern("PostgreSQL Connection String",
		regexp.MustCompile(`(?i)postgres(ql)?://[^:]+:[^@]+@[^/]+`),
		"HIGH", true)
	
	sid.addPattern("MongoDB Connection String",
		regexp.MustCompile(`(?i)mongodb(\+srv)?://[^:]+:[^@]+@[^/]+`),
		"HIGH", true)
	
	// === 密码 ===
	sid.addPattern("Password in URL",
		regexp.MustCompile(`(?i)[a-z]+://[^:]+:([^@]{6,})@`),
		"HIGH", true)
	
	sid.addPattern("Password Config",
		regexp.MustCompile(`(?i)(password|passwd|pwd)['"]?\s*[:=]\s*['"]([^'"\s]{6,})`),
		"MEDIUM", true)
	
	// === 邮箱 ===
	sid.addPattern("Email Address",
		regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		"LOW", false)
	
	// === 手机号（中国）===
	sid.addPattern("Chinese Phone Number",
		regexp.MustCompile(`1[3-9]\d{9}`),
		"LOW", true)
	
	// === 身份证号（中国）===
	sid.addPattern("Chinese ID Card",
		regexp.MustCompile(`[1-9]\d{5}(18|19|20)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]`),
		"HIGH", true)
	
	// === IP地址 ===
	sid.addPattern("Internal IP Address",
		regexp.MustCompile(`(10\.\d{1,3}\.\d{1,3}\.\d{1,3})|(172\.(1[6-9]|2\d|3[0-1])\.\d{1,3}\.\d{1,3})|(192\.168\.\d{1,3}\.\d{1,3})`),
		"MEDIUM", false)
	
	// === 配置文件 ===
	sid.addPattern("Database Host",
		regexp.MustCompile(`(?i)(db[_-]?host|database[_-]?host)['"]?\s*[:=]\s*['"]?([a-zA-Z0-9.-]+)`),
		"MEDIUM", false)
	
	sid.addPattern("Redis Connection",
		regexp.MustCompile(`(?i)redis://[^:]+:[^@]+@[^/]+`),
		"HIGH", true)
	
	// === 社交媒体/第三方服务 ===
	sid.addPattern("GitHub Token",
		regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
		"HIGH", true)
	
	sid.addPattern("Slack Token",
		regexp.MustCompile(`xox[pborsa]-[0-9]{12}-[0-9]{12}-[0-9]{12}-[a-z0-9]{32}`),
		"HIGH", true)
	
	sid.addPattern("Stripe API Key",
		regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24}`),
		"HIGH", true)
	
	sid.addPattern("PayPal Braintree Token",
		regexp.MustCompile(`access_token\$production\$[0-9a-z]{16}\$[0-9a-f]{32}`),
		"HIGH", true)
	
	// === 其他敏感信息 ===
	sid.addPattern("Base64 Encoded Data (Potential Secret)",
		regexp.MustCompile(`(?i)(secret|key|token|password)['"]?\s*[:=]\s*['"]?([A-Za-z0-9+/]{40,}={0,2})`),
		"MEDIUM", true)
	
	sid.addPattern("Hex Encoded Secret",
		regexp.MustCompile(`(?i)(secret|key|token)['"]?\s*[:=]\s*['"]?([a-fA-F0-9]{32,})`),
		"MEDIUM", true)
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
	
	// 解析JSON
	var config RuleConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析规则文件失败: %v", err)
	}
	
	// 清空现有规则（可选）
	sid.patterns = make(map[string]*SensitivePattern)
	
	// 加载新规则
	loadedCount := 0
	for name, rule := range config.Rules {
		// 编译正则表达式
		regex, err := regexp.Compile(rule.Pattern)
		if err != nil {
			fmt.Printf("警告: 规则 '%s' 的正则表达式编译失败: %v\n", name, err)
			continue
		}
		
		// 添加到检测器
		sid.patterns[name] = &SensitivePattern{
			Name:        name,
			Pattern:     regex,
			Severity:    rule.Severity,
			Mask:        rule.Mask,
			Description: rule.Description,
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
	
	// 解析JSON
	var config RuleConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析规则文件失败: %v", err)
	}
	
	// 合并规则
	loadedCount := 0
	for name, rule := range config.Rules {
		// 编译正则表达式
		regex, err := regexp.Compile(rule.Pattern)
		if err != nil {
			fmt.Printf("警告: 规则 '%s' 的正则表达式编译失败: %v\n", name, err)
			continue
		}
		
		// 添加到检测器（会覆盖同名规则）
		sid.patterns[name] = &SensitivePattern{
			Name:        name,
			Pattern:     regex,
			Severity:    rule.Severity,
			Mask:        rule.Mask,
			Description: rule.Description,
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
					fullValue := match[0]
					
					// 如果有捕获组，使用最后一个捕获组作为值
					if len(match) > 1 {
						fullValue = match[len(match)-1]
					}
					
					// 脱敏处理
					displayValue := fullValue
					if pattern.Mask {
						displayValue = sid.maskValue(fullValue)
					}
					
					info := &SensitiveInfo{
						Type:       pattern.Name,
						Value:      displayValue,
						FullValue:  fullValue,
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
		export["value"] = finding.Value
		export["location"] = finding.Location
		export["severity"] = finding.Severity
		export["source_url"] = finding.SourceURL
		export["line_number"] = finding.LineNumber
		
		exports = append(exports, export)
	}
	
	return exports
}

