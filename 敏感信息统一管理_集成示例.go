package main

// =====================================================
// 敏感信息统一管理 - main.go集成示例
// GogoSpider v4.2
// =====================================================

import (
	"fmt"
	"log"
	"time"
	
	"spider-golang/config"
	"spider-golang/core"
)

// 示例1: 基本用法 - 使用统一导出
func example1_BasicUsage() {
	fmt.Println("=== 示例1: 基本用法 ===\n")
	
	// 1. 创建配置
	cfg := config.NewDefaultConfig()
	cfg.TargetURL = "https://testphp.vulnweb.com"
	cfg.SensitiveDetectionSettings.Enabled = true
	cfg.SensitiveDetectionSettings.RulesFile = "sensitive_rules_standard.json"
	
	// 2. 创建并启动爬虫
	spider := core.NewSpider(cfg)
	defer spider.Close()
	
	// 3. 加载敏感信息规则
	if err := spider.MergeSensitiveRules(cfg.SensitiveDetectionSettings.RulesFile); err != nil {
		log.Printf("⚠️  警告: 加载敏感规则失败: %v\n", err)
	}
	
	// 4. 开始爬取
	if err := spider.Start(cfg.TargetURL); err != nil {
		log.Fatalf("爬取失败: %v", err)
	}
	
	// 5. 生成输出文件名
	timestamp := time.Now().Format("20060102_150405")
	domain := extractDomain(cfg.TargetURL)
	baseFilename := fmt.Sprintf("sensitive_%s_%s", domain, timestamp)
	
	// 🆕 6. 使用统一导出（推荐方式）
	fmt.Println("\n📊 导出敏感信息报告...")
	if err := spider.ExportSensitiveInfoUnified(".", baseFilename); err != nil {
		log.Printf("统一导出敏感信息失败: %v", err)
	}
	
	fmt.Println("\n✅ 完成！请查看当前目录下的敏感信息报告文件")
}

// 示例2: 在现有main.go中的集成位置
func example2_IntegrationInMain() {
	fmt.Println("=== 示例2: main.go集成示例 ===\n")
	
	// ... [省略前面的代码：创建spider、爬取等] ...
	
	// 生成输出文件名
	timestamp := time.Now().Format("20060102_150405")
	domain := "example_com" // extractDomain(cfg.TargetURL)
	baseFilename := fmt.Sprintf("spider_%s_%s", domain, timestamp)
	
	// ========================================
	// 在保存其他结果后，添加这部分代码
	// ========================================
	
	// 文件1: 详细数据文件
	// saveDetailedResults(...)
	
	// 文件2: 所有链接
	// saveAllLinks(...)
	
	// 文件3: 范围内链接
	// saveInScopeLinks(...)
	
	// 🆕 文件4-8: 敏感信息统一导出（如果启用）
	enableSensitiveDetection := true // 从配置或命令行获取
	if enableSensitiveDetection {
		fmt.Println("\n📊 导出敏感信息报告（统一格式）...")
		
		// 🔧 新方式：一次调用导出所有格式
		// 这会生成5个文件：.txt, .json, .csv, .html, _summary.txt
		// if err := spider.ExportSensitiveInfoUnified(".", baseFilename); err != nil {
		// 	log.Printf("统一导出敏感信息失败: %v", err)
		// }
		
		// 🔧 旧方式（保持兼容，但已废弃）
		// sensitiveFile := baseFilename + "_sensitive.txt"
		// if err := spider.SaveSensitiveInfoToFile(sensitiveFile); err != nil {
		// 	log.Printf("保存敏感信息失败: %v", err)
		// }
		
		fmt.Println("✅ 敏感信息已统一导出")
	}
	
	fmt.Println("\n[+] 所有结果已保存到当前目录")
}

// 示例3: 批量扫描中的使用
func example3_BatchScan() {
	fmt.Println("=== 示例3: 批量扫描集成 ===\n")
	
	// 在批量扫描的每个URL处理完成后
	urls := []string{
		"https://example1.com",
		"https://example2.com",
	}
	
	for i, targetURL := range urls {
		fmt.Printf("\n[%d/%d] 扫描: %s\n", i+1, len(urls), targetURL)
		
		// 创建Spider并扫描...
		// cfg := config.NewDefaultConfig()
		// cfg.TargetURL = targetURL
		// spider := core.NewSpider(cfg)
		// spider.Start(targetURL)
		
		// 生成输出文件名
		timestamp := time.Now().Format("20060102_150405")
		domain := extractDomain(targetURL)
		baseFilename := fmt.Sprintf("batch_%s_%s", domain, timestamp)
		
		// 🆕 统一导出敏感信息
		// if err := spider.ExportSensitiveInfoUnified("./batch_results", baseFilename); err != nil {
		// 	log.Printf("导出敏感信息失败: %v", err)
		// }
		
		// 保存其他结果...
		fmt.Printf("✅ 完成扫描: %s\n", targetURL)
	}
}

// 示例4: 自定义配置
func example4_CustomConfiguration() {
	fmt.Println("=== 示例4: 自定义配置 ===\n")
	
	// 创建自定义配置的敏感信息管理器
	// spider := core.NewSpider(cfg)
	// ... 爬取 ...
	
	// 方式1: 直接使用Spider的统一导出（推荐）
	// spider.ExportSensitiveInfoUnified("./custom_output", "custom_name")
	
	// 方式2: 手动创建管理器（高级用法）
	/*
	manager := core.NewSensitiveInfoManager(core.SensitiveInfoManagerConfig{
		TargetDomain:  "example.com",
		OutputDir:     "./reports",
		BaseFilename:  "security_scan_2025",
		Detector:      spider.GetSensitiveDetector(),
	})
	
	// 收集并去重
	manager.CollectFindings()
	
	// 导出所有格式
	if err := manager.ExportAll(); err != nil {
		log.Printf("导出失败: %v", err)
	}
	
	// 或只导出特定格式
	manager.ExportHTML()  // 只导出HTML
	manager.ExportCSV()   // 只导出CSV
	*/
	
	fmt.Println("✅ 自定义配置导出完成")
}

// 示例5: 查看导出的文件
func example5_ViewResults() {
	fmt.Println("=== 示例5: 查看结果 ===\n")
	
	baseFilename := "sensitive_example_com_20251028_153000"
	
	fmt.Println("导出的文件：")
	fmt.Printf("1. %s.txt        - 详细文本报告（最完整）\n", baseFilename)
	fmt.Printf("2. %s.json       - JSON格式（程序化处理）\n", baseFilename)
	fmt.Printf("3. %s.csv        - CSV格式（Excel友好）\n", baseFilename)
	fmt.Printf("4. %s.html       - HTML报告（可视化）\n", baseFilename)
	fmt.Printf("5. %s_summary.txt - 快速摘要（推荐首先查看）\n", baseFilename)
	
	fmt.Println("\n推荐查看顺序：")
	fmt.Println("1️⃣  先看 _summary.txt 了解总体情况")
	fmt.Println("2️⃣  再看 .html 查看详细可视化报告")
	fmt.Println("3️⃣  用 .csv 在Excel中做数据分析")
	fmt.Println("4️⃣  用 .json 进行程序化处理或集成")
	
	fmt.Println("\n命令行查看：")
	fmt.Printf("  cat %s_summary.txt\n", baseFilename)
	fmt.Printf("  open %s.html        # macOS\n", baseFilename)
	fmt.Printf("  start %s.html       # Windows\n", baseFilename)
}

// 工具函数
func extractDomain(urlStr string) string {
	// 简化版本，实际应使用net/url解析
	domain := urlStr
	domain = removePrefix(domain, "http://")
	domain = removePrefix(domain, "https://")
	
	// 只取第一个斜杠之前的部分
	if idx := indexOf(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	
	// 替换冒号（端口号）
	domain = replaceAll(domain, ":", "_")
	
	return domain
}

func removePrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i <= len(s)-len(old) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}

// 主函数示例
func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║       敏感信息统一管理 - 集成示例（GogoSpider v4.2）          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝\n")
	
	fmt.Println("请选择示例：")
	fmt.Println("  1. 基本用法")
	fmt.Println("  2. main.go集成示例")
	fmt.Println("  3. 批量扫描集成")
	fmt.Println("  4. 自定义配置")
	fmt.Println("  5. 查看结果说明")
	fmt.Println()
	
	// 运行所有示例（仅展示代码，不实际执行）
	// example1_BasicUsage()
	example2_IntegrationInMain()
	// example3_BatchScan()
	// example4_CustomConfiguration()
	example5_ViewResults()
	
	fmt.Println("\n" + "═"*64)
	fmt.Println("💡 提示：这是集成示例代码，请根据需要复制到您的main.go中")
	fmt.Println("📖 详细文档：敏感信息统一管理_使用指南.md")
	fmt.Println("═"*64)
}

