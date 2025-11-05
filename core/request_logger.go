package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// RequestLog 请求日志
type RequestLog struct {
	Timestamp   time.Time         `json:"timestamp"`   // 请求时间戳
	Method      string            `json:"method"`      // 请求方法(GET/POST/PUT等)
	URL         string            `json:"url"`         // 完整URL
	Path        string            `json:"path"`        // URL路径
	Query       map[string]string `json:"query"`       // 查询参数
	Headers     map[string]string `json:"headers"`     // 请求头(可选)
	Body        string            `json:"body"`        // 请求体(POST等)
	StatusCode  int               `json:"status_code"` // 响应状态码
	ResponseTime int64             `json:"response_time_ms"` // 响应时间(毫秒)
	Error       string            `json:"error,omitempty"` // 错误信息(如果有)
}

// RequestLogger 请求日志记录器
type RequestLogger struct {
	logs     []RequestLog
	mutex    sync.Mutex
	enabled  bool
	maxLogs  int // 最大保存日志数量(防止内存溢出)
}

// NewRequestLogger 创建请求日志记录器
func NewRequestLogger(enabled bool, maxLogs int) *RequestLogger {
	if maxLogs <= 0 {
		maxLogs = 100000 // 默认最大10万条
	}
	
	return &RequestLogger{
		logs:    make([]RequestLog, 0),
		enabled: enabled,
		maxLogs: maxLogs,
	}
}

// LogRequest 记录请求
func (rl *RequestLogger) LogRequest(method, urlStr string, headers map[string]string, body string) {
	if !rl.enabled {
		return
	}
	
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	// 检查是否超过最大限制
	if len(rl.logs) >= rl.maxLogs {
		// 移除最旧的日志
		rl.logs = rl.logs[1:]
	}
	
	// 解析URL和参数
	parsedURL, err := url.Parse(urlStr)
	query := make(map[string]string)
	path := urlStr
	
	if err == nil {
		path = parsedURL.Path
		// 解析查询参数
		for key, values := range parsedURL.Query() {
			if len(values) > 0 {
				query[key] = values[0] // 只保存第一个值
			}
		}
	}
	
	log := RequestLog{
		Timestamp: time.Now(),
		Method:    method,
		URL:       urlStr,
		Path:      path,
		Query:     query,
		Headers:   headers,
		Body:      body,
	}
	
	rl.logs = append(rl.logs, log)
}

// LogResponse 记录响应信息
func (rl *RequestLogger) LogResponse(urlStr string, statusCode int, responseTime int64, err error) {
	if !rl.enabled {
		return
	}
	
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	// 查找最后一个匹配的请求并更新
	for i := len(rl.logs) - 1; i >= 0; i-- {
		if rl.logs[i].URL == urlStr && rl.logs[i].StatusCode == 0 {
			rl.logs[i].StatusCode = statusCode
			rl.logs[i].ResponseTime = responseTime
			if err != nil {
				rl.logs[i].Error = err.Error()
			}
			break
		}
	}
}

// GetLogs 获取所有日志
func (rl *RequestLogger) GetLogs() []RequestLog {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	// 返回副本
	logs := make([]RequestLog, len(rl.logs))
	copy(logs, rl.logs)
	return logs
}

// GetStatistics 获取统计信息
func (rl *RequestLogger) GetStatistics() map[string]interface{} {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	stats := make(map[string]interface{})
	
	// 总请求数
	stats["total_requests"] = len(rl.logs)
	
	// 按方法统计
	methodCount := make(map[string]int)
	statusCount := make(map[int]int)
	var totalResponseTime int64
	errorCount := 0
	
	for _, log := range rl.logs {
		methodCount[log.Method]++
		if log.StatusCode > 0 {
			statusCount[log.StatusCode]++
		}
		totalResponseTime += log.ResponseTime
		if log.Error != "" {
			errorCount++
		}
	}
	
	stats["methods"] = methodCount
	stats["status_codes"] = statusCount
	stats["error_count"] = errorCount
	
	if len(rl.logs) > 0 {
		stats["avg_response_time_ms"] = totalResponseTime / int64(len(rl.logs))
	}
	
	return stats
}

// SaveToFile 保存请求日志到文件(文本格式)
func (rl *RequestLogger) SaveToFile(filename string) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	if len(rl.logs) == 0 {
		return fmt.Errorf("没有请求日志可保存")
	}
	
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()
	
	// 文件头
	file.WriteString("═══════════════════════════════════════════════════════\n")
	file.WriteString("  GogoSpider - 请求日志详情\n")
	file.WriteString(fmt.Sprintf("  生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	file.WriteString(fmt.Sprintf("  总请求数: %d\n", len(rl.logs)))
	file.WriteString("═══════════════════════════════════════════════════════\n\n")
	
	// 按时间顺序保存每个请求
	for i, log := range rl.logs {
		file.WriteString(fmt.Sprintf("【请求 %d】\n", i+1))
		file.WriteString(fmt.Sprintf("时间: %s\n", log.Timestamp.Format("2006-01-02 15:04:05.000")))
		file.WriteString(fmt.Sprintf("方法: %s\n", log.Method))
		file.WriteString(fmt.Sprintf("URL: %s\n", log.URL))
		file.WriteString(fmt.Sprintf("路径: %s\n", log.Path))
		
		// 查询参数
		if len(log.Query) > 0 {
			file.WriteString("查询参数:\n")
			// 排序参数名以保持一致性
			paramNames := make([]string, 0, len(log.Query))
			for name := range log.Query {
				paramNames = append(paramNames, name)
			}
			sort.Strings(paramNames)
			
			for _, name := range paramNames {
				file.WriteString(fmt.Sprintf("  %s = %s\n", name, log.Query[name]))
			}
		}
		
		// 请求头(可选)
		if len(log.Headers) > 0 {
			file.WriteString("请求头:\n")
			for key, value := range log.Headers {
				// 只显示关键头
				if strings.Contains(strings.ToLower(key), "content") ||
				   strings.Contains(strings.ToLower(key), "authorization") ||
				   strings.Contains(strings.ToLower(key), "cookie") {
					file.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
				}
			}
		}
		
		// 请求体(POST等)
		if log.Body != "" {
			file.WriteString("请求体:\n")
			if len(log.Body) > 500 {
				file.WriteString(fmt.Sprintf("  %s...(省略)\n", log.Body[:500]))
			} else {
				file.WriteString(fmt.Sprintf("  %s\n", log.Body))
			}
		}
		
		// 响应信息
		if log.StatusCode > 0 {
			file.WriteString(fmt.Sprintf("状态码: %d\n", log.StatusCode))
			file.WriteString(fmt.Sprintf("响应时间: %d ms\n", log.ResponseTime))
		}
		
		// 错误信息
		if log.Error != "" {
			file.WriteString(fmt.Sprintf("错误: %s\n", log.Error))
		}
		
		file.WriteString("\n" + strings.Repeat("─", 55) + "\n\n")
	}
	
	// 统计摘要
	stats := rl.getInternalStatistics()
	file.WriteString("\n═══════════════════════════════════════════════════════\n")
	file.WriteString("【统计摘要】\n")
	file.WriteString("═══════════════════════════════════════════════════════\n")
	file.WriteString(fmt.Sprintf("总请求数: %d\n", len(rl.logs)))
	file.WriteString("\n按方法统计:\n")
	for method, count := range stats.MethodCount {
		file.WriteString(fmt.Sprintf("  %s: %d\n", method, count))
	}
	file.WriteString("\n按状态码统计:\n")
	for code, count := range stats.StatusCount {
		file.WriteString(fmt.Sprintf("  %d: %d\n", code, count))
	}
	if stats.ErrorCount > 0 {
		file.WriteString(fmt.Sprintf("\n失败请求数: %d\n", stats.ErrorCount))
	}
	if stats.AvgResponseTime > 0 {
		file.WriteString(fmt.Sprintf("平均响应时间: %d ms\n", stats.AvgResponseTime))
	}
	file.WriteString("═══════════════════════════════════════════════════════\n")
	
	return nil
}

// SaveToJSON 保存请求日志到JSON文件
func (rl *RequestLogger) SaveToJSON(filename string) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	if len(rl.logs) == 0 {
		return fmt.Errorf("没有请求日志可保存")
	}
	
	// 构建完整的日志数据
	output := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339),
		"total_requests": len(rl.logs),
		"statistics":     rl.getInternalStatistics(),
		"logs":           rl.logs,
	}
	
	// 转换为JSON
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON编码失败: %v", err)
	}
	
	// 写入文件
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}
	
	return nil
}

// internalStats 内部统计结构
type internalStats struct {
	MethodCount     map[string]int
	StatusCount     map[int]int
	ErrorCount      int
	AvgResponseTime int64
}

// getInternalStatistics 获取内部统计(不加锁,内部使用)
func (rl *RequestLogger) getInternalStatistics() internalStats {
	methodCount := make(map[string]int)
	statusCount := make(map[int]int)
	var totalResponseTime int64
	errorCount := 0
	
	for _, log := range rl.logs {
		methodCount[log.Method]++
		if log.StatusCode > 0 {
			statusCount[log.StatusCode]++
		}
		totalResponseTime += log.ResponseTime
		if log.Error != "" {
			errorCount++
		}
	}
	
	avgResponseTime := int64(0)
	if len(rl.logs) > 0 {
		avgResponseTime = totalResponseTime / int64(len(rl.logs))
	}
	
	return internalStats{
		MethodCount:     methodCount,
		StatusCount:     statusCount,
		ErrorCount:      errorCount,
		AvgResponseTime: avgResponseTime,
	}
}

// PrintSummary 打印统计摘要
func (rl *RequestLogger) PrintSummary() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	if len(rl.logs) == 0 {
		fmt.Println("\n[请求日志] 没有记录任何请求")
		return
	}
	
	stats := rl.getInternalStatistics()
	
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 请求日志统计")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n🎯 总请求数: %d\n", len(rl.logs))
	
	fmt.Println("\n📋 按方法统计:")
	for method, count := range stats.MethodCount {
		percentage := float64(count) / float64(len(rl.logs)) * 100
		fmt.Printf("  %-6s: %5d (%.1f%%)\n", method, count, percentage)
	}
	
	fmt.Println("\n📈 按状态码统计:")
	// 按状态码排序
	codes := make([]int, 0, len(stats.StatusCount))
	for code := range stats.StatusCount {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	
	for _, code := range codes {
		count := stats.StatusCount[code]
		percentage := float64(count) / float64(len(rl.logs)) * 100
		statusEmoji := "✅"
		if code >= 400 {
			statusEmoji = "❌"
		} else if code >= 300 {
			statusEmoji = "↪️"
		}
		fmt.Printf("  %s %d: %5d (%.1f%%)\n", statusEmoji, code, count, percentage)
	}
	
	if stats.ErrorCount > 0 {
		fmt.Printf("\n⚠️  失败请求: %d (%.1f%%)\n", stats.ErrorCount, 
			float64(stats.ErrorCount)/float64(len(rl.logs))*100)
	}
	
	if stats.AvgResponseTime > 0 {
		fmt.Printf("\n⏱️  平均响应时间: %d ms\n", stats.AvgResponseTime)
	}
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// Clear 清空日志
func (rl *RequestLogger) Clear() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.logs = make([]RequestLog, 0)
}

// Enable 启用请求日志
func (rl *RequestLogger) Enable() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.enabled = true
}

// Disable 禁用请求日志
func (rl *RequestLogger) Disable() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.enabled = false
}

// IsEnabled 检查是否启用
func (rl *RequestLogger) IsEnabled() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	return rl.enabled
}

