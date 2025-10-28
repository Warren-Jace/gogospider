package core

import (
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// URL过滤管理器使用示例
// ============================================================================

// ExampleBasicUsage 基础使用示例
func ExampleBasicUsage() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         示例1：基础使用（平衡模式）                           ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
	
	// 创建过滤管理器
	manager := NewURLFilterManagerWithPreset(PresetBalanced, "example.com")
	
	// 测试URL列表
	testURLs := []string{
		"https://example.com/",                    // 首页
		"https://example.com/api/users",           // API端点
		"https://example.com/admin/dashboard",     // 管理页面
		"https://example.com/static/logo.png",     // 静态资源
		"https://example.com/app.js",              // JS文件
		"https://external.com/page",               // 外部链接
		"function",                                 // JavaScript关键字
		"#ffffff",                                  // 颜色值
	}
	
	fmt.Println("开始测试URL过滤...\n")
	
	for i, url := range testURLs {
		result := manager.Filter(url, nil)
		
		icon := "✓"
		if !result.Allowed || result.Action != FilterAllow {
			icon = "✗"
		}
		
		actionStr := ""
		switch result.Action {
		case FilterAllow:
			actionStr = "[允许]"
		case FilterReject:
			actionStr = "[拒绝]"
		case FilterDegrade:
			actionStr = "[降级]"
		}
		
		fmt.Printf("%d. [%s] %s %s\n", i+1, icon, actionStr, url)
		fmt.Printf("   原因: %s\n", result.Reason)
		if result.Score > 0 {
			fmt.Printf("   分数: %.1f\n", result.Score)
		}
		fmt.Println()
	}
	
	// 打印统计
	manager.PrintStatistics()
}

// ExampleAPIOnlyMode API专用模式示例
func ExampleAPIOnlyMode() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         示例2：API专用模式                                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
	
	manager := NewURLFilterManagerWithPreset(PresetAPIOnly, "api.example.com")
	
	testURLs := []string{
		"https://api.example.com/v1/users",        // API - 允许
		"https://api.example.com/rest/products",   // API - 允许
		"https://api.example.com/about",           // 普通页面 - 拒绝
		"https://api.example.com/logo.png",        // 静态资源 - 拒绝
	}
	
	for _, url := range testURLs {
		if manager.ShouldCrawl(url) {
			fmt.Printf("✓ 允许爬取: %s\n", url)
		} else {
			result := manager.Filter(url, nil)
			fmt.Printf("✗ 拒绝爬取: %s (%s)\n", url, result.Reason)
		}
	}
}

// ExampleCustomBuilder 自定义构建示例
func ExampleCustomBuilder() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         示例3：自定义配置（构建器模式）                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
	
	manager := NewFilterManagerBuilder("example.com").
		WithMode(FilterModeBalanced).
		WithCaching(true, 10000).
		WithEarlyStop(true).
		WithTrace(true, 100). // 启用追踪用于调试
		AddBasicFormat().
		AddBlacklist().
		AddScope(ScopeFilterConfig{
			AllowSubdomains:    true,
			AllowHTTP:          true,
			AllowHTTPS:         true,
			ExternalLinkAction: FilterDegrade,
		}).
		AddTypeClassifier(TypeClassifierConfig{
			StaticResourceAction: FilterDegrade,
			JSFileAction:         FilterAllow,
			CSSFileAction:        FilterDegrade,
		}).
		AddBusinessValue(35.0, 75.0). // 自定义阈值
		Build()
	
	fmt.Println("自定义配置已创建")
	fmt.Printf("  - 模式: %s\n", manager.config.Mode)
	fmt.Printf("  - 缓存: %v (大小: %d)\n", manager.config.EnableCaching, manager.config.CacheSize)
	fmt.Printf("  - 早停: %v\n", manager.config.EnableEarlyStop)
	fmt.Printf("  - 追踪: %v\n", manager.config.EnableTrace)
	fmt.Printf("  - 过滤器数量: %d\n", len(manager.filters))
	
	// 测试
	testURL := "https://example.com/api/users"
	result := manager.Filter(testURL, nil)
	
	fmt.Printf("\n测试URL: %s\n", testURL)
	fmt.Printf("结果: %s\n", result.Reason)
	
	// 查看详细追踪
	explanation := manager.ExplainURL(testURL)
	fmt.Println("\n详细追踪:")
	fmt.Println(explanation)
}

// ExampleDynamicAdjustment 动态调整示例
func ExampleDynamicAdjustment() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         示例4：动态调整过滤器                                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
	
	manager := NewURLFilterManagerWithPreset(PresetBalanced, "example.com")
	
	// 初始状态
	fmt.Println("初始配置:")
	fmt.Printf("  - 启用的过滤器: %v\n", manager.ListFilters())
	
	testURL := "https://example.com/test"
	result1 := manager.Filter(testURL, nil)
	fmt.Printf("\n测试1: %s\n", result1.Reason)
	
	// 禁用黑名单过滤器
	fmt.Println("\n禁用黑名单过滤器...")
	manager.DisableFilter("Blacklist")
	
	result2 := manager.Filter(testURL, nil)
	fmt.Printf("测试2: %s\n", result2.Reason)
	
	// 切换到严格模式
	fmt.Println("\n切换到严格模式...")
	manager.SetMode(FilterModeStrict)
	
	result3 := manager.Filter(testURL, nil)
	fmt.Printf("测试3: %s\n", result3.Reason)
	
	// 查看统计
	manager.PrintStatistics()
}

// ExampleBatchFiltering 批量过滤示例
func ExampleBatchFiltering() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         示例5：批量过滤                                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
	
	manager := NewURLFilterManagerWithPreset(PresetBalanced, "example.com")
	
	// 模拟从页面提取的URL列表
	urls := []string{
		"https://example.com/",
		"https://example.com/about",
		"https://example.com/api/users",
		"https://example.com/api/products",
		"https://example.com/admin/login",
		"https://example.com/static/app.js",
		"https://example.com/images/logo.png",
		"https://example.com/css/style.css",
		"https://external.com/resource",
		"function",
		"margin",
		"#ffffff",
	}
	
	fmt.Printf("批量过滤 %d 个URL...\n\n", len(urls))
	
	results := manager.FilterBatch(urls, nil)
	
	// 分类统计
	allowed := 0
	rejected := 0
	degraded := 0
	
	for _, result := range results {
		switch result.Action {
		case FilterAllow:
			allowed++
		case FilterReject:
			rejected++
		case FilterDegrade:
			degraded++
		}
	}
	
	fmt.Printf("结果统计:\n")
	fmt.Printf("  - 允许爬取: %d\n", allowed)
	fmt.Printf("  - 拒绝:     %d\n", rejected)
	fmt.Printf("  - 降级:     %d\n", degraded)
	
	fmt.Println("\n详细结果:")
	for url, result := range results {
		switch result.Action {
		case FilterAllow:
			fmt.Printf("  ✓ [允许] %s\n", url)
		case FilterReject:
			fmt.Printf("  ✗ [拒绝] %s - %s\n", url, result.Reason)
		case FilterDegrade:
			fmt.Printf("  ⚠ [降级] %s - %s\n", url, result.Reason)
		}
	}
}

// ExampleModeComparison 模式对比示例
func ExampleModeComparison() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         示例6：不同模式对比                                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
	
	testURLs := []string{
		"https://example.com/api/users",
		"https://example.com/page?id=1",
		"https://example.com/logo.png",
		"https://external.com/page",
	}
	
	presets := []FilterPreset{PresetStrict, PresetBalanced, PresetLoose}
	
	for _, preset := range presets {
		manager := NewURLFilterManagerWithPreset(preset, "example.com")
		
		fmt.Printf("\n【%s 模式】\n", preset)
		fmt.Println(strings.Repeat("-", 60))
		
		allowed := 0
		rejected := 0
		degraded := 0
		
		for _, url := range testURLs {
			result := manager.Filter(url, nil)
			
			switch result.Action {
			case FilterAllow:
				allowed++
				fmt.Printf("  ✓ %s\n", url)
			case FilterReject:
				rejected++
				fmt.Printf("  ✗ %s (%s)\n", url, result.Reason)
			case FilterDegrade:
				degraded++
				fmt.Printf("  ⚠ %s (%s)\n", url, result.Reason)
			}
		}
		
		fmt.Printf("\n统计: 允许:%d, 拒绝:%d, 降级:%d\n", allowed, rejected, degraded)
	}
}

// ============================================================================
// 集成测试示例
// ============================================================================

// TestFilterIntegration 集成测试
func TestFilterIntegration(targetDomain string) {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         URL过滤管理器 - 集成测试                              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
	
	// 创建管理器
	manager := NewURLFilterManagerWithPreset(PresetBalanced, targetDomain)
	
	// 测试用例
	testCases := []struct {
		url          string
		expectAllow  bool
		expectAction FilterAction
		description  string
	}{
		{
			url:          fmt.Sprintf("https://%s/", targetDomain),
			expectAllow:  true,
			expectAction: FilterAllow,
			description:  "首页应该允许",
		},
		{
			url:          fmt.Sprintf("https://%s/api/users", targetDomain),
			expectAllow:  true,
			expectAction: FilterAllow,
			description:  "API端点应该允许",
		},
		{
			url:          "function",
			expectAllow:  false,
			expectAction: FilterReject,
			description:  "JavaScript关键字应该拒绝",
		},
		{
			url:          fmt.Sprintf("https://%s/logo.png", targetDomain),
			expectAllow:  true,
			expectAction: FilterDegrade,
			description:  "静态资源应该降级",
		},
		{
			url:          "https://external.com/page",
			expectAllow:  true,
			expectAction: FilterDegrade,
			description:  "外部链接应该降级",
		},
	}
	
	passed := 0
	failed := 0
	
	for i, tc := range testCases {
		result := manager.Filter(tc.url, nil)
		
		success := (result.Allowed == tc.expectAllow) && (result.Action == tc.expectAction)
		
		if success {
			passed++
			fmt.Printf("✓ 测试%d通过: %s\n", i+1, tc.description)
		} else {
			failed++
			fmt.Printf("✗ 测试%d失败: %s\n", i+1, tc.description)
			fmt.Printf("   期望: Allowed=%v, Action=%v\n", tc.expectAllow, tc.expectAction)
			fmt.Printf("   实际: Allowed=%v, Action=%v\n", result.Allowed, result.Action)
			fmt.Printf("   原因: %s\n", result.Reason)
		}
	}
	
	fmt.Printf("\n测试结果: %d/%d 通过\n", passed, len(testCases))
	
	if failed == 0 {
		fmt.Println("🎉 所有测试通过！")
	} else {
		fmt.Printf("⚠️  %d 个测试失败\n", failed)
	}
}

// ============================================================================
// 性能测试示例
// ============================================================================

// BenchmarkFilterPerformance 性能基准测试
func BenchmarkFilterPerformance() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         性能基准测试                                           ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
	
	// 测试URL
	testURL := "https://example.com/api/users?id=123&name=test"
	iterations := 10000
	
	// 测试1：无缓存
	manager1 := NewFilterManagerBuilder("example.com").
		WithCaching(false, 0).
		WithEarlyStop(false).
		AddBasicFormat().
		AddBlacklist().
		AddScope(ScopeFilterConfig{
			AllowSubdomains:    true,
			ExternalLinkAction: FilterDegrade,
		}).
		AddTypeClassifier(TypeClassifierConfig{
			StaticResourceAction: FilterDegrade,
			JSFileAction:         FilterAllow,
		}).
		AddBusinessValue(30.0, 70.0).
		Build()
	
	start := time.Now()
	for i := 0; i < iterations; i++ {
		manager1.Filter(testURL, nil)
	}
	duration1 := time.Since(start)
	
	// 测试2：有缓存
	manager2 := NewFilterManagerBuilder("example.com").
		WithCaching(true, 10000).
		WithEarlyStop(false).
		AddBasicFormat().
		AddBlacklist().
		AddScope(ScopeFilterConfig{
			AllowSubdomains:    true,
			ExternalLinkAction: FilterDegrade,
		}).
		AddTypeClassifier(TypeClassifierConfig{
			StaticResourceAction: FilterDegrade,
			JSFileAction:         FilterAllow,
		}).
		AddBusinessValue(30.0, 70.0).
		Build()
	
	start = time.Now()
	for i := 0; i < iterations; i++ {
		manager2.Filter(testURL, nil)
	}
	duration2 := time.Since(start)
	
	// 测试3：启用早停
	manager3 := NewFilterManagerBuilder("example.com").
		WithCaching(true, 10000).
		WithEarlyStop(true).
		AddBasicFormat().
		AddBlacklist().
		AddScope(ScopeFilterConfig{
			AllowSubdomains:    true,
			ExternalLinkAction: FilterReject, // 早停测试：拒绝会提前返回
		}).
		AddTypeClassifier(TypeClassifierConfig{
			StaticResourceAction: FilterDegrade,
			JSFileAction:         FilterAllow,
		}).
		AddBusinessValue(30.0, 70.0).
		Build()
	
	// 使用外部链接测试早停
	externalURL := "https://external.com/page"
	start = time.Now()
	for i := 0; i < iterations; i++ {
		manager3.Filter(externalURL, nil)
	}
	duration3 := time.Since(start)
	
	// 输出结果
	fmt.Printf("迭代次数: %d\n\n", iterations)
	
	fmt.Printf("测试1 - 无缓存, 无早停:\n")
	fmt.Printf("  总耗时: %v\n", duration1)
	fmt.Printf("  平均耗时: %v/次\n", duration1/time.Duration(iterations))
	
	fmt.Printf("\n测试2 - 有缓存, 无早停:\n")
	fmt.Printf("  总耗时: %v\n", duration2)
	fmt.Printf("  平均耗时: %v/次\n", duration2/time.Duration(iterations))
	fmt.Printf("  性能提升: %.1f%%\n", float64(duration1-duration2)/float64(duration1)*100)
	
	fmt.Printf("\n测试3 - 有缓存, 有早停:\n")
	fmt.Printf("  总耗时: %v\n", duration3)
	fmt.Printf("  平均耗时: %v/次\n", duration3/time.Duration(iterations))
	fmt.Printf("  性能提升: %.1f%%\n", float64(duration1-duration3)/float64(duration1)*100)
}

// ============================================================================
// 运行所有示例
// ============================================================================

// RunAllExamples 运行所有示例
func RunAllExamples() {
	ExampleBasicUsage()
	ExampleAPIOnlyMode()
	ExampleCustomBuilder()
	ExampleDynamicAdjustment()
	// BenchmarkFilterPerformance() // 性能测试可选
	
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         所有示例运行完成！                                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

