package core

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// RExRepository 结构定义 - 对应 regex.yaml
type RExRepository struct {
	RegularExpressions []RExCategory `yaml:"regular_expresions"`
}

// RExCategory 规则分类
type RExCategory struct {
	Name    string      `yaml:"name"`
	Regexes []RExRegex  `yaml:"regexes"`
}

// RExRegex 单个正则规则
type RExRegex struct {
	Name             string `yaml:"name"`
	Regex            string `yaml:"regex"`
	Example          string `yaml:"example"`
	FalsePositives   bool   `yaml:"falsePositives"`
	CaseInsensitive  bool   `yaml:"caseinsensitive"`
	ExtraGrep        string `yaml:"extra_grep"`
}

// RExRepositoryAdapter RExpository适配器
// 将 regex.yaml 中的规则转换为 GogoSpider 格式
type RExRepositoryAdapter struct {
	yamlFile           string
	repository         *RExRepository
	skipFalsePositives bool // 是否跳过高误报规则
}

// NewRExRepositoryAdapter 创建适配器
func NewRExRepositoryAdapter(yamlFile string) *RExRepositoryAdapter {
	return &RExRepositoryAdapter{
		yamlFile:           yamlFile,
		skipFalsePositives: true, // 默认跳过高误报规则
	}
}

// SetSkipFalsePositives 设置是否跳过高误报规则
func (adapter *RExRepositoryAdapter) SetSkipFalsePositives(skip bool) {
	adapter.skipFalsePositives = skip
}

// LoadFromYAML 从 YAML 文件加载规则
func (adapter *RExRepositoryAdapter) LoadFromYAML() error {
	data, err := os.ReadFile(adapter.yamlFile)
	if err != nil {
		return fmt.Errorf("读取 YAML 文件失败: %v", err)
	}

	adapter.repository = &RExRepository{}
	if err := yaml.Unmarshal(data, adapter.repository); err != nil {
		return fmt.Errorf("解析 YAML 文件失败: %v", err)
	}

	return nil
}

// ConvertToGogoSpiderRules 转换为 GogoSpider 规则格式
// 返回可直接加载到 SensitiveInfoDetector 的规则配置
func (adapter *RExRepositoryAdapter) ConvertToGogoSpiderRules() (map[string]*SensitivePattern, error) {
	if adapter.repository == nil {
		return nil, fmt.Errorf("请先调用 LoadFromYAML() 加载规则")
	}

	rules := make(map[string]*SensitivePattern)
	loadedCount := 0
	skippedCount := 0

	for _, category := range adapter.repository.RegularExpressions {
		for _, rex := range category.Regexes {
			// 跳过高误报规则（可配置）
			if adapter.skipFalsePositives && rex.FalsePositives {
				skippedCount++
				continue
			}

			// 处理大小写不敏感
			regexPattern := rex.Regex
			if rex.CaseInsensitive {
				regexPattern = "(?i)" + regexPattern
			}

			// 编译正则表达式
			compiled, err := regexp.Compile(regexPattern)
			if err != nil {
				fmt.Printf("警告: [%s] %s 正则编译失败: %v\n", category.Name, rex.Name, err)
				continue
			}

			// 确定严重程度
			severity := adapter.determineSeverity(category.Name, rex.Name)

			// 确定是否需要脱敏
			mask := adapter.shouldMask(category.Name, rex.Name)

			// 创建规则名称 (分类 - 规则名)
			ruleName := fmt.Sprintf("[%s] %s", category.Name, rex.Name)

			rules[ruleName] = &SensitivePattern{
				Name:        ruleName,
				Pattern:     compiled,
				Severity:    severity,
				Mask:        mask,
				Description: rex.Example,
			}

			loadedCount++
		}
	}

	fmt.Printf("[RExpository] 加载完成: 成功 %d 条, 跳过 %d 条（高误报）\n", loadedCount, skippedCount)
	return rules, nil
}

// determineSeverity 根据分类和规则名称确定严重程度
func (adapter *RExRepositoryAdapter) determineSeverity(category string, name string) string {
	// 高危规则关键词
	highSeverityKeywords := []string{
		"private key", "secret key", "password", "api key", "access key",
		"token", "credential", "auth", "aws", "github", "slack",
		"stripe", "paypal", "ssh", "rsa", "pgp", "jwt",
	}

	// 低危规则关键词
	lowSeverityKeywords := []string{
		"email", "ip", "username", "url", "base64",
	}

	lowerName := strings.ToLower(name)
	lowerCategory := strings.ToLower(category)

	// 检查高危关键词
	for _, keyword := range highSeverityKeywords {
		if strings.Contains(lowerName, keyword) || strings.Contains(lowerCategory, keyword) {
			return "HIGH"
		}
	}

	// 检查低危关键词
	for _, keyword := range lowSeverityKeywords {
		if strings.Contains(lowerName, keyword) || strings.Contains(lowerCategory, keyword) {
			return "LOW"
		}
	}

	// 默认中危
	return "MEDIUM"
}

// shouldMask 确定是否需要脱敏
func (adapter *RExRepositoryAdapter) shouldMask(category string, name string) bool {
	// 不需要脱敏的关键词
	noMaskKeywords := []string{
		"email", "url", "domain", "ip", "username",
	}

	lowerName := strings.ToLower(name)

	for _, keyword := range noMaskKeywords {
		if strings.Contains(lowerName, keyword) {
			return false
		}
	}

	// 默认脱敏（安全第一）
	return true
}

// LoadIntoDetector 直接加载到检测器
func (adapter *RExRepositoryAdapter) LoadIntoDetector(detector *SensitiveInfoDetector) error {
	rules, err := adapter.ConvertToGogoSpiderRules()
	if err != nil {
		return err
	}

	// 合并规则到检测器
	for name, pattern := range rules {
		detector.patterns[name] = pattern
	}

	return nil
}

// GetStatistics 获取加载统计
func (adapter *RExRepositoryAdapter) GetStatistics() map[string]interface{} {
	if adapter.repository == nil {
		return map[string]interface{}{
			"error": "未加载规则",
		}
	}

	stats := make(map[string]interface{})
	totalRules := 0
	falsePositiveRules := 0
	categoryStats := make(map[string]int)

	for _, category := range adapter.repository.RegularExpressions {
		categoryCount := 0
		for _, rex := range category.Regexes {
			totalRules++
			categoryCount++
			if rex.FalsePositives {
				falsePositiveRules++
			}
		}
		categoryStats[category.Name] = categoryCount
	}

	stats["total_rules"] = totalRules
	stats["false_positive_rules"] = falsePositiveRules
	stats["categories"] = len(adapter.repository.RegularExpressions)
	stats["category_breakdown"] = categoryStats
	stats["skip_false_positives"] = adapter.skipFalsePositives

	return stats
}

// ExportCategoryNames 导出所有分类名称
func (adapter *RExRepositoryAdapter) ExportCategoryNames() []string {
	if adapter.repository == nil {
		return []string{}
	}

	names := make([]string, 0, len(adapter.repository.RegularExpressions))
	for _, category := range adapter.repository.RegularExpressions {
		names = append(names, category.Name)
	}

	return names
}

// ExportRulesByCategory 导出指定分类的规则
func (adapter *RExRepositoryAdapter) ExportRulesByCategory(categoryName string) []RExRegex {
	if adapter.repository == nil {
		return []RExRegex{}
	}

	for _, category := range adapter.repository.RegularExpressions {
		if category.Name == categoryName {
			return category.Regexes
		}
	}

	return []RExRegex{}
}

// PrintSummary 打印加载摘要
func (adapter *RExRepositoryAdapter) PrintSummary() {
	stats := adapter.GetStatistics()
	
	if _, ok := stats["error"]; ok {
		fmt.Println("❌ 未加载规则")
		return
	}

	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              RExpository 规则加载摘要                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n📊 规则统计:\n")
	fmt.Printf("  总规则数: %d\n", stats["total_rules"])
	fmt.Printf("  规则分类: %d\n", stats["categories"])
	fmt.Printf("  高误报规则: %d\n", stats["false_positive_rules"])
	fmt.Printf("  跳过高误报: %v\n", stats["skip_false_positives"])
	
	fmt.Printf("\n📂 分类明细:\n")
	categoryStats := stats["category_breakdown"].(map[string]int)
	for name, count := range categoryStats {
		fmt.Printf("  - %-30s: %d 条\n", name, count)
	}
	
	fmt.Println("\n" + strings.Repeat("═", 64))
}

