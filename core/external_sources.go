package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ExternalSourceConfig 外部数据源配置
type ExternalSourceConfig struct {
	EnableWaybackMachine bool   // 启用Wayback Machine
	EnableVirusTotal     bool   // 启用VirusTotal
	EnableCommonCrawl    bool   // 启用CommonCrawl
	
	VirusTotalAPIKey     string // VirusTotal API密钥
	MaxResultsPerSource  int    // 每个数据源最大结果数
	Timeout              int    // 超时时间（秒）
}

// ExternalDataSource 外部数据源接口
type ExternalDataSource interface {
	FetchURLs(domain string) ([]string, error)
	GetName() string
}

// WaybackMachineSource Wayback Machine数据源
type WaybackMachineSource struct {
	maxResults int
	timeout    time.Duration
	client     *http.Client
}

// NewWaybackMachineSource 创建Wayback Machine数据源
func NewWaybackMachineSource(maxResults int, timeout time.Duration) *WaybackMachineSource {
	return &WaybackMachineSource{
		maxResults: maxResults,
		timeout:    timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetName 获取数据源名称
func (wm *WaybackMachineSource) GetName() string {
	return "Wayback Machine"
}

// FetchURLs 从Wayback Machine获取历史URL
func (wm *WaybackMachineSource) FetchURLs(domain string) ([]string, error) {
	// 构建API URL
	apiURL := fmt.Sprintf(
		"http://web.archive.org/cdx/search/cdx?url=%s/*&output=json&fl=original&collapse=urlkey&limit=%d",
		domain,
		wm.maxResults,
	)
	
	// 发送请求
	resp, err := wm.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("请求Wayback Machine失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Wayback Machine返回错误: %d", resp.StatusCode)
	}
	
	// 解析JSON响应
	var results [][]string
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	
	// 提取URL（跳过第一行header）
	urls := make([]string, 0)
	for i, result := range results {
		if i == 0 {
			continue // 跳过header
		}
		if len(result) > 0 {
			urls = append(urls, result[0])
		}
	}
	
	return urls, nil
}

// VirusTotalSource VirusTotal数据源
type VirusTotalSource struct {
	apiKey     string
	maxResults int
	timeout    time.Duration
	client     *http.Client
}

// NewVirusTotalSource 创建VirusTotal数据源
func NewVirusTotalSource(apiKey string, maxResults int, timeout time.Duration) *VirusTotalSource {
	return &VirusTotalSource{
		apiKey:     apiKey,
		maxResults: maxResults,
		timeout:    timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetName 获取数据源名称
func (vt *VirusTotalSource) GetName() string {
	return "VirusTotal"
}

// FetchURLs 从VirusTotal获取URL
func (vt *VirusTotalSource) FetchURLs(domain string) ([]string, error) {
	if vt.apiKey == "" {
		return nil, fmt.Errorf("VirusTotal API密钥未配置")
	}
	
	// 构建API URL
	apiURL := fmt.Sprintf("https://www.virustotal.com/api/v3/domains/%s/urls?limit=%d", domain, vt.maxResults)
	
	// 创建请求
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	
	// 添加API密钥
	req.Header.Set("x-apikey", vt.apiKey)
	
	// 发送请求
	resp, err := vt.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求VirusTotal失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("VirusTotal返回错误: %d, %s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var result struct {
		Data []struct {
			Attributes struct {
				URL string `json:"url"`
			} `json:"attributes"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	
	// 提取URL
	urls := make([]string, 0)
	for _, item := range result.Data {
		if item.Attributes.URL != "" {
			urls = append(urls, item.Attributes.URL)
		}
	}
	
	return urls, nil
}

// CommonCrawlSource CommonCrawl数据源
type CommonCrawlSource struct {
	maxResults int
	timeout    time.Duration
	client     *http.Client
}

// NewCommonCrawlSource 创建CommonCrawl数据源
func NewCommonCrawlSource(maxResults int, timeout time.Duration) *CommonCrawlSource {
	return &CommonCrawlSource{
		maxResults: maxResults,
		timeout:    timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetName 获取数据源名称
func (cc *CommonCrawlSource) GetName() string {
	return "CommonCrawl"
}

// FetchURLs 从CommonCrawl获取URL
func (cc *CommonCrawlSource) FetchURLs(domain string) ([]string, error) {
	// CommonCrawl Index API
	apiURL := fmt.Sprintf(
		"http://index.commoncrawl.org/CC-MAIN-2024-10-index?url=%s&output=json&limit=%d",
		url.QueryEscape(domain+"/*"),
		cc.maxResults,
	)
	
	resp, err := cc.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("请求CommonCrawl失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("CommonCrawl返回错误: %d", resp.StatusCode)
	}
	
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// CommonCrawl返回JSONL格式（每行一个JSON）
	lines := strings.Split(string(body), "\n")
	urls := make([]string, 0)
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		var result struct {
			URL string `json:"url"`
		}
		
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		
		if result.URL != "" {
			urls = append(urls, result.URL)
		}
	}
	
	return urls, nil
}

// ExternalSourceManager 外部数据源管理器
type ExternalSourceManager struct {
	sources []ExternalDataSource
	config  ExternalSourceConfig
}

// NewExternalSourceManager 创建外部数据源管理器
func NewExternalSourceManager(config ExternalSourceConfig) *ExternalSourceManager {
	manager := &ExternalSourceManager{
		sources: make([]ExternalDataSource, 0),
		config:  config,
	}
	
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	
	maxResults := config.MaxResultsPerSource
	if maxResults == 0 {
		maxResults = 1000
	}
	
	// 根据配置添加数据源
	if config.EnableWaybackMachine {
		manager.sources = append(manager.sources,
			NewWaybackMachineSource(maxResults, timeout))
	}
	
	if config.EnableVirusTotal && config.VirusTotalAPIKey != "" {
		manager.sources = append(manager.sources,
			NewVirusTotalSource(config.VirusTotalAPIKey, maxResults, timeout))
	}
	
	if config.EnableCommonCrawl {
		manager.sources = append(manager.sources,
			NewCommonCrawlSource(maxResults, timeout))
	}
	
	return manager
}

// FetchAllURLs 从所有数据源获取URL
func (esm *ExternalSourceManager) FetchAllURLs(domain string) map[string][]string {
	results := make(map[string][]string)
	
	for _, source := range esm.sources {
		fmt.Printf("📡 正在从 %s 获取历史URL...\n", source.GetName())
		
		urls, err := source.FetchURLs(domain)
		if err != nil {
			fmt.Printf("  ⚠️  %s 获取失败: %v\n", source.GetName(), err)
			continue
		}
		
		results[source.GetName()] = urls
		fmt.Printf("  ✅ %s 发现 %d 个URL\n", source.GetName(), len(urls))
	}
	
	return results
}

// GetUniqueURLs 获取去重后的所有URL
func (esm *ExternalSourceManager) GetUniqueURLs(domain string) []string {
	allResults := esm.FetchAllURLs(domain)
	
	// 使用map去重
	uniqueURLs := make(map[string]bool)
	for _, urls := range allResults {
		for _, url := range urls {
			uniqueURLs[url] = true
		}
	}
	
	// 转换为数组
	result := make([]string, 0, len(uniqueURLs))
	for url := range uniqueURLs {
		result = append(result, url)
	}
	
	return result
}

// PrintStatistics 打印统计信息
func (esm *ExternalSourceManager) PrintStatistics(domain string) {
	results := esm.FetchAllURLs(domain)
	
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("          外部数据源统计")
	fmt.Println(strings.Repeat("=", 70))
	
	totalURLs := 0
	for source, urls := range results {
		fmt.Printf("%-20s: %d 个URL\n", source, len(urls))
		totalURLs += len(urls)
	}
	
	fmt.Println(strings.Repeat("-", 70))
	
	// 去重统计
	uniqueURLs := make(map[string]bool)
	for _, urls := range results {
		for _, url := range urls {
			uniqueURLs[url] = true
		}
	}
	
	fmt.Printf("总计URL:              %d 个\n", totalURLs)
	fmt.Printf("去重后URL:            %d 个\n", len(uniqueURLs))
	if totalURLs > 0 {
		deduplicationRate := float64(totalURLs-len(uniqueURLs)) / float64(totalURLs) * 100
		fmt.Printf("去重率:               %.1f%%\n", deduplicationRate)
	}
	
	fmt.Println(strings.Repeat("=", 70))
}

