package core

import (
	"fmt"
	"sync"
)

// AdaptivePriorityLearner 自适应优先级学习器
// 根据爬取结果动态调整优先级权重，使爬虫越爬越聪明
type AdaptivePriorityLearner struct {
	mutex sync.RWMutex
	
	// 学习统计
	totalURLs        int     // 总爬取URL数
	highValueHits    int     // 高价值URL命中次数（分数>=80）
	midValueHits     int     // 中等价值URL命中次数（50-80）
	lowValueHits     int     // 低价值URL命中次数（<50）
	
	// 发现统计
	totalLinksFound  int     // 发现的总链接数
	totalAPIsFound   int     // 发现的API数
	totalFormsFound  int     // 发现的表单数
	
	// 响应统计
	avgResponseTime  float64 // 平均响应时间（毫秒）
	successRate      float64 // 成功率（2xx响应）
	
	// 权重调整历史
	weightAdjustments []WeightAdjustment
	
	// 学习参数
	learningRate     float64 // 学习率（0.1-0.5）
	adjustmentCount  int     // 调整次数
}

// WeightAdjustment 权重调整记录
type WeightAdjustment struct {
	Iteration   int     // 迭代次数
	Reason      string  // 调整原因
	OldWeights  Weights // 调整前的权重
	NewWeights  Weights // 调整后的权重
	Performance PerformanceMetrics // 性能指标
}

// Weights 权重快照
type Weights struct {
	Depth         float64
	Internal      float64
	Params        float64
	Recent        float64
	PathValue     float64
	BusinessValue float64
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	HighValueRate  float64 // 高价值URL占比
	APIDiscoveryRate float64 // API发现率
	AvgLinksPerPage float64 // 平均每页链接数
	SuccessRate    float64 // 成功率
}

// NewAdaptivePriorityLearner 创建自适应学习器
func NewAdaptivePriorityLearner(learningRate float64) *AdaptivePriorityLearner {
	if learningRate <= 0 || learningRate > 1 {
		learningRate = 0.15 // 默认学习率15%
	}
	
	return &AdaptivePriorityLearner{
		learningRate:      learningRate,
		weightAdjustments: make([]WeightAdjustment, 0),
	}
}

// LearnFromResults 从爬取结果中学习
func (l *AdaptivePriorityLearner) LearnFromResults(results []*Result) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	if len(results) == 0 {
		return
	}
	
	// 更新统计
	for _, result := range results {
		l.totalURLs++
		
		// 评估URL价值（基于发现的内容）
		value := l.evaluateURLValue(result)
		
		if value >= 80 {
			l.highValueHits++
		} else if value >= 50 {
			l.midValueHits++
		} else {
			l.lowValueHits++
		}
		
		// 统计发现
		l.totalLinksFound += len(result.Links)
		l.totalAPIsFound += len(result.APIs)
		l.totalFormsFound += len(result.Forms)
		
		// 更新成功率
		if result.StatusCode >= 200 && result.StatusCode < 300 {
			newSuccessCount := l.successRate * float64(l.totalURLs-1) + 1
			l.successRate = newSuccessCount / float64(l.totalURLs)
		} else {
			l.successRate = l.successRate * float64(l.totalURLs-1) / float64(l.totalURLs)
		}
	}
}

// evaluateURLValue 评估URL的价值（0-100）
func (l *AdaptivePriorityLearner) evaluateURLValue(result *Result) float64 {
	score := 50.0 // 基础分数
	
	// 根据发现的内容加分
	if len(result.APIs) > 0 {
		score += 20.0 // 发现API，高价值
	}
	
	if len(result.Forms) > 0 {
		score += 15.0 // 发现表单，较高价值
	}
	
	if len(result.Links) > 10 {
		score += 10.0 // 发现大量链接，中等价值
	} else if len(result.Links) > 5 {
		score += 5.0
	}
	
	// 根据状态码调整
	if result.StatusCode == 200 {
		score += 5.0
	} else if result.StatusCode >= 400 {
		score -= 20.0
	}
	
	// 限制范围
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// ShouldAdjustWeights 判断是否应该调整权重
func (l *AdaptivePriorityLearner) ShouldAdjustWeights() (bool, string) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	// 至少爬取20个URL后才开始调整
	if l.totalURLs < 20 {
		return false, "样本量不足"
	}
	
	// 每爬取50个URL调整一次
	if l.totalURLs%50 != 0 {
		return false, "未达到调整间隔"
	}
	
	// 计算高价值URL占比
	highValueRate := float64(l.highValueHits) / float64(l.totalURLs)
	
	// 如果高价值URL占比过低（<20%），需要调整
	if highValueRate < 0.2 {
		return true, fmt.Sprintf("高价值URL占比过低(%.1f%%)", highValueRate*100)
	}
	
	// 如果低价值URL占比过高（>50%），需要调整
	lowValueRate := float64(l.lowValueHits) / float64(l.totalURLs)
	if lowValueRate > 0.5 {
		return true, fmt.Sprintf("低价值URL占比过高(%.1f%%)", lowValueRate*100)
	}
	
	// 如果API发现率高，但参数权重不够
	apiRate := float64(l.totalAPIsFound) / float64(l.totalURLs)
	if apiRate > 0.3 {
		return true, fmt.Sprintf("API发现率较高(%.1f%%),可增强参数权重", apiRate*100)
	}
	
	return false, "当前权重表现良好"
}

// AdjustWeights 调整优先级权重
func (l *AdaptivePriorityLearner) AdjustWeights(scheduler *URLPriorityScheduler) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	// 记录调整前的权重
	oldWeights := Weights{
		Depth:         scheduler.W1_Depth,
		Internal:      scheduler.W2_Internal,
		Params:        scheduler.W3_Params,
		Recent:        scheduler.W4_Recent,
		PathValue:     scheduler.W5_PathValue,
		BusinessValue: 0, // 暂不支持
	}
	
	newWeights := oldWeights
	adjustmentReason := ""
	adjusted := false
	
	// 计算各项指标
	highValueRate := float64(l.highValueHits) / float64(l.totalURLs)
	lowValueRate := float64(l.lowValueHits) / float64(l.totalURLs)
	apiRate := float64(l.totalAPIsFound) / float64(l.totalURLs)
	
	// 调整策略1: 高价值URL占比过低，增加路径价值权重
	if highValueRate < 0.2 {
		adjustment := 1.0 + l.learningRate
		newWeights.PathValue *= adjustment
		adjustmentReason = fmt.Sprintf("高价值URL占比低(%.1f%%)，增加路径价值权重", highValueRate*100)
		adjusted = true
	}
	
	// 调整策略2: API发现率高，增加参数权重
	if apiRate > 0.3 {
		adjustment := 1.0 + l.learningRate
		newWeights.Params *= adjustment
		if adjustmentReason != "" {
			adjustmentReason += "; "
		}
		adjustmentReason += fmt.Sprintf("API发现率高(%.1f%%)，增加参数权重", apiRate*100)
		adjusted = true
	}
	
	// 调整策略3: 低价值URL占比过高，降低深度权重
	if lowValueRate > 0.5 {
		adjustment := 1.0 - l.learningRate*0.5
		newWeights.Depth *= adjustment
		if adjustmentReason != "" {
			adjustmentReason += "; "
		}
		adjustmentReason += fmt.Sprintf("低价值URL占比高(%.1f%%)，降低深度权重", lowValueRate*100)
		adjusted = true
	}
	
	// 调整策略4: 成功率低，增加域内链接权重
	if l.successRate < 0.7 {
		adjustment := 1.0 + l.learningRate*0.8
		newWeights.Internal *= adjustment
		if adjustmentReason != "" {
			adjustmentReason += "; "
		}
		adjustmentReason += fmt.Sprintf("成功率低(%.1f%%)，增加域内链接权重", l.successRate*100)
		adjusted = true
	}
	
	if !adjusted {
		return false
	}
	
	// 应用新权重
	scheduler.SetWeights(
		newWeights.Depth,
		newWeights.Internal,
		newWeights.Params,
		newWeights.Recent,
		newWeights.PathValue,
	)
	
	// 记录调整
	l.adjustmentCount++
	l.weightAdjustments = append(l.weightAdjustments, WeightAdjustment{
		Iteration:  l.adjustmentCount,
		Reason:     adjustmentReason,
		OldWeights: oldWeights,
		NewWeights: newWeights,
		Performance: PerformanceMetrics{
			HighValueRate:    highValueRate,
			APIDiscoveryRate: apiRate,
			AvgLinksPerPage:  float64(l.totalLinksFound) / float64(l.totalURLs),
			SuccessRate:      l.successRate,
		},
	})
	
	// 打印调整信息
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("【自适应学习】第 %d 次权重调整\n", l.adjustmentCount)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("  调整原因: %s\n", adjustmentReason)
	fmt.Printf("\n  权重变化:\n")
	fmt.Printf("    深度权重:     %.2f → %.2f (%.1f%%)\n", 
		oldWeights.Depth, newWeights.Depth, 
		(newWeights.Depth-oldWeights.Depth)/oldWeights.Depth*100)
	fmt.Printf("    域内权重:     %.2f → %.2f (%.1f%%)\n", 
		oldWeights.Internal, newWeights.Internal,
		(newWeights.Internal-oldWeights.Internal)/oldWeights.Internal*100)
	fmt.Printf("    参数权重:     %.2f → %.2f (%.1f%%)\n", 
		oldWeights.Params, newWeights.Params,
		(newWeights.Params-oldWeights.Params)/oldWeights.Params*100)
	fmt.Printf("    路径价值权重: %.2f → %.2f (%.1f%%)\n", 
		oldWeights.PathValue, newWeights.PathValue,
		(newWeights.PathValue-oldWeights.PathValue)/oldWeights.PathValue*100)
	fmt.Printf("\n  性能指标:\n")
	fmt.Printf("    高价值URL占比: %.1f%%\n", highValueRate*100)
	fmt.Printf("    API发现率:     %.1f%%\n", apiRate*100)
	fmt.Printf("    成功率:        %.1f%%\n", l.successRate*100)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	
	return true
}

// GetStatistics 获取学习统计
func (l *AdaptivePriorityLearner) GetStatistics() map[string]interface{} {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	
	stats["total_urls"] = l.totalURLs
	stats["high_value_hits"] = l.highValueHits
	stats["mid_value_hits"] = l.midValueHits
	stats["low_value_hits"] = l.lowValueHits
	
	if l.totalURLs > 0 {
		stats["high_value_rate"] = float64(l.highValueHits) / float64(l.totalURLs)
		stats["mid_value_rate"] = float64(l.midValueHits) / float64(l.totalURLs)
		stats["low_value_rate"] = float64(l.lowValueHits) / float64(l.totalURLs)
	}
	
	stats["total_links_found"] = l.totalLinksFound
	stats["total_apis_found"] = l.totalAPIsFound
	stats["total_forms_found"] = l.totalFormsFound
	stats["success_rate"] = l.successRate
	stats["adjustment_count"] = l.adjustmentCount
	
	return stats
}

// PrintReport 打印学习报告
func (l *AdaptivePriorityLearner) PrintReport() {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("       自适应优先级学习报告")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	fmt.Printf("\n【学习统计】\n")
	fmt.Printf("  总爬取URL数:   %d\n", l.totalURLs)
	fmt.Printf("  高价值URL:     %d (%.1f%%)\n", 
		l.highValueHits, 
		float64(l.highValueHits)/float64(l.totalURLs)*100)
	fmt.Printf("  中等价值URL:   %d (%.1f%%)\n", 
		l.midValueHits,
		float64(l.midValueHits)/float64(l.totalURLs)*100)
	fmt.Printf("  低价值URL:     %d (%.1f%%)\n", 
		l.lowValueHits,
		float64(l.lowValueHits)/float64(l.totalURLs)*100)
	
	fmt.Printf("\n【发现统计】\n")
	fmt.Printf("  总链接数:      %d\n", l.totalLinksFound)
	fmt.Printf("  总API数:       %d\n", l.totalAPIsFound)
	fmt.Printf("  总表单数:      %d\n", l.totalFormsFound)
	
	if l.totalURLs > 0 {
		fmt.Printf("  平均链接/页:   %.1f\n", float64(l.totalLinksFound)/float64(l.totalURLs))
		fmt.Printf("  API发现率:     %.1f%%\n", float64(l.totalAPIsFound)/float64(l.totalURLs)*100)
	}
	
	fmt.Printf("\n【性能指标】\n")
	fmt.Printf("  成功率:        %.1f%%\n", l.successRate*100)
	fmt.Printf("  权重调整次数:  %d\n", l.adjustmentCount)
	
	// 显示权重调整历史
	if len(l.weightAdjustments) > 0 {
		fmt.Printf("\n【权重调整历史】\n")
		for i, adj := range l.weightAdjustments {
			fmt.Printf("\n  调整 %d:\n", i+1)
			fmt.Printf("    原因: %s\n", adj.Reason)
			fmt.Printf("    高价值率: %.1f%% → API率: %.1f%%\n",
				adj.Performance.HighValueRate*100,
				adj.Performance.APIDiscoveryRate*100)
		}
	}
	
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

