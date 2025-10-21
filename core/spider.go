package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	
	"spider-golang/config"
)

// Spider 主爬虫协调器
type Spider struct {
	config             *config.Config
	staticCrawler      StaticCrawler
	dynamicCrawler     DynamicCrawler
	jsAnalyzer         *JSAnalyzer
	paramHandler       *ParamHandler
	duplicateHandler   *DuplicateHandler
	smartDeduplication *SmartDeduplication
	hiddenPathDiscovery *HiddenPathDiscovery
	cdnDetector        *CDNDetector // CDN检测器
	workerPool         *WorkerPool  // 并发工作池
	results            []*Result
	externalLinks      []string // 记录外部链接
	hiddenPaths        []string // 记录隐藏路径
	securityFindings   []string // 记录安全发现
	crossDomainJS      []string // 记录跨域JS发现的URL
	mutex              sync.Mutex
	targetDomain       string          // 目标域名
	visitedURLs        map[string]bool // 已访问URL
}

// NewSpider 创建爬虫实例
func NewSpider(cfg *config.Config) *Spider {
	// 创建结果通道和停止通道
	resultChan := make(chan Result, 100)
	stopChan := make(chan struct{})
	
	// 计算并发worker数量（默认10个，可配置）
	workerCount := 10
	if cfg.DepthSettings.MaxDepth > 2 {
		workerCount = 15 // 深度爬取时增加worker数
	}
	
	// 速率限制（每秒最多20个请求，避免过载）
	maxQPS := 20
	
	spider := &Spider{
		config:             cfg,
		staticCrawler:      NewStaticCrawler(cfg, resultChan, stopChan),
		dynamicCrawler:     NewDynamicCrawler(),
		jsAnalyzer:         NewJSAnalyzer(),
		paramHandler:       NewParamHandler(),
		duplicateHandler:   NewDuplicateHandler(cfg.DeduplicationSettings.SimilarityThreshold),
		smartDeduplication: NewSmartDeduplication(), // 初始化智能去重
		cdnDetector:        NewCDNDetector(), // 初始化CDN检测器
		workerPool:         NewWorkerPool(workerCount, maxQPS), // 初始化工作池
		hiddenPathDiscovery: nil, // 将在Start方法中初始化，需要用户代理
		results:            make([]*Result, 0),
		externalLinks:      make([]string, 0),
		hiddenPaths:        make([]string, 0),
		securityFindings:   make([]string, 0),
		crossDomainJS:      make([]string, 0),
		visitedURLs:        make(map[string]bool),
	}
	
	// 配置各个组件
	spider.staticCrawler.Configure(cfg)
	spider.dynamicCrawler.Configure(cfg)
	
	// 设置JS分析器的目标域名
	spider.jsAnalyzer.SetTargetDomain(cfg.TargetURL)
	
	return spider
}

// Start 开始爬取
func (s *Spider) Start(targetURL string) error {
	// 解析目标URL并提取域名
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("无效的URL: %v", err)
	}
	s.targetDomain = parsedURL.Host
	
	// 设置JS分析器的目标域名
	s.jsAnalyzer.SetTargetDomain(s.targetDomain)
	
	// 检查是否重复
	if s.duplicateHandler.IsDuplicateURL(targetURL) {
		return fmt.Errorf("URL已处理过: %s", targetURL)
	}
	
	fmt.Printf("开始爬取URL: %s\n", targetURL)
	fmt.Printf("限制域名范围: %s\n", s.targetDomain)
	fmt.Printf("跨域JS分析: 已启用（支持CDN和同源域名）\n")
	
	// 初始化隐藏路径发现器
	userAgent := ""
	if len(s.config.AntiDetectionSettings.UserAgents) > 0 {
		userAgent = s.config.AntiDetectionSettings.UserAgents[0]
	}
	s.hiddenPathDiscovery = NewHiddenPathDiscovery(targetURL, userAgent)
	
	// 开始隐藏路径发现
	fmt.Println("开始隐藏路径发现...")
	hiddenPaths := s.hiddenPathDiscovery.DiscoverAllHiddenPaths()
	s.mutex.Lock()
	s.hiddenPaths = append(s.hiddenPaths, hiddenPaths...)
	s.mutex.Unlock()
	fmt.Printf("发现 %d 个隐藏路径\n", len(hiddenPaths))
	
	// 根据配置决定使用哪种爬虫策略
	if s.config.StrategySettings.EnableStaticCrawler {
		fmt.Println("使用静态爬虫...")
		result, err := s.staticCrawler.Crawl(parsedURL)
		if err != nil {
			fmt.Printf("静态爬虫错误: %v\n", err)
		} else {
			s.addResult(result)
			fmt.Printf("静态爬虫完成，发现 %d 个链接, %d 个资源, %d 个表单, %d 个API\n", 
				len(result.Links), len(result.Assets), len(result.Forms), len(result.APIs))
		}
	}
	
	// 如果启用了动态爬虫且静态爬虫未找到足够内容，使用动态爬虫
	if s.config.StrategySettings.EnableDynamicCrawler && 
		(len(s.results) == 0 || s.shouldUseDynamicCrawler()) {
		fmt.Println("使用动态爬虫...")
		result, err := s.dynamicCrawler.Crawl(parsedURL)
		if err != nil {
			fmt.Printf("动态爬虫错误: %v\n", err)
		} else {
			s.addResult(result)
			fmt.Printf("动态爬虫完成，发现 %d 个链接, %d 个资源, %d 个表单, %d 个API\n", 
				len(result.Links), len(result.Assets), len(result.Forms), len(result.APIs))
		}
	}
	
	// 处理发现的参数
	s.processParams(targetURL)
	
	// 分析跨域JS文件（在递归爬取之前）
	s.processCrossDomainJS()
	
	// 如果启用了递归爬取，继续爬取发现的链接
	if s.config.DepthSettings.MaxDepth > 1 {
		s.crawlRecursively()
	}
	
	return nil
}

// shouldUseDynamicCrawler 判断是否需要使用动态爬虫
func (s *Spider) shouldUseDynamicCrawler() bool {
	// 如果没有发现足够的链接或API，可能需要动态爬虫
	if len(s.results) == 0 {
		return true
	}
	
	// 检查最近的结果
	lastResult := s.results[len(s.results)-1]
	// 降低触发动态爬虫的阈值
	if len(lastResult.Links) < 10 && len(lastResult.APIs) < 5 {
		return true
	}
	
	return false
}

// addResult 添加爬取结果
func (s *Spider) addResult(result *Result) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.results = append(s.results, result)
}

// GetResults 获取所有爬取结果
func (s *Spider) GetResults() []*Result {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// 返回结果副本
	results := make([]*Result, len(s.results))
	copy(results, s.results)
	return results
}

// processParams 处理参数变体生成和安全分析
func (s *Spider) processParams(rawURL string) []string {
	// 提取参数
	params, err := s.paramHandler.ExtractParams(rawURL)
	if err != nil {
		// 如果提取参数失败，至少返回原始URL
		return []string{rawURL}
	}
	
	// 对每个参数进行安全分析
	for paramName := range params {
		risk, level := s.paramHandler.AnalyzeParameterSecurity(paramName)
		if level >= 2 { // 中等风险以上
			finding := fmt.Sprintf("SECURITY_PARAM: %s - %s (Risk Level: %d)", paramName, risk, level)
			s.mutex.Lock()
			s.securityFindings = append(s.securityFindings, finding)
			s.mutex.Unlock()
			fmt.Printf("安全发现: %s\n", finding)
		}
	}
	
	// 生成参数变体
	variations := s.paramHandler.GenerateParamVariations(rawURL)
	
	// 生成安全测试变体
	securityVariations := s.paramHandler.GenerateSecurityTestVariations(rawURL)
	variations = append(variations, securityVariations...)
	
	// 如果没有生成变体，返回原始URL
	if len(variations) == 0 {
		return []string{rawURL}
	}
	
	// 打印生成的变体
	fmt.Printf("为URL %s 生成 %d 个参数变体（包括安全测试）\n", rawURL, len(variations))
	for i, variation := range variations {
		if i < 10 { // 只显示前10个，避免输出过多
			fmt.Printf("  变体: %s\n", variation)
		}
	}
	if len(variations) > 10 {
		fmt.Printf("  ... 还有 %d 个变体\n", len(variations)-10)
	}
	
	return variations
}

// processCrossDomainJS 处理跨域JS文件
func (s *Spider) processCrossDomainJS() {
	fmt.Println("\n开始分析跨域JS文件...")
	
	// 收集所有资源链接
	allAssets := make(map[string]bool)
	
	s.mutex.Lock()
	for _, result := range s.results {
		for _, asset := range result.Assets {
			allAssets[asset] = true
		}
		// 也检查Links中的JS文件
		for _, link := range result.Links {
			if strings.HasSuffix(strings.ToLower(link), ".js") {
				allAssets[link] = true
			}
		}
	}
	s.mutex.Unlock()
	
	// 过滤出需要分析的JS文件
	jsToAnalyze := make([]string, 0)
	for asset := range allAssets {
		// 检查是否为JS文件
		if !strings.HasSuffix(strings.ToLower(asset), ".js") {
			continue
		}
		
		// 解析URL
		parsedURL, err := url.Parse(asset)
		if err != nil {
			continue
		}
		
		domain := parsedURL.Host
		if domain == "" {
			continue
		}
		
		// 检查是否需要分析
		shouldAnalyze := false
		reason := ""
		
		// 1. 是目标域名 - 已经正常爬取了，不需要特殊处理
		if domain == s.targetDomain {
			continue
		}
		
		// 2. 是同源域名
		if s.cdnDetector.IsSameBaseDomain(domain, s.targetDomain) {
			shouldAnalyze = true
			reason = "同源域名"
		}
		
		// 3. 是已知CDN
		if s.cdnDetector.IsCDN(domain) {
			shouldAnalyze = true
			cdnInfo := s.cdnDetector.GetCDNInfo(domain)
			reason = cdnInfo
		}
		
		if shouldAnalyze {
			jsToAnalyze = append(jsToAnalyze, asset)
			fmt.Printf("  发现跨域JS: %s (%s)\n", asset, reason)
		}
	}
	
	if len(jsToAnalyze) == 0 {
		fmt.Println("未发现需要分析的跨域JS文件")
		return
	}
	
	fmt.Printf("准备分析 %d 个跨域JS文件...\n", len(jsToAnalyze))
	
	// 分析每个JS文件
	totalURLsFound := 0
	for _, jsURL := range jsToAnalyze {
		urls := s.analyzeExternalJS(jsURL)
		if len(urls) > 0 {
			fmt.Printf("  从 %s 提取了 %d 个URL\n", jsURL, len(urls))
			totalURLsFound += len(urls)
			
			// 添加到跨域JS发现列表
			s.mutex.Lock()
			s.crossDomainJS = append(s.crossDomainJS, urls...)
			s.mutex.Unlock()
			
			// 添加到爬取队列（如果启用递归爬取）
			if s.config.DepthSettings.MaxDepth > 1 {
				for _, u := range urls {
					// 添加到结果中（作为发现的链接）
					if len(s.results) > 0 {
						s.results[0].Links = append(s.results[0].Links, u)
					}
				}
			}
		}
	}
	
	fmt.Printf("跨域JS分析完成！共从 %d 个JS文件中提取了 %d 个目标域名URL\n\n", len(jsToAnalyze), totalURLsFound)
}

// analyzeExternalJS 下载并分析外部JS文件
func (s *Spider) analyzeExternalJS(jsURL string) []string {
	// 使用HTTP客户端下载JS内容
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	req, err := http.NewRequest("GET", jsURL, nil)
	if err != nil {
		fmt.Printf("    创建请求失败: %v\n", err)
		return []string{}
	}
	
	// 设置User-Agent
	if len(s.config.AntiDetectionSettings.UserAgents) > 0 {
		req.Header.Set("User-Agent", s.config.AntiDetectionSettings.UserAgents[0])
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	}
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("    下载失败: %v\n", err)
		return []string{}
	}
	defer resp.Body.Close()
	
	// 检查状态码
	if resp.StatusCode != 200 {
		fmt.Printf("    HTTP %d\n", resp.StatusCode)
		return []string{}
	}
	
	// 限制文件大小（最大5MB）
	const maxSize = 5 * 1024 * 1024
	limitedReader := &io.LimitedReader{R: resp.Body, N: maxSize}
	
	// 读取内容
	content, err := ioutil.ReadAll(limitedReader)
	if err != nil {
		fmt.Printf("    读取内容失败: %v\n", err)
		return []string{}
	}
	
	// 使用JS分析器提取URL
	urls := s.jsAnalyzer.ExtractRelativeURLs(string(content))
	
	return urls
}

// crawlRecursively 递归爬取发现的链接（并发版本）
func (s *Spider) crawlRecursively() {
	fmt.Println("开始并发递归爬取...")
	
	// 收集所有发现的链接
	allLinks := make(map[string]bool)
	externalLinks := make([]string, 0)
	
	s.mutex.Lock()
	for _, result := range s.results {
		for _, link := range result.Links {
			// 解析链接域名
			parsedURL, err := url.Parse(link)
			if err != nil {
				continue
			}
			
			// 检查是否为同一域名
			if parsedURL.Host == s.targetDomain || parsedURL.Host == "" {
				// 规范化URL
				normalizedURL, err := s.paramHandler.NormalizeURL(link)
				if err == nil {
					allLinks[normalizedURL] = true
				} else {
					allLinks[link] = true
				}
			} else {
				// 记录外部链接但不爬取
				externalLinks = append(externalLinks, link)
			}
		}
	}
	s.mutex.Unlock()
	
	// 记录外部链接
	if len(externalLinks) > 0 {
		s.mutex.Lock()
		s.externalLinks = append(s.externalLinks, externalLinks...)
		s.mutex.Unlock()
		fmt.Printf("发现 %d 个外部链接（已记录但不爬取）\n", len(externalLinks))
	}
	
	// 限制递归深度
	if s.config.DepthSettings.MaxDepth <= 1 {
		return
	}
	
	// 过滤并准备待爬取的URL
	tasksToSubmit := make([]string, 0)
	for link := range allLinks {
		// 检查是否重复
		s.mutex.Lock()
		if s.visitedURLs[link] {
			s.mutex.Unlock()
			continue
		}
		s.visitedURLs[link] = true
		s.mutex.Unlock()
		
		// 检查是否已被去重处理器处理过
		if s.duplicateHandler.IsDuplicateURL(link) {
			continue
		}
		
		// 验证链接格式
		if !IsValidURL(link) {
			continue
		}
		
		tasksToSubmit = append(tasksToSubmit, link)
		
		// 限制最大爬取数量
		if len(tasksToSubmit) >= 100 {
			break
		}
	}
	
	if len(tasksToSubmit) == 0 {
		fmt.Println("没有需要递归爬取的链接")
		return
	}
	
	fmt.Printf("准备并发爬取 %d 个链接...\n", len(tasksToSubmit))
	
	// 启动工作池
	s.workerPool.Start(func(task Task) (*Result, error) {
		return s.crawlURL(task.URL)
	})
	
	// 提交所有任务
	for _, link := range tasksToSubmit {
		task := Task{
			URL:   link,
			Depth: s.config.DepthSettings.MaxDepth - 1,
		}
		if err := s.workerPool.Submit(task); err != nil {
			fmt.Printf("提交任务失败 %s: %v\n", link, err)
		}
	}
	
	// 显示进度
	go s.showProgress()
	
	// 等待所有任务完成
	s.workerPool.Wait()
	
	// 收集结果
	results := s.workerPool.GetResults()
	s.mutex.Lock()
	s.results = append(s.results, results...)
	s.mutex.Unlock()
	
	// 停止工作池
	s.workerPool.Stop()
	
	// 显示统计
	stats := s.workerPool.GetStats()
	fmt.Printf("\n并发爬取完成！\n")
	fmt.Printf("  总任务: %d\n", stats["total"])
	fmt.Printf("  成功: %d\n", stats["completed"]-stats["failed"])
	fmt.Printf("  失败: %d\n", stats["failed"])
}

// crawlURL 爬取单个URL（供工作池使用）
func (s *Spider) crawlURL(targetURL string) (*Result, error) {
	// 解析URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("URL解析失败: %v", err)
	}
	
	// 使用静态爬虫
	result, err := s.staticCrawler.Crawl(parsedURL)
	if err != nil {
		// 如果静态爬虫失败，尝试动态爬虫
		if s.config.StrategySettings.EnableDynamicCrawler {
			result, err = s.dynamicCrawler.Crawl(parsedURL)
			if err != nil {
				return nil, fmt.Errorf("爬取失败: %v", err)
			}
		} else {
			return nil, fmt.Errorf("静态爬虫失败: %v", err)
		}
	}
	
	return result, nil
}

// showProgress 显示爬取进度
func (s *Spider) showProgress() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			progress := s.workerPool.GetProgress()
			stats := s.workerPool.GetStats()
			
			// 计算进度条
			barWidth := 30
			filled := int(progress * float64(barWidth) / 100)
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			
			fmt.Printf("\r[进度] %s %.1f%% (%d/%d)", bar, progress, stats["completed"], stats["total"])
			
			if stats["completed"] >= stats["total"] {
				fmt.Println()
				return
			}
		}
	}
}

// Stop 停止爬取
func (s *Spider) Stop() {
	fmt.Println("停止爬取...")
	s.staticCrawler.Stop()
	s.dynamicCrawler.Stop()
}

// IsValidURL 检查URL是否为有效的HTTP/HTTPS链接
func IsValidURL(url string) bool {
	// 检查是否为空
	if url == "" {
		return false
	}
	
	// 检查是否为javascript:或mailto:等非HTTP链接
	if strings.HasPrefix(url, "javascript:") || 
	   strings.HasPrefix(url, "mailto:") || 
	   strings.HasPrefix(url, "tel:") || 
	   strings.HasPrefix(url, "sms:") || 
	   strings.HasPrefix(url, "ftp:") || 
	   strings.HasPrefix(url, "file:") {
		return false
	}
	
	// 检查是否为相对链接（不包含协议）
	if !strings.Contains(url, "://") && !strings.HasPrefix(url, "//") {
		return true
	}
	
	// 检查是否为HTTP/HTTPS协议
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return true
	}
	
	// 其他情况视为无效
	return false
}

// ExportResults 导出结果
func (s *Spider) ExportResults() map[string]interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	exportData := make(map[string]interface{})
	exportData["total_results"] = len(s.results)
	
	// 统计信息
	totalLinks := 0
	totalAssets := 0
	totalForms := 0
	totalAPIs := 0
	
	// 详细结果数据
	detailedResults := make([]map[string]interface{}, 0)
	allLinks := make([]string, 0)
	allAPIs := make([]string, 0)
	
	// 使用智能去重处理所有URL和表单
	for _, result := range s.results {
		totalLinks += len(result.Links)
		totalAssets += len(result.Assets)
		totalForms += len(result.Forms)
		totalAPIs += len(result.APIs)
		
		// 保存详细结果
		resultData := make(map[string]interface{})
		resultData["url"] = result.URL
		resultData["status_code"] = result.StatusCode
		resultData["content_type"] = result.ContentType
		resultData["links"] = result.Links
		resultData["assets"] = result.Assets
		resultData["forms"] = result.Forms
		resultData["apis"] = result.APIs
		detailedResults = append(detailedResults, resultData)
		
		// 处理链接进行智能去重
		for _, link := range result.Links {
			s.smartDeduplication.ProcessURL(link)
		}
		
		// 处理表单进行智能去重
		for _, form := range result.Forms {
			s.smartDeduplication.ProcessForm(form)
		}
		
		// 收集所有链接
		allLinks = append(allLinks, result.Links...)
		allAPIs = append(allAPIs, result.APIs...)
	}
	
	exportData["total_links"] = totalLinks
	exportData["total_assets"] = totalAssets
	exportData["total_forms"] = totalForms
	exportData["total_apis"] = totalAPIs
	exportData["detailed_results"] = detailedResults
	exportData["links"] = allLinks
	exportData["apis"] = allAPIs
	exportData["external_links"] = s.externalLinks
	exportData["hidden_paths"] = s.hiddenPaths
	exportData["security_findings"] = s.securityFindings
	exportData["cross_domain_js_urls"] = s.crossDomainJS
	exportData["total_hidden_paths"] = len(s.hiddenPaths)
	exportData["total_security_findings"] = len(s.securityFindings)
	exportData["total_cross_domain_js_urls"] = len(s.crossDomainJS)
	
	// 添加智能去重统计
	exportData["deduplication_stats"] = s.smartDeduplication.GetDeduplicationStats()
	exportData["unique_url_patterns"] = s.smartDeduplication.GetUniqueURLs()
	exportData["unique_form_patterns"] = s.smartDeduplication.GetUniqueForms()
	
	return exportData
}