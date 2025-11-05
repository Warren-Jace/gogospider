package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Rule struct {
	Pattern     string `json:"pattern"`
	Severity    string `json:"severity"`
	Mask        bool   `json:"mask"`
	Description string `json:"description"`
}

type Config struct {
	Description string          `json:"description"`
	Version     string          `json:"version"`
	Rules       map[string]Rule `json:"rules"`
}

func main() {
	// 读取规则文件
	data, err := os.ReadFile("sensitive_rules.json")
	if err != nil {
		fmt.Printf("❌ 读取失败: %v\n", err)
		os.Exit(1)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("❌ 解析失败: %v\n", err)
		os.Exit(1)
	}

	// 优化规则
	fixed := 0
	for name, rule := range config.Rules {
		// 移除末尾的 \n
		if strings.HasSuffix(rule.Pattern, "\\n") {
			rule.Pattern = strings.TrimSuffix(rule.Pattern, "\\n")
			config.Rules[name] = rule
			fixed++
		}
	}

	// 保存
	output, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile("sensitive_rules.json", output, 0644); err != nil {
		fmt.Printf("❌ 保存失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 优化完成！\n")
	fmt.Printf("   总规则数: %d\n", len(config.Rules))
	fmt.Printf("   修复规则: %d (移除末尾\\n)\n", fixed)
}



