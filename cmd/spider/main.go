package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"spider-golang/config"
	"spider-golang/core"
)

var (
	targetURL       string
	mode            string
	maxDepth        int
	maxPages        int
	timeout         int
	workers         int
	cookieFile      string
	customHeaders   string
	proxy           string
	userAgent       string
	ignoreRobots    bool
	allowSubdomains bool
	outputDir       string
	chromePath      string
	enableFuzzing   bool
	fuzzParams      string
	fuzzDict        string
	configFile      string
	// v2.6 新增：日志和监控参数
	logLevel        string
	logFile         string
	logFormat       string
	showMetrics     bool
	// v2.6 新增：易用性参数（借鉴竞品）
	useStdin        bool
	simpleMode      bool
	outputFormat    string
	showVersion     bool
	
	// 🆕 v2.9 新增：企业级功能参数
	// JSON输出
	enableJSON      bool
	jsonMode        string
	outputFile      string
	includeAllFields bool
	
	// 速率控制
	enableRateLimit bool
	requestsPerSec  int
	burstSize       int
	minDelay        int
	maxDelay        int
	adaptiveRate    bool
	minRate         int
	maxRate         int
	
	// 外部数据源
	enableWayback   bool
	enableVT        bool
	vtAPIKey        string
	enableCC        bool
	externalTimeout int
	
	// Scope控制
	includeDomains  string
	excludeDomains  string
	includePaths    string
	excludePaths    string
	includeRegex    string
	excludeRegex    string
	includeExt      string
	excludeExt      string
	
	// 管道模式
	enablePipeline  bool
	quietMode       bool
)

func init() {
	flag.StringVar(&targetURL, "url", "", "目标URL（必需）")
	flag.StringVar(&mode, "mode", "smart", "爬取模式: static, dynamic, smart（默认）")
	flag.IntVar(&maxDepth, "depth", 3, "最大爬取深度")
	flag.IntVar(&maxPages, "max-pages", 100, "最大爬取页面数")
	flag.IntVar(&timeout, "timeout", 30, "请求超时时间（秒）")
	flag.IntVar(&workers, "workers", 10, "并发工作线程数")
	flag.StringVar(&cookieFile, "cookie-file", "", "Cookie文件路径")
	flag.StringVar(&customHeaders, "headers", "", "自定义HTTP头（JSON格式）")
	flag.StringVar(&proxy, "proxy", "", "代理服务器地址")
	flag.StringVar(&userAgent, "user-agent", "", "自定义User-Agent")
	flag.BoolVar(&ignoreRobots, "ignore-robots", false, "忽略robots.txt")
	flag.BoolVar(&allowSubdomains, "allow-subdomains", false, "允许爬取子域名")
	flag.StringVar(&outputDir, "output", "./", "输出目录")
	flag.StringVar(&chromePath, "chrome-path", "", "Chrome浏览器路径")
	flag.BoolVar(&enableFuzzing, "fuzz", false, "启用参数模糊测试")
	flag.StringVar(&fuzzParams, "fuzz-params", "", "要fuzz的参数列表（逗号分隔）")
	flag.StringVar(&fuzzDict, "fuzz-dict", "", "Fuzz字典文件路径")
	flag.StringVar(&configFile, "config", "", "配置文件路径")
	// v2.6 新增参数
	flag.StringVar(&logLevel, "log-level", "info", "日志级别: debug, info, warn, error")
	flag.StringVar(&logFile, "log-file", "", "日志文件路径（空表示输出到控制台）")
	flag.StringVar(&logFormat, "log-format", "json", "日志格式: json, text")
	flag.BoolVar(&showMetrics, "show-metrics", false, "显示实时监控指标")
	// v2.6 新增：易用性参数（借鉴 Hakrawler/Katana）
	flag.BoolVar(&useStdin, "stdin", false, "从标准输入读取URL（支持pipeline）")
	flag.BoolVar(&simpleMode, "simple", false, "简洁模式（只输出URL，适合pipeline）")
	flag.StringVar(&outputFormat, "format", "text", "输出格式: text, json, urls-only")
	flag.BoolVar(&showVersion, "version", false, "显示版本信息")
	
	// 🆕 v2.9 新增：企业级功能参数
	// JSON输出参数
	flag.BoolVar(&enableJSON, "json", false, "启用JSON输出格式")
	flag.StringVar(&jsonMode, "json-mode", "line", "JSON模式: compact, pretty, line")
	flag.StringVar(&outputFile, "output-file", "", "输出文件路径（为空则输出到stdout）")
	flag.BoolVar(&includeAllFields, "include-all", false, "JSON输出包含所有字段")
	
	// 速率控制参数
	flag.BoolVar(&enableRateLimit, "rate-limit-enable", false, "启用速率限制")
	flag.IntVar(&requestsPerSec, "rate-limit", 100, "每秒最大请求数（设置后自动启用速率限制）")
	flag.IntVar(&burstSize, "burst", 10, "允许的突发请求数")
	flag.IntVar(&minDelay, "min-delay", 0, "最小请求间隔（毫秒）")
	flag.IntVar(&maxDelay, "max-delay", 0, "最大请求间隔（毫秒）")
	flag.BoolVar(&adaptiveRate, "adaptive-rate", false, "启用自适应速率控制")
	flag.IntVar(&minRate, "min-rate", 10, "自适应最小速率")
	flag.IntVar(&maxRate, "max-rate", 200, "自适应最大速率")
	
	// 外部数据源参数
	flag.BoolVar(&enableWayback, "wayback", false, "从Wayback Machine获取历史URL")
	flag.BoolVar(&enableVT, "virustotal", false, "从VirusTotal获取URL")
	flag.StringVar(&vtAPIKey, "vt-api-key", "", "VirusTotal API密钥")
	flag.BoolVar(&enableCC, "commoncrawl", false, "从CommonCrawl获取URL")
	flag.IntVar(&externalTimeout, "external-timeout", 30, "外部数据源超时（秒）")
	
	// Scope控制参数
	flag.StringVar(&includeDomains, "include-domains", "", "包含的域名列表（逗号分隔，支持*.example.com）")
	flag.StringVar(&excludeDomains, "exclude-domains", "", "排除的域名列表（逗号分隔）")
	flag.StringVar(&includePaths, "include-paths", "", "包含的路径模式（逗号分隔，支持/api/*）")
	flag.StringVar(&excludePaths, "exclude-paths", "", "排除的路径模式（逗号分隔）")
	flag.StringVar(&includeRegex, "include-regex", "", "包含的URL正则表达式")
	flag.StringVar(&excludeRegex, "exclude-regex", "", "排除的URL正则表达式")
	flag.StringVar(&includeExt, "include-ext", "", "包含的文件扩展名（逗号分隔）")
	flag.StringVar(&excludeExt, "exclude-ext", "", "排除的文件扩展名（逗号分隔）")
	
	// 管道模式参数
	flag.BoolVar(&enablePipeline, "pipeline", false, "启用管道模式")
	flag.BoolVar(&quietMode, "quiet", false, "静默模式（日志输出到stderr）")
}


func main() {
	// 🔧 优化：添加panic恢复机制
	defer func() {
		if r := recover(); r != nil {
			log.Printf("程序panic: %v", r)
			log.Printf("请查看日志文件或使用 -log-level debug 获取详细信息")
			os.Exit(1)
		}
	}()
	
	flag.Parse()

	// v2.6: 处理 version 命令
	if showVersion {
		printVersion()
		os.Exit(0)
	}

	// v2.6: 处理 stdin 模式（借鉴 Hakrawler）
	if useStdin {
		handleStdinMode()
		return
	}

	// 简洁模式下不显示横幅
	if !simpleMode {
		printBanner()
	}

	// 🔧 优化：加载配置（支持配置文件）
	var cfg *config.Config
	
	if configFile != "" {
		// 从配置文件加载
		loadedCfg, err := loadConfigFile(configFile)
		if err != nil {
			log.Fatalf("加载配置文件失败: %v", err)
		}
		cfg = loadedCfg
		if !simpleMode {
			fmt.Printf("[*] 已加载配置文件: %s\n", configFile)
		}
	} else {
		// 使用默认配置
		cfg = config.NewDefaultConfig()
	}

	// 命令行参数覆盖配置文件
	if targetURL != "" {
		cfg.TargetURL = targetURL
	}
	if maxDepth != 3 {
		cfg.DepthSettings.MaxDepth = maxDepth
	}
	if proxy != "" {
		cfg.AntiDetectionSettings.Proxies = []string{proxy}
	}
	if userAgent != "" {
		cfg.AntiDetectionSettings.UserAgents = []string{userAgent}
	}
	// 参数爆破功能已移除
	// if enableFuzzing {
	// 	cfg.StrategySettings.EnableParamFuzzing = true
	// 	cfg.StrategySettings.EnablePOSTParamFuzzing = true
	// }
	
	// v2.6: 配置日志设置
	if logLevel != "info" {
		cfg.LogSettings.Level = strings.ToUpper(logLevel)
	}
	if logFile != "" {
		cfg.LogSettings.OutputFile = logFile
	}
	if logFormat != "json" {
		cfg.LogSettings.Format = logFormat
	}
	if showMetrics {
		cfg.LogSettings.ShowMetrics = true
	}
	
	// 🆕 v2.9: 应用新功能参数到配置
	// JSON输出配置
	if enableJSON {
		cfg.OutputSettings.Format = "json"
		cfg.OutputSettings.JSONMode = jsonMode
		cfg.OutputSettings.IncludeAll = includeAllFields
	}
	if outputFile != "" {
		cfg.OutputSettings.OutputFile = outputFile
	}
	
	// 速率控制配置
	if requestsPerSec != 100 || enableRateLimit {
		cfg.RateLimitSettings.Enabled = true
		cfg.RateLimitSettings.RequestsPerSecond = requestsPerSec
	}
	if burstSize != 10 {
		cfg.RateLimitSettings.BurstSize = burstSize
	}
	if minDelay > 0 {
		cfg.RateLimitSettings.MinDelay = minDelay
	}
	if maxDelay > 0 {
		cfg.RateLimitSettings.MaxDelay = maxDelay
	}
	if adaptiveRate {
		cfg.RateLimitSettings.Adaptive = true
		cfg.RateLimitSettings.AdaptiveMinRate = minRate
		cfg.RateLimitSettings.AdaptiveMaxRate = maxRate
	}
	
	// 外部数据源配置
	if enableWayback || enableVT || enableCC {
		cfg.ExternalSourceSettings.Enabled = true
		cfg.ExternalSourceSettings.EnableWaybackMachine = enableWayback
		cfg.ExternalSourceSettings.EnableVirusTotal = enableVT
		cfg.ExternalSourceSettings.VirusTotalAPIKey = vtAPIKey
		cfg.ExternalSourceSettings.EnableCommonCrawl = enableCC
		cfg.ExternalSourceSettings.Timeout = externalTimeout
	}
	
	// Scope控制配置
	if includeDomains != "" || excludeDomains != "" || includePaths != "" || 
	   excludePaths != "" || includeRegex != "" || excludeRegex != "" ||
	   includeExt != "" || excludeExt != "" {
		cfg.ScopeSettings.Enabled = true
		
		if includeDomains != "" {
			cfg.ScopeSettings.IncludeDomains = strings.Split(includeDomains, ",")
		}
		if excludeDomains != "" {
			cfg.ScopeSettings.ExcludeDomains = strings.Split(excludeDomains, ",")
		}
		if includePaths != "" {
			cfg.ScopeSettings.IncludePaths = strings.Split(includePaths, ",")
		}
		if excludePaths != "" {
			cfg.ScopeSettings.ExcludePaths = strings.Split(excludePaths, ",")
		}
		if includeRegex != "" {
			cfg.ScopeSettings.IncludeRegex = includeRegex
		}
		if excludeRegex != "" {
			cfg.ScopeSettings.ExcludeRegex = excludeRegex
		}
		if includeExt != "" {
			cfg.ScopeSettings.IncludeExtensions = strings.Split(includeExt, ",")
		}
		if excludeExt != "" {
			cfg.ScopeSettings.ExcludeExtensions = strings.Split(excludeExt, ",")
		}
	}
	
	// 管道模式配置
	if enablePipeline || useStdin {
		cfg.PipelineSettings.Enabled = true
		cfg.PipelineSettings.EnableStdin = useStdin || enablePipeline
		cfg.PipelineSettings.EnableStdout = true
		cfg.PipelineSettings.Quiet = quietMode
	}

	// 参数验证
	if cfg.TargetURL == "" {
		fmt.Println("错误: 必须指定目标URL")
		flag.Usage()
		os.Exit(1)
	}
	
	// 配置验证（优化：确保配置有效）
	if err := cfg.Validate(); err != nil {
		fmt.Printf("配置验证失败: %v\n", err)
		os.Exit(1)
	}

	// 创建爬虫实例
	spider := core.NewSpider(cfg)
	defer spider.Close() // 确保资源清理

	// 启动爬取
	fmt.Printf("\n[*] 开始爬取: %s\n", cfg.TargetURL)
	fmt.Printf("[*] 最大深度: %d\n", cfg.DepthSettings.MaxDepth)
	fmt.Printf("[*] 静态爬虫: %v\n", cfg.StrategySettings.EnableStaticCrawler)
	fmt.Printf("[*] 动态爬虫: %v\n", cfg.StrategySettings.EnableDynamicCrawler)
	fmt.Printf("[*] 纯爬虫模式: 专注URL发现（已禁用参数爆破）\n")
	fmt.Println()

	startTime := time.Now()
	err := spider.Start(cfg.TargetURL)
	if err != nil {
		log.Fatalf("爬取失败: %v", err)
	}
	elapsed := time.Since(startTime)

	// 获取结果
	results := spider.GetResults()

	// 生成输出文件名
	timestamp := time.Now().Format("20060102_150405")
	domain := extractDomain(cfg.TargetURL)
	baseFilename := fmt.Sprintf("spider_%s_%s", domain, timestamp)

	// 保存结果
	if err := saveResults(results, baseFilename+".txt"); err != nil {
		log.Printf("保存结果失败: %v", err)
	}

	// 保存URL列表（旧版，为了兼容性保留）
	if err := saveURLs(results, baseFilename+"_urls.txt"); err != nil {
		log.Printf("保存URL列表失败: %v", err)
	}
	
	// 保存所有类型的URL到不同文件（新增：增强版）
	if err := saveAllURLs(results, baseFilename); err != nil {
		log.Printf("保存分类URL失败: %v", err)
	}

	// 🆕 v2.8: 保存去重后的URL（忽略参数值）
	uniqueURLFile := baseFilename + "_unique_urls.txt"
	if err := spider.SaveUniqueURLsToFile(uniqueURLFile); err != nil {
		log.Printf("保存去重URL失败: %v", err)
	}
	
	// 🆕 结构化去重: 保存结构化去重后的URL（识别路径变量+参数值）
	// 先收集所有URL到结构化去重器
	spider.CollectAllURLsForStructureDedup()
	
	// 保存结构化去重后的URL
	structureUniqueFile := baseFilename + "_structure_unique_urls.txt"
	if err := spider.SaveStructureUniqueURLsToFile(structureUniqueFile); err != nil {
		log.Printf("保存结构化去重URL失败: %v", err)
	}
	
	// 打印统计信息
	if !simpleMode {
		printStats(results, elapsed)
		
		// v2.9: 打印URL模式去重报告
		spider.PrintURLPatternDedupReport()
		
		// v2.7: 打印业务感知过滤器报告
		spider.PrintBusinessFilterReport()
		
		// 🆕 v2.8: 打印URL去重报告
		spider.PrintURLDeduplicationReport()
		
		// 🆕 结构化去重: 打印结构化去重报告
		spider.PrintStructureDeduplicationReport()
		
		fmt.Printf("\n[+] 结果已保存到当前目录\n")
	}
	
	// v2.6: 处理不同的输出格式（借鉴 Katana）
	handleOutputFormat(results)
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   ███████╗██████╗ ██╗██████╗ ███████╗██████╗                ║
║   ██╔════╝██╔══██╗██║██╔══██╗██╔════╝██╔══██╗               ║
║   ███████╗██████╔╝██║██║  ██║█████╗  ██████╔╝               ║
║   ╚════██║██╔═══╝ ██║██║  ██║██╔══╝  ██╔══██╗               ║
║   ███████║██║     ██║██████╔╝███████╗██║  ██║               ║
║   ╚══════╝╚═╝     ╚═╝╚═════╝ ╚══════╝╚═╝  ╚═╝               ║
║                                                               ║
║            Spider Ultimate - 智能Web爬虫系统                 ║
║              Version 2.10 - Pure Crawler                      ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}

func extractDomain(urlStr string) string {
	urlStr = strings.TrimPrefix(urlStr, "http://")
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.Split(urlStr, "/")[0]
	urlStr = strings.ReplaceAll(urlStr, ":", "_")
	return urlStr
}

// isInTargetDomain 检查URL是否属于目标域名
func isInTargetDomain(urlStr, targetDomain string) bool {
	// 忽略mailto等特殊协议
	if strings.HasPrefix(urlStr, "mailto:") || 
	   strings.HasPrefix(urlStr, "tel:") ||
	   strings.HasPrefix(urlStr, "javascript:") {
		return false
	}
	
	// 提取URL的域名部分
	urlDomain := strings.TrimPrefix(urlStr, "http://")
	urlDomain = strings.TrimPrefix(urlDomain, "https://")
	urlDomain = strings.Split(urlDomain, "/")[0]
	urlDomain = strings.Split(urlDomain, ":")[0] // 移除端口号
	
	// 清理目标域名（移除端口号）
	cleanTargetDomain := strings.Split(targetDomain, ":")[0]
	cleanTargetDomain = strings.ReplaceAll(cleanTargetDomain, "_", ":") // extractDomain会替换冒号
	
	// 完全匹配
	if urlDomain == cleanTargetDomain {
		return true
	}
	
	// 子域名匹配（例如：api.example.com 匹配 example.com）
	if strings.HasSuffix(urlDomain, "."+cleanTargetDomain) {
		return true
	}
	
	return false
}

func saveResults(results []*core.Result, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, result := range results {
		output := fmt.Sprintf("[GET] %s | 状态码: %d | 类型: %s\n", 
			result.URL, result.StatusCode, result.ContentType)
		file.WriteString(output)

		// 保存发现的链接
		if len(result.Links) > 0 {
			file.WriteString(fmt.Sprintf("  链接数: %d\n", len(result.Links)))
		}

		// 保存表单信息
		if len(result.Forms) > 0 {
			file.WriteString(fmt.Sprintf("  表单数: %d\n", len(result.Forms)))
			for _, form := range result.Forms {
				file.WriteString(fmt.Sprintf("    - %s %s\n", form.Method, form.Action))
			}
		}

		// 保存POST请求
		if len(result.POSTRequests) > 0 {
			file.WriteString(fmt.Sprintf("  POST请求数: %d\n", len(result.POSTRequests)))
			for _, post := range result.POSTRequests {
				file.WriteString(fmt.Sprintf("    - [POST] %s\n", post.URL))
				if len(post.Parameters) > 0 {
					paramsJSON, _ := json.Marshal(post.Parameters)
					file.WriteString(fmt.Sprintf("      参数: %s\n", string(paramsJSON)))
				}
			}
		}

		// 保存API
		if len(result.APIs) > 0 {
			file.WriteString(fmt.Sprintf("  API数: %d\n", len(result.APIs)))
			for _, api := range result.APIs {
				file.WriteString(fmt.Sprintf("    - %s\n", api))
			}
		}

		file.WriteString("\n")
	}

	return nil
}

func saveURLs(results []*core.Result, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	urlSet := make(map[string]bool)
	
	// 获取目标域名（从第一个结果的URL中提取）
	var targetDomain string
	if len(results) > 0 {
		targetDomain = extractDomain(results[0].URL)
	}
	
	// 收集所有URL：爬取的页面URL + 发现的链接
	for _, result := range results {
		// 添加页面URL
		if !urlSet[result.URL] && isInTargetDomain(result.URL, targetDomain) {
			file.WriteString(result.URL + "\n")
			urlSet[result.URL] = true
		}
		
		// 添加发现的所有链接（只添加目标域名的链接）
		for _, link := range result.Links {
			if !urlSet[link] && isInTargetDomain(link, targetDomain) {
				file.WriteString(link + "\n")
				urlSet[link] = true
			}
		}
	}

	return nil
}

// saveAllURLs 保存所有类型的URL到不同文件（新增：增强版URL保存）
func saveAllURLs(results []*core.Result, baseFilename string) error {
	// 获取目标域名
	var targetDomain string
	if len(results) > 0 {
		targetDomain = extractDomain(results[0].URL)
	}
	
	// 1. 保存所有URL（最完整）
	allURLs := make(map[string]bool)
	paramURLs := make(map[string]bool)
	apiURLs := make(map[string]bool)
	formURLs := make(map[string]bool)
	
	for _, result := range results {
		// 收集爬取的页面URL（只保存目标域名的URL）
		if isInTargetDomain(result.URL, targetDomain) {
			allURLs[result.URL] = true
			
			if strings.Contains(result.URL, "?") {
				paramURLs[result.URL] = true
			}
		}
		
		// 收集发现的链接（只保存目标域名的链接）
		for _, link := range result.Links {
			if isInTargetDomain(link, targetDomain) {
				allURLs[link] = true
				if strings.Contains(link, "?") {
					paramURLs[link] = true
				}
			}
		}
		
		// 收集API（只保存目标域名的API）
		for _, api := range result.APIs {
			if isInTargetDomain(api, targetDomain) {
				allURLs[api] = true
				apiURLs[api] = true
			}
		}
		
		// 收集表单URL（只保存目标域名的表单URL）
		for _, form := range result.Forms {
			if form.Action != "" && isInTargetDomain(form.Action, targetDomain) {
				allURLs[form.Action] = true
				formURLs[form.Action] = true
			}
		}
	}
	
	// 保存所有URL到主文件
	if err := writeURLsToFile(allURLs, baseFilename+"_all_urls.txt"); err != nil {
		return fmt.Errorf("保存全部URL失败: %v", err)
	}
	
	// 保存带参数的URL（方便参数Fuzz）
	if len(paramURLs) > 0 {
		if err := writeURLsToFile(paramURLs, baseFilename+"_params.txt"); err != nil {
			log.Printf("警告: 保存参数URL失败: %v", err)
		}
	}
	
	// 保存API URL（方便API测试）
	if len(apiURLs) > 0 {
		if err := writeURLsToFile(apiURLs, baseFilename+"_apis.txt"); err != nil {
			log.Printf("警告: 保存API URL失败: %v", err)
		}
	}
	
	// 保存表单URL（方便表单测试）
	if len(formURLs) > 0 {
		if err := writeURLsToFile(formURLs, baseFilename+"_forms.txt"); err != nil {
			log.Printf("警告: 保存表单URL失败: %v", err)
		}
	}
	
	// 收集POST请求
	postRequests := make([]*core.POSTRequest, 0)
	for _, result := range results {
		if len(result.POSTRequests) > 0 {
			for i := range result.POSTRequests {
				postRequests = append(postRequests, &result.POSTRequests[i])
			}
		}
	}
	
	// 保存POST请求（新增：增强版）
	if len(postRequests) > 0 {
		if err := savePOSTRequests(postRequests, baseFilename+"_post_requests.txt"); err != nil {
			log.Printf("警告: 保存POST请求失败: %v", err)
		}
	}
	
	// 打印保存统计
	fmt.Printf("\n[+] URL保存完成:\n")
	fmt.Printf("  - %s_all_urls.txt  : %d 个URL（全部）\n", baseFilename, len(allURLs))
	if len(paramURLs) > 0 {
		fmt.Printf("  - %s_params.txt    : %d 个URL（带参数）\n", baseFilename, len(paramURLs))
	}
	if len(apiURLs) > 0 {
		fmt.Printf("  - %s_apis.txt      : %d 个URL（API接口）\n", baseFilename, len(apiURLs))
	}
	if len(formURLs) > 0 {
		fmt.Printf("  - %s_forms.txt     : %d 个URL（表单）\n", baseFilename, len(formURLs))
	}
	if len(postRequests) > 0 {
		fmt.Printf("  - %s_post_requests.txt : %d 个POST请求\n", baseFilename, len(postRequests))
	}
	
	return nil
}

// savePOSTRequests 保存POST请求到文件
func savePOSTRequests(requests []*core.POSTRequest, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	for i, req := range requests {
		if i > 0 {
			file.WriteString("\n")
		}
		
		// 写入请求方法和URL
		file.WriteString(fmt.Sprintf("%s %s\n", req.Method, req.URL))
		
		// 写入Content-Type
		if req.ContentType != "" {
			file.WriteString(fmt.Sprintf("  Content-Type: %s\n", req.ContentType))
		}
		
		// 写入参数
		if len(req.Parameters) > 0 {
			file.WriteString("  Parameters:\n")
			// 排序参数名以保持一致性
			paramNames := make([]string, 0, len(req.Parameters))
			for name := range req.Parameters {
				paramNames = append(paramNames, name)
			}
			sort.Strings(paramNames)
			
			for _, name := range paramNames {
				file.WriteString(fmt.Sprintf("    %s=%s\n", name, req.Parameters[name]))
			}
		}
		
		// 写入请求体
		if req.Body != "" {
			file.WriteString("  Body: ")
			// 如果Body太长，只显示前200个字符
			if len(req.Body) > 200 {
				file.WriteString(req.Body[:200] + "...\n")
			} else {
				file.WriteString(req.Body + "\n")
			}
		}
		
		// 写入来源信息
		if req.FromForm {
			file.WriteString(fmt.Sprintf("  From Form: %s\n", req.FormAction))
		}
	}
	
	return nil
}

// writeURLsToFile 将URL集合写入文件
func writeURLsToFile(urls map[string]bool, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// 转换为切片并排序（方便查看和对比）
	urlList := make([]string, 0, len(urls))
	for url := range urls {
		urlList = append(urlList, url)
	}
	sort.Strings(urlList)
	
	// 写入文件
	for _, url := range urlList {
		file.WriteString(url + "\n")
	}
	
	return nil
}

func printStats(results []*core.Result, elapsed time.Duration) {
	stats := map[string]int{
		"总页面":     0,
		"总链接":     0,
		"总表单":     0,
		"POST请求":  0,
		"API接口":   0,
		"带参数":     0,
		"静态资源":    0,
	}

	uniqueURLs := make(map[string]bool)
	totalLinks := 0
	totalForms := 0
	totalPOST := 0
	totalAPIs := 0

	for _, result := range results {
		uniqueURLs[result.URL] = true
		stats["总页面"]++

		totalLinks += len(result.Links)
		totalForms += len(result.Forms)
		totalPOST += len(result.POSTRequests)
		totalAPIs += len(result.APIs)

		if strings.Contains(result.URL, "?") {
			stats["带参数"]++
		}

		// 简单判断静态资源
		ext := strings.ToLower(filepath.Ext(result.URL))
		if ext == ".js" || ext == ".css" || ext == ".jpg" || ext == ".png" || 
		   ext == ".gif" || ext == ".svg" || ext == ".woff" || ext == ".ttf" {
			stats["静态资源"]++
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("                        爬取统计")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("爬取页面数:    %d\n", stats["总页面"])
	fmt.Printf("唯一URL数:     %d\n", len(uniqueURLs))
	fmt.Printf("发现链接数:    %d\n", totalLinks)
	fmt.Printf("发现表单数:    %d\n", totalForms)
	fmt.Printf("POST请求数:    %d\n", totalPOST)
	fmt.Printf("API接口数:     %d\n", totalAPIs)
	fmt.Printf("带参数URL:     %d\n", stats["带参数"])
	fmt.Printf("静态资源:      %d\n", stats["静态资源"])
	fmt.Printf("耗时:          %.2f秒\n", elapsed.Seconds())
	if elapsed.Seconds() > 0 {
		fmt.Printf("平均速度:      %.2f 页/秒\n", float64(stats["总页面"])/elapsed.Seconds())
	}
	fmt.Println(strings.Repeat("=", 60))
}

// printVersion 显示版本信息
func printVersion() {
	fmt.Println("Spider Ultimate v2.10 - Pure Crawler Edition")
	fmt.Println("Build: 2025-10-25")
	fmt.Println("Go Version: " + strings.TrimPrefix(filepath.Base(os.Args[0]), "go"))
	fmt.Println("")
	fmt.Println("Features:")
	fmt.Println("  ✓ 静态+动态双引擎爬虫")
	fmt.Println("  ✓ AJAX请求拦截")
	fmt.Println("  ✓ JavaScript深度分析")
	fmt.Println("  ✓ 跨域JS分析（60+CDN）")
	fmt.Println("  ✓ 智能表单识别")
	fmt.Println("  ✓ URL模式去重 🆕")
	fmt.Println("  ✓ 业务感知过滤 🆕")
	fmt.Println("  ✓ DOM相似度检测")
	fmt.Println("  ✓ 技术栈检测")
	fmt.Println("  ✓ 敏感信息检测")
	fmt.Println("  ✓ 结构化日志系统")
	fmt.Println("  ✓ Pipeline支持")
	fmt.Println("")
	fmt.Println("Positioning: Pure Web Crawler - Focus on URL Discovery")
	fmt.Println("GitHub: https://github.com/Warren-Jace/gogospider")
}

// handleStdinMode 处理 stdin 模式（v2.6 新增，借鉴 Hakrawler）
func handleStdinMode() {
	// 从 stdin 读取 URL
	scanner := bufio.NewScanner(os.Stdin)
	urlCount := 0
	
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url == "" {
			continue
		}
		
		urlCount++
		
		// 为每个 URL 创建配置
		cfg := config.NewDefaultConfig()
		cfg.TargetURL = url
		
		if maxDepth != 3 {
			cfg.DepthSettings.MaxDepth = maxDepth
		}
		if logLevel != "info" {
			cfg.LogSettings.Level = strings.ToUpper(logLevel)
		}
		// 参数爆破功能已移除
		// if enableFuzzing {
		// 	cfg.StrategySettings.EnableParamFuzzing = true
		// }
		if proxy != "" {
			cfg.AntiDetectionSettings.Proxies = []string{proxy}
		}
		
		// 验证配置
		if err := cfg.Validate(); err != nil {
			if !simpleMode {
				log.Printf("配置验证失败 %s: %v", url, err)
			}
			continue
		}
		
		// 🔧 修复：创建爬虫后立即关闭，避免资源泄漏
		func() {
			spider := core.NewSpider(cfg)
			defer spider.Close() // 在匿名函数结束时立即关闭
			
			// 爬取
			err := spider.Start(url)
			if err != nil && !simpleMode {
				log.Printf("爬取失败 %s: %v", url, err)
				return
			}
			
			// 获取结果
			results := spider.GetResults()
			
			// 简洁模式：只输出 URL
			if simpleMode {
				for _, result := range results {
					fmt.Println(result.URL)
				}
			} else {
				// 正常模式：显示统计
				fmt.Printf("[%d] %s - 发现 %d 个结果\n", urlCount, url, len(results))
			}
		}()
	}
	
	if err := scanner.Err(); err != nil {
		log.Fatalf("读取输入失败: %v", err)
	}
	
	if !simpleMode {
		fmt.Printf("\n总计处理 %d 个URL\n", urlCount)
	}
}

// handleOutputFormat 处理输出格式（v2.6 新增，借鉴 Katana）
func handleOutputFormat(results []*core.Result) {
	switch outputFormat {
	case "json":
		// JSON 格式输出
		output := map[string]interface{}{
			"version": "2.6",
			"timestamp": time.Now().Format(time.RFC3339),
			"total": len(results),
			"results": results,
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Printf("JSON 编码失败: %v", err)
			return
		}
		fmt.Println(string(data))
		
	case "urls-only":
		// 只输出 URL（去重）
		urlSet := make(map[string]bool)
		for _, result := range results {
			if !urlSet[result.URL] {
				fmt.Println(result.URL)
				urlSet[result.URL] = true
			}
			// 也输出发现的链接
			for _, link := range result.Links {
				if !urlSet[link] {
					fmt.Println(link)
					urlSet[link] = true
				}
			}
		}
		
	case "text":
		// 默认文本格式（已经在前面处理）
		// 不需要额外操作
	}
}

// loadConfigFile 加载配置文件（v2.9新增）
func loadConfigFile(filename string) (*config.Config, error) {
	// 读取文件
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}
	
	// 解析JSON
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}
	
	// 验证配置
	if err := cfg.ValidateAndFix(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}
	
	return &cfg, nil
}
