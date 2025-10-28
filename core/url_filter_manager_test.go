package core

import (
	"fmt"
	"testing"
)

// ============================================================================
// 单元测试
// ============================================================================

// TestBasicFormatFilter 测试基础格式过滤器
func TestBasicFormatFilter(t *testing.T) {
	filter := NewBasicFormatFilter()
	ctx := &FilterContext{}
	
	tests := []struct {
		url      string
		expected bool
		reason   string
	}{
		{"", false, "空URL应该被拒绝"},
		{"javascript:alert(1)", false, "javascript协议应该被拒绝"},
		{"https://example.com", true, "正常URL应该通过"},
		{"http://test.com/page", true, "HTTP URL应该通过"},
	}
	
	for _, tt := range tests {
		result := filter.Filter(tt.url, ctx)
		if result.Allowed != tt.expected {
			t.Errorf("%s: 期望 %v, 得到 %v (原因: %s)", 
				tt.reason, tt.expected, result.Allowed, result.Reason)
		}
	}
}

// TestBlacklistFilter 测试黑名单过滤器
func TestBlacklistFilter(t *testing.T) {
	filter := NewBlacklistFilter()
	ctx := &FilterContext{}
	
	tests := []struct {
		url      string
		expected bool
		reason   string
	}{
		{"function", false, "JS关键字应该被拒绝"},
		{"margin", false, "CSS属性应该被拒绝"},
		{"https://example.com/api/get-user", true, "包含get但不完全等于应该通过"},
		{"https://example.com/margin-trading", true, "包含margin但不完全等于应该通过"},
		{"123", false, "纯数字应该被拒绝"},
		{"#ffffff", false, "颜色值应该被拒绝"},
	}
	
	for _, tt := range tests {
		result := filter.Filter(tt.url, ctx)
		if result.Allowed != tt.expected {
			t.Errorf("%s: URL=%s, 期望 %v, 得到 %v (原因: %s)", 
				tt.reason, tt.url, tt.expected, result.Allowed, result.Reason)
		}
	}
}

// TestScopeFilter 测试作用域过滤器
func TestScopeFilter(t *testing.T) {
	config := ScopeFilterConfig{
		TargetDomain:       "example.com",
		AllowSubdomains:    true,
		AllowHTTP:          true,
		AllowHTTPS:         true,
		ExternalLinkAction: FilterDegrade,
	}
	filter := NewScopeFilter(config)
	ctx := &FilterContext{}
	
	tests := []struct {
		url            string
		expectedAllow  bool
		expectedAction FilterAction
		reason         string
	}{
		{"https://example.com/page", true, FilterAllow, "目标域名应该允许"},
		{"https://api.example.com/users", true, FilterAllow, "子域名应该允许"},
		{"https://external.com/page", true, FilterDegrade, "外部链接应该降级"},
		{"/relative/path", true, FilterAllow, "相对路径应该允许"},
	}
	
	for _, tt := range tests {
		result := filter.Filter(tt.url, ctx)
		if result.Allowed != tt.expectedAllow {
			t.Errorf("%s: URL=%s, 期望Allowed=%v, 得到 %v", 
				tt.reason, tt.url, tt.expectedAllow, result.Allowed)
		}
		if result.Action != tt.expectedAction {
			t.Errorf("%s: URL=%s, 期望Action=%v, 得到 %v", 
				tt.reason, tt.url, tt.expectedAction, result.Action)
		}
	}
}

// TestTypeClassifierFilter 测试类型分类过滤器
func TestTypeClassifierFilter(t *testing.T) {
	config := TypeClassifierConfig{
		StaticResourceAction: FilterDegrade,
		JSFileAction:         FilterAllow,
		CSSFileAction:        FilterDegrade,
	}
	filter := NewTypeClassifierFilter(config)
	ctx := &FilterContext{}
	
	tests := []struct {
		url            string
		expectedAllow  bool
		expectedAction FilterAction
		reason         string
	}{
		{"https://example.com/page", true, FilterAllow, "普通页面应该允许"},
		{"https://example.com/app.js", true, FilterAllow, "JS文件应该允许"},
		{"https://example.com/style.css", true, FilterDegrade, "CSS文件应该降级"},
		{"https://example.com/logo.png", true, FilterDegrade, "图片应该降级"},
		{"https://example.com/api/data.json", true, FilterAllow, "JSON API应该允许"},
	}
	
	for _, tt := range tests {
		result := filter.Filter(tt.url, ctx)
		if result.Allowed != tt.expectedAllow {
			t.Errorf("%s: URL=%s, 期望Allowed=%v, 得到 %v (原因: %s)", 
				tt.reason, tt.url, tt.expectedAllow, result.Allowed, result.Reason)
		}
		if result.Action != tt.expectedAction {
			t.Errorf("%s: URL=%s, 期望Action=%v, 得到 %v (原因: %s)", 
				tt.reason, tt.url, tt.expectedAction, result.Action, result.Reason)
		}
	}
}

// TestBusinessValueFilter 测试业务价值过滤器
func TestBusinessValueFilter(t *testing.T) {
	filter := NewBusinessValueFilter(30.0, 70.0)
	ctx := &FilterContext{}
	
	tests := []struct {
		url           string
		expectedAllow bool
		minScore      float64
		reason        string
	}{
		{"https://example.com/admin/users", true, 70.0, "admin应该高分"},
		{"https://example.com/api/orders", true, 70.0, "API应该高分"},
		{"https://example.com/track/pixel", false, 0.0, "track应该低分"},
		{"https://example.com/page", true, 30.0, "普通页面应该中等分"},
	}
	
	for _, tt := range tests {
		result := filter.Filter(tt.url, ctx)
		if result.Allowed != tt.expectedAllow {
			t.Errorf("%s: URL=%s, 期望 %v, 得到 %v (分数: %.1f, 原因: %s)", 
				tt.reason, tt.url, tt.expectedAllow, result.Allowed, result.Score, result.Reason)
		}
		if tt.expectedAllow && result.Score < tt.minScore {
			t.Errorf("%s: URL=%s, 期望分数>%.1f, 得到 %.1f", 
				tt.reason, tt.url, tt.minScore, result.Score)
		}
	}
}

// ============================================================================
// 集成测试
// ============================================================================

// TestURLFilterManager 测试过滤管理器
func TestURLFilterManager(t *testing.T) {
	manager := NewURLFilterManagerWithPreset(PresetBalanced, "example.com")
	
	tests := []struct {
		url            string
		expectedAllow  bool
		expectedAction FilterAction
		description    string
	}{
		{
			url:            "https://example.com/",
			expectedAllow:  true,
			expectedAction: FilterAllow,
			description:    "首页应该允许",
		},
		{
			url:            "https://example.com/api/users",
			expectedAllow:  true,
			expectedAction: FilterAllow,
			description:    "API端点应该允许",
		},
		{
			url:            "function",
			expectedAllow:  false,
			expectedAction: FilterReject,
			description:    "JS关键字应该拒绝",
		},
		{
			url:            "https://example.com/logo.png",
			expectedAllow:  true,
			expectedAction: FilterDegrade,
			description:    "静态资源应该降级",
		},
		{
			url:            "https://external.com/page",
			expectedAllow:  true,
			expectedAction: FilterDegrade,
			description:    "外部链接应该降级",
		},
	}
	
	for i, tt := range tests {
		result := manager.Filter(tt.url, nil)
		
		if result.Allowed != tt.expectedAllow {
			t.Errorf("测试%d失败: %s\n  期望Allowed=%v, 得到=%v\n  URL=%s\n  原因=%s",
				i+1, tt.description, tt.expectedAllow, result.Allowed, tt.url, result.Reason)
		}
		
		if result.Action != tt.expectedAction {
			t.Errorf("测试%d失败: %s\n  期望Action=%v, 得到=%v\n  URL=%s\n  原因=%s",
				i+1, tt.description, tt.expectedAction, result.Action, tt.url, result.Reason)
		}
	}
	
	// 打印统计
	t.Log("测试完成，统计信息：")
	manager.PrintStatistics()
}

// TestFilterPipeline 测试过滤器管道顺序
func TestFilterPipeline(t *testing.T) {
	manager := NewURLFilterManager(FilterManagerConfig{
		Enabled:         true,
		EnableEarlyStop: false, // 不早停，检查所有过滤器
		EnableTrace:     true,  // 启用追踪
	})
	
	// 注册过滤器
	manager.RegisterFilter(NewBasicFormatFilter())
	manager.RegisterFilter(NewBlacklistFilter())
	
	// 测试一个会被黑名单拒绝的URL
	url := "function"
	result := manager.Filter(url, nil)
	
	// 检查追踪
	traces := manager.GetRecentTraces(1)
	if len(traces) == 0 {
		t.Fatal("应该有追踪记录")
	}
	
	trace := traces[0]
	
	// 验证执行了2个过滤器
	if len(trace.Steps) != 2 {
		t.Errorf("期望执行2个过滤器，实际执行 %d 个", len(trace.Steps))
	}
	
	// 验证执行顺序
	if trace.Steps[0].FilterName != "BasicFormat" {
		t.Errorf("第一个应该是BasicFormat，实际是 %s", trace.Steps[0].FilterName)
	}
	if trace.Steps[1].FilterName != "Blacklist" {
		t.Errorf("第二个应该是Blacklist，实际是 %s", trace.Steps[1].FilterName)
	}
	
	// 验证最终结果
	if result.Allowed {
		t.Error("function应该被拒绝")
	}
}

// TestEarlyStop 测试早停优化
func TestEarlyStop(t *testing.T) {
	// 创建管理器，启用早停
	manager := NewURLFilterManager(FilterManagerConfig{
		Enabled:         true,
		EnableEarlyStop: true,
		EnableTrace:     true,
	})
	
	manager.RegisterFilter(NewBasicFormatFilter())
	manager.RegisterFilter(NewBlacklistFilter())
	manager.RegisterFilter(NewBusinessValueFilter(30.0, 70.0))
	
	// 测试被黑名单拒绝的URL
	url := "function"
	result := manager.Filter(url, nil)
	
	// 检查追踪
	traces := manager.GetRecentTraces(1)
	if len(traces) == 0 {
		t.Fatal("应该有追踪记录")
	}
	
	trace := traces[0]
	
	// 启用早停时，应该在Blacklist就停止
	// 不应该执行BusinessValue
	if len(trace.Steps) > 2 {
		t.Errorf("早停优化失败：期望执行<=2个过滤器，实际执行 %d 个", len(trace.Steps))
		for i, step := range trace.Steps {
			t.Logf("  步骤%d: %s", i+1, step.FilterName)
		}
	}
}

// TestBatchFiltering 测试批量过滤
func TestBatchFiltering(t *testing.T) {
	manager := NewURLFilterManagerWithPreset(PresetBalanced, "example.com")
	
	urls := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"function",
		"https://example.com/logo.png",
	}
	
	results := manager.FilterBatch(urls, nil)
	
	if len(results) != len(urls) {
		t.Errorf("结果数量不匹配：期望 %d, 得到 %d", len(urls), len(results))
	}
	
	// 验证specific结果
	if !results["https://example.com/page1"].Allowed {
		t.Error("page1应该被允许")
	}
	
	if results["function"].Allowed {
		t.Error("function应该被拒绝")
	}
	
	if results["https://example.com/logo.png"].Action != FilterDegrade {
		t.Errorf("logo.png应该被降级，实际: %v", results["https://example.com/logo.png"].Action)
	}
}

// TestPresets 测试所有预设模式
func TestPresets(t *testing.T) {
	presets := []FilterPreset{
		PresetStrict,
		PresetBalanced,
		PresetLoose,
		PresetAPIOnly,
		PresetDeepScan,
	}
	
	testURL := "https://example.com/api/users"
	
	for _, preset := range presets {
		manager := NewURLFilterManagerWithPreset(preset, "example.com")
		result := manager.Filter(testURL, nil)
		
		// API端点在所有模式下都应该被允许
		if !result.Allowed {
			t.Errorf("预设%s: API端点应该被允许，但被拒绝了（原因: %s）", 
				preset, result.Reason)
		}
		
		t.Logf("预设%s: %s (分数: %.1f)", preset, result.Reason, result.Score)
	}
}

// TestAPIOnlyMode 测试API专用模式
func TestAPIOnlyMode(t *testing.T) {
	manager := NewURLFilterManagerWithPreset(PresetAPIOnly, "api.example.com")
	
	tests := []struct {
		url          string
		expectAllow  bool
		description  string
	}{
		{"https://api.example.com/v1/users", true, "API端点应该允许"},
		{"https://api.example.com/rest/orders", true, "REST API应该允许"},
		{"https://api.example.com/about", false, "非API页面应该拒绝"},
		{"https://api.example.com/logo.png", false, "静态资源应该拒绝"},
	}
	
	for _, tt := range tests {
		result := manager.Filter(tt.url, nil)
		if result.Allowed != tt.expectAllow {
			t.Errorf("%s: 期望 %v, 得到 %v (原因: %s)", 
				tt.description, tt.expectAllow, result.Allowed, result.Reason)
		}
	}
}

// ============================================================================
// 性能基准测试
// ============================================================================

// BenchmarkFilterSingle 单个URL过滤基准
func BenchmarkFilterSingle(b *testing.B) {
	manager := NewURLFilterManagerWithPreset(PresetBalanced, "example.com")
	url := "https://example.com/api/users?id=123"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Filter(url, nil)
	}
}

// BenchmarkFilterWithCaching 启用缓存的基准测试
func BenchmarkFilterWithCaching(b *testing.B) {
	manager := NewFilterManagerBuilder("example.com").
		WithCaching(true, 10000).
		WithEarlyStop(true).
		AddBasicFormat().
		AddBlacklist().
		AddScope(ScopeFilterConfig{
			AllowSubdomains:    true,
			ExternalLinkAction: FilterDegrade,
		}).
		Build()
	
	url := "https://example.com/api/users?id=123"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Filter(url, nil)
	}
}

// BenchmarkFilterEarlyStop 早停优化基准测试
func BenchmarkFilterEarlyStop(b *testing.B) {
	manager := NewFilterManagerBuilder("example.com").
		WithEarlyStop(true).
		AddBasicFormat().
		AddBlacklist().
		AddScope(ScopeFilterConfig{
			AllowSubdomains:    true,
			ExternalLinkAction: FilterReject, // 会触发早停
		}).
		AddTypeClassifier(TypeClassifierConfig{}).
		AddBusinessValue(30.0, 70.0).
		Build()
	
	// 使用外部链接测试（会在Scope被拒绝，早停）
	url := "https://external.com/page"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Filter(url, nil)
	}
}

// ============================================================================
// 示例运行函数
// ============================================================================

// RunAllTests 运行所有测试示例（手动调用）
func RunAllTests() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         URL过滤管理器 - 测试套件                              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
	
	// 运行集成测试
	TestFilterIntegration("example.com")
	
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("测试完成！")
	fmt.Println(strings.Repeat("=", 70))
}

