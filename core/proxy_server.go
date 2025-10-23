package core

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ProxyServer HTTP代理服务器
type ProxyServer struct {
	// 监听地址
	listenAddr string
	
	// 爬虫实例（用于实时分析）
	spider *Spider
	
	// 拦截的请求记录
	interceptedRequests  []*InterceptedRequest
	interceptedResponses []*InterceptedResponse
	mutex                sync.RWMutex
	
	// 过滤器配置
	targetDomain string   // 只记录目标域名的请求
	filters      []string // URL过滤规则
	
	// 统计信息
	stats *ProxyStats
	
	// 服务器实例
	server   *http.Server
	running  bool
	stopChan chan struct{}
}

// InterceptedRequest 拦截的请求
type InterceptedRequest struct {
	Timestamp   time.Time
	Method      string
	URL         string
	Headers     map[string]string
	Body        string
	ContentType string
}

// InterceptedResponse 拦截的响应
type InterceptedResponse struct {
	Timestamp   time.Time
	URL         string
	StatusCode  int
	Headers     map[string]string
	Body        string
	ContentType string
	Size        int64
}

// ProxyStats 代理统计信息
type ProxyStats struct {
	TotalRequests   int64
	TotalResponses  int64
	TotalBytes      int64
	StartTime       time.Time
	RequestsByHost  map[string]int64
	RequestsByType  map[string]int64
	mutex           sync.RWMutex
}

// NewProxyServer 创建代理服务器
func NewProxyServer(listenAddr string, targetDomain string) *ProxyServer {
	return &ProxyServer{
		listenAddr:           listenAddr,
		targetDomain:         targetDomain,
		interceptedRequests:  make([]*InterceptedRequest, 0),
		interceptedResponses: make([]*InterceptedResponse, 0),
		filters:              make([]string, 0),
		stats: &ProxyStats{
			StartTime:      time.Now(),
			RequestsByHost: make(map[string]int64),
			RequestsByType: make(map[string]int64),
		},
		running:  false,
		stopChan: make(chan struct{}),
	}
}

// SetSpider 设置爬虫实例（用于实时分析）
func (ps *ProxyServer) SetSpider(spider *Spider) {
	ps.spider = spider
}

// AddFilter 添加URL过滤规则
func (ps *ProxyServer) AddFilter(filter string) {
	ps.filters = append(ps.filters, filter)
}

// Start 启动代理服务器
func (ps *ProxyServer) Start() error {
	if ps.running {
		return fmt.Errorf("代理服务器已在运行")
	}
	
	ps.running = true
	
	// 创建HTTP服务器
	ps.server = &http.Server{
		Addr:    ps.listenAddr,
		Handler: http.HandlerFunc(ps.handleRequest),
		// 禁用HTTP/2，简化代理实现
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	
	log.Printf("代理服务器启动在: %s", ps.listenAddr)
	log.Printf("目标域名: %s", ps.targetDomain)
	log.Printf("配置浏览器代理为: http://%s", ps.listenAddr)
	
	// 在goroutine中启动服务器
	go func() {
		if err := ps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("代理服务器错误: %v", err)
		}
	}()
	
	return nil
}

// Stop 停止代理服务器
func (ps *ProxyServer) Stop() error {
	if !ps.running {
		return fmt.Errorf("代理服务器未运行")
	}
	
	ps.running = false
	close(ps.stopChan)
	
	if ps.server != nil {
		return ps.server.Close()
	}
	
	return nil
}

// handleRequest 处理代理请求
func (ps *ProxyServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// 更新统计
	ps.updateStats(r)
	
	// 处理CONNECT方法（HTTPS隧道）
	if r.Method == http.MethodConnect {
		ps.handleHTTPSConnect(w, r)
		return
	}
	
	// 处理普通HTTP请求
	ps.handleHTTPRequest(w, r)
}

// handleHTTPRequest 处理HTTP请求
func (ps *ProxyServer) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// 判断是否需要拦截
	if !ps.shouldIntercept(r.URL.String()) {
		// 直接转发，不记录
		ps.forwardRequest(w, r)
		return
	}
	
	// 记录请求
	interceptedReq := ps.recordRequest(r)
	
	// 转发请求并获取响应
	resp, err := ps.sendRequest(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("代理请求失败: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	
	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("读取响应失败: %v", err), http.StatusBadGateway)
		return
	}
	
	// 记录响应
	ps.recordResponse(r.URL.String(), resp, bodyBytes)
	
	// 实时分析（如果配置了爬虫）
	if ps.spider != nil && resp.StatusCode == http.StatusOK {
		ps.analyzeResponse(r.URL.String(), resp, string(bodyBytes))
	}
	
	// 转发响应给客户端
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(bodyBytes)
	
	// 输出拦截信息
	log.Printf("[拦截] %s %s - %d (%d bytes)", 
		interceptedReq.Method, 
		interceptedReq.URL, 
		resp.StatusCode, 
		len(bodyBytes))
}

// handleHTTPSConnect 处理HTTPS CONNECT隧道
func (ps *ProxyServer) handleHTTPSConnect(w http.ResponseWriter, r *http.Request) {
	// 对于HTTPS，我们只能建立隧道，无法看到内容（除非做中间人）
	// 这里实现简单的隧道转发
	
	targetConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("无法连接目标: %v", err), http.StatusBadGateway)
		return
	}
	defer targetConn.Close()
	
	// 劫持连接
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "不支持劫持", http.StatusInternalServerError)
		return
	}
	
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, fmt.Sprintf("劫持失败: %v", err), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()
	
	// 告诉客户端连接已建立
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	
	// 双向转发数据
	go io.Copy(targetConn, clientConn)
	io.Copy(clientConn, targetConn)
	
	log.Printf("[HTTPS隧道] %s", r.Host)
}

// sendRequest 发送代理请求
func (ps *ProxyServer) sendRequest(r *http.Request) (*http.Response, error) {
	// 创建新的请求
	outReq := &http.Request{
		Method: r.Method,
		URL:    r.URL,
		Header: r.Header.Clone(),
		Body:   r.Body,
	}
	
	// 移除代理相关的头
	outReq.Header.Del("Proxy-Connection")
	outReq.Header.Del("Proxy-Authenticate")
	outReq.Header.Del("Proxy-Authorization")
	
	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不自动跟随重定向
		},
	}
	
	return client.Do(outReq)
}

// forwardRequest 直接转发请求（不记录）
func (ps *ProxyServer) forwardRequest(w http.ResponseWriter, r *http.Request) {
	resp, err := ps.sendRequest(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("转发失败: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	
	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// shouldIntercept 判断是否应该拦截此请求
func (ps *ProxyServer) shouldIntercept(urlStr string) bool {
	// 如果没有设置目标域名，拦截所有请求
	if ps.targetDomain == "" {
		return true
	}
	
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	
	// 检查是否匹配目标域名
	if !strings.Contains(parsedURL.Host, ps.targetDomain) {
		return false
	}
	
	// 检查过滤规则
	for _, filter := range ps.filters {
		if strings.Contains(urlStr, filter) {
			return false
		}
	}
	
	return true
}

// recordRequest 记录请求
func (ps *ProxyServer) recordRequest(r *http.Request) *InterceptedRequest {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	// 读取请求体（如果有）
	var bodyStr string
	if r.Body != nil {
		bodyBytes, _ := io.ReadAll(r.Body)
		bodyStr = string(bodyBytes)
		// 重新设置Body
		r.Body = io.NopCloser(strings.NewReader(bodyStr))
	}
	
	// 记录请求头
	headers := make(map[string]string)
	for key, values := range r.Header {
		headers[key] = strings.Join(values, ", ")
	}
	
	req := &InterceptedRequest{
		Timestamp:   time.Now(),
		Method:      r.Method,
		URL:         r.URL.String(),
		Headers:     headers,
		Body:        bodyStr,
		ContentType: r.Header.Get("Content-Type"),
	}
	
	ps.interceptedRequests = append(ps.interceptedRequests, req)
	
	return req
}

// recordResponse 记录响应
func (ps *ProxyServer) recordResponse(urlStr string, resp *http.Response, body []byte) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	// 记录响应头
	headers := make(map[string]string)
	for key, values := range resp.Header {
		headers[key] = strings.Join(values, ", ")
	}
	
	response := &InterceptedResponse{
		Timestamp:   time.Now(),
		URL:         urlStr,
		StatusCode:  resp.StatusCode,
		Headers:     headers,
		Body:        string(body),
		ContentType: resp.Header.Get("Content-Type"),
		Size:        int64(len(body)),
	}
	
	ps.interceptedResponses = append(ps.interceptedResponses, response)
	
	// 更新统计
	ps.stats.mutex.Lock()
	ps.stats.TotalBytes += response.Size
	ps.stats.TotalResponses++
	ps.stats.mutex.Unlock()
}

// analyzeResponse 实时分析响应
func (ps *ProxyServer) analyzeResponse(urlStr string, resp *http.Response, body string) {
	// 如果是HTML内容，进行分析
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		// 提取链接、表单等
		// 这里可以调用爬虫的分析功能
		log.Printf("[分析] HTML页面: %s (%d字节)", urlStr, len(body))
		
		// TODO: 集成到爬虫的分析流程
		// ps.spider.analyzeHTML(urlStr, body)
	}
}

// updateStats 更新统计信息
func (ps *ProxyServer) updateStats(r *http.Request) {
	ps.stats.mutex.Lock()
	defer ps.stats.mutex.Unlock()
	
	ps.stats.TotalRequests++
	
	// 按主机统计
	host := r.URL.Host
	if host == "" {
		host = r.Host
	}
	ps.stats.RequestsByHost[host]++
	
	// 按类型统计
	ps.stats.RequestsByType[r.Method]++
}

// GetInterceptedRequests 获取拦截的请求
func (ps *ProxyServer) GetInterceptedRequests() []*InterceptedRequest {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	requests := make([]*InterceptedRequest, len(ps.interceptedRequests))
	copy(requests, ps.interceptedRequests)
	return requests
}

// GetInterceptedResponses 获取拦截的响应
func (ps *ProxyServer) GetInterceptedResponses() []*InterceptedResponse {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	responses := make([]*InterceptedResponse, len(ps.interceptedResponses))
	copy(responses, ps.interceptedResponses)
	return responses
}

// GetStatistics 获取统计信息
func (ps *ProxyServer) GetStatistics() map[string]interface{} {
	ps.stats.mutex.RLock()
	defer ps.stats.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_requests"] = ps.stats.TotalRequests
	stats["total_responses"] = ps.stats.TotalResponses
	stats["total_bytes"] = ps.stats.TotalBytes
	stats["uptime"] = time.Since(ps.stats.StartTime).Seconds()
	stats["requests_by_host"] = ps.stats.RequestsByHost
	stats["requests_by_type"] = ps.stats.RequestsByType
	
	return stats
}

// ExportToHAR 导出为HAR格式（用于分析）
func (ps *ProxyServer) ExportToHAR(filename string) error {
	// TODO: 实现HAR导出
	return fmt.Errorf("HAR导出功能待实现")
}

// Clear 清除所有拦截记录
func (ps *ProxyServer) Clear() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	ps.interceptedRequests = make([]*InterceptedRequest, 0)
	ps.interceptedResponses = make([]*InterceptedResponse, 0)
}

// IsRunning 检查服务器是否运行中
func (ps *ProxyServer) IsRunning() bool {
	return ps.running
}

// PrintStatistics 打印统计信息
func (ps *ProxyServer) PrintStatistics() {
	stats := ps.GetStatistics()
	
	fmt.Println("\n=== 代理服务器统计 ===")
	fmt.Printf("总请求数: %d\n", stats["total_requests"])
	fmt.Printf("总响应数: %d\n", stats["total_responses"])
	fmt.Printf("总流量: %.2f MB\n", float64(stats["total_bytes"].(int64))/1024/1024)
	fmt.Printf("运行时间: %.2f 秒\n", stats["uptime"])
	
	fmt.Println("\n按主机统计:")
	if hostStats, ok := stats["requests_by_host"].(map[string]int64); ok {
		for host, count := range hostStats {
			fmt.Printf("  %s: %d 请求\n", host, count)
		}
	}
	
	fmt.Println("\n按方法统计:")
	if typeStats, ok := stats["requests_by_type"].(map[string]int64); ok {
		for method, count := range typeStats {
			fmt.Printf("  %s: %d 请求\n", method, count)
		}
	}
}

