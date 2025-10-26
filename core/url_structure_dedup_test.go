package core

import (
	"testing"
)

func TestURLStructureDeduplicator_NormalizeURL(t *testing.T) {
	dedup := NewURLStructureDeduplicator()
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// 路径中的纯数字
		{
			name:     "纯数字路径段",
			input:    "http://example.com/user/123",
			expected: "http://example.com/user/{num}",
		},
		{
			name:     "多个纯数字路径段",
			input:    "http://example.com/category/456/product/789",
			expected: "http://example.com/category/{num}/product/{num}",
		},
		
		// 带分隔符的数字
		{
			name:     "RateProduct-2.html",
			input:    "http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-2.html",
			expected: "http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-{num}.html",
		},
		{
			name:     "RateProduct-100.html",
			input:    "http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-100.html",
			expected: "http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-{num}.html",
		},
		{
			name:     "BuyProduct-2/",
			input:    "http://testphp.vulnweb.com/Mod_Rewrite_Shop/BuyProduct-2/",
			expected: "http://testphp.vulnweb.com/Mod_Rewrite_Shop/BuyProduct-{num}/",
		},
		{
			name:     "BuyProduct-999/",
			input:    "http://testphp.vulnweb.com/Mod_Rewrite_Shop/BuyProduct-999/",
			expected: "http://testphp.vulnweb.com/Mod_Rewrite_Shop/BuyProduct-{num}/",
		},
		{
			name:     "product_456",
			input:    "http://example.com/shop/product_456",
			expected: "http://example.com/shop/product_{num}",
		},
		
		// UUID
		{
			name:     "UUID格式",
			input:    "http://api.example.com/order/a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			expected: "http://api.example.com/order/{uuid}",
		},
		
		// 长哈希
		{
			name:     "长哈希值",
			input:    "http://cdn.example.com/file/a1b2c3d4e5f6789012345678",
			expected: "http://cdn.example.com/file/{hash}",
		},
		
		// 参数值
		{
			name:     "单参数",
			input:    "http://example.com/item?id=123",
			expected: "http://example.com/item?id=",
		},
		{
			name:     "多参数",
			input:    "http://example.com/search?type=1&sort=asc&page=5",
			expected: "http://example.com/search?page=&sort=&type=", // 按字母排序
		},
		
		// 混合场景
		{
			name:     "路径变量+参数",
			input:    "http://shop.example.com/product-123/detail?color=red&size=M",
			expected: "http://shop.example.com/product-{num}/detail?color=&size=",
		},
		
		// 普通URL（不需要归一化）
		{
			name:     "普通URL",
			input:    "http://example.com/about",
			expected: "http://example.com/about",
		},
		{
			name:     "普通URL带扩展名",
			input:    "http://example.com/contact.html",
			expected: "http://example.com/contact.html",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dedup.NormalizeURL(tt.input)
			if err != nil {
				t.Errorf("NormalizeURL() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("NormalizeURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestURLStructureDeduplicator_AddURL(t *testing.T) {
	dedup := NewURLStructureDeduplicator()
	
	// 测试添加相似URL
	urls := []string{
		"http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-2.html",
		"http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-3.html",
		"http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-100.html",
	}
	
	// 第一个URL应该是新结构
	isNew, pattern := dedup.AddURL(urls[0])
	if !isNew {
		t.Error("第一个URL应该是新结构")
	}
	expectedPattern := "http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-{num}.html"
	if pattern != expectedPattern {
		t.Errorf("Pattern = %v, want %v", pattern, expectedPattern)
	}
	
	// 后续相似URL应该不是新结构
	for i := 1; i < len(urls); i++ {
		isNew, pattern := dedup.AddURL(urls[i])
		if isNew {
			t.Errorf("URL %s 应该被识别为相似结构", urls[i])
		}
		if pattern != expectedPattern {
			t.Errorf("Pattern = %v, want %v", pattern, expectedPattern)
		}
	}
	
	// 验证统计
	stats := dedup.GetStatistics()
	if stats["total_urls"] != 3 {
		t.Errorf("total_urls = %d, want 3", stats["total_urls"])
	}
	if stats["unique_structures"] != 1 {
		t.Errorf("unique_structures = %d, want 1", stats["unique_structures"])
	}
	
	// 验证只返回一个唯一结构
	uniqueStructures := dedup.GetUniqueStructures()
	if len(uniqueStructures) != 1 {
		t.Errorf("GetUniqueStructures() 返回 %d 个URL, want 1", len(uniqueStructures))
	}
	if uniqueStructures[0] != urls[0] {
		t.Errorf("代表URL = %v, want %v", uniqueStructures[0], urls[0])
	}
}

func TestURLStructureDeduplicator_MultipleStructures(t *testing.T) {
	dedup := NewURLStructureDeduplicator()
	
	// 添加不同结构的URL
	urls := []string{
		// 结构1: RateProduct
		"http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-2.html",
		"http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-3.html",
		
		// 结构2: BuyProduct
		"http://testphp.vulnweb.com/Mod_Rewrite_Shop/BuyProduct-2/",
		"http://testphp.vulnweb.com/Mod_Rewrite_Shop/BuyProduct-3/",
		
		// 结构3: artists.php
		"http://testphp.vulnweb.com/artists.php?artist=1",
		"http://testphp.vulnweb.com/artists.php?artist=2",
		
		// 结构4: 普通页面
		"http://testphp.vulnweb.com/about.html",
	}
	
	for _, url := range urls {
		dedup.AddURL(url)
	}
	
	// 验证统计
	stats := dedup.GetStatistics()
	if stats["total_urls"] != 7 {
		t.Errorf("total_urls = %d, want 7", stats["total_urls"])
	}
	if stats["unique_structures"] != 4 {
		t.Errorf("unique_structures = %d, want 4", stats["unique_structures"])
	}
	
	// 验证返回4个唯一结构
	uniqueStructures := dedup.GetUniqueStructures()
	if len(uniqueStructures) != 4 {
		t.Errorf("GetUniqueStructures() 返回 %d 个URL, want 4", len(uniqueStructures))
	}
	
	// 验证代表URL（每个结构的第一个URL）
	expected := []string{
		"http://testphp.vulnweb.com/Mod_Rewrite_Shop/BuyProduct-2/",
		"http://testphp.vulnweb.com/Mod_Rewrite_Shop/RateProduct-2.html",
		"http://testphp.vulnweb.com/about.html",
		"http://testphp.vulnweb.com/artists.php?artist=1",
	}
	
	// 因为返回的是排序后的，所以需要匹配
	for _, expectedURL := range expected {
		found := false
		for _, actualURL := range uniqueStructures {
			if actualURL == expectedURL {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("未找到期望的URL: %s", expectedURL)
		}
	}
}

func TestNormalizePathSegment(t *testing.T) {
	dedup := NewURLStructureDeduplicator()
	
	tests := []struct {
		input    string
		expected string
	}{
		// 纯数字
		{"123", "{num}"},
		{"456789", "{num}"},
		
		// 带分隔符的数字
		{"product-123", "product-{num}"},
		{"item_456", "item_{num}"},
		{"RateProduct-2", "RateProduct-{num}"},
		{"BuyProduct-999", "BuyProduct-{num}"},
		
		// 带扩展名
		{"file-123.html", "file-{num}.html"},
		{"page-456.php", "page-{num}.php"},
		
		// UUID
		{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", "{uuid}"},
		
		// 长哈希
		{"a1b2c3d4e5f6789012345678abcdef", "{hash}"},
		
		// 不应该改变的
		{"about", "about"},
		{"contact.html", "contact.html"},
		{"product", "product"},
		{"item-abc", "item-abc"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dedup.normalizePathSegment(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePathSegment(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

