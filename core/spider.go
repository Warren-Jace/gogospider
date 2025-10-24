package core

import (
	"fmt"
	"io"
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
	
	// 新增优化组件
	formFiller         *SmartFormFiller        // 智能表单填充器
	advancedScope      *AdvancedScope          // 高级作用域控制
	perfOptimizer      *PerformanceOptimizer   // 性能优化器
	
	// 高级功能组件
	techDetector       *TechStackDetector      // 技术栈检测器
	sensitiveDetector  *SensitiveInfoDetector  // 敏感信息检测器
	passiveCrawler     *PassiveCrawler         // 被动爬取器
	subdomainExtractor *SubdomainExtractor     // 子域名提取器
	domSimilarity      *DOMSimilarityDetector  // DOM相似度检测器
	sitemapCrawler     *SitemapCrawler         // Sitemap爬取器
	assetClassifier    *AssetClassifier        // 静态资源分类器
	ipDetector         *IPDetector             // IP地址检测器
	
	results            []*Result
	sitemapURLs        []string // 从sitemap发现的URL
	robotsURLs         []string // 从robots.txt发现的URL
	externalLinks      []string // 记录外部链接
	hiddenPaths        []string // 记录隐藏路径
	securityFindings   []string // 记录安全发现
	crossDomainJS      []string // 记录跨域JS发现的URL
	detectedTechs      []*TechInfo // 检测到的技术栈
	sensitiveFindings  []*SensitiveInfo // 敏感信息发现
	mutex              sync.Mutex
	targetDomain       string          // 目标域名
	visitedURLs        map[string]bool // 已访问URL
	
	// 资源管理（优化：防止泄漏）
	done               chan struct{}   // 完成信号
	wg                 sync.WaitGroup  // 等待所有goroutine完成
	closed             bool            // 是否已关闭
	closeMux           sync.Mutex      // 关闭锁
}

// NewSpider 创建爬虫实例
func NewSpider(cfg *config.Config) *Spider {
	// 创建结果通道和停止通道
	resultChan := make(chan Result, 100)
	stopChan := make(chan struct{})
	
	// 计算并发worker数量（默认20个，可配置）
	workerCount := 20
	if cfg.DepthSettings.MaxDepth > 2 {
		workerCount = 30 // 深度爬取时增加worker数
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
		
		// 初始化新增组件
		formFiller:         NewSmartFormFiller(),          // 智能表单填充器
		advancedScope:      nil,                           // 将在Start中初始化
		perfOptimizer:      NewPerformanceOptimizer(500),  // 性能优化器（限制500MB）
		
		// 初始化高级功能组件
		techDetector:       NewTechStackDetector(),        // 技术栈检测器
		sensitiveDetector:  NewSensitiveInfoDetector(),    // 敏感信息检测器
		passiveCrawler:     nil,                           // 按需创建
		domSimilarity:      NewDOMSimilarityDetector(0.85), // DOM相似度检测器（阈值85%）
		sitemapCrawler:     NewSitemapCrawler(),           // Sitemap爬取器
		assetClassifier:    NewAssetClassifier(),          // 静态资源分类器
		ipDetector:         NewIPDetector(),               // IP地址检测器
		
		hiddenPathDiscovery: nil, // 将在Start方法中初始化，需要用户代理
		results:            make([]*Result, 0),
		externalLinks:      make([]string, 0),
		hiddenPaths:        make([]string, 0),
		securityFindings:   make([]string, 0),
		crossDomainJS:      make([]string, 0),
		detectedTechs:      make([]*TechInfo, 0),
		sensitiveFindings:  make([]*SensitiveInfo, 0),
		sitemapURLs:        make([]string, 0),
		robotsURLs:         make([]string, 0),
		visitedURLs:        make(map[string]bool),
		
		// 初始化资源管理
		done:               make(chan struct{}),
		closed:             false,
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
	// 确保资源清理（优化：防止泄漏）
	defer s.cleanup()
	
	// 解析目标URL并提取域名
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("无效的URL: %v", err)
	}
	s.targetDomain = parsedURL.Host
	
	// 设置JS分析器的目标域名
	s.jsAnalyzer.SetTargetDomain(s.targetDomain)
	
	// 初始化高级作用域控制
	s.advancedScope = NewAdvancedScope(s.targetDomain)
	s.advancedScope.SetMode(ScopeRDN) // 根域名模式
	s.advancedScope.PresetStaticFilterScope() // 过滤静态资源
	
	// 初始化子域名提取器
	s.subdomainExtractor = NewSubdomainExtractor(targetURL)
	
	// 检查是否重复
	if s.duplicateHandler.IsDuplicateURL(targetURL) {
		return fmt.Errorf("URL已处理过: %s", targetURL)
	}
	
	fmt.Printf("开始爬取URL: %s\n", targetURL)
	fmt.Printf("限制域名范围: %s\n", s.targetDomain)
	fmt.Printf("\n【已启用功能】Spider Enhanced v2.2\n")
	fmt.Printf("  ✓ 跨域JS分析（支持60+个CDN）\n")
	fmt.Printf("  ✓ 智能表单填充（支持20+种字段类型）\n")
	fmt.Printf("  ✓ 作用域精确控制（10个过滤维度）\n")
	fmt.Printf("  ✓ 性能优化（对象池+连接池）\n")
	fmt.Printf("  ✓ 技术栈识别（15+种框架）\n")
	fmt.Printf("  ✓ 敏感信息检测（30+种模式）\n")
	fmt.Printf("  ✓ JavaScript事件触发（点击、悬停、输入、滚动）\n")
	fmt.Printf("  ✓ AJAX请求拦截（动态URL捕获）🆕\n")
	fmt.Printf("  ✓ 增强JS分析（对象、路由、配置）🆕\n")
	fmt.Printf("  ✓ 静态资源分类（7种类型）🆕\n")
	fmt.Printf("  ✓ IP地址检测（内网泄露识别）🆕\n")
	fmt.Printf("  ✓ URL优先级排序（智能爬取策略）🆕\n")
	fmt.Printf("\n爬取配置:\n")
	fmt.Printf("  深度: %d 层 | 并发: 20-30 | 最大URL: 500\n", s.config.DepthSettings.MaxDepth)
	fmt.Printf("\n")
	
	// 初始化隐藏路径发现器
	userAgent := ""
	if len(s.config.AntiDetectionSettings.UserAgents) > 0 {
		userAgent = s.config.AntiDetectionSettings.UserAgents[0]
	}
	s.hiddenPathDiscovery = NewHiddenPathDiscovery(targetURL, userAgent)
	
	// === 优化：先爬取sitemap.xml和robots.txt ===
	fmt.Println("开始爬取sitemap.xml和robots.txt...")
	sitemapURLs, robotsInfo := s.sitemapCrawler.GetAllURLs(targetURL)
	s.mutex.Lock()
	s.sitemapURLs = sitemapURLs
	s.robotsURLs = append(robotsInfo.DisallowPaths, robotsInfo.AllowPaths...)
	s.mutex.Unlock()
	
	if len(sitemapURLs) > 0 {
		fmt.Printf("  [Sitemap] 发现 %d 个URL\n", len(sitemapURLs))
	}
	if len(robotsInfo.DisallowPaths) > 0 {
		fmt.Printf("  [robots.txt] 发现 %d 个Disallow路径（渗透测试重点）\n", len(robotsInfo.DisallowPaths))
	}
	if len(robotsInfo.SitemapURLs) > 0 {
		fmt.Printf("  [robots.txt] 发现 %d 个额外sitemap\n", len(robotsInfo.SitemapURLs))
	}
	
	// 将sitemap和robots中的URL添加到待爬取列表
	for _, u := range sitemapURLs {
		s.visitedURLs[u] = false // 标记为待爬取
	}
	for _, u := range robotsInfo.DisallowPaths {
		s.visitedURLs[u] = false // Disallow路径也要爬取
	}
	
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
	
	// 如果启用了动态爬虫，总是使用（Phase 2/3优化：捕获AJAX和JS动态内容）
	if s.config.StrategySettings.EnableDynamicCrawler {
		fmt.Println("使用动态爬虫（捕获AJAX和动态JS内容）...")
		result, err := s.dynamicCrawler.Crawl(parsedURL)
		if err != nil {
			fmt.Printf("动态爬虫错误: %v\n", err)
		} else {
			s.addResult(result)
			fmt.Printf("动态爬虫完成，发现 %d 个链接, %d 个资源, %d 个表单, %d 个API\n", 
				len(result.Links), len(result.Assets), len(result.Forms), len(result.APIs))
		}
	}
	
	// 处理发现的参数（包括GET参数爆破）
	paramFuzzURLs := s.processParams(targetURL)
	
	// 将参数爆破生成的URL添加到第一个结果的Links中，以便递归爬取（优化：修复并发安全）
	s.mutex.Lock()
	if len(paramFuzzURLs) > 1 && len(s.results) > 0 {
		// 添加到第一个结果的Links中（作为发现的链接）
		s.results[0].Links = append(s.results[0].Links, paramFuzzURLs...)
		s.mutex.Unlock()
		fmt.Printf("  [参数爆破] 已将 %d 个爆破URL添加到爬取队列\n", len(paramFuzzURLs))
	} else {
		s.mutex.Unlock()
	}
	
	// 处理发现的表单（包括POST参数爆破）
	s.processForms(targetURL)
	
	// 分析跨域JS文件（在递归爬取之前）
	s.processCrossDomainJS()
	
	// 如果启用了递归爬取，继续爬取发现的链接（真正的多层递归）
	if s.config.DepthSettings.MaxDepth > 1 {
		s.crawlRecursivelyMultiLayer()
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
	// 降低触发动态爬虫的阈值（更容易触发）
	if len(lastResult.Links) < 20 && len(lastResult.APIs) < 10 {
		return true
	}
	
	return false
}

// addResult 添加爬取结果（增强版：包含DOM相似度检测、技术栈检测和敏感信息检测）
func (s *Spider) addResult(result *Result) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// 如果有HTML内容，先进行DOM相似度检测
	if result.HTMLContent != "" && s.domSimilarity != nil {
		isSimilar, record := s.domSimilarity.CheckSimilarity(result.URL, result.HTMLContent)
		if isSimilar && record != nil {
			fmt.Printf("  [DOM相似度] 发现相似页面！相似度: %.1f%%, 相似于: %s\n", 
				record.Similarity*100, record.SimilarToURL)
			fmt.Printf("  [DOM相似度] 原因: %s\n", record.Reason)
			fmt.Printf("  [DOM相似度] ✓ 跳过重复爬取，节省资源\n")
			// 相似页面仍然记录，但标记为已跳过
			result.IsSimilar = true
			result.SimilarToURL = record.SimilarToURL
		}
	}
	
	s.results = append(s.results, result)
	
	// 如果有HTML内容，进行高级检测
	if result.HTMLContent != "" {
		// 技术栈检测
		if s.techDetector != nil {
			techs := s.techDetector.DetectFromContent(result.HTMLContent, result.Headers)
			s.detectedTechs = append(s.detectedTechs, techs...)
			
			if len(techs) > 0 {
				techNames := make([]string, 0)
				for _, tech := range techs {
					if tech.Version != "" {
						techNames = append(techNames, tech.Name+" "+tech.Version)
					} else {
						techNames = append(techNames, tech.Name)
					}
				}
				fmt.Printf("  [技术栈] 检测到: %s\n", strings.Join(techNames, ", "))
			}
		}
		
		// 敏感信息检测
		if s.sensitiveDetector != nil {
			// 扫描HTML内容
			findings := s.sensitiveDetector.Scan(result.HTMLContent, result.URL)
			s.sensitiveFindings = append(s.sensitiveFindings, findings...)
			
			// 扫描HTTP头
			if len(result.Headers) > 0 {
				headerContent := ""
				for key, value := range result.Headers {
					headerContent += key + ": " + value + "\n"
				}
				headerFindings := s.sensitiveDetector.Scan(headerContent, result.URL+" (Headers)")
				s.sensitiveFindings = append(s.sensitiveFindings, headerFindings...)
				findings = append(findings, headerFindings...)
			}
			
			if len(findings) > 0 {
				highCount := 0
				for _, finding := range findings {
					if finding.Severity == "HIGH" {
						highCount++
					}
				}
				
				if highCount > 0 {
					fmt.Printf("  [敏感信息] ⚠️  发现 %d 处高危敏感信息！\n", highCount)
				} else if len(findings) > 0 {
					fmt.Printf("  [敏感信息] 发现 %d 处敏感信息\n", len(findings))
				}
			}
		}
	}
}

// addResultWithDetection 添加结果并进行检测
func (s *Spider) addResultWithDetection(result *Result, response *http.Response, htmlContent string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.results = append(s.results, result)
	
	// 技术栈检测
	if response != nil && s.techDetector != nil {
		techs := s.techDetector.Detect(response, htmlContent)
		s.detectedTechs = append(s.detectedTechs, techs...)
		
		if len(techs) > 0 {
			fmt.Printf("  [技术栈] 检测到: ")
			techNames := make([]string, 0)
			for _, tech := range techs {
				if tech.Version != "" {
					techNames = append(techNames, tech.Name+" "+tech.Version)
				} else {
					techNames = append(techNames, tech.Name)
				}
			}
			fmt.Printf("%s\n", strings.Join(techNames, ", "))
		}
	}
	
	// 敏感信息检测
	if s.sensitiveDetector != nil {
		findings := s.sensitiveDetector.Scan(htmlContent, result.URL)
		s.sensitiveFindings = append(s.sensitiveFindings, findings...)
		
		if len(findings) > 0 {
			highCount := 0
			for _, finding := range findings {
				if finding.Severity == "HIGH" {
					highCount++
				}
			}
			
			if highCount > 0 {
				fmt.Printf("  [敏感信息] ⚠️  发现 %d 处高危敏感信息！\n", highCount)
			} else {
				fmt.Printf("  [敏感信息] 发现 %d 处敏感信息\n", len(findings))
			}
		}
	}
	
	// 子域名提取
	if s.subdomainExtractor != nil && htmlContent != "" {
		// 从HTML内容提取子域名
		subdomains := s.subdomainExtractor.ExtractFromHTML(htmlContent)
		if len(subdomains) > 0 {
			fmt.Printf("  [子域名] 发现 %d 个新子域名\n", len(subdomains))
		}
		
		// 从URL本身提取
		s.subdomainExtractor.ExtractFromURL(result.URL)
	}
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
	
	// === 新增：对无参数URL进行参数爆破 ===
	if len(params) == 0 && s.config.StrategySettings.EnableParamFuzzing {
		fmt.Printf("  [参数爆破] 检测到无参数URL，开始参数枚举...\n")
		
		// 生成参数爆破列表
		fuzzList := s.paramHandler.GenerateParameterFuzzList(rawURL)
		
		// 应用限制（避免生成过多URL）
		if s.config.StrategySettings.ParamFuzzLimit > 0 && len(fuzzList) > s.config.StrategySettings.ParamFuzzLimit {
			fuzzList = fuzzList[:s.config.StrategySettings.ParamFuzzLimit]
			fmt.Printf("  [参数爆破] 限制爆破数量为 %d 个（可配置）\n", s.config.StrategySettings.ParamFuzzLimit)
		}
		
		if len(fuzzList) > 0 {
			fmt.Printf("  [参数爆破] 为无参数URL生成 %d 个参数爆破变体\n", len(fuzzList))
			fmt.Printf("  [参数爆破] 示例: %s\n", fuzzList[0])
			if len(fuzzList) > 1 {
				fmt.Printf("  [参数爆破] 示例: %s\n", fuzzList[1])
			}
			if len(fuzzList) > 2 {
				fmt.Printf("  [参数爆破] 示例: %s\n", fuzzList[2])
			}
			fmt.Printf("  [参数爆破] ... 还有 %d 个爆破URL\n", len(fuzzList)-3)
			
			return fuzzList
		}
		
		// 如果爆破失败，返回原始URL
		return []string{rawURL}
	}
	
	// === 原有逻辑：对有参数URL进行变体生成（纯爬虫模式，不含攻击payload） ===
	// 对每个参数进行信息分析（仅记录，不用于攻击）
	for paramName := range params {
		risk, level := s.paramHandler.AnalyzeParameterSecurity(paramName)
		if level >= 2 { // 中等风险以上
			finding := fmt.Sprintf("PARAM_INFO: %s - %s (Risk Level: %d)", paramName, risk, level)
			s.mutex.Lock()
			s.securityFindings = append(s.securityFindings, finding)
			s.mutex.Unlock()
			fmt.Printf("  [参数分析] %s\n", finding)
		}
	}
	
	// 生成参数变体（只使用正常值，不含攻击payload）
	variations := s.paramHandler.GenerateParamVariations(rawURL)
	
	// === 移除：安全测试变体（攻击性payload） ===
	// 作为纯爬虫工具，不应发送SQL注入、XSS等攻击性payload
	// securityVariations := s.paramHandler.GenerateSecurityTestVariations(rawURL)
	// variations = append(variations, securityVariations...)
	
	// 如果没有生成变体，返回原始URL
	if len(variations) == 0 {
		return []string{rawURL}
	}
	
	// 打印生成的变体
	fmt.Printf("  [参数变体] 为URL生成 %d 个参数变体（正常测试值）\n", len(variations))
	for i, variation := range variations {
		if i < 5 { // 只显示前5个，避免输出过多
			fmt.Printf("    变体: %s\n", variation)
		}
	}
	if len(variations) > 5 {
		fmt.Printf("    ... 还有 %d 个变体\n", len(variations)-5)
	}
	
	return variations
}

// processForms 处理表单（包括POST参数爆破）
func (s *Spider) processForms(targetURL string) {
	// 检查是否启用POST参数爆破
	if !s.config.StrategySettings.EnablePOSTParamFuzzing {
		return
	}
	
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// 收集所有空表单或无效表单
	emptyForms := make([]string, 0)
	totalForms := 0
	
	for _, result := range s.results {
		totalForms += len(result.Forms)
		
		for _, form := range result.Forms {
			// 检查表单是否有有效字段
			hasValidFields := false
			for _, field := range form.Fields {
				// 跳过提交按钮和普通按钮
				fieldTypeLower := strings.ToLower(field.Type)
				if fieldTypeLower != "submit" && fieldTypeLower != "button" && field.Name != "" {
					hasValidFields = true
					break
				}
			}
			
			// 如果表单没有有效字段，添加到爆破列表
			if !hasValidFields {
				emptyForms = append(emptyForms, form.Action)
			}
		}
		
		// 同样检查POST请求
		for _, postReq := range result.POSTRequests {
			// 如果POST请求没有参数或参数为空
			if len(postReq.Parameters) == 0 {
				emptyForms = append(emptyForms, postReq.URL)
			}
		}
	}
	
	// 去重
	uniqueEmptyForms := make(map[string]bool)
	for _, formURL := range emptyForms {
		uniqueEmptyForms[formURL] = true
	}
	
	if len(uniqueEmptyForms) == 0 {
		if totalForms > 0 {
			fmt.Printf("  [POST爆破] 发现 %d 个表单，全部有字段，无需爆破\n", totalForms)
		}
		return
	}
	
	fmt.Printf("  [POST爆破] 检测到 %d 个空表单，开始POST参数爆破...\n", len(uniqueEmptyForms))
	
	// 对每个空表单生成POST爆破请求
	totalPOSTFuzzRequests := 0
	for formURL := range uniqueEmptyForms {
		// 生成POST参数爆破列表
		postFuzzList := s.paramHandler.GeneratePOSTParameterFuzzList(formURL)
		
		// 应用限制
		if s.config.StrategySettings.POSTParamFuzzLimit > 0 && len(postFuzzList) > s.config.StrategySettings.POSTParamFuzzLimit {
			postFuzzList = postFuzzList[:s.config.StrategySettings.POSTParamFuzzLimit]
		}
		
		if len(postFuzzList) > 0 {
			// 添加到第一个结果的POSTRequests中
			if len(s.results) > 0 {
				s.results[0].POSTRequests = append(s.results[0].POSTRequests, postFuzzList...)
			}
			totalPOSTFuzzRequests += len(postFuzzList)
		}
	}
	
	if totalPOSTFuzzRequests > 0 {
		fmt.Printf("  [POST爆破] 为 %d 个空表单生成 %d 个POST爆破请求\n", len(uniqueEmptyForms), totalPOSTFuzzRequests)
		fmt.Printf("  [POST爆破] 示例: POST %s {username=admin, password=admin123}\n", emptyForms[0])
		if len(uniqueEmptyForms) > 1 {
			fmt.Printf("  [POST爆破] 示例: POST %s {search=test, q=admin}\n", emptyForms[0])
		}
	}
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

// analyzeExternalJS 下载并分析外部JS文件（使用性能优化）
func (s *Spider) analyzeExternalJS(jsURL string) []string {
	// 使用性能优化的HTTP客户端
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
	
	// 使用优化的HTTP客户端（带连接池）
	resp, err := s.perfOptimizer.DoRequest(req)
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
	
	// 使用Buffer池读取内容
	buf := s.perfOptimizer.GetBuffer()
	defer s.perfOptimizer.PutBuffer(buf)
	
	// 限制文件大小（最大5MB）
	const maxSize = 5 * 1024 * 1024
	limitedReader := &io.LimitedReader{R: resp.Body, N: maxSize}
	
	_, err = buf.ReadFrom(limitedReader)
	if err != nil {
		fmt.Printf("    读取内容失败: %v\n", err)
		return []string{}
	}
	
	// 使用增强的JS分析器提取URL
	jsCode := buf.String()
	
	// 使用增强分析
	enhancedResult := s.jsAnalyzer.EnhancedAnalyze(jsCode)
	
	// 合并所有发现的URL
	urls := make([]string, 0)
	seen := make(map[string]bool)
	
	for category, categoryURLs := range enhancedResult {
		for _, url := range categoryURLs {
			if !seen[url] {
				seen[url] = true
				urls = append(urls, url)
			}
		}
		
		// 打印各类别的发现
		if len(categoryURLs) > 0 {
			fmt.Printf("    [%s] 发现 %d 个URL\n", category, len(categoryURLs))
		}
	}
	
	return urls
}

// crawlRecursively 递归爬取发现的链接（单层爬取，已废弃）
// 请使用 crawlRecursivelyMultiLayer
func (s *Spider) crawlRecursively() {
	s.crawlRecursivelyMultiLayer()
}

// crawlRecursivelyMultiLayer 真正的多层递归爬取（修复深度问题）
func (s *Spider) crawlRecursivelyMultiLayer() {
	fmt.Println("开始多层递归爬取...")
	
	currentDepth := 1
	totalCrawled := 0
	
	// 循环爬取每一层，直到达到最大深度
	for currentDepth < s.config.DepthSettings.MaxDepth {
		currentDepth++
		
		fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("【第 %d 层爬取】最大深度: %d\n", currentDepth, s.config.DepthSettings.MaxDepth)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		
		// 收集当前层需要爬取的链接
		layerLinks := s.collectLinksForLayer(currentDepth)
		
		if len(layerLinks) == 0 {
			fmt.Printf("第 %d 层没有新链接，递归结束\n", currentDepth)
			break
		}
		
		fmt.Printf("第 %d 层准备爬取 %d 个链接...\n", currentDepth, len(layerLinks))
		
		// 爬取当前层的所有链接
		newResults := s.crawlLayer(layerLinks, currentDepth)
		
		// 合并结果
		s.mutex.Lock()
		s.results = append(s.results, newResults...)
		s.mutex.Unlock()
		
		totalCrawled += len(layerLinks)
		fmt.Printf("第 %d 层爬取完成！本层爬取 %d 个URL，累计 %d 个\n", 
			currentDepth, len(layerLinks), totalCrawled)
		
		// 检查是否达到URL限制
		if totalCrawled >= 500 {
			fmt.Printf("已达到URL限制(500)，递归结束\n")
			break
		}
	}
	
	fmt.Printf("\n多层递归爬取完成！总共爬取 %d 个URL，深度 %d 层\n", totalCrawled, currentDepth)
}

// collectLinksForLayer 收集指定层需要爬取的链接
func (s *Spider) collectLinksForLayer(targetDepth int) []string {
	allLinks := make(map[string]bool)
	externalLinks := make([]string, 0)
	
	s.mutex.Lock()
	// 从所有结果中收集链接
	for _, result := range s.results {
		for _, link := range result.Links {
			// 检查是否已访问
			if s.visitedURLs[link] {
				continue
			}
			
			// 解析链接
			parsedURL, err := url.Parse(link)
			if err != nil {
				continue
			}
			
			// 作用域检查
			inScope, _ := s.advancedScope.InScope(link)
			if inScope {
				// 规范化URL
				normalizedURL, err := s.paramHandler.NormalizeURL(link)
				if err == nil {
					allLinks[normalizedURL] = true
				} else {
					allLinks[link] = true
				}
			} else {
				if parsedURL.Host != s.targetDomain && parsedURL.Host != "" {
					externalLinks = append(externalLinks, link)
				}
			}
		}
	}
	s.mutex.Unlock()
	
	// 记录外部链接
	if len(externalLinks) > 0 {
		s.mutex.Lock()
		s.externalLinks = append(s.externalLinks, externalLinks...)
		s.mutex.Unlock()
		fmt.Printf("  发现 %d 个外部链接（已记录但不爬取）\n", len(externalLinks))
	}
	
	// 转换为列表并优先级排序
	tasksToSubmit := make([]string, 0)
	for link := range allLinks {
		// 去重检查
		if s.duplicateHandler.IsDuplicateURL(link) {
			continue
		}
		
		// 验证格式
		if !IsValidURL(link) {
			continue
		}
		
		tasksToSubmit = append(tasksToSubmit, link)
		
		// 每层限制100个URL
		if len(tasksToSubmit) >= 100 {
			break
		}
	}
	
	// 优先级排序
	tasksToSubmit = s.prioritizeURLs(tasksToSubmit)
	
	return tasksToSubmit
}

// crawlLayer 爬取一层的所有链接
func (s *Spider) crawlLayer(links []string, depth int) []*Result {
	results := make([]*Result, 0)
	
	// 标记为已访问
	s.mutex.Lock()
	for _, link := range links {
		s.visitedURLs[link] = true
	}
	s.mutex.Unlock()
	
	// 为每层创建新的工作池（修复：避免复用已关闭的工作池）
	layerWorkerPool := NewWorkerPool(30, 20)
	
	// 启动工作池
	layerWorkerPool.Start(func(task Task) (*Result, error) {
		return s.crawlURL(task.URL)
	})
	
	// 提交所有任务
	for _, link := range links {
		task := Task{
			URL:   link,
			Depth: depth,
		}
		if err := layerWorkerPool.Submit(task); err != nil {
			fmt.Printf("  提交任务失败 %s: %v\n", link, err)
		}
	}
	
	// 等待完成（不显示进度，避免干扰）
	layerWorkerPool.Wait()
	
	// 收集结果
	results = layerWorkerPool.GetResults()
	
	// 停止工作池
	layerWorkerPool.Stop()
	
	// 显示统计
	stats := layerWorkerPool.GetStats()
	fmt.Printf("  本层统计 - 总任务: %d, 成功: %d, 失败: %d\n", 
		stats["total"], stats["completed"]-stats["failed"], stats["failed"])
	
	return results
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

// ImportFromBurp 从Burp Suite文件导入
func (s *Spider) ImportFromBurp(filename string) error {
	fmt.Printf("从Burp Suite导入流量: %s\n", filename)
	
	// 创建被动爬取器
	s.passiveCrawler = NewPassiveCrawler("burp")
	
	// 加载Burp文件
	err := s.passiveCrawler.LoadFromBurp(filename)
	if err != nil {
		return err
	}
	
	// 过滤目标域名的URL
	targetURLs := s.passiveCrawler.FilterByDomain(s.targetDomain)
	fmt.Printf("过滤后得到目标域名URL: %d个\n", len(targetURLs))
	
	// 将导入的URL和表单加入结果
	passiveResult := s.passiveCrawler.ExportToResult(s.targetDomain)
	s.addResult(passiveResult)
	
	return nil
}

// ImportFromHAR 从HAR文件导入
func (s *Spider) ImportFromHAR(filename string) error {
	fmt.Printf("从HAR文件导入流量: %s\n", filename)
	
	// 创建被动爬取器
	s.passiveCrawler = NewPassiveCrawler("har")
	
	// 加载HAR文件
	err := s.passiveCrawler.LoadFromHAR(filename)
	if err != nil {
		return err
	}
	
	// 过滤目标域名的URL
	targetURLs := s.passiveCrawler.FilterByDomain(s.targetDomain)
	fmt.Printf("过滤后得到目标域名URL: %d个\n", len(targetURLs))
	
	// 将导入的URL和表单加入结果
	passiveResult := s.passiveCrawler.ExportToResult(s.targetDomain)
	s.addResult(passiveResult)
	
	return nil
}

// Stop 停止爬取
func (s *Spider) Stop() {
	fmt.Println("停止爬取...")
	s.staticCrawler.Stop()
	s.dynamicCrawler.Stop()
	
	// 关闭性能优化器
	if s.perfOptimizer != nil {
		s.perfOptimizer.Close()
	}
}

// Close 优雅关闭爬虫，释放所有资源（实现 io.Closer 接口）
func (s *Spider) Close() error {
	s.closeMux.Lock()
	defer s.closeMux.Unlock()
	
	// 防止重复关闭
	if s.closed {
		return nil
	}
	
	fmt.Println("\n正在关闭爬虫，清理资源...")
	
	// 停止爬取
	s.Stop()
	
	// 等待所有 goroutine 完成
	s.wg.Wait()
	
	// 关闭 done channel
	close(s.done)
	
	// 标记为已关闭
	s.closed = true
	
	fmt.Println("资源清理完成")
	return nil
}

// cleanup 内部清理方法（在 Start 中使用 defer 调用）
func (s *Spider) cleanup() {
	// Close 方法已经处理了所有清理工作
	s.Close()
}

// prioritizeURLs URL优先级排序（Phase 3新增）
func (s *Spider) prioritizeURLs(urls []string) []string {
	highPriority := make([]string, 0)    // 高优先级
	mediumPriority := make([]string, 0)  // 中优先级
	lowPriority := make([]string, 0)     // 低优先级
	
	for _, url := range urls {
		urlLower := strings.ToLower(url)
		
		// 高优先级：带多个参数的URL、admin/api/login等敏感路径
		if (strings.Count(url, "=") >= 2) ||
			strings.Contains(urlLower, "/admin") ||
			strings.Contains(urlLower, "/api/") ||
			strings.Contains(urlLower, "/login") ||
			strings.Contains(urlLower, "/user") ||
			strings.Contains(urlLower, "/account") {
			highPriority = append(highPriority, url)
		} else if strings.Contains(url, "?") {
			// 中优先级：带参数的URL
			mediumPriority = append(mediumPriority, url)
		} else {
			// 低优先级：普通URL
			lowPriority = append(lowPriority, url)
		}
	}
	
	// 合并：高 → 中 → 低
	result := make([]string, 0, len(urls))
	result = append(result, highPriority...)
	result = append(result, mediumPriority...)
	result = append(result, lowPriority...)
	
	return result
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
		resultData["post_requests"] = result.POSTRequests // 添加POST请求数据
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
	
	// 添加新功能统计
	if s.advancedScope != nil {
		exportData["scope_stats"] = s.advancedScope.GetStatistics()
	}
	if s.perfOptimizer != nil {
		exportData["performance_stats"] = s.perfOptimizer.GetStatistics()
	}
	if s.formFiller != nil {
		exportData["form_filler_stats"] = s.formFiller.GetStatistics()
	}
	
	// 添加高级功能统计
	exportData["detected_technologies"] = s.detectedTechs
	exportData["tech_stack_summary"] = s.techDetector.GetTechStackSummary(s.detectedTechs)
	exportData["sensitive_findings"] = s.sensitiveFindings
	exportData["sensitive_stats"] = s.sensitiveDetector.GetStatistics()
	exportData["total_sensitive_findings"] = len(s.sensitiveFindings)
	
	// 被动爬取统计（如果使用）
	if s.passiveCrawler != nil {
		exportData["passive_stats"] = s.passiveCrawler.GetStatistics()
	}
	
	// 子域名提取统计
	if s.subdomainExtractor != nil {
		exportData["subdomains"] = s.subdomainExtractor.ExportSubdomains()
		exportData["subdomain_stats"] = s.subdomainExtractor.GetStatistics()
		exportData["total_subdomains"] = s.subdomainExtractor.GetSubdomainCount()
	}
	
	// DOM相似度检测统计
	if s.domSimilarity != nil {
		exportData["dom_similarity_stats"] = s.domSimilarity.GetStatistics()
		exportData["similar_pages"] = s.domSimilarity.GetSimilarPages()
		exportData["total_similar_pages"] = len(s.domSimilarity.GetSimilarPages())
	}
	
	// Sitemap和robots.txt统计
	exportData["sitemap_urls"] = s.sitemapURLs
	exportData["robots_urls"] = s.robotsURLs
	exportData["total_sitemap_urls"] = len(s.sitemapURLs)
	exportData["total_robots_urls"] = len(s.robotsURLs)
	
	// === 新增：静态资源分类 ===
	allAssets := make([]string, 0)
	for _, result := range s.results {
		allAssets = append(allAssets, result.Assets...)
	}
	
	if s.assetClassifier != nil {
		classifiedAssets := s.assetClassifier.ClassifyAssets(allAssets)
		exportData["classified_assets"] = classifiedAssets
		exportData["assets_stats"] = s.assetClassifier.GetAssetStats(classifiedAssets)
	}
	
	// === 新增：IP链接分类 ===
	// 重置allLinks用于IP检测（包含所有链接源）
	allLinks = make([]string, 0)
	for _, result := range s.results {
		allLinks = append(allLinks, result.Links...)
	}
	// 也检查外部链接和其他链接源
	allLinks = append(allLinks, s.externalLinks...)
	allLinks = append(allLinks, s.crossDomainJS...)
	
	if s.ipDetector != nil {
		classifiedIPs := s.ipDetector.ClassifyIPLinks(allLinks)
		exportData["ip_links"] = map[string]interface{}{
			"private_ips":   classifiedIPs["private_ip"],
			"public_ips":    classifiedIPs["public_ip"],
			"private_count": len(classifiedIPs["private_ip"]),
			"public_count":  len(classifiedIPs["public_ip"]),
			"total_count":   len(classifiedIPs["private_ip"]) + len(classifiedIPs["public_ip"]),
			"has_leak":      len(classifiedIPs["private_ip"]) > 0,
		}
	}
	
	return exportData
}