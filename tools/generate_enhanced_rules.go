package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type RExRepository struct {
	RegularExpressions []RExCategory `yaml:"regular_expresions"`
}

type RExCategory struct {
	Name    string     `yaml:"name"`
	Regexes []RExRegex `yaml:"regexes"`
}

type RExRegex struct {
	Name            string `yaml:"name"`
	Regex           string `yaml:"regex"`
	FalsePositives  bool   `yaml:"falsePositives"`
	CaseInsensitive bool   `yaml:"caseinsensitive"`
}

type GogoRule struct {
	Pattern     string `json:"pattern"`
	Severity    string `json:"severity"`
	Mask        bool   `json:"mask"`
	Description string `json:"description"`
}

type GogoRulesConfig struct {
	Description string              `json:"description"`
	Version     string              `json:"version"`
	Rules       map[string]GogoRule `json:"rules"`
}

func main() {
	fmt.Println("🚀 生成增强版规则...")

	// 加载 RExpository
	rexData, err := os.ReadFile("regex.yaml")
	if err != nil {
		fmt.Printf("❌ 读取 regex.yaml 失败: %v\n", err)
		os.Exit(1)
	}

	var rex RExRepository
	if err := yaml.Unmarshal(rexData, &rex); err != nil {
		fmt.Printf("❌ 解析 regex.yaml 失败: %v\n", err)
		os.Exit(1)
	}

	// 加载标准版
	stdData, err := os.ReadFile("sensitive_rules_standard.json")
	if err != nil {
		fmt.Printf("❌ 读取标准版规则失败: %v\n", err)
		os.Exit(1)
	}

	var existingRules map[string]interface{}
	if err := json.Unmarshal(stdData, &existingRules); err != nil {
		fmt.Printf("❌ 解析标准版规则失败: %v\n", err)
		os.Exit(1)
	}

	// 创建增强版
	enhanced := &GogoRulesConfig{
		Description: "GogoSpider 增强规则集 - 集成 RExpository 高价值规则",
		Version:     "4.0",
		Rules:       make(map[string]GogoRule),
	}

	// 复制标准版规则
	existingRulesMap := existingRules["rules"].(map[string]interface{})
	for name, ruleData := range existingRulesMap {
		if strings.HasPrefix(name, "_") {
			continue
		}
		ruleMap := ruleData.(map[string]interface{})
		enhanced.Rules[name] = GogoRule{
			Pattern:     ruleMap["pattern"].(string),
			Severity:    ruleMap["severity"].(string),
			Mask:        ruleMap["mask"].(bool),
			Description: ruleMap["description"].(string),
		}
	}

	// 添加高价值规则
	highValueRules := getHighValueRuleNames()
	addedCount := 0

	for _, category := range rex.RegularExpressions {
		for _, rule := range category.Regexes {
			if rule.FalsePositives {
				continue
			}

			shouldAdd, exists := highValueRules[rule.Name]
			if !exists || !shouldAdd {
				continue
			}

			if ruleExists(rule.Name, enhanced.Rules) {
				continue
			}

			pattern := rule.Regex
			if rule.CaseInsensitive {
				pattern = "(?i)" + pattern
			}

			enhanced.Rules[rule.Name] = GogoRule{
				Pattern:     pattern,
				Severity:    determineSeverity(rule.Name),
				Mask:        shouldMask(rule.Name),
				Description: fmt.Sprintf("[REx] %s", rule.Name),
			}
			addedCount++
		}
	}

	// 保存
	data, _ := json.MarshalIndent(enhanced, "", "  ")
	if err := os.WriteFile("sensitive_rules_enhanced.json", data, 0644); err != nil {
		fmt.Printf("❌ 保存失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 生成完成！\n")
	fmt.Printf("   标准版规则: %d\n", len(existingRulesMap)-countComments(existingRulesMap))
	fmt.Printf("   新增规则: %d\n", addedCount)
	fmt.Printf("   增强版总计: %d\n", len(enhanced.Rules))
}

func countComments(rules map[string]interface{}) int {
	count := 0
	for name := range rules {
		if strings.HasPrefix(name, "_") {
			count++
		}
	}
	return count
}

func getHighValueRuleNames() map[string]bool {
	return map[string]bool{
		"Github App Token":                           true,
		"Github OAuth Access Token":                  true,
		"Github Personal Access Token":               true,
		"Github Refresh Token":                       true,
		"GitHub Fine-Grained Personal Access Token":  true,
		"Gitlab Personal Access Token":               true,
		"GitLab Pipeline Trigger Token":              true,
		"GitLab Runner Registration Token":           true,
		"OpenAI API Token":                           true,
		"Npm Access Token":                           true,
		"PyPI upload token":                          true,
		"Heroku API Key":                             true,
		"Telegram Bot API Token":                     true,
		"Twilio API Key":                             true,
		"Sendgrid API Key":                           true,
		"Mailchimp API Key":                          true,
		"Mailgun API Key":                            true,
		"Cloudinary Basic Auth":                      true,
		"Postman API Key":                            true,
		"Grafana API Key":                            true,
		"Grafana cloud api token":                    true,
		"Grafana service account token":              true,
		"Alibaba Access Key ID":                      true,
		"Alibaba Secret Key":                         true,
		"Age Secret Key":                             true,
		"Doppler API Key":                            true,
		"Linear API Key":                             true,
		"PlanetScale OAuth token":                    true,
		"Planetscale API Key":                        true,
		"Planetscale Password":                       true,
		"Pulumi API Key":                             true,
		"Rubygem API Key":                            true,
		"Readme API token":                           true,
		"Sendinblue API Key":                         true,
		"Square Access Token":                        true,
		"Square API Key":                             true,
		"Typeform API Key":                           true,
		"EasyPost API Key":                           true,
		"EasyPost test API Key":                      true,
		"Dynatrace API Key":                          true,
		"Duffel API Key":                             true,
		"Frame.io API Key":                           true,
		"Mapbox API Key":                             true,
		"Microsoft Teams Webhook":                    true,
		"Clojars API Key":                            true,
		"Google Drive Oauth":                         true,
		"Google Oauth Access Token":                  true,
		"Google (GCP) Service-account":               true,
		"AWS MWS Key":                                true,
		"AWS AppSync GraphQL Key":                    true,
		"Hashicorp Terraform user/org API Key":       true,
		"Databricks API Key":                         true,
		"Prefect API token":                          true,
		"Airtable API Key":                           true,
		"Asana Client ID":                            true,
		"Atlassian API Key":                          true,
		"Dropbox API Key":                            true,
		"Facebook Access Token":                      true,
		"Facebook Client ID":                         true,
		"Facebook Oauth":                             true,
		"Facebook Secret Key":                        true,
		"LinkedIn Client ID":                         true,
		"LinkedIn Secret Key":                        true,
		"Twitter Client ID":                          true,
		"Twitter Bearer Token":                       true,
		"Twitter Oauth":                              true,
		"Twitter Secret Key":                         true,
		"Discord API Key, Client ID & Client Secret": true,
		"Plaid API Token":                            true,
		"PayPal Braintree Access Token":              true,
		"Basic Auth Credentials":                     true,
	}
}

func ruleExists(name string, rules map[string]GogoRule) bool {
	if _, exists := rules[name]; exists {
		return true
	}
	lowerName := strings.ToLower(name)
	for existingName := range rules {
		if strings.Contains(strings.ToLower(existingName), lowerName) ||
			strings.Contains(lowerName, strings.ToLower(existingName)) {
			return true
		}
	}
	return false
}

func determineSeverity(name string) string {
	name = strings.ToLower(name)
	highKeywords := []string{"secret", "password", "token", "key", "auth", "credential"}
	lowKeywords := []string{"client id", "username"}

	for _, kw := range highKeywords {
		if strings.Contains(name, kw) {
			return "HIGH"
		}
	}
	for _, kw := range lowKeywords {
		if strings.Contains(name, kw) {
			return "LOW"
		}
	}
	return "MEDIUM"
}

func shouldMask(name string) bool {
	name = strings.ToLower(name)
	noMaskKeywords := []string{"client id", "username", "email", "url"}

	for _, kw := range noMaskKeywords {
		if strings.Contains(name, kw) {
			return false
		}
	}
	return true
}

