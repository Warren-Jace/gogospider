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
	"sync"
	"time"

	"spider-golang/config"
	"spider-golang/core"
)

// printUsage 打印自定义的帮助信息
func printUsage() {
	fmt.Fprintf(os.Stderr, `
╔════════════════════════════════════════════════════════════════╗
║            GogoSpider v3.3 - 智能Web爬虫工具                   ║
║                   简洁命令行指南                               ║
╚════════════════════════════════════════════════════════════════╝

📖 使用方法:
  spider [选项]
  spider -config <配置文件>          # 推荐：使用配置文件

═══════════════════════════════════════════════════════════════

🎯 核心参数（必选其一）:

  -url string
        目标URL（单URL扫描模式）
  
  -batch-file string
        批量URL文件（批量扫描模式，每行一个URL）
        支持配置文件: -batch-file targets.txt -config my_config.json
  
  -config string
        配置文件路径（推荐使用，包含所有详细配置）
        示例: spider -config config.json

  -version
        显示版本信息

═══════════════════════════════════════════════════════════════

⚙️  常用参数（可选，会覆盖配置文件）:

  -depth int
        最大爬取深度 (默认: 3)
  
  -proxy string
        代理服务器 (如: http://127.0.0.1:8080)
  
  -log-level string
        日志级别: debug/info/warn/error (默认: info)

═══════════════════════════════════════════════════════════════

📋 更多配置请使用配置文件:

  🔹 Cookie认证      → anti_detection_settings.cookie_file
  🔹 HTTPS证书      → anti_detection_settings.insecure_skip_verify
  🔹 静态文件过滤    → scope_settings.exclude_extensions
  🔹 黑名单设置      → blacklist_settings.domains
  🔹 速率控制        → rate_limit_settings
  🔹 敏感信息检测    → sensitive_detection_settings
  🔹 ...更多配置     → 查看 config.json

💡 提示: 配置文件更强大、更易维护！

═══════════════════════════════════════════════════════════════

🚀 快速开始:

  1️⃣  最简单的使用（单URL）:
     spider -url https://example.com

  2️⃣  使用配置文件（推荐）:
     spider -config config.json

  3️⃣  批量扫描（支持配置文件）:
     spider -batch-file targets.txt -config my_config.json

  4️⃣  带Cookie认证（配置文件中设置）:
     # 在配置文件中添加:
     # "cookie_file": "cookies.json"
     spider -config config_with_cookie.json

  5️⃣  忽略HTTPS证书错误（配置文件中设置）:
     # 在配置文件中添加:
     # "insecure_skip_verify": true
     spider -config config_insecure.json

═══════════════════════════════════════════════════════════════

📚 详细文档:

  📄 配置文件示例:  config.json（开箱即用）
  📄 配置指南:      CONFIG_GUIDE.md
  📄 快速迁移:      快速迁移指南_v3.3.md
  📄 更新日志:      CHANGELOG_v3.3.md
  📄 项目主页:      https://github.com/Warren-Jace/gogospider

═══════════════════════════════════════════════════════════════

💬 核心理念:
  
  ✅ 命令行 = 快速简单
  ✅ 配置文件 = 完整强大
  ✅ 二者结合 = 灵活高效

  推荐做法: 为不同场景准备不同的配置文件！

═══════════════════════════════════════════════════════════════

`)
}

var (
	targetURL       string
	mode            string
	maxDepth        int
	maxPages        int
	timeout         int
	workers         int
	// ✅ 修复2: cookieFile变量已移除,改用配置文件
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
	
	// 🆕 v2.10: 敏感信息检测参数
	enableSensitiveDetection bool
	sensitiveScanBody        bool
	sensitiveScanHeaders     bool
	sensitiveMinSeverity     string
	sensitiveOutputFile      string
	sensitiveRealTime        bool
	sensitiveRulesFile       string // 外部规则文件
	
	// 🆕 v2.11: 批量扫描参数
	batchFile               string // 批量URL文件
	batchConcurrency        int    // 批量扫描并发数
	
	// ✅ 修复2: cookieString变量已移除,改用配置文件
)

func init() {
	// 自定义帮助信息
	flag.Usage = printUsage
	
	flag.StringVar(&targetURL, "url", "", "目标URL（必需）")
	flag.StringVar(&mode, "mode", "smart", "爬取模式: static, dynamic, smart（默认）")
	flag.IntVar(&maxDepth, "depth", 3, "最大爬取深度")
	flag.IntVar(&maxPages, "max-pages", 100, "最大爬取页面数")
	flag.IntVar(&timeout, "timeout", 30, "请求超时时间（秒）")
	flag.IntVar(&workers, "workers", 10, "并发工作线程数")
	// ✅ 修复2: Cookie参数已移除,请在配置文件中配置 anti_detection_settings.cookie_file
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
	
	// 🆕 v2.10: 敏感信息检测参数
	flag.BoolVar(&enableSensitiveDetection, "sensitive-detect", true, "启用敏感信息检测（默认开启）")
	flag.BoolVar(&sensitiveScanBody, "sensitive-scan-body", true, "扫描响应体中的敏感信息")
	flag.BoolVar(&sensitiveScanHeaders, "sensitive-scan-headers", true, "扫描响应头中的敏感信息")
	flag.StringVar(&sensitiveMinSeverity, "sensitive-min-severity", "LOW", "最低严重级别: LOW, MEDIUM, HIGH")
	flag.StringVar(&sensitiveOutputFile, "sensitive-output", "", "敏感信息输出文件路径")
	flag.BoolVar(&sensitiveRealTime, "sensitive-realtime", true, "实时输出敏感信息发现")
	flag.StringVar(&sensitiveRulesFile, "sensitive-rules", "", "外部敏感信息规则文件（JSON格式）")
	
	// 🆕 v2.11: 批量扫描参数
	flag.StringVar(&batchFile, "batch-file", "", "批量扫描URL列表文件（每行一个URL）")
	flag.IntVar(&batchConcurrency, "batch-concurrency", 5, "批量扫描并发数（默认5）")
	
	// ✅ 修复2: Cookie字符串参数已移除,请在配置文件中配置 anti_detection_settings.cookie_string
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
	
	// 🆕 v2.11: 处理批量扫描模式
	if batchFile != "" {
		handleBatchScanMode()
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
	
	// ✅ 修复1: 批量扫描和URL二选一的逻辑验证
	// 如果既没有配置URL也没有批量文件,报错
	if cfg.TargetURL == "" {
		fmt.Println("错误: 必须指定目标URL（-url）或使用批量扫描（-batch-file）")
		flag.Usage()
		os.Exit(1)
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
	
	// 🆕 v2.10: 敏感信息检测配置
	cfg.SensitiveDetectionSettings.Enabled = enableSensitiveDetection
	cfg.SensitiveDetectionSettings.ScanResponseBody = sensitiveScanBody
	cfg.SensitiveDetectionSettings.ScanResponseHeaders = sensitiveScanHeaders
	cfg.SensitiveDetectionSettings.MinSeverity = strings.ToUpper(sensitiveMinSeverity)
	cfg.SensitiveDetectionSettings.OutputFile = sensitiveOutputFile
	cfg.SensitiveDetectionSettings.RealTimeOutput = sensitiveRealTime

	// 参数验证已在上方完成（批量扫描和URL二选一）
	
	// 配置验证（优化：确保配置有效）
	if err := cfg.Validate(); err != nil {
		fmt.Printf("配置验证失败: %v\n", err)
		os.Exit(1)
	}

	// 创建爬虫实例
	spider := core.NewSpider(cfg)
	defer spider.Close() // 确保资源清理
	
	// ✅ 修复2: 从配置文件加载Cookie
	if cfg.AntiDetectionSettings.CookieFile != "" {
		fmt.Printf("⏳ 正在加载Cookie文件: %s\n", cfg.AntiDetectionSettings.CookieFile)
		if err := spider.LoadCookieFromFile(cfg.AntiDetectionSettings.CookieFile); err != nil {
			fmt.Printf("⚠️  警告: 加载Cookie文件失败: %v\n", err)
		} else {
			cookieManager := spider.GetCookieManager()
			if cookieManager != nil {
				cookieManager.PrintSummary()
			}
		}
	}
	
	if cfg.AntiDetectionSettings.CookieString != "" {
		fmt.Printf("⏳ 正在加载Cookie字符串...\n")
		if err := spider.LoadCookieFromString(cfg.AntiDetectionSettings.CookieString); err != nil {
			fmt.Printf("⚠️  警告: 加载Cookie字符串失败: %v\n", err)
		} else {
			fmt.Printf("✅ Cookie字符串加载成功\n")
		}
	}
	
	// 🆕 v2.11: 加载敏感信息规则文件
	if enableSensitiveDetection {
		// 确定要加载的规则文件路径
		rulesFile := sensitiveRulesFile
		if rulesFile == "" {
			// 如果用户没有指定，使用配置中的默认规则文件
			rulesFile = cfg.SensitiveDetectionSettings.RulesFile
		}
		
		// 如果有规则文件路径，尝试加载
		if rulesFile != "" {
			if err := spider.MergeSensitiveRules(rulesFile); err != nil {
				fmt.Printf("⚠️  警告: 加载敏感规则失败: %v\n", err)
				fmt.Printf("💡 提示: 请使用 -sensitive-rules 参数指定规则文件，或确保默认文件存在\n")
				fmt.Printf("    推荐: -sensitive-rules sensitive_rules_standard.json\n")
			} else {
				fmt.Printf("✅ 已加载敏感信息规则文件: %s\n", rulesFile)
			}
		} else {
			fmt.Printf("⚠️  警告: 敏感信息检测已启用，但未指定规则文件\n")
			fmt.Printf("💡 请使用 -sensitive-rules 参数指定规则文件\n")
			fmt.Printf("    示例: -sensitive-rules sensitive_rules_standard.json\n")
		}
	}

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
	
	// 🆕 v2.11: 保存敏感信息到独立文件
	if enableSensitiveDetection {
		// 保存文本格式
		sensitiveFile := baseFilename + "_sensitive.txt"
		if err := spider.SaveSensitiveInfoToFile(sensitiveFile); err != nil {
			log.Printf("保存敏感信息失败: %v", err)
		}
		
		// 保存JSON格式（如果指定了输出文件）
		if sensitiveOutputFile != "" {
			if err := spider.SaveSensitiveInfoToJSON(sensitiveOutputFile); err != nil {
				log.Printf("保存敏感信息JSON失败: %v", err)
			}
		} else {
			// 默认也保存JSON格式
			sensitiveJSONFile := baseFilename + "_sensitive.json"
			if err := spider.SaveSensitiveInfoToJSON(sensitiveJSONFile); err != nil {
				log.Printf("保存敏感信息JSON失败: %v", err)
			}
		}
	}
	
	// 打印统计信息
	if !simpleMode {
		printStats(results, elapsed)
		
		// 🆕 v3.2: 打印重定向检测报告
		spider.PrintRedirectReport()
		
		// 🆕 v3.2: 打印登录墙检测报告
		spider.PrintLoginWallReport()
		
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
║           GogoSpider - 智能Web爬虫系统                       ║
║     Version 3.4 - Hybrid Strategy with Adaptive Learning     ║
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
	fmt.Println("GogoSpider v3.4 - Hybrid Strategy with Adaptive Learning")
	fmt.Println("Build: 2025-10-26")
	fmt.Println("Go Version: " + strings.TrimPrefix(filepath.Base(os.Args[0]), "go"))
	fmt.Println("")
	fmt.Println("✨ v3.4 核心创新:")
	fmt.Println("  ✓ 混合调度策略 - BFS+优先级+自适应学习（业界首创）")
	fmt.Println("  ✓ 自适应学习 - 越爬越聪明，动态调整优先级权重")
	fmt.Println("  ✓ 6维优先级权重 - 可根据场景精细调整")
	fmt.Println("  ✓ 配置文件统一 - 从3个简化为1个，配置项50+")
	fmt.Println("  ✓ 性能提升20% - API发现率95%+，高价值URL发现+40%")
	fmt.Println("  ✓ 完全向下兼容 - 旧配置无需修改")
	fmt.Println("")
	fmt.Println("✨ v3.3 核心改进（继承）:")
	fmt.Println("  ✓ 配置简化 - Cookie/证书统一在配置文件")
	fmt.Println("  ✓ 批量扫描 - 支持配置文件")
	fmt.Println("  ✓ 静态资源智能过滤 - 只记录不请求(70%效率提升)")
	fmt.Println("")
	fmt.Println("🎯 核心功能:")
	fmt.Println("  ✓ 静态+动态双引擎爬虫")
	fmt.Println("  ✓ AJAX请求拦截")
	fmt.Println("  ✓ JavaScript深度分析")
	fmt.Println("  ✓ 跨域JS分析（60+CDN）")
	fmt.Println("  ✓ 智能表单识别")
	fmt.Println("  ✓ URL模式去重")
	fmt.Println("  ✓ 业务感知过滤")
	fmt.Println("  ✓ DOM相似度检测")
	fmt.Println("  ✓ 技术栈检测")
	fmt.Println("  ✓ 敏感信息检测")
	fmt.Println("  ✓ 结构化日志系统")
	fmt.Println("  ✓ Pipeline支持")
	fmt.Println("")
	fmt.Println("💡 理念: 命令行快速简单，配置文件完整强大")
	fmt.Println("📚 文档: spider --help 或查看 使用指南_v3.3.md")
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
	
	// ✅ 修复: 不在这里验证，等命令行参数应用后再验证
	// 因为target_url可能通过-url参数提供
	
	return &cfg, nil
}

// handleBatchScanMode 处理批量扫描模式（v2.11 新增）
func handleBatchScanMode() {
	fmt.Printf("\n╔════════════════════════════════════════════════╗\n")
	fmt.Printf("║     GogoSpider - 批量扫描模式               ║\n")
	fmt.Printf("╚════════════════════════════════════════════════╝\n\n")
	
	// ✅ 优化1: 批量模式支持配置文件
	var baseCfg *config.Config
	if configFile != "" {
		loadedCfg, err := loadConfigFile(configFile)
		if err != nil {
			log.Fatalf("加载配置文件失败: %v", err)
		}
		baseCfg = loadedCfg
		fmt.Printf("[*] 已加载配置文件: %s\n", configFile)
	} else {
		baseCfg = config.NewDefaultConfig()
	}
	
	// 读取URL列表文件
	file, err := os.Open(batchFile)
	if err != nil {
		log.Fatalf("打开URL列表文件失败: %v", err)
	}
	defer file.Close()
	
	// 读取所有URL
	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url == "" || strings.HasPrefix(url, "#") {
			continue // 跳过空行和注释行
		}
		urls = append(urls, url)
	}
	
	if err := scanner.Err(); err != nil {
		log.Fatalf("读取URL列表失败: %v", err)
	}
	
	if len(urls) == 0 {
		log.Fatalf("URL列表为空")
	}
	
	fmt.Printf("[批量扫描] 共读取 %d 个URL，并发数: %d\n\n", len(urls), batchConcurrency)
	
	// 创建并发控制
	sem := make(chan struct{}, batchConcurrency)
	var wg sync.WaitGroup
	var successCount, failCount int
	var mu sync.Mutex
	
	startTime := time.Now()
	
	// 遍历每个URL进行扫描
	for i, url := range urls {
		wg.Add(1)
		go func(index int, targetURL string) {
			defer wg.Done()
			
			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()
			
			fmt.Printf("\n[%d/%d] 开始扫描: %s\n", index+1, len(urls), targetURL)
			
			// ✅ 优化1: 使用基础配置的副本，避免并发问题
			cfg := *baseCfg // 复制配置
			cfg.TargetURL = targetURL
			
			// ✅ 优化1: 命令行参数覆盖配置文件(如果指定)
			if maxDepth != 3 {
				cfg.DepthSettings.MaxDepth = maxDepth
			}
			if proxy != "" {
				cfg.AntiDetectionSettings.Proxies = []string{proxy}
			}
			if userAgent != "" {
				cfg.AntiDetectionSettings.UserAgents = []string{userAgent}
			}
			if logLevel != "info" {
				cfg.LogSettings.Level = strings.ToUpper(logLevel)
			}
			
			// 批量模式特殊配置
			cfg.SensitiveDetectionSettings.RealTimeOutput = false // 批量模式下关闭实时输出
			
			// 配置验证
			if err := cfg.Validate(); err != nil {
				fmt.Printf("  ❌ 配置验证失败: %v\n", err)
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}
			
			// 创建爬虫实例
			spider := core.NewSpider(&cfg)
			defer spider.Close()
			
			// ✅ 优化1: 加载Cookie(如果配置文件中指定)
			if cfg.AntiDetectionSettings.CookieFile != "" {
				if err := spider.LoadCookieFromFile(cfg.AntiDetectionSettings.CookieFile); err != nil {
					fmt.Printf("  ⚠️  警告: 加载Cookie文件失败: %v\n", err)
				}
			}
			if cfg.AntiDetectionSettings.CookieString != "" {
				if err := spider.LoadCookieFromString(cfg.AntiDetectionSettings.CookieString); err != nil {
					fmt.Printf("  ⚠️  警告: 加载Cookie字符串失败: %v\n", err)
				}
			}
			
			// 加载敏感信息规则文件
			if cfg.SensitiveDetectionSettings.Enabled {
				rulesFile := cfg.SensitiveDetectionSettings.RulesFile
				if rulesFile != "" {
					if err := spider.MergeSensitiveRules(rulesFile); err != nil {
						fmt.Printf("  ⚠️  警告: 加载敏感规则失败: %v\n", err)
					}
				}
			}
			
			// 执行爬取
			err := spider.Start(targetURL)
			if err != nil {
				fmt.Printf("  ❌ 爬取失败: %v\n", err)
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}
			
			// 获取结果
			results := spider.GetResults()
			
			// 生成输出文件名
			timestamp := time.Now().Format("20060102_150405")
			domain := extractDomain(targetURL)
			baseFilename := fmt.Sprintf("batch_%s_%s", domain, timestamp)
			
			// 保存结果
			if err := saveResults(results, baseFilename+".txt"); err != nil {
				fmt.Printf("  警告: 保存结果失败: %v\n", err)
			}
			
			// 保存URL列表
			if err := saveAllURLs(results, baseFilename); err != nil {
				fmt.Printf("  警告: 保存URL失败: %v\n", err)
			}
			
			// 保存敏感信息
			if enableSensitiveDetection {
				sensitiveFile := baseFilename + "_sensitive.txt"
				if err := spider.SaveSensitiveInfoToFile(sensitiveFile); err != nil {
					fmt.Printf("  警告: 保存敏感信息失败: %v\n", err)
				}
				
				sensitiveJSONFile := baseFilename + "_sensitive.json"
				if err := spider.SaveSensitiveInfoToJSON(sensitiveJSONFile); err != nil {
					fmt.Printf("  警告: 保存敏感信息JSON失败: %v\n", err)
				}
			}
			
			// 统计
			linkCount := 0
			for _, r := range results {
				linkCount += len(r.Links)
			}
			
			fmt.Printf("  ✅ 完成: 爬取了 %d 个页面，发现 %d 个链接\n", len(results), linkCount)
			
			mu.Lock()
			successCount++
			mu.Unlock()
			
		}(i, url)
	}
	
	// 等待所有任务完成
	wg.Wait()
	
	elapsed := time.Since(startTime)
	
	// 打印总结
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("  批量扫描完成！\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("  总URL数: %d\n", len(urls))
	fmt.Printf("  成功: %d\n", successCount)
	fmt.Printf("  失败: %d\n", failCount)
	fmt.Printf("  耗时: %.2f秒\n", elapsed.Seconds())
	fmt.Printf("  平均速度: %.2f URL/秒\n", float64(len(urls))/elapsed.Seconds())
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Printf("[+] 所有结果已保存到当前目录（batch_*）\n")
}
