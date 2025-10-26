package core

import (
	"testing"
)

// TestSmartURLValidator_BusinessURLs 测试业务URL应该通过
func TestSmartURLValidator_BusinessURLs(t *testing.T) {
	v := NewSmartURLValidator()

	// 这些URL在旧版本中被误杀，但在新版本中应该通过
	validURLs := []struct {
		url    string
		reason string
	}{
		{"http://example.com/api/users", "包含api但不是JS关键字"},
		{"http://example.com/admin/config", "包含admin但不是JS关键字"},
		{"http://example.com/user/profile", "包含user但不是JS关键字"},
		{"http://example.com/search?q=test", "包含search但不是JS关键字"},
		{"http://example.com/home", "包含home但不是JS关键字"},
		{"http://example.com/application_list", "包含application但不是MIME类型"},
		{"http://example.com/text/editor", "包含text但不是MIME类型"},
		{"http://example.com/api/json/export", "包含json但不是MIME类型"},
		{"http://example.com/ui/ly_harbor/home", "真实业务路径"},
		{"http://example.com/v1/api/data", "版本化API路径"},
		{"http://example.com/ws", "WebSocket路径（短路径）"},
		{"http://example.com/doc", "文档路径"},
		{"http://x.lydaas.com/api/document/portal_banner_advertising_query", "真实的长API路径"},
		{"http://x.lydaas.com/ui/ly_harbor/blank/harbor_portal", "真实的UI路径"},
		{"/api", "相对路径"},
		{"/a", "单字符路径（可能是路由）"},
		{"/123", "纯数字路径（可能是ID）"},
		{"http://example.com/path/with/many/segments", "多段路径"},
		{"http://example.com/file.php", "PHP文件"},
		{"http://example.com/api/v1/users?id=123&name=test", "带参数的API"},
		{"http://example.com/get-user-info", "包含get但不是代码"},
		{"http://example.com/data-export", "包含data但不是代码"},
		{"http://example.com/user-list", "包含list但不是代码"},
	}

	for _, tc := range validURLs {
		valid, reason := v.IsValidBusinessURL(tc.url)
		if !valid {
			t.Errorf("URL应该通过但被过滤:\n  URL: %s\n  说明: %s\n  过滤原因: %s\n",
				tc.url, tc.reason, reason)
		}
	}
}

// TestSmartURLValidator_InvalidURLs 测试应该被过滤的URL
func TestSmartURLValidator_InvalidURLs(t *testing.T) {
	v := NewSmartURLValidator()

	invalidURLs := []struct {
		url    string
		reason string
	}{
		{"javascript:alert(1)", "JavaScript协议"},
		{"<script>alert(1)</script>", "HTML标签"},
		{"function() { return true; }", "JavaScript函数"},
		{"var x = 123;", "JavaScript变量声明"},
		{"let y = 456;", "JavaScript let声明"},
		{"const z = 789;", "JavaScript const声明"},
		{"#", "纯符号"},
		{"?", "纯符号"},
		{"", "空URL"},
		{"   ", "纯空格"},
		{"console.log('test')", "JavaScript代码"},
		{"{{variable}}", "模板语法"},
		{"<%=value%>", "模板语法"},
		{"<?php echo $x; ?>", "PHP代码标签"},
		{"window.location.href", "JavaScript对象"},
		{"document.getElementById", "JavaScript DOM操作"},
		{"http://example.com/path?code=function(){}=>return", "URL中包含JS代码"},
		{"data:text/html,<script>alert(1)</script>", "data协议"},
		{"blob:https://example.com/123", "blob协议"},
	}

	for _, tc := range invalidURLs {
		valid, _ := v.IsValidBusinessURL(tc.url)
		if valid {
			t.Errorf("URL应该被过滤但通过了:\n  URL: %s\n  说明: %s\n",
				tc.url, tc.reason)
		}
	}
}

// TestSmartURLValidator_EdgeCases 测试边缘情况
func TestSmartURLValidator_EdgeCases(t *testing.T) {
	v := NewSmartURLValidator()

	testCases := []struct {
		url      string
		expected bool
		desc     string
	}{
		// 短路径
		{"/ui", true, "短路径应该通过"},
		{"/v1", true, "版本路径应该通过"},
		{"/id", true, "ID路径应该通过"},

		// 数字路径
		{"/123", true, "纯数字路径应该通过"},
		{"/user/123/profile", true, "包含数字的路径应该通过"},

		// 多段路径
		{"http://example.com/a/b/c/d/e/f", true, "多段路径应该通过"},

		// 带扩展名
		{"http://example.com/file.php", true, "PHP文件应该通过"},
		{"http://example.com/page.jsp", true, "JSP文件应该通过"},
		{"http://example.com/api.do", true, "DO文件应该通过"},
		{"http://example.com/index.html", true, "HTML文件应该通过"},

		// 带参数
		{"http://example.com/search?q=test&page=1", true, "带参数应该通过"},
		{"http://example.com/api?id=123&name=test&type=json", true, "多参数应该通过"},

		// 带锚点
		{"http://example.com/page#section1", true, "带锚点应该通过"},

		// URL编码
		{"http://example.com/path%20with%20space", true, "正常URL编码应该通过"},
		{"http://example.com/%E4%B8%AD%E6%96%87", true, "中文URL编码应该通过"},

		// 特殊字符（合理的）
		{"http://example.com/user-profile", true, "包含连字符应该通过"},
		{"http://example.com/user_profile", true, "包含下划线应该通过"},
		{"http://example.com/api/v1.0/users", true, "包含点号应该通过"},

		// 过长URL（应该被过滤）
		{"http://example.com/" + string(make([]byte, 600)), false, "超长URL应该被过滤"},

		// 过度编码（应该被过滤）
		{"http://example.com/%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20", false, "过度URL编码应该被过滤"},
	}

	for _, tc := range testCases {
		valid, reason := v.IsValidBusinessURL(tc.url)
		if valid != tc.expected {
			status := "通过"
			if !valid {
				status = "被过滤"
			}
			t.Errorf("%s:\n  预期=%v, 实际=%v (URL%s)\n  URL=%s\n  原因=%s\n",
				tc.desc, tc.expected, valid, status, tc.url, reason)
		}
	}
}

// TestSmartURLValidator_FilterURLs 测试批量过滤
func TestSmartURLValidator_FilterURLs(t *testing.T) {
	v := NewSmartURLValidator()

	urls := []string{
		"http://example.com/api/users",        // 有效
		"javascript:alert(1)",                 // 无效
		"http://example.com/admin/config",     // 有效
		"<script>alert(1)</script>",           // 无效
		"http://example.com/search?q=test",    // 有效
		"function() { return true; }",         // 无效
		"http://example.com/home",             // 有效
		"#",                                   // 无效
		"http://example.com/api/json/export",  // 有效
	}

	filtered := v.FilterURLs(urls)

	expectedCount := 5 // 预期有5个有效URL
	if len(filtered) != expectedCount {
		t.Errorf("过滤后的URL数量不符: 预期=%d, 实际=%d\n过滤结果: %v",
			expectedCount, len(filtered), filtered)
	}

	// 验证所有过滤后的URL都是有效的
	for _, url := range filtered {
		if valid, _ := v.IsValidBusinessURL(url); !valid {
			t.Errorf("过滤结果中包含无效URL: %s", url)
		}
	}
}

// TestSmartURLValidator_Statistics 测试统计功能
func TestSmartURLValidator_Statistics(t *testing.T) {
	v := NewSmartURLValidator()

	// 测试一些URL
	testURLs := []string{
		"http://example.com/api/users",        // 有效
		"javascript:alert(1)",                 // 无效协议
		"http://example.com/admin",            // 有效
		"<script>alert(1)</script>",           // HTML标签
		"function() {}",                       // JS代码
		"#",                                   // 符号
		"http://example.com/path",             // 有效
	}

	for _, url := range testURLs {
		v.IsValidBusinessURL(url)
	}

	stats := v.GetStatistics()

	// 验证统计数据
	if stats["total_checked"] != len(testURLs) {
		t.Errorf("总检查数不符: 预期=%d, 实际=%d", len(testURLs), stats["total_checked"])
	}

	expectedPassed := 3
	if stats["total_passed"] != expectedPassed {
		t.Errorf("通过数不符: 预期=%d, 实际=%d", expectedPassed, stats["total_passed"])
	}

	// 验证各类过滤原因的统计
	if stats["filtered_by_scheme"] < 1 {
		t.Error("应该有至少1个被无效协议过滤的URL")
	}

	if stats["filtered_by_html_tag"] < 1 {
		t.Error("应该有至少1个被HTML标签过滤的URL")
	}

	if stats["filtered_by_js_code"] < 1 {
		t.Error("应该有至少1个被JavaScript代码过滤的URL")
	}

	// 测试重置统计
	v.ResetStatistics()
	statsAfterReset := v.GetStatistics()

	if statsAfterReset["total_checked"] != 0 {
		t.Error("重置后统计数据应该为0")
	}
}

// TestSmartURLValidator_Compat 测试兼容层
func TestSmartURLValidator_Compat(t *testing.T) {
	v := NewSmartURLValidatorCompat()

	// 测试兼容接口（只返回bool）
	testCases := []struct {
		url      string
		expected bool
	}{
		{"http://example.com/api/users", true},
		{"javascript:alert(1)", false},
		{"http://example.com/admin", true},
		{"<script>alert(1)</script>", false},
	}

	for _, tc := range testCases {
		valid := v.IsValidBusinessURL(tc.url) // 兼容接口，只返回bool
		if valid != tc.expected {
			t.Errorf("兼容层接口测试失败: URL=%s, 预期=%v, 实际=%v",
				tc.url, tc.expected, valid)
		}
	}

	// 测试批量过滤
	urls := []string{
		"http://example.com/api/users",
		"javascript:alert(1)",
		"http://example.com/admin",
	}

	filtered := v.FilterURLs(urls)
	if len(filtered) != 2 {
		t.Errorf("兼容层批量过滤失败: 预期=2, 实际=%d", len(filtered))
	}
}

// TestSmartURLValidator_RealWorldExamples 测试真实世界的URL
func TestSmartURLValidator_RealWorldExamples(t *testing.T) {
	v := NewSmartURLValidator()

	// 基于实际爬取结果的URL
	realURLs := []struct {
		url      string
		expected bool
		desc     string
	}{
		// 从 spider_x.lydaas.com_20251026_220336_urls.txt 中提取的真实URL
		{"http://x.lydaas.com", true, "首页"},
		{"https://x.lydaas.com/ui/ly_harbor/home/harbor_portal", true, "UI页面"},
		{"https://x.lydaas.com/ui/ly_harbor/blank/harbor_portal", true, "空白页面"},
		{"https://x.lydaas.com/api/ly_harbor/reportCenter_rule", true, "API接口"},
		{"https://x.lydaas.com/api/document/portal_banner_advertising_query", true, "查询API"},
		{"https://x.lydaas.com/api/document/query_portal_search_hot_word", true, "搜索API"},
		{"https://x.lydaas.com/api/document/query_portal_search_hot_word_all", true, "全部搜索API"},
		{"https://x.lydaas.com/api/document/portal_category_query", true, "分类查询API"},
		{"https://x.lydaas.com/api/document/portal_solution_query", true, "解决方案API"},

		// 可能被旧版本误杀的URL（应该通过）
		{"http://x.lydaas.com/admin", true, "管理后台"},
		{"http://x.lydaas.com/user/list", true, "用户列表"},
		{"http://x.lydaas.com/search", true, "搜索页面"},
		{"http://x.lydaas.com/api/data/export", true, "数据导出"},
		{"http://x.lydaas.com/config/system", true, "系统配置"},
		{"http://x.lydaas.com/application/manage", true, "应用管理"},
		{"http://x.lydaas.com/text/format", true, "文本格式化"},

		// 应该被过滤的垃圾URL
		{"?%2A=", false, "特殊参数"},
		{"?%24=", false, "特殊参数"},
		{"#BFBFBF", false, "颜色代码"},
	}

	for _, tc := range realURLs {
		valid, reason := v.IsValidBusinessURL(tc.url)
		if valid != tc.expected {
			status := "通过"
			if !valid {
				status = "被过滤"
			}
			t.Errorf("真实URL测试失败 - %s:\n  预期=%v, 实际=%v (URL%s)\n  URL=%s\n  原因=%s\n",
				tc.desc, tc.expected, valid, status, tc.url, reason)
		}
	}
}

// BenchmarkSmartURLValidator 性能基准测试
func BenchmarkSmartURLValidator(b *testing.B) {
	v := NewSmartURLValidator()
	testURL := "http://example.com/api/users?id=123&name=test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.IsValidBusinessURL(testURL)
	}
}

// BenchmarkSmartURLValidator_Batch 批量性能基准测试
func BenchmarkSmartURLValidator_Batch(b *testing.B) {
	v := NewSmartURLValidator()
	urls := []string{
		"http://example.com/api/users",
		"http://example.com/admin/config",
		"http://example.com/search?q=test",
		"javascript:alert(1)",
		"<script>alert(1)</script>",
		"http://example.com/home",
		"http://example.com/api/json/export",
		"function() {}",
		"#",
		"http://example.com/path",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.FilterURLs(urls)
	}
}

