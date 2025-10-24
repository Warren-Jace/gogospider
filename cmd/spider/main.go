package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
}

func main() {
	flag.Parse()

	// 显示横幅
	printBanner()

	// 加载配置
	cfg := config.NewDefaultConfig()

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
	if enableFuzzing {
		cfg.StrategySettings.EnableParamFuzzing = true
		cfg.StrategySettings.EnablePOSTParamFuzzing = true
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
	fmt.Printf("[*] 参数爆破: %v\n", cfg.StrategySettings.EnableParamFuzzing)
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

	// 保存URL列表
	if err := saveURLs(results, baseFilename+"_urls.txt"); err != nil {
		log.Printf("保存URL列表失败: %v", err)
	}

	// 打印统计信息
	printStats(results, elapsed)

	fmt.Printf("\n[+] 结果已保存到当前目录\n")
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
║                     Version 2.5                               ║
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
	for _, result := range results {
		if !urlSet[result.URL] {
			file.WriteString(result.URL + "\n")
			urlSet[result.URL] = true
		}
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

