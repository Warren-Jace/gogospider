package core

import (
	"fmt"
	"strings"
)

// PrintLayeredDeduplicationReport 打印分层去重详细报告
func (s *Spider) PrintLayeredDeduplicationReport() {
	if s.layeredDedup == nil {
		return
	}
	
	stats := s.layeredDedup.GetStatistics()
	
	fmt.Println("\n" + strings.Repeat("═", 70))
	fmt.Println("  🎯 分层去重策略统计报告 (v3.6)")
	fmt.Println(strings.Repeat("═", 70))
	
	// 总览
	fmt.Printf("\n【总览】\n")
	fmt.Printf("  总URL数量: %d\n", stats.TotalURLs)
	fmt.Printf("  节省请求: %d 个\n", stats.SavedRequests)
	
	if stats.TotalURLs > 0 {
		effectiveRate := float64(stats.SavedRequests) / float64(stats.TotalURLs) * 100
		fmt.Printf("  去重效率: %.1f%%\n", effectiveRate)
	}
	
	// 按类型分类统计
	fmt.Printf("\n【URL分类统计】\n")
	
	if stats.RESTfulURLs > 0 {
		fmt.Printf("  🔵 RESTful路径: %d 个\n", stats.RESTfulURLs)
		fmt.Printf("     策略: 保留所有路径变体（避免丢失独立业务端点）\n")
		fmt.Printf("     示例: /api/user/123/profile, /product/buy-1/\n")
	}
	
	if stats.AJAXAPIs > 0 {
		fmt.Printf("  🟢 AJAX/API接口: %d 个\n", stats.AJAXAPIs)
		fmt.Printf("     策略: 每个端点独立保留（避免API遗漏）\n")
		fmt.Printf("     示例: /ajax/artists.php, /api/v1/data\n")
	}
	
	if stats.FileParamURLs > 0 {
		fmt.Printf("  🟡 文件参数URL: %d 个\n", stats.FileParamURLs)
		fmt.Printf("     策略: 保留编码差异样本（检测路径穿越）\n")
		fmt.Printf("     示例: ?file=./path.jpg, ?file=%%2F..%%2F\n")
		if stats.ParameterVariations > 0 {
			fmt.Printf("     参数编码变体: %d 个\n", stats.ParameterVariations)
		}
	}
	
	if stats.NormalURLs > 0 {
		fmt.Printf("  ⚪ 普通URL: %d 个\n", stats.NormalURLs)
		fmt.Printf("     策略: 标准模式去重\n")
	}
	
	// POST请求统计
	fmt.Printf("\n【POST请求统计】\n")
	fmt.Printf("  去重后数量: %d 个\n", stats.POSTRequests)
	if stats.DuplicatePOSTs > 0 {
		fmt.Printf("  重复数量: %d 个（已去重）\n", stats.DuplicatePOSTs)
		fmt.Printf("  去重率: %.1f%%\n", 
			float64(stats.DuplicatePOSTs)/float64(stats.POSTRequests+stats.DuplicatePOSTs)*100)
	}
	
	// 对比旧版本
	fmt.Printf("\n【对比旧版本】\n")
	oldDedupeRate := 60.0 // 旧版本的去重率
	if stats.TotalURLs > 0 {
		newDedupeRate := float64(stats.SavedRequests) / float64(stats.TotalURLs) * 100
		improvement := oldDedupeRate - newDedupeRate
		
		if improvement > 0 {
			fmt.Printf("  旧版去重率: %.1f%% (过度去重)\n", oldDedupeRate)
			fmt.Printf("  新版去重率: %.1f%% (智能去重)\n", newDedupeRate)
			fmt.Printf("  ✅ 多保留了 %.1f%% 的有效URL\n", improvement)
		}
	}
	
	// 推荐建议
	fmt.Printf("\n【推荐使用】\n")
	fmt.Printf("  ✅ 安全测试: 使用去重后的URL进行漏洞扫描\n")
	fmt.Printf("  ✅ API测试: 特别关注 AJAX/API 类型的URL\n")
	fmt.Printf("  ✅ 文件包含: 重点测试文件参数URL的编码变体\n")
	fmt.Printf("  ✅ RESTful: 测试所有路径变体的越权访问\n")
	
	fmt.Println(strings.Repeat("═", 70))
}

// GetLayeredDeduplicationStats 获取分层去重统计（给外部调用）
func (s *Spider) GetLayeredDeduplicationStats() *LayeredDeduplicationStats {
	if s.layeredDedup == nil {
		return nil
	}
	stats := s.layeredDedup.GetStatistics()
	return &stats
}

// PrintLayeredDeduplicationComparison 打印与原始结果的对比
func (s *Spider) PrintLayeredDeduplicationComparison(originalCount int, dedupedCount int) {
	fmt.Println("\n" + strings.Repeat("─", 70))
	fmt.Println("  📊 URL去重效果对比")
	fmt.Println(strings.Repeat("─", 70))
	
	fmt.Printf("  原始URL数量: %d\n", originalCount)
	fmt.Printf("  去重后数量: %d\n", dedupedCount)
	
	if originalCount > 0 {
		saved := originalCount - dedupedCount
		savedRate := float64(saved) / float64(originalCount) * 100
		
		fmt.Printf("  减少数量: %d\n", saved)
		fmt.Printf("  去重率: %.1f%%\n", savedRate)
		
		// 评估去重效果
		if savedRate > 70 {
			fmt.Printf("  ⚠️ 警告: 去重率过高，可能丢失有效URL\n")
		} else if savedRate > 40 && savedRate <= 70 {
			fmt.Printf("  ✅ 正常: 去重效果适中\n")
		} else if savedRate > 20 && savedRate <= 40 {
			fmt.Printf("  ℹ️ 提示: 去重率较低，URL较多样化\n")
		} else {
			fmt.Printf("  ℹ️ 提示: 去重率很低，URL高度多样化\n")
		}
	}
	
	fmt.Println(strings.Repeat("─", 70))
}

