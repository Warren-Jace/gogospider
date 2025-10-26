package core

import (
	"container/heap"
	"net/url"
	"strings"
	"sync"
	"time"
)

// URLPriority URL优先级项
type URLPriority struct {
	URL          string
	Priority     float64
	Depth        int
	DiscoveryTime time.Time
	IsInternal   bool
	HasParams    bool
	Index        int // heap需要的索引
}

// PriorityQueue 优先级队列（实现heap.Interface）
type PriorityQueue []*URLPriority

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// 优先级高的排前面
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*URLPriority)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // 避免内存泄漏
	item.Index = -1 // 标记已移除
	*pq = old[0 : n-1]
	return item
}

// URLPriorityScheduler URL优先级调度器
type URLPriorityScheduler struct {
	queue          PriorityQueue
	visited        map[string]bool
	targetDomain   string
	mutex          sync.Mutex
	
	// 权重配置
	W1_Depth       float64 // 深度权重
	W2_Internal    float64 // 域内权重
	W3_Params      float64 // 参数权重
	W4_Recent      float64 // 新鲜度权重
	W5_PathValue   float64 // 路径价值权重
}

// NewURLPriorityScheduler 创建优先级调度器
func NewURLPriorityScheduler(targetDomain string) *URLPriorityScheduler {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)
	
	return &URLPriorityScheduler{
		queue:        pq,
		visited:      make(map[string]bool),
		targetDomain: targetDomain,
		
		// 默认权重配置（可调整）
		W1_Depth:     3.0,  // 深度影响较大
		W2_Internal:  2.0,  // 域内链接重要
		W3_Params:    1.5,  // 带参数的URL有价值
		W4_Recent:    1.0,  // 新发现的稍加权
		W5_PathValue: 4.0,  // 路径价值最重要
	}
}

// CalculatePriority 计算URL优先级
// priority = W1*(1/depth) + W2*(is_internal) + W3*(has_params) + W4*(recent) + W5*(path_value)
func (s *URLPriorityScheduler) CalculatePriority(urlStr string, depth int) float64 {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return 0.0
	}
	
	priority := 0.0
	
	// 1. 深度因子（越浅优先级越高）
	// depth=1 → 1.0, depth=2 → 0.5, depth=3 → 0.33...
	if depth > 0 {
		priority += s.W1_Depth * (1.0 / float64(depth))
	} else {
		priority += s.W1_Depth
	}
	
	// 2. 域内/域外因子
	isInternal := s.isInternalURL(parsedURL.Host)
	if isInternal {
		priority += s.W2_Internal
	}
	
	// 3. 参数因子（带参数的URL更有价值）
	hasParams := parsedURL.RawQuery != ""
	if hasParams {
		paramCount := len(parsedURL.Query())
		if paramCount >= 3 {
			priority += s.W3_Params * 2.0 // 多参数加倍
		} else if paramCount >= 2 {
			priority += s.W3_Params * 1.5
		} else {
			priority += s.W3_Params
		}
	}
	
	// 4. 新鲜度因子（最近发现的URL）
	// 这里简化处理，实际可以基于时间戳
	priority += s.W4_Recent * 0.5
	
	// 5. 路径价值因子（核心功能）
	pathValue := s.evaluatePathValue(parsedURL.Path)
	priority += s.W5_PathValue * pathValue
	
	return priority
}

// evaluatePathValue 评估路径的业务价值（0.0-3.0）
func (s *URLPriorityScheduler) evaluatePathValue(path string) float64 {
	pathLower := strings.ToLower(path)
	
	// 极高价值路径（3.0）
	highValueKeywords := []string{
		"admin", "phpmyadmin", "cpanel", "login", "auth",
		"token", "key", "secret", ".env", "config",
		"backup", "database", "dump", "export", "phpinfo",
	}
	for _, keyword := range highValueKeywords {
		if strings.Contains(pathLower, keyword) {
			return 3.0
		}
	}
	
	// 高价值路径（2.0）
	mediumHighKeywords := []string{
		"api", "graphql", "rest", "admin", "manage",
		"upload", "download", "file", "editor", "dashboard",
		"user", "account", "profile", "payment", "order",
	}
	for _, keyword := range mediumHighKeywords {
		if strings.Contains(pathLower, keyword) {
			return 2.0
		}
	}
	
	// 中等价值路径（1.0）
	mediumKeywords := []string{
		"search", "register", "signup", "contact", "message",
		"cart", "checkout", "product", "category", "post",
		"article", "comment", "setting", "preference",
	}
	for _, keyword := range mediumKeywords {
		if strings.Contains(pathLower, keyword) {
			return 1.0
		}
	}
	
	// 低价值路径（0.3）
	lowKeywords := []string{
		"about", "help", "faq", "terms", "privacy",
		"image", "img", "css", "js", "static",
	}
	for _, keyword := range lowKeywords {
		if strings.Contains(pathLower, keyword) {
			return 0.3
		}
	}
	
	// 默认值
	return 0.5
}

// isInternalURL 判断是否为内部URL
func (s *URLPriorityScheduler) isInternalURL(host string) bool {
	if host == "" {
		return true // 相对路径视为内部
	}
	
	cleanTarget := strings.TrimPrefix(s.targetDomain, "http://")
	cleanTarget = strings.TrimPrefix(cleanTarget, "https://")
	cleanTarget = strings.Split(cleanTarget, ":")[0]
	
	cleanHost := strings.Split(host, ":")[0]
	
	if cleanHost == cleanTarget {
		return true
	}
	
	// 子域名也算内部
	if strings.HasSuffix(cleanHost, "."+cleanTarget) {
		return true
	}
	
	return false
}

// AddURL 添加URL到优先级队列
func (s *URLPriorityScheduler) AddURL(urlStr string, depth int) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// 检查是否已访问
	if s.visited[urlStr] {
		return false
	}
	
	// 解析URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	
	// 计算优先级
	priority := s.CalculatePriority(urlStr, depth)
	
	// 创建优先级项
	item := &URLPriority{
		URL:           urlStr,
		Priority:      priority,
		Depth:         depth,
		DiscoveryTime: time.Now(),
		IsInternal:    s.isInternalURL(parsedURL.Host),
		HasParams:     parsedURL.RawQuery != "",
	}
	
	// 添加到堆
	heap.Push(&s.queue, item)
	
	return true
}

// PopURL 从队列中取出优先级最高的URL
func (s *URLPriorityScheduler) PopURL() *URLPriority {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.queue.Len() == 0 {
		return nil
	}
	
	item := heap.Pop(&s.queue).(*URLPriority)
	s.visited[item.URL] = true
	
	return item
}

// PopBatch 批量取出URL（用于并发爬取）
func (s *URLPriorityScheduler) PopBatch(batchSize int) []*URLPriority {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	batch := make([]*URLPriority, 0, batchSize)
	
	for i := 0; i < batchSize && s.queue.Len() > 0; i++ {
		item := heap.Pop(&s.queue).(*URLPriority)
		s.visited[item.URL] = true
		batch = append(batch, item)
	}
	
	return batch
}

// Size 队列大小
func (s *URLPriorityScheduler) Size() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.queue.Len()
}

// IsVisited 检查URL是否已访问
func (s *URLPriorityScheduler) IsVisited(urlStr string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.visited[urlStr]
}

// GetStatistics 获取统计信息
func (s *URLPriorityScheduler) GetStatistics() map[string]interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	stats := make(map[string]interface{})
	stats["queue_size"] = s.queue.Len()
	stats["visited_count"] = len(s.visited)
	stats["total_processed"] = len(s.visited)
	
	// 统计队列中的URL分布
	depthDist := make(map[int]int)
	internalCount := 0
	externalCount := 0
	paramsCount := 0
	
	for _, item := range s.queue {
		depthDist[item.Depth]++
		if item.IsInternal {
			internalCount++
		} else {
			externalCount++
		}
		if item.HasParams {
			paramsCount++
		}
	}
	
	stats["depth_distribution"] = depthDist
	stats["internal_urls"] = internalCount
	stats["external_urls"] = externalCount
	stats["urls_with_params"] = paramsCount
	
	return stats
}

// PrintStatistics 打印统计信息
func (s *URLPriorityScheduler) PrintStatistics() {
	stats := s.GetStatistics()
	
	println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	println("      优先级队列统计")
	println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	println("  队列大小:", stats["queue_size"].(int))
	println("  已访问:", stats["visited_count"].(int))
	println("  域内URL:", stats["internal_urls"].(int))
	println("  域外URL:", stats["external_urls"].(int))
	println("  带参数:", stats["urls_with_params"].(int))
	println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// SetWeights 设置权重配置
func (s *URLPriorityScheduler) SetWeights(w1, w2, w3, w4, w5 float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.W1_Depth = w1
	s.W2_Internal = w2
	s.W3_Params = w3
	s.W4_Recent = w4
	s.W5_PathValue = w5
}

// GetTopURLs 获取优先级最高的N个URL（不移除）
func (s *URLPriorityScheduler) GetTopURLs(n int) []*URLPriority {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	result := make([]*URLPriority, 0, n)
	count := n
	if count > s.queue.Len() {
		count = s.queue.Len()
	}
	
	for i := 0; i < count; i++ {
		result = append(result, s.queue[i])
	}
	
	return result
}

