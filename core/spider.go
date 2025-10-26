package core

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"spider-golang/config"
)

// Spider 主爬虫协调器
type Spider struct {
	config              *config.Config
	staticCrawler       StaticCrawler
	dynamicCrawler      DynamicCrawler
	jsAnalyzer          *JSAnalyzer
	paramHandler        *ParamHandler
	duplicateHandler    *DuplicateHandler
	smartDeduplication  *SmartDeduplication
	smartParamDedup     *SmartParamDeduplicator // 智能参数值去重器（v2.6.1）
	businessFilter      *BusinessAwareURLFilter  // 业务感知过滤器（v2.7）
	urlPatternDedup     *URLPatternDeduplicator  // URL模式去重器（v2.9）
	hiddenPathDiscovery *HiddenPathDiscovery
	cdnDetector         *CDNDetector // CDN检测器
	workerPool          *WorkerPool  // 并发工作池

	// 新增优化组件
	formFiller    *SmartFormFiller      // 智能表单填充器
	advancedScope *AdvancedScope        // 高级作用域控制
	perfOptimizer *PerformanceOptimizer // 性能优化器

	// 高级功能组件
	techDetector       *TechStackDetector     // 技术栈检测器
	sensitiveDetector  *SensitiveInfoDetector // 敏感信息检测器
	passiveCrawler     *PassiveCrawler        // 被动爬取器
	subdomainExtractor *SubdomainExtractor    // 子域名提取器
	domSimilarity      *DOMSimilarityDetector // DOM相似度检测器
	sitemapCrawler     *SitemapCrawler        // Sitemap爬取器
	assetClassifier    *AssetClassifier       // 静态资源分类器
	ipDetector         *IPDetector            // IP地址检测器
	
	// 🆕 v2.7+ 新增组件
	cssAnalyzer         *CSSAnalyzer            // CSS分析器
	resourceClassifier  *ResourceClassifier     // 资源分类器
	urlDeduplicator     *URLDeduplicator        // URL去重器（忽略参数值）
	priorityScheduler   *URLPriorityScheduler   // 优先级调度器（可选）

	results           []*Result
	sitemapURLs       []string         // 从sitemap发现的URL
	robotsURLs        []string         // 从robots.txt发现的URL
	externalLinks     []string         // 记录外部链接
	hiddenPaths       []string         // 记录隐藏路径
	securityFindings  []string         // 记录安全发现
	crossDomainJS     []string         // 记录跨域JS发现的URL
	detectedTechs     []*TechInfo      // 检测到的技术栈
	sensitiveFindings []*SensitiveInfo // 敏感信息发现
	mutex             sync.Mutex
	targetDomain      string          // 目标域名
	visitedURLs       map[string]bool // 已访问URL

	// 资源管理（优化：防止泄漏）
	done     chan struct{}  // 完成信号
	wg       sync.WaitGroup // 等待所有goroutine完成
	closed   bool           // 是否已关闭
	closeMux sync.Mutex     // 关闭锁

	// v2.6: 日志和监控
	logger Logger // 结构化日志记录器
}

// NewSpider 创建爬虫实例
func NewSpider(cfg *config.Config) *Spider {
	// v2.6: 创建日志记录器
	var logOutput io.Writer = os.Stdout
	if cfg.LogSettings.OutputFile != "" {
		file, err := os.OpenFile(cfg.LogSettings.OutputFile,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("无法打开日志文件 %s: %v，使用标准输出", cfg.LogSettings.OutputFile, err)
		} else {
			logOutput = file
		}
	}

	logLevel := parseLogLevel(cfg.LogSettings.Level)
	logger := NewLogger(logLevel, logOutput)

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
		smartDeduplication: NewSmartDeduplication(),                                                                                                             // 初始化智能去重
		smartParamDedup:    NewSmartParamDeduplicator(cfg.DeduplicationSettings.MaxParamValueVariantsPerGroup, cfg.DeduplicationSettings.EnableSmartParamDedup), // v2.6.1: 智能参数值去重
		businessFilter:     NewBusinessAwareURLFilter(BusinessFilterConfig{                                                                                       // v2.7: 业务感知过滤器
			MinBusinessScore:        cfg.DeduplicationSettings.BusinessFilterMinScore,
			HighValueThreshold:      cfg.DeduplicationSettings.BusinessFilterHighValueThreshold,
			MaxSamePatternLowValue:  cfg.DeduplicationSettings.BusinessFilterMaxLowValue,
			MaxSamePatternMidValue:  cfg.DeduplicationSettings.BusinessFilterMaxMidValue,
			MaxSamePatternHighValue: cfg.DeduplicationSettings.BusinessFilterMaxHighValue,
			EnableAdaptiveLearning:  cfg.DeduplicationSettings.BusinessFilterAdaptiveLearning,
			Enabled:                 cfg.DeduplicationSettings.EnableBusinessAwareFilter,
		}),
		urlPatternDedup: NewURLPatternDeduplicator(), // v2.9: URL模式去重器
		cdnDetector:     NewCDNDetector(),            // 初始化CDN检测器
		workerPool:      NewWorkerPool(workerCount, maxQPS), // 初始化工作池

		// 初始化新增组件
		formFiller:    NewSmartFormFiller(),         // 智能表单填充器
		advancedScope: nil,                          // 将在Start中初始化
		perfOptimizer: NewPerformanceOptimizer(500), // 性能优化器（限制500MB）

		// 初始化高级功能组件
		techDetector:      NewTechStackDetector(),         // 技术栈检测器
		sensitiveDetector: NewSensitiveInfoDetector(),     // 敏感信息检测器
		passiveCrawler:    nil,                            // 按需创建
		domSimilarity:     NewDOMSimilarityDetector(0.85), // DOM相似度检测器（阈值85%）
		sitemapCrawler:    NewSitemapCrawler(),            // Sitemap爬取器
		assetClassifier:   NewAssetClassifier(),           // 静态资源分类器
		ipDetector:        NewIPDetector(),                // IP地址检测器
		
		// 🆕 v2.7+ 新增组件
		cssAnalyzer:        NewCSSAnalyzer(),              // CSS分析器
		resourceClassifier: nil,                           // 将在Start中初始化（需要目标域名）
		urlDeduplicator:    NewURLDeduplicator(),         // URL去重器
		priorityScheduler:  nil,                           // 将在Start中初始化（可选，需要配置）

		hiddenPathDiscovery: nil, // 将在Start方法中初始化，需要用户代理
		results:             make([]*Result, 0),
		externalLinks:       make([]string, 0),
		hiddenPaths:         make([]string, 0),
		securityFindings:    make([]string, 0),
		crossDomainJS:       make([]string, 0),
		detectedTechs:       make([]*TechInfo, 0),
		sensitiveFindings:   make([]*SensitiveInfo, 0),
		sitemapURLs:         make([]string, 0),
		robotsURLs:          make([]string, 0),
		visitedURLs:         make(map[string]bool),

		// 初始化资源管理
		done:   make(chan struct{}),
		closed: false,

		// v2.6: 初始化日志
		logger: logger,
	}

	// 配置各个组件
	spider.staticCrawler.Configure(cfg)
	spider.dynamicCrawler.Configure(cfg)

	// 设置JS分析器的目标域名
	spider.jsAnalyzer.SetTargetDomain(cfg.TargetURL)

	return spider
}

// parseLogLevel 解析日志级别字符串为 slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
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
	
	// 设置CSS分析器的目标域名
	s.cssAnalyzer.SetTargetDomain(s.targetDomain)
	
	// 初始化资源分类器
	s.resourceClassifier = NewResourceClassifier(s.targetDomain)
	
	// 🆕 初始化优先级调度器（如果配置启用）
	// 可以通过配置文件控制是否使用优先级队列模式
	s.priorityScheduler = NewURLPriorityScheduler(s.targetDomain)

	// 初始化高级作用域控制
	s.advancedScope = NewAdvancedScope(s.targetDomain)
	s.advancedScope.SetMode(ScopeRDN)         // 根域名模式
	s.advancedScope.PresetStaticFilterScope() // 过滤静态资源

	// 初始化子域名提取器
	s.subdomainExtractor = NewSubdomainExtractor(targetURL)

	// 检查是否重复
	if s.duplicateHandler.IsDuplicateURL(targetURL) {
		return fmt.Errorf("URL已处理过: %s", targetURL)
	}

	// v2.6: 使用结构化日志
	s.logger.Info("开始爬取",
		"url", targetURL,
		"target_domain", s.targetDomain,
		"max_depth", s.config.DepthSettings.MaxDepth,
		"version", "v2.6")

	// 显示功能清单（保留用户友好的格式）
	fmt.Printf("\n【已启用功能】Spider Ultimate v2.6\n")
	fmt.Printf("  ✓ 跨域JS分析（支持60+个CDN）\n")
	fmt.Printf("  ✓ 智能表单填充（支持20+种字段类型）\n")
	fmt.Printf("  ✓ 作用域精确控制（10个过滤维度）\n")
	fmt.Printf("  ✓ 性能优化（对象池+连接池）\n")
	fmt.Printf("  ✓ 技术栈识别（15+种框架）\n")
	fmt.Printf("  ✓ 敏感信息检测（30+种模式）\n")
	fmt.Printf("  ✓ JavaScript事件触发（点击、悬停、输入、滚动）\n")
	fmt.Printf("  ✓ AJAX请求拦截（动态URL捕获）\n")
	fmt.Printf("  ✓ 增强JS分析（对象、路由、配置）\n")
	fmt.Printf("  ✓ 静态资源分类（7种类型）\n")
	fmt.Printf("  ✓ IP地址检测（内网泄露识别）\n")
	fmt.Printf("  ✓ URL优先级排序（智能爬取策略）\n")
	fmt.Printf("  ✓ 结构化日志系统（分级、文件、JSON）🆕\n")
	fmt.Printf("\n爬取配置:\n")
	fmt.Printf("  深度: %d 层 | 并发: 20-30 | 日志: %s\n",
		s.config.DepthSettings.MaxDepth, s.config.LogSettings.Level)
	fmt.Printf("\n")

	// 初始化隐藏路径发现器
	userAgent := ""
	if len(s.config.AntiDetectionSettings.UserAgents) > 0 {
		userAgent = s.config.AntiDetectionSettings.UserAgents[0]
	}
	s.hiddenPathDiscovery = NewHiddenPathDiscovery(targetURL, userAgent)

	// === 优化：先爬取sitemap.xml和robots.txt ===
	s.logger.Info("开始爬取sitemap和robots.txt", "target", targetURL)
	sitemapURLs, robotsInfo := s.sitemapCrawler.GetAllURLs(targetURL)
	s.mutex.Lock()
	s.sitemapURLs = sitemapURLs
	s.robotsURLs = append(robotsInfo.DisallowPaths, robotsInfo.AllowPaths...)
	s.mutex.Unlock()

	s.logger.Info("sitemap和robots.txt爬取完成",
		"sitemap_urls", len(sitemapURLs),
		"disallow_paths", len(robotsInfo.DisallowPaths),
		"allow_paths", len(robotsInfo.AllowPaths),
		"extra_sitemaps", len(robotsInfo.SitemapURLs))

	// 将sitemap和robots中的URL添加到待爬取列表
	for _, u := range sitemapURLs {
		s.visitedURLs[u] = false // 标记为待爬取
	}
	for _, u := range robotsInfo.DisallowPaths {
		s.visitedURLs[u] = false // Disallow路径也要爬取
	}

	// 开始隐藏路径发现
	s.logger.Info("开始扫描隐藏路径")
	hiddenPaths := s.hiddenPathDiscovery.DiscoverAllHiddenPaths()
	s.mutex.Lock()
	s.hiddenPaths = append(s.hiddenPaths, hiddenPaths...)
	s.mutex.Unlock()
	s.logger.Info("隐藏路径扫描完成", "count", len(hiddenPaths))

	// 根据配置决定使用哪种爬虫策略
	if s.config.StrategySettings.EnableStaticCrawler {
		s.logger.Info("使用静态爬虫", "url", targetURL)
		result, err := s.staticCrawler.Crawl(parsedURL)
		if err != nil {
			s.logger.Error("静态爬虫失败", "url", targetURL, "error", err)
		} else {
			s.addResult(result)
			s.logger.Info("静态爬虫完成",
				"url", targetURL,
				"links", len(result.Links),
				"assets", len(result.Assets),
				"forms", len(result.Forms),
				"apis", len(result.APIs))
		}
	}

	// 如果启用了动态爬虫，总是使用（Phase 2/3优化：捕获AJAX和JS动态内容）
	if s.config.StrategySettings.EnableDynamicCrawler {
		s.logger.Info("使用动态爬虫", "url", targetURL, "mode", "ajax_intercept")
		result, err := s.dynamicCrawler.Crawl(parsedURL)
		if err != nil {
			s.logger.Error("动态爬虫失败", "url", targetURL, "error", err)
		} else {
			s.addResult(result)
			s.logger.Info("动态爬虫完成",
				"url", targetURL,
				"links", len(result.Links),
				"assets", len(result.Assets),
				"forms", len(result.Forms),
				"apis", len(result.APIs))
		}
	}

	// 参数爆破功能已移除，专注于纯爬虫
	// 不再生成参数爆破URL，只爬取真实发现的链接

	// 分析跨域JS文件（在递归爬取之前）
	s.processCrossDomainJS()

	// 如果启用了递归爬取，继续爬取发现的链接
	if s.config.DepthSettings.MaxDepth > 1 {
		// 🆕 v2.8: 支持两种爬取模式
		// 模式1: BFS（广度优先，默认）- 稳定可靠
		// 模式2: Priority Queue（优先级队列）- 智能调度
		
		// 🆕 从配置中读取爬取模式
		usePriorityQueue := s.config.StrategySettings.UsePriorityQueue
		
		if usePriorityQueue && s.priorityScheduler != nil {
			// 使用优先级队列模式（实验性）
			s.logger.Info("使用优先级队列模式爬取")
			s.crawlWithPriorityQueue()
		} else {
			// 使用BFS模式（默认，推荐）
			s.crawlRecursivelyMultiLayer()
		}
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
	
	// 🆕 将域内URL添加到去重器
	if s.urlDeduplicator != nil && result != nil {
		// 添加当前页面URL
		if result.URL != "" {
			s.urlDeduplicator.AddURL(result.URL)
		}
		
		// 添加发现的所有链接（域内的）
		if len(result.Links) > 0 {
			s.urlDeduplicator.AddURLs(result.Links)
		}
		
		// 添加API端点
		if len(result.APIs) > 0 {
			s.urlDeduplicator.AddURLs(result.APIs)
		}
		
		// 添加表单action
		for _, form := range result.Forms {
			if form.Action != "" {
				s.urlDeduplicator.AddURL(form.Action)
			}
		}
		
		// 添加POST请求URL
		for _, postReq := range result.POSTRequests {
			if postReq.URL != "" {
				s.urlDeduplicator.AddURL(postReq.URL)
			}
		}
	}

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
	
	// v2.7: 业务感知过滤器 - 自适应学习
	if s.config.DeduplicationSettings.EnableBusinessAwareFilter && 
	   s.config.DeduplicationSettings.BusinessFilterAdaptiveLearning {
		// 计算响应时间（如果可用）
		responseTime := 0.0
		
		// 检查是否发现新内容
		hasNewLinks := len(result.Links) > 0
		hasNewForms := len(result.Forms) > 0
		hasNewAPIs := len(result.APIs) > 0
		
		// 更新爬取结果，用于自适应学习
		s.businessFilter.UpdateCrawlResult(
			result.URL,
			result.StatusCode,
			responseTime,
			hasNewLinks,
			hasNewForms,
			hasNewAPIs,
		)
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

	// v2.6.1: 打印智能参数值去重统计
	if s.config.DeduplicationSettings.EnableSmartParamDedup && s.smartParamDedup != nil {
		s.smartParamDedup.PrintStatistics()
	}

	// 返回结果副本
	results := make([]*Result, len(s.results))
	copy(results, s.results)
	return results
}

// processParams 已废弃（参数爆破功能已移除）
// 保留定义以避免编译错误，但不再使用
func (s *Spider) processParams(rawURL string) []string {
	// 参数爆破功能已移除，直接返回原始URL
	return []string{rawURL}
}

/*
// 以下是原 processParams 的代码（已废弃）
func (s *Spider) processParamsOLD(rawURL string) []string {
	// 提取参数
	params, err := s.paramHandler.ExtractParams(rawURL)
	if err != nil {
		// 如果提取参数失败，至少返回原始URL
		return []string{rawURL}
	}

	// ===  修复：防止参数爆破无限递归 ===
	// 检测URL是否可能是参数爆破生成的（避免对爆破结果再次爆破）
	if len(params) > 0 && s.config.StrategySettings.EnableParamFuzzing {
		// 完整的参数爆破参数列表（来自GenerateParameterFuzzList）
		fuzzedParamNames := []string{
			// 通用参数
			"id", "page", "limit", "offset", "sort", "order", "search", "q", "query",
			"filter", "category", "type", "status", "action", "method", "format",
			// 用户相关
			"user", "username", "userid", "uid", "email", "password", "pass", "pwd",
			"token", "auth", "session", "key", "api_key", "access_token",
			// 文件相关
			"file", "filename", "path", "dir", "folder", "upload", "download",
			"image", "img", "pic", "photo", "document", "doc", "pdf",
			// 其他常见参数
			"debug", "test", "dev", "admin", "config",
		}

		// 测试值列表（用于检测参数值是否是测试值）
		testValues := []string{"1", "test", "admin", "null", "../", "", "false", "true"}

		fuzzedParamCount := 0
		originalParamCount := 0
		testValueCount := 0

		parsedURL, _ := url.Parse(rawURL)
		if parsedURL != nil {
			queryParams := parsedURL.Query()
			for paramName, values := range queryParams {
				// 检查参数名是否是爆破参数
				isFuzzedParam := false
				for _, fuzzName := range fuzzedParamNames {
					if paramName == fuzzName {
						isFuzzedParam = true
						fuzzedParamCount++
						break
					}
				}
				if !isFuzzedParam {
					originalParamCount++
				}

				// 检查参数值是否是测试值
				if len(values) > 0 {
					paramValue := values[0]
					for _, testVal := range testValues {
						if paramValue == testVal || strings.Contains(paramValue, testVal) {
							testValueCount++
							break
						}
					}
				}
			}

			// 判断是否是爆破生成的URL（优化后的检测规则）：
			// 核心原则：只跳过"纯粹由爆破生成"的URL，不误杀真实URL
			//
			// 规则1：包含2个以上爆破参数（如 ?id=1&page=1）
			// 规则2：只有爆破参数，没有原始参数（如 ?search=1, ?limit=）
			//
			// 不跳过的情况：
			// - 真实参数+爆破参数的组合（如 ?article_id=123&id=1）
			// - 真实参数碰巧值是测试值（如 ?article_id=1）
			shouldSkip := false
			skipReason := ""

			if fuzzedParamCount >= 2 {
				// 规则1：多个爆破参数，明显是爆破生成的
				shouldSkip = true
				skipReason = fmt.Sprintf("包含%d个爆破参数", fuzzedParamCount)
			} else if fuzzedParamCount >= 1 && originalParamCount == 0 {
				// 规则2：只有爆破参数，没有原始参数
				// 例如：?search=1, ?limit=, ?id=1
				shouldSkip = true
				skipReason = fmt.Sprintf("只包含爆破参数（%d个），无原始参数", fuzzedParamCount)
			}

			if shouldSkip {
				fmt.Printf("为URL %s 生成 %d 个参数变体\n", rawURL, 1)
				fmt.Printf("  变体: %s\n", rawURL)
				fmt.Printf("  [参数爆破] 检测到该URL可能是爆破生成的（%s），跳过再次爆破\n", skipReason)
				return []string{rawURL}
			}
		}
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
*/

// processForms 已废弃（POST参数爆破功能已移除）
// 保留定义以避免编译错误，但不再使用
func (s *Spider) processForms(targetURL string) {
	// POST参数爆破功能已移除
	return
}

/*
// 以下是原 processForms 的代码（已废弃）
func (s *Spider) processFormsOLD(targetURL string) {
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
*/

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
	skippedBySmart := 0 // 统计智能去重跳过的数量
	skippedByBusiness := 0 // 统计业务感知过滤器跳过的数量
	skippedByPattern := 0 // 统计URL模式去重跳过的数量
	skippedByResourceType := 0 // 🆕 统计资源分类跳过的数量（静态资源/域外）
	
	for link := range allLinks {
		// 🆕 v2.7+: 资源分类检查（最优先）
		if s.resourceClassifier != nil {
			resType, shouldRequest := s.resourceClassifier.ClassifyURL(link)
			if !shouldRequest {
				// 静态资源和域外URL只收集不请求
				skippedByResourceType++
				if skippedByResourceType <= 5 {
					typeStr := s.resourceClassifier.GetResourceTypeString(resType)
					s.logger.Debug("资源分类跳过",
						"url", link,
						"type", typeStr,
						"reason", "只收集不请求")
				}
				continue
			}
		}
		
		// v2.9: URL模式去重检查
		shouldProcess, _, reason := s.urlPatternDedup.ShouldProcess(link, "GET")
		if !shouldProcess {
			skippedByPattern++
			if skippedByPattern <= 3 { // 只打印前3个，避免日志过多
				s.logger.Debug("URL模式去重跳过",
					"url", link,
					"reason", reason)
			}
			continue
		}
		
		// 去重检查
		if s.duplicateHandler.IsDuplicateURL(link) {
			continue
		}

		// v2.6.1: 智能参数值去重检查
		if s.config.DeduplicationSettings.EnableSmartParamDedup {
			shouldCrawl, reason := s.smartParamDedup.ShouldCrawl(link)
			if !shouldCrawl {
				skippedBySmart++
				if skippedBySmart <= 5 { // 只打印前5个，避免日志过多
					fmt.Printf("  [智能去重] 跳过: %s\n  原因: %s\n", link, reason)
				}
				continue
			}
		}

		// v2.7: 业务感知过滤检查
		if s.config.DeduplicationSettings.EnableBusinessAwareFilter {
			shouldCrawl, reason, score := s.businessFilter.ShouldCrawlURL(link)
			if !shouldCrawl {
				skippedByBusiness++
				if skippedByBusiness <= 5 { // 只打印前5个，避免日志过多
					s.logger.Debug("业务感知过滤跳过URL",
						"url", link,
						"reason", reason,
						"score", score)
				}
				continue
			}
			// 记录高价值URL
			if score >= s.config.DeduplicationSettings.BusinessFilterHighValueThreshold {
				s.logger.Info("发现高价值URL",
					"url", link,
					"score", score,
					"reason", reason)
			}
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

	// v2.6.1: 打印智能去重统计
	if skippedBySmart > 0 {
		fmt.Printf("  [智能去重] 本层跳过 %d 个相似参数值URL\n", skippedBySmart)
	}
	
	// v2.7: 打印业务感知过滤统计
	if skippedByBusiness > 0 {
		fmt.Printf("  [业务感知] 本层过滤 %d 个低价值URL\n", skippedByBusiness)
	}
	
	// v2.9: 打印URL模式去重统计
	if skippedByPattern > 0 {
		fmt.Printf("  [URL模式去重] 本层跳过 %d 个重复模式URL\n", skippedByPattern)
	}
	
	// 🆕 v2.7+: 打印资源分类统计
	if skippedByResourceType > 0 {
		fmt.Printf("  [资源分类] 本层跳过 %d 个静态资源/域外URL（已收集不请求）\n", skippedByResourceType)
	}

	// 优先级排序（🆕 传入实际深度，用于精确优先级计算）
	tasksToSubmit = s.prioritizeURLsWithDepth(tasksToSubmit, targetDepth)

	return tasksToSubmit
}

// prioritizeURLsWithDepth 带深度参数的优先级排序
func (s *Spider) prioritizeURLsWithDepth(urls []string, depth int) []string {
	// 如果有优先级调度器，使用精确计算（混合决策模式）
	if s.priorityScheduler != nil {
		return s.prioritizeURLsWithPreciseCalculation(urls, depth)
	}
	
	// 否则使用简单分类
	return s.prioritizeURLs(urls)
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

// prioritizeURLs URL优先级排序（v2.8增强：BFS + 优先级混合决策）
func (s *Spider) prioritizeURLs(urls []string) []string {
	// 🆕 v2.8混合决策：如果有优先级调度器，使用精确优先级计算
	if s.priorityScheduler != nil {
		return s.prioritizeURLsWithPreciseCalculation(urls, 2) // depth默认2（会在调用处传入实际深度）
	}
	
	// 向下兼容：使用原有的简单三级分类
	highPriority := make([]string, 0)   // 高优先级
	mediumPriority := make([]string, 0) // 中优先级
	lowPriority := make([]string, 0)    // 低优先级

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

// prioritizeURLsWithPreciseCalculation 🆕 使用精确优先级计算排序（混合决策核心）
func (s *Spider) prioritizeURLsWithPreciseCalculation(urls []string, depth int) []string {
	type URLWithPriority struct {
		URL      string
		Priority float64
	}
	
	urlsWithPriority := make([]URLWithPriority, 0, len(urls))
	
	// 计算每个URL的精确优先级
	for _, url := range urls {
		priority := s.priorityScheduler.CalculatePriority(url, depth)
		urlsWithPriority = append(urlsWithPriority, URLWithPriority{
			URL:      url,
			Priority: priority,
		})
	}
	
	// 按优先级从高到低排序
	sort.Slice(urlsWithPriority, func(i, j int) bool {
		return urlsWithPriority[i].Priority > urlsWithPriority[j].Priority
	})
	
	// 打印本层优先级TOP3（让用户看到混合决策的效果）
	if len(urlsWithPriority) > 0 {
		fmt.Printf("\n  [混合决策] 本层优先级TOP3（BFS框架 + 智能排序）:\n")
		topCount := 3
		if len(urlsWithPriority) < 3 {
			topCount = len(urlsWithPriority)
		}
		for i := 0; i < topCount; i++ {
			fmt.Printf("    %d. [优先级:%.2f] %s\n", 
				i+1, urlsWithPriority[i].Priority, urlsWithPriority[i].URL)
		}
		
		if len(urlsWithPriority) > 3 {
			fmt.Printf("    ... 还有 %d 个URL按优先级排序\n", len(urlsWithPriority)-3)
		}
	}
	
	// 提取排序后的URL列表
	result := make([]string, 0, len(urls))
	for _, item := range urlsWithPriority {
		result = append(result, item.URL)
	}
	
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

	// v2.7: 业务感知过滤器统计
	if s.businessFilter != nil && s.config.DeduplicationSettings.EnableBusinessAwareFilter {
		exportData["business_filter_stats"] = s.businessFilter.GetStatistics()
		exportData["business_top_patterns"] = s.businessFilter.GetTopPatterns(20)
	}
	
	// 🆕 v2.8: URL去重统计
	if s.urlDeduplicator != nil {
		exportData["url_deduplication"] = s.urlDeduplicator.GetStatistics()
		exportData["unique_url_patterns"] = s.urlDeduplicator.GetUniquePatterns()
		exportData["all_urls_with_variants"] = s.urlDeduplicator.GetAllURLs()
	}

	return exportData
}

// SaveUniqueURLsToFile 保存去重后的URL到文件（给其他工具使用）
func (s *Spider) SaveUniqueURLsToFile(filepath string) error {
	if s.urlDeduplicator == nil {
		return fmt.Errorf("URL去重器未初始化")
	}
	
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// 获取唯一的URL模式
	uniquePatterns := s.urlDeduplicator.GetUniquePatterns()
	
	if len(uniquePatterns) == 0 {
		return fmt.Errorf("没有URL可保存")
	}
	
	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()
	
	// 写入URL（每行一个）
	for _, pattern := range uniquePatterns {
		_, err := file.WriteString(pattern + "\n")
		if err != nil {
			return fmt.Errorf("写入文件失败: %v", err)
		}
	}
	
	// 打印统计
	stats := s.urlDeduplicator.GetStatistics()
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("  ✅ URL去重文件已保存: %s\n", filepath)
	fmt.Printf("  唯一URL模式: %d 个\n", stats["unique_patterns"])
	fmt.Printf("  原始URL总数: %d 个\n", stats["total_urls"])
	if stats["total_urls"] > stats["unique_patterns"] {
		reduction := stats["total_urls"] - stats["unique_patterns"]
		reductionPercent := float64(reduction) / float64(stats["total_urls"]) * 100
		fmt.Printf("  去重效果: 减少 %d 个 (%.1f%%)\n", reduction, reductionPercent)
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	
	return nil
}

// PrintURLDeduplicationReport 打印URL去重详细报告
func (s *Spider) PrintURLDeduplicationReport() {
	if s.urlDeduplicator != nil {
		s.urlDeduplicator.PrintReport()
		// 显示前10个最多变体的URL模式
		s.urlDeduplicator.PrintDetailedReport(10)
	}
}

// PrintBusinessFilterReport 打印业务感知过滤器的详细报告
func (s *Spider) PrintBusinessFilterReport() {
	if s.businessFilter != nil && s.config.DeduplicationSettings.EnableBusinessAwareFilter {
		s.businessFilter.PrintReport()
	}
}

// PrintURLPatternDedupReport 打印URL模式去重报告
func (s *Spider) PrintURLPatternDedupReport() {
	if s.urlPatternDedup != nil {
		s.urlPatternDedup.PrintReport()
	}
}

// crawlWithPriorityQueue 🆕 使用优先级队列模式爬取（实验性）
func (s *Spider) crawlWithPriorityQueue() {
	fmt.Println("\n开始优先级队列模式爬取...")
	fmt.Println("算法：BFS + 优先级调度（智能排序）")
	
	// 将所有已发现的URL添加到优先级队列
	s.mutex.Lock()
	for _, result := range s.results {
		for _, link := range result.Links {
			// 计算深度（简化：都视为深度2）
			s.priorityScheduler.AddURL(link, 2)
		}
	}
	s.mutex.Unlock()
	
	fmt.Printf("优先级队列初始化完成，队列大小: %d\n", s.priorityScheduler.Size())
	
	totalCrawled := 0
	maxURLs := 500 // 限制最大爬取数量
	
	// 循环从队列中取URL爬取
	for totalCrawled < maxURLs && s.priorityScheduler.Size() > 0 {
		// 批量取出高优先级URL
		batchSize := 30 // 每批30个（匹配worker数量）
		batch := s.priorityScheduler.PopBatch(batchSize)
		
		if len(batch) == 0 {
			break
		}
		
		fmt.Printf("\n批次爬取: %d个URL（优先级排序）\n", len(batch))
		
		// 显示前3个URL的优先级
		for i := 0; i < len(batch) && i < 3; i++ {
			fmt.Printf("  [优先级: %.2f] %s\n", batch[i].Priority, batch[i].URL)
		}
		if len(batch) > 3 {
			fmt.Printf("  ... 还有 %d 个URL\n", len(batch)-3)
		}
		
		// 提取URL列表
		urls := make([]string, 0, len(batch))
		for _, item := range batch {
			urls = append(urls, item.URL)
		}
		
		// 爬取这批URL
		newResults := s.crawlLayer(urls, batch[0].Depth)
		
		// 合并结果
		s.mutex.Lock()
		s.results = append(s.results, newResults...)
		
		// 将新发现的URL添加到优先级队列
		for _, result := range newResults {
			for _, newLink := range result.Links {
				if !s.priorityScheduler.IsVisited(newLink) {
					// 新链接的深度 = 当前深度 + 1
					newDepth := batch[0].Depth + 1
					if newDepth <= s.config.DepthSettings.MaxDepth {
						s.priorityScheduler.AddURL(newLink, newDepth)
					}
				}
			}
		}
		s.mutex.Unlock()
		
		totalCrawled += len(batch)
		fmt.Printf("已爬取: %d个，队列剩余: %d个\n", totalCrawled, s.priorityScheduler.Size())
		
		// 检查是否达到最大深度
		if batch[0].Depth >= s.config.DepthSettings.MaxDepth {
			fmt.Printf("已达到最大深度 %d，停止爬取\n", s.config.DepthSettings.MaxDepth)
			break
		}
	}
	
	// 打印最终统计
	fmt.Printf("\n优先级队列爬取完成！总共爬取 %d 个URL\n", totalCrawled)
	s.priorityScheduler.PrintStatistics()
}
