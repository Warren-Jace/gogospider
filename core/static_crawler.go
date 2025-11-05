package core

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"spider-golang/config"
)

// StaticCrawlerImpl 静态爬虫实现
type StaticCrawlerImpl struct {
	collector        *colly.Collector
	config           *config.Config
	resultChan       chan<- Result
	stopChan         chan struct{}
	duplicateHandler *DuplicateHandler
	paramHandler     *ParamHandler
	cookieManager    *CookieManager       // Cookie管理器（v3.2新增）
	redirectManager  *RedirectManager     // 重定向管理器（v3.2新增）
	urlValidator     URLValidatorInterface // 🔧 修复：改为接口类型，支持v2.0验证器
	spider           SpiderRecorder       // Spider引用（v3.7新增，用于实时记录URL）
	urlNormalizer    *URLNormalizer       // 🆕 v4.0：URL规范化处理器
	urlQualityFilter *URLQualityFilter    // 🆕 v4.0：URL质量过滤器
}


// NewStaticCrawler 创建新的静态爬虫实例
func NewStaticCrawler(config *config.Config, resultChan chan<- Result, stopChan chan struct{}) StaticCrawler {
	c := colly.NewCollector(
		colly.MaxDepth(config.DepthSettings.MaxDepth),
		colly.Async(true),
	)
	
	// 设置并发限制
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5, // 增加并发数
		Delay:       time.Duration(500) * time.Millisecond, // 减少延迟
	})
	
	// 创建去重处理器
	duplicateHandler := NewDuplicateHandler(0.9) // 使用默认相似度阈值
	
	// 创建参数处理器
	paramHandler := NewParamHandler()
	
	return &StaticCrawlerImpl{
		collector:        c,
		config:           config,
		resultChan:       resultChan,
		stopChan:         stopChan,
		duplicateHandler: duplicateHandler,
		paramHandler:     paramHandler,
		urlValidator:     NewSmartURLValidatorCompat(), // 🔧 修复：使用v2.0智能验证器
		urlQualityFilter: NewURLQualityFilter(),        // 🆕 v4.0：URL质量过滤器
	}
}

// Configure 配置爬虫
func (s *StaticCrawlerImpl) Configure(config *config.Config) {
	s.config = config
	
	// 更新并发限制
	s.collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5, // 增加并发数
		Delay:       time.Duration(500) * time.Millisecond, // 减少延迟
	})
}

// SetCookieManager 设置Cookie管理器（v3.2新增）
func (s *StaticCrawlerImpl) SetCookieManager(cm *CookieManager) {
	s.cookieManager = cm
}

// SetRedirectManager 设置重定向管理器（v3.2新增）
func (s *StaticCrawlerImpl) SetRedirectManager(rm *RedirectManager) {
	s.redirectManager = rm
}

// SetSpider 设置Spider引用（v3.7新增，实现Crawler接口）
func (s *StaticCrawlerImpl) SetSpider(spider SpiderRecorder) {
	s.spider = spider
	
	// 🆕 v4.5: 使用Spider的共享去重器（修复多实例问题）
	if spider != nil {
		sharedDedup := spider.GetDuplicateHandler()
		if sharedDedup != nil {
			s.duplicateHandler = sharedDedup
			fmt.Printf("🔧 [静态爬虫] 使用共享去重器 (地址: %p)\n", sharedDedup)
		}
	}
}

// Crawl 执行爬取
func (s *StaticCrawlerImpl) Crawl(startURL *url.URL) (*Result, error) {
	result := &Result{
		URL:          startURL.String(),
		Links:        make([]string, 0),
		Assets:       make([]string, 0),
		Forms:        make([]Form, 0),
		APIs:         make([]string, 0),
		POSTRequests: make([]POSTRequest, 0),
	}
	
	// 为每次Crawl创建新的collector实例，避免WaitGroup重用问题
	collector := colly.NewCollector(
		colly.MaxDepth(s.config.DepthSettings.MaxDepth),
		colly.Async(true),
	)
	
	// ✅ 修复5: 配置HTTPS证书验证
	if s.config.AntiDetectionSettings.InsecureSkipVerify {
		collector.WithTransport(&http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		})
	}
	
	// 设置并发限制
	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		Delay:       time.Duration(500) * time.Millisecond,
	})
	
	// 设置请求前回调，实现User-Agent轮换、域名范围检查和Cookie应用
	collector.OnRequest(func(r *colly.Request) {
		// 🆕 v4.5: 在Colly层面阻止重复请求（关键修复！）
		// 必须在OnRequest最开始就检查，避免发起HTTP请求
		if s.duplicateHandler != nil && s.duplicateHandler.IsDuplicateURL(r.URL.String()) {
			// URL已经被访问过，中止本次请求
			r.Abort()
			return
		}
		
		// 检查域名范围限制
		if s.config.StrategySettings.DomainScope != "" {
			requestURL, err := url.Parse(r.URL.String())
			if err != nil {
				fmt.Printf("解析URL失败 %s: %v\n", r.URL.String(), err)
				r.Abort()
				return
			}
			
			// 检查是否在允许的域名范围内
			if !strings.Contains(requestURL.Host, s.config.StrategySettings.DomainScope) {
				fmt.Printf("URL超出域名范围，已记录但不爬取: %s\n", r.URL.String())
				// 记录外部链接但不发送请求
				r.Abort()
				return
			}
		}
		
		// 如果配置了User-Agent列表，则随机选择一个
		if len(s.config.AntiDetectionSettings.UserAgents) > 0 {
			// 简单随机选择User-Agent
			rand.Seed(time.Now().UnixNano())
			randIndex := rand.Intn(len(s.config.AntiDetectionSettings.UserAgents))
			userAgent := s.config.AntiDetectionSettings.UserAgents[randIndex]
			r.Headers.Set("User-Agent", userAgent)
		}
		
		// 🆕 v3.2: 应用Cookie（如果已加载）
		if s.cookieManager != nil && s.cookieManager.GetCookieCount() > 0 {
			cookieHeader := s.cookieManager.GetCookieHeader()
			if cookieHeader != "" {
				r.Headers.Set("Cookie", cookieHeader)
			}
		}
		
	// 🆕 v4.4: 记录请求日志和开始时间
	if s.spider != nil {
		logger := s.spider.GetRequestLogger()
		if logger != nil && logger.IsEnabled() {
			// 转换Headers
			headers := make(map[string]string)
			for key, values := range *r.Headers {
				if len(values) > 0 {
					headers[key] = values[0]
				}
			}
			logger.LogRequest("GET", r.URL.String(), headers, "")
			
			// 记录请求开始时间（存储在请求上下文中）
			r.Ctx.Put("request_start_time", time.Now())
		}
	}
	})
	
	// 设置HTML回调 - 提取所有可能包含URL的元素
	// 1. 提取 <a href> 链接
	linkCount := 0
	validCount := 0
	duplicateCount := 0
	invalidCount := 0
	
	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		linkCount++
		
		// 🆕 v3.7: 检查特殊协议链接并记录
		if strings.HasPrefix(link, "mailto:") {
			if s.spider != nil {
				s.spider.RecordSpecialLink(link, "mailto")
			}
			invalidCount++
			return
		}
		if strings.HasPrefix(link, "tel:") {
			if s.spider != nil {
				s.spider.RecordSpecialLink(link, "tel")
			}
			invalidCount++
			return
		}
		if strings.HasPrefix(link, "ftp://") {
			if s.spider != nil {
				s.spider.RecordSpecialLink(link, "ftp")
			}
			invalidCount++
			return
		}
		if strings.HasPrefix(link, "ws://") || strings.HasPrefix(link, "wss://") {
			if s.spider != nil {
				protocol := "ws"
				if strings.HasPrefix(link, "wss://") {
					protocol = "wss"
				}
				s.spider.RecordSpecialLink(link, protocol)
			}
			invalidCount++
			return
		}
		if strings.HasPrefix(link, "data:") {
			if s.spider != nil {
				s.spider.RecordSpecialLink(link, "data")
			}
			invalidCount++
			return
		}
		
		// 特殊处理：如果是javascript:协议，提取其中的URL
		if strings.HasPrefix(link, "javascript:") {
			// 简单直接的提取：从javascript:函数调用中提取参数
			// 例如：javascript:loadSomething('artists.php'); → artists.php
			
			// 匹配 函数名('参数')
			funcCallPattern := regexp.MustCompile(`\w+\s*\(\s*['"]([^'"]+)['"]`)
			matches := funcCallPattern.FindAllStringSubmatch(link, -1)
			
			foundAny := false
			for _, match := range matches {
				if len(match) > 1 {
					extractedURL := match[1]
					// 转换为绝对URL
					absURL := e.Request.AbsoluteURL(extractedURL)
					// ✅ v4.1：使用过滤器
					if absURL != "" && s.addLinkWithFilter(result, extractedURL, absURL) {
						validCount++
						foundAny = true
						fmt.Printf("    [JS提取] 从javascript:协议提取URL: %s → %s\n", extractedURL, absURL)
					}
				}
			}
			
			if !foundAny {
				invalidCount++
			}
			return
		}
		
		// 检查URL有效性
		if !IsValidURL(link) {
			invalidCount++
			return
		}
		
		absoluteURL := e.Request.AbsoluteURL(link)
		if absoluteURL == "" {
			invalidCount++
			return
		}
		
		// ✅ v4.0: 使用统一的过滤函数添加链接
		if s.addLinkWithFilter(result, link, absoluteURL) {
			validCount++
			
			// ✅ v4.0: 协议相对URL处理 - 生成http和https两个版本
			if strings.HasPrefix(link, "//") {
				normalizedURLs := s.normalizeURLWithProtocolVariants(link, e.Request.URL)
				for _, nURL := range normalizedURLs {
					if nURL != absoluteURL {
						// 添加协议变体（也会经过过滤）
						if s.addLinkWithFilter(result, link, nURL) {
							validCount++
						}
					}
				}
			}
		} else {
			invalidCount++
			return
		}
		
		// 🆕 v3.7: 使用资源分类器判断URL类型
		if s.spider != nil {
			classifier := s.spider.GetResourceClassifier()
			if classifier != nil {
				resourceType, shouldRequest := classifier.ClassifyURL(absoluteURL)
				
				if !shouldRequest {
					// 静态资源：记录到静态资源列表（但URL已在Links中）
					s.spider.RecordStaticResource(absoluteURL, resourceType)
					invalidCount++
					// 这里不return，继续标记validCount，因为URL已成功记录
				}
				// 需要请求的资源继续处理
			}
		}
		
		validCount++
	})
	
	// 添加详细调试日志
	collector.OnScraped(func(r *colly.Response) {
		fmt.Printf("\n[静态爬虫] 页面爬取完成: %s\n", r.Request.URL)
		fmt.Printf("[静态爬虫] 发现 %d 个<a>标签\n", linkCount)
		fmt.Printf("[静态爬虫] 有效链接: %d个 | 重复过滤: %d个 | 无效链接: %d个\n", 
			validCount, duplicateCount, invalidCount)
		fmt.Printf("[静态爬虫] 最终收集: %d 个链接\n\n", len(result.Links))
	})
	
	// 2. 提取 <form action> 表单提交地址
	collector.OnHTML("form[action]", func(e *colly.HTMLElement) {
		action := e.Attr("action")
		if action != "" && !strings.HasPrefix(action, "javascript:") {
			absoluteURL := e.Request.AbsoluteURL(action)
			if absoluteURL != "" {
				// ✅ v4.0：应用质量过滤
				if s.urlQualityFilter == nil || func() bool {
					valid, _ := s.urlQualityFilter.IsHighQualityURL(absoluteURL)
					return valid
				}() {
                    _ = s.addLinkWithFilter(result, action, absoluteURL)
				}
			}
		}
	})
	
	// 3. 提取 <iframe src> 框架地址
	collector.OnHTML("iframe[src]", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		if src != "" && !strings.HasPrefix(src, "javascript:") {
			absoluteURL := e.Request.AbsoluteURL(src)
			if absoluteURL != "" {
				// ✅ v4.0：应用质量过滤
				if s.urlQualityFilter == nil || func() bool {
					valid, _ := s.urlQualityFilter.IsHighQualityURL(absoluteURL)
					return valid
				}() {
                    _ = s.addLinkWithFilter(result, src, absoluteURL)
				}
			}
		}
	})
	
	// 4. 提取 <frame src> 框架地址
	collector.OnHTML("frame[src]", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		if src != "" && !strings.HasPrefix(src, "javascript:") {
			absoluteURL := e.Request.AbsoluteURL(src)
			// ✅ v4.1：应用质量过滤
			if absoluteURL != "" {
				s.addLinkWithFilter(result, src, absoluteURL)
			}
		}
	})
	
	// 5. 提取 <embed src> 嵌入资源
	collector.OnHTML("embed[src]", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		if src != "" {
			absoluteURL := e.Request.AbsoluteURL(src)
			// ✅ v4.1：应用质量过滤
			if absoluteURL != "" {
				s.addLinkWithFilter(result, src, absoluteURL)
			}
		}
	})
	
	// 6. 提取 <object data> 对象数据
	collector.OnHTML("object[data]", func(e *colly.HTMLElement) {
		data := e.Attr("data")
		if data != "" {
			absoluteURL := e.Request.AbsoluteURL(data)
			// ✅ v4.1：应用质量过滤
			if absoluteURL != "" {
				s.addLinkWithFilter(result, data, absoluteURL)
			}
		}
	})
	
	// 7. 提取 <meta http-equiv="refresh"> 重定向
	collector.OnHTML("meta[http-equiv='refresh']", func(e *colly.HTMLElement) {
		content := e.Attr("content")
		if content != "" {
			// 解析格式: "0;URL='http://example.com'" 或 "0;url=http://example.com"
			parts := strings.Split(content, ";")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(strings.ToLower(part), "url=") {
					urlStr := strings.TrimPrefix(strings.ToLower(part), "url=")
					urlStr = strings.Trim(urlStr, "'\"")
					absoluteURL := e.Request.AbsoluteURL(urlStr)
					// ✅ v4.1：应用质量过滤
					if absoluteURL != "" {
						s.addLinkWithFilter(result, urlStr, absoluteURL)
					}
					break
				}
			}
		}
	})
	
	// 8. 提取 <area href> 图像映射区域
	collector.OnHTML("area[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		if href != "" && !strings.HasPrefix(href, "javascript:") {
			absoluteURL := e.Request.AbsoluteURL(href)
			// ✅ v4.1：应用质量过滤
			if absoluteURL != "" {
				s.addLinkWithFilter(result, href, absoluteURL)
			}
		}
	})
	
	// 9. 提取 <base href> 基础URL（影响相对路径解析）
	collector.OnHTML("base[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		if href != "" {
			absoluteURL := e.Request.AbsoluteURL(href)
			// ✅ v4.1：应用质量过滤
			if absoluteURL != "" {
				s.addLinkWithFilter(result, href, absoluteURL)
			}
		}
	})
	
	// 10. 提取 data-* 属性中的URL（常见于SPA应用）
	collector.OnHTML("[data-url], [data-href], [data-src], [data-link], [data-ajax], [data-target]", func(e *colly.HTMLElement) {
		for _, attr := range []string{"data-url", "data-href", "data-src", "data-link", "data-ajax", "data-target"} {
			if val := e.Attr(attr); val != "" && !strings.HasPrefix(val, "javascript:") && !strings.HasPrefix(val, "#") {
				if strings.HasPrefix(val, "http") || strings.HasPrefix(val, "/") {
					absoluteURL := e.Request.AbsoluteURL(val)
					// ✅ v4.1：应用质量过滤
					if absoluteURL != "" {
						s.addLinkWithFilter(result, val, absoluteURL)
					}
				}
			}
		}
	})
	
	// 11. 提取 onclick/onmouseover 等事件处理器中的URL（新增）
	collector.OnHTML("[onclick], [onmouseover], [onmousedown], [ondblclick]", func(e *colly.HTMLElement) {
		for _, eventAttr := range []string{"onclick", "onmouseover", "onmousedown", "ondblclick"} {
			if eventCode := e.Attr(eventAttr); eventCode != "" {
				// 从事件代码中提取URL（已包含质量过滤）
				urls := s.extractURLsFromJSCode(eventCode)
				for _, url := range urls {
					absoluteURL := e.Request.AbsoluteURL(url)
					// ✅ v4.1：应用质量过滤（extractURLsFromJSCode已经过滤，这里再次确认）
					if absoluteURL != "" {
						s.addLinkWithFilter(result, url, absoluteURL)
					}
				}
			}
		}
	})
	
	// 12. 提取所有<button>和带role="button"的元素（新增）
	collector.OnHTML("button, [role='button']", func(e *colly.HTMLElement) {
		// 检查data属性
		for _, attr := range []string{"data-url", "data-href", "data-target", "data-action"} {
			if val := e.Attr(attr); val != "" && !strings.HasPrefix(val, "#") {
				if strings.HasPrefix(val, "http") || strings.HasPrefix(val, "/") {
					absoluteURL := e.Request.AbsoluteURL(val)
					// ✅ v4.1：应用质量过滤
					if absoluteURL != "" {
						s.addLinkWithFilter(result, val, absoluteURL)
					}
				}
			}
		}
	})
	
	// 🆕 v3.7: 设置资源回调（增强版：实时分类和记录）
	collector.OnHTML("link[href], script[src], img[src]", func(e *colly.HTMLElement) {
		var assetURL string
		if e.Name == "link" {
			assetURL = e.Attr("href")
		} else {
			assetURL = e.Attr("src")
		}
		absoluteURL := e.Request.AbsoluteURL(assetURL)
		if absoluteURL != "" && !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
			// 🆕 v3.7: 使用资源分类器判断并记录
			if s.spider != nil {
				classifier := s.spider.GetResourceClassifier()
				if classifier != nil {
					resourceType, shouldRequest := classifier.ClassifyURL(absoluteURL)
					if !shouldRequest {
						// 静态资源：只记录不请求
						s.spider.RecordStaticResource(absoluteURL, resourceType)
					}
				}
			}
			result.Assets = append(result.Assets, absoluteURL)
		}
	})
	
	// 🆕 提取 srcset 属性（响应式图片）- 新功能
	collector.OnHTML("img[srcset], source[srcset]", func(e *colly.HTMLElement) {
		srcset := e.Attr("srcset")
		if srcset == "" {
			return
		}
		
		// 解析srcset格式: "url1 320w, url2 640w, url3 1024w"
		// 或: "url1 1x, url2 2x, url3 3x"
		parts := strings.Split(srcset, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			
			// 提取URL（第一个空格前的部分）
			fields := strings.Fields(part)
			if len(fields) > 0 {
				urlStr := fields[0]
				absoluteURL := e.Request.AbsoluteURL(urlStr)
				if absoluteURL != "" && !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
					result.Assets = append(result.Assets, absoluteURL)
				}
			}
		}
	})
	
	// 🆕 提取 picture 标签内的所有源 - 新功能
	collector.OnHTML("picture", func(e *colly.HTMLElement) {
		// 提取 source 标签
		e.ForEach("source[srcset]", func(_ int, source *colly.HTMLElement) {
			srcset := source.Attr("srcset")
			if srcset != "" {
				parts := strings.Split(srcset, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					fields := strings.Fields(part)
					if len(fields) > 0 {
						urlStr := fields[0]
						absoluteURL := e.Request.AbsoluteURL(urlStr)
						if absoluteURL != "" && !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
							result.Assets = append(result.Assets, absoluteURL)
						}
					}
				}
			}
		})
		
		// 提取 img 标签（fallback）
		e.ForEach("img[src]", func(_ int, img *colly.HTMLElement) {
			src := img.Attr("src")
			if src != "" {
				absoluteURL := e.Request.AbsoluteURL(src)
				if absoluteURL != "" && !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
					result.Assets = append(result.Assets, absoluteURL)
				}
			}
		})
	})
	
	// 设置表单回调（增强版：捕获所有表单 + POST请求生成）
	collector.OnHTML("form", func(e *colly.HTMLElement) {
		action := e.Attr("action")
		method := strings.ToUpper(e.Attr("method"))
		enctype := e.Attr("enctype")
		if method == "" {
			method = "GET" // 默认为GET
		}
		if enctype == "" {
			enctype = "application/x-www-form-urlencoded"
		}
		
		// 如果action为空，使用当前页面URL
		if action == "" {
			action = e.Request.URL.String()
		} else {
			// 转换为绝对URL
			action = e.Request.AbsoluteURL(action)
		}
		
		// 收集表单字段
		fields := make([]FormField, 0)
		e.ForEach("input, select, textarea", func(_ int, el *colly.HTMLElement) {
			fieldName := el.Attr("name")
			if fieldName == "" {
				return // 跳过没有name的字段
			}
			
			field := FormField{
				Name:     fieldName,
				Type:     el.Attr("type"),
				Value:    el.Attr("value"),
				Required: el.Attr("required") != "",
			}
			
			// 如果type为空，根据标签设置默认type
			if field.Type == "" {
				switch el.Name {
				case "textarea":
					field.Type = "textarea"
				case "select":
					field.Type = "select"
				default:
					field.Type = "text"
				}
			}
			
			fields = append(fields, field)
		})
		
		formData := Form{
			Action: action,
			Method: method,
			Fields: fields,
		}
		
		// 所有表单都添加，不再检查是否有参数
		result.Forms = append(result.Forms, formData)
		
		// === 新增：生成POST请求数据 ===
		postReq := s.generatePOSTRequestFromForm(&formData, enctype)
		if postReq != nil {
			result.POSTRequests = append(result.POSTRequests, *postReq)
			
			// 打印POST请求信息
			if method == "POST" {
				fmt.Printf("  [静态爬虫] 发现POST表单: %s\n", action)
				fmt.Printf("    字段数: %d, 参数: %d\n", len(fields), len(postReq.Parameters))
			}
		}
		
        // 如果是带参数的action，也添加到链接列表（统一过滤）
        if strings.Contains(action, "?") {
            _ = s.addLinkWithFilter(result, action, action)
        }
	})
	
	// 设置API端点回调
	collector.OnHTML("script[src]", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		absoluteURL := e.Request.AbsoluteURL(src)
		if absoluteURL != "" && (strings.Contains(absoluteURL, "api") || strings.Contains(absoluteURL, "json")) {
			// 检查是否为重复的API端点
			if !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
				result.APIs = append(result.APIs, absoluteURL)
			}
		}
	})
	
	// 设置响应回调
	collector.OnResponse(func(r *colly.Response) {
		result.StatusCode = r.StatusCode
		result.ContentType = r.Headers.Get("Content-Type")
		
	// 🆕 v4.4: 记录响应日志（修复：计算实际响应时间）
	if s.spider != nil {
		logger := s.spider.GetRequestLogger()
		if logger != nil && logger.IsEnabled() {
			// 计算响应时间
			var responseTime int64 = 0
			if startTime := r.Request.Ctx.GetAny("request_start_time"); startTime != nil {
				if t, ok := startTime.(time.Time); ok {
					responseTime = time.Since(t).Milliseconds()
				}
			}
			logger.LogResponse(r.Request.URL.String(), r.StatusCode, responseTime, nil)
		}
	}
		
		// 🆕 v3.2: 检测重定向（通过响应码和Location头）
		if s.redirectManager != nil {
			if r.StatusCode >= 300 && r.StatusCode < 400 {
				locationHeader := r.Headers.Get("Location")
				if locationHeader != "" {
					redirectInfo := s.redirectManager.RecordRedirect(
						r.Request.URL.String(),
						locationHeader,
						r.StatusCode,
					)
					
					// 如果是认证重定向，记录并可能警告
					if redirectInfo.IsAuthRedirect {
						fmt.Printf("⚠️  [认证重定向] %s → %s\n", 
							redirectInfo.OriginalURL, redirectInfo.FinalURL)
					}
				}
			}
		}
		
		// 保存HTML内容和Headers供高级检测使用
		result.HTMLContent = string(r.Body)
		result.Headers = make(map[string]string)
		for key, values := range *r.Headers {
			if len(values) > 0 {
				result.Headers[key] = values[0]
			}
		}
		
        // === 优化1：提取响应头中的URL（统一过滤） ===
        headerURLs := s.extractURLsFromHeaders(r)
        for _, u := range headerURLs {
            _ = s.addLinkWithFilter(result, u, u)
        }
		
		// === 优化2：提取内联JavaScript中的URL ===
		if strings.Contains(result.ContentType, "text/html") {
			inlineURLs := s.extractURLsFromInlineScripts(string(r.Body), r.Request.URL.String())
			for _, u := range inlineURLs {
				absoluteURL := r.Request.AbsoluteURL(u)
				if absoluteURL != "" {
					// ✅ v4.0：应用质量过滤器
					if s.urlQualityFilter != nil {
						if valid, _ := s.urlQualityFilter.IsHighQualityURL(absoluteURL); !valid {
							continue
						}
					}
					
                    _ = s.addLinkWithFilter(result, u, absoluteURL)
				}
			}
			
			// === 优化3：提取CSS中的URL ===
			cssURLs := s.extractURLsFromCSS(string(r.Body))
			for _, u := range cssURLs {
				absoluteURL := r.Request.AbsoluteURL(u)
				if absoluteURL != "" && !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
					result.Assets = append(result.Assets, absoluteURL)
				}
			}
		}
		
		// 提取参数
		params, err := s.paramHandler.ExtractParams(r.Request.URL.String())
		if err != nil {
			return
		}
		
		// 如果有查询参数，生成变体URL
		if len(params) > 0 {
			variations := s.paramHandler.GenerateParamVariations(r.Request.URL.String())
			fmt.Printf("为URL %s 生成 %d 个参数变体\n", r.Request.URL.String(), len(variations))
			
			// 可以将变体URL添加到结果中或进一步爬取
			// 这里简化处理，只打印
			for _, variation := range variations {
				fmt.Printf("  变体: %s\n", variation)
			}
		}
	})
	
	// 设置错误回调
	collector.OnError(func(r *colly.Response, err error) {
		fmt.Printf("请求错误 %s: %v\n", r.Request.URL, err)
		
	// 🆕 v4.4: 记录错误日志（修复：计算实际响应时间）
	if s.spider != nil {
		logger := s.spider.GetRequestLogger()
		if logger != nil && logger.IsEnabled() {
			statusCode := 0
			if r != nil {
				statusCode = r.StatusCode
			}
			
			// 计算响应时间
			var responseTime int64 = 0
			if startTime := r.Request.Ctx.GetAny("request_start_time"); startTime != nil {
				if t, ok := startTime.(time.Time); ok {
					responseTime = time.Since(t).Milliseconds()
				}
			}
			
			logger.LogResponse(r.Request.URL.String(), statusCode, responseTime, err)
		}
	}
	})
	
	// 开始爬取
	err := collector.Visit(startURL.String())
	if err != nil {
		return nil, fmt.Errorf("访问URL失败 %s: %v", startURL.String(), err)
	}
	
	// 等待所有请求完成
	collector.Wait()
	
	return result, nil
}

// Stop 停止爬取
func (s *StaticCrawlerImpl) Stop() {
	// 等待所有请求完成
	s.collector.Wait()
}

// generatePOSTRequestFromForm 从表单生成POST请求数据
func (s *StaticCrawlerImpl) generatePOSTRequestFromForm(form *Form, enctype string) *POSTRequest {
	if form == nil || len(form.Fields) == 0 {
		return nil
	}
	
	// 使用SmartFormFiller填充表单
	formFiller := NewSmartFormFiller()
	formFiller.FillForm(form, "normal")
	
	// 构建参数map（过滤掉submit和button类型）
	parameters := make(map[string]string)
	for _, field := range form.Fields {
		if field.Name != "" && field.Value != "" {
			// 过滤掉提交按钮和普通按钮
			fieldTypeLower := strings.ToLower(field.Type)
			if fieldTypeLower == "submit" || fieldTypeLower == "button" {
				continue
			}
			parameters[field.Name] = field.Value
		}
	}
	
	// 如果没有参数，返回nil
	if len(parameters) == 0 {
		return nil
	}
	
	// 构建请求体
	body := ""
	requestURL := form.Action
	
	if form.Method == "POST" || form.Method == "PUT" || form.Method == "PATCH" {
		// POST请求：构建请求体
		values := url.Values{}
		for key, value := range parameters {
			values.Add(key, value)
		}
		body = values.Encode()
	} else if form.Method == "GET" {
		// GET请求：将参数添加到URL
		parsedURL, err := url.Parse(form.Action)
		if err == nil {
			query := parsedURL.Query()
			for key, value := range parameters {
				query.Set(key, value)
			}
			parsedURL.RawQuery = query.Encode()
			requestURL = parsedURL.String()
		}
	}
	
	return &POSTRequest{
		URL:         requestURL,
		Method:      form.Method,
		Parameters:  parameters,
		Body:        body,
		ContentType: enctype,
		FromForm:    true,
		FormAction:  form.Action,
	}
}

// addLinkWithFilter 添加链接到结果，应用所有过滤器（v4.0统一入口）
// 这是添加链接的唯一正确方式，确保所有过滤器都被应用
func (s *StaticCrawlerImpl) addLinkWithFilter(result *Result, rawURL string, absoluteURL string) bool {
	// 质量过滤（第一道防线）
	if s.urlQualityFilter != nil {
		if valid, _ := s.urlQualityFilter.IsHighQualityURL(absoluteURL); !valid {
			return false
		}
	}
	
	// URL验证器（第二道防线）
	if s.urlValidator != nil {
		if !s.urlValidator.IsValidBusinessURL(absoluteURL) {
			return false
		}
	}
	
	// 去重检查
	if s.duplicateHandler.IsDuplicateURL(absoluteURL) {
		return false
	}
	
	// 添加到结果
	result.Links = append(result.Links, absoluteURL)
	return true
}

// normalizeURLWithProtocolVariants 规范化URL并返回协议变体
// 🆕 v4.0：处理协议相对URL，生成http和https两个版本
func (s *StaticCrawlerImpl) normalizeURLWithProtocolVariants(rawURL string, baseURL *url.URL) []string {
	// 创建URL规范化处理器
	normalizer, err := NewURLNormalizer(baseURL.String())
	if err != nil {
		return nil
	}
	
	// 获取所有规范化的URL（协议相对URL会返回2个版本）
	normalized := normalizer.NormalizeURL(rawURL)
	
	// 过滤和验证
	filtered := make([]string, 0, len(normalized))
	for _, u := range normalized {
		// 应用质量过滤
		if s.urlQualityFilter != nil {
			if valid, _ := s.urlQualityFilter.IsHighQualityURL(u); !valid {
				continue
			}
		}
		
		// 应用URL验证器
		if s.urlValidator != nil {
			if !s.urlValidator.IsValidBusinessURL(u) {
				continue
			}
		}
		
		filtered = append(filtered, u)
	}
	
	return filtered
}

// resolveURL 将相对URL转换为绝对URL
// 🆕 v4.0：返回规范化后的URL列表（协议相对URL会返回http和https两个版本）
func resolveURL(baseURL *url.URL, relativeURL string) string {
	// 创建URL规范化处理器
	normalizer, err := NewURLNormalizer(baseURL.String())
	if err != nil {
		// 降级到旧逻辑
		return resolveURLLegacy(baseURL, relativeURL)
	}
	
	// 使用新的规范化处理器
	normalized := normalizer.NormalizeURL(relativeURL)
	if len(normalized) > 0 {
		return normalized[0] // 返回第一个（主要的）
	}
	
	return ""
}

// resolveURLLegacy 旧版URL解析逻辑（向下兼容）
func resolveURLLegacy(baseURL *url.URL, relativeURL string) string {
	// 如果relativeURL已经是绝对URL，直接返回
	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}
	
	// 解析相对URL
	parsedURL, err := url.Parse(relativeURL)
	if err != nil {
		return ""
	}
	
	// 如果relativeURL是协议相对URL（以//开头）
	// 🔧 v4.0修复：默认使用HTTPS（更安全）
	if strings.HasPrefix(relativeURL, "//") {
		return baseURL.Scheme + ":" + relativeURL
	}
	
	// 如果relativeURL是绝对路径（以/开头）
	if strings.HasPrefix(relativeURL, "/") {
		return baseURL.Scheme + "://" + baseURL.Host + relativeURL
	}
	
	// 处理相对路径（不以/开头）
	if !strings.HasPrefix(relativeURL, "/") && baseURL.Path != "" {
		// 获取基础路径的目录部分
		basePathDir := path.Dir(baseURL.Path)
		if basePathDir == "." {
			basePathDir = "/"
		}
		// 确保路径以/结尾
		if !strings.HasSuffix(basePathDir, "/") {
			basePathDir += "/"
		}
		return baseURL.Scheme + "://" + baseURL.Host + basePathDir + relativeURL
	}
	
	// 否则，将相对URL解析为绝对URL
	absoluteURL := baseURL.ResolveReference(parsedURL)
	return absoluteURL.String()
}

// ParseHTML 解析HTML内容
func (s *StaticCrawlerImpl) ParseHTML(htmlContent string, baseURL *url.URL) (*Result, error) {
	result := &Result{
		URL:    baseURL.String(),
		Links:  make([]string, 0),
		Assets: make([]string, 0),
		Forms:  make([]Form, 0),
		APIs:   make([]string, 0),
	}
	
	// 使用goquery解析HTML内容
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("解析HTML内容失败: %v", err)
	}
	
    // 提取链接（统一过滤）
	doc.Find("a[href]").Each(func(i int, selection *goquery.Selection) {
		link := selection.AttrOr("href", "")
		// 验证链接格式，避免处理javascript:和mailto:等非HTTP链接
		if !IsValidURL(link) {
			return
		}
		
		// 转换为绝对URL
		absoluteURL := resolveURL(baseURL, link)
		if absoluteURL != "" {
            _ = s.addLinkWithFilter(result, link, absoluteURL)
		}
	})
	
    // 提取资源链接（Assets中记录，Links不再重复记录）
	doc.Find("link[href], script[src], img[src]").Each(func(i int, selection *goquery.Selection) {
		var assetURL string
		if selection.Is("link") {
			assetURL = selection.AttrOr("href", "")
		} else {
			assetURL = selection.AttrOr("src", "")
		}
		
		// 转换为绝对URL
		absoluteURL := resolveURL(baseURL, assetURL)
		if absoluteURL != "" {
            // 仅记录到 Assets
            result.Assets = append(result.Assets, absoluteURL)
		}
	})
	
	// 提取表单
	forms := s.extractForms(htmlContent, baseURL.String())
	result.Forms = append(result.Forms, forms...)
	
	// 提取API端点
	doc.Find("script[src]").Each(func(i int, selection *goquery.Selection) {
		src := selection.AttrOr("src", "")
		absoluteURL := resolveURL(baseURL, src)
		if absoluteURL != "" && (strings.Contains(absoluteURL, "api") || strings.Contains(absoluteURL, "json") || strings.Contains(absoluteURL, "API")) {
			// 检查是否为重复的API端点
			if !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
				result.APIs = append(result.APIs, absoluteURL)
			}
		}
	})
	
	return result, nil
}

// extractForms 从HTML中提取表单
func (s *StaticCrawlerImpl) extractForms(htmlContent string, baseURL string) []Form {
	// 解析HTML文档
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return []Form{}
	}
	
	forms := make([]Form, 0)
	
	// 查找所有表单元素
	doc.Find("form").Each(func(i int, form *goquery.Selection) {
		extractedForm := Form{
			Action: form.AttrOr("action", ""),
			Method: strings.ToUpper(form.AttrOr("method", "GET")), // 转换为大写
			Fields: make([]FormField, 0),
		}
		
		// 解析表单字段
		form.Find("input, select, textarea").Each(func(j int, field *goquery.Selection) {
			formField := FormField{
				Name:     field.AttrOr("name", ""),
				Type:     field.AttrOr("type", "text"),
				Value:    field.AttrOr("value", ""),
				Required: field.AttrOr("required", "") != "",
			}
			
			// 为没有值的字段设置默认值
			if formField.Value == "" && (formField.Type == "text" || formField.Type == "password" || 
				formField.Type == "hidden" || formField.Type == "search" || formField.Type == "email" || 
				formField.Type == "url" || formField.Type == "tel") {
				formField.Value = "param_value"
			}
			
			extractedForm.Fields = append(extractedForm.Fields, formField)
		})
		
		// 处理表单action，确保是完整URL
		if extractedForm.Action != "" {
			// 如果action是相对路径，转换为绝对路径
			if !strings.HasPrefix(extractedForm.Action, "http") {
				resolvedURL, err := url.Parse(baseURL)
				if err == nil {
					baseURLPath, err := url.Parse(extractedForm.Action)
					if err == nil {
						extractedForm.Action = resolvedURL.ResolveReference(baseURLPath).String()
					}
				}
			}
		} else {
			// 如果没有action，使用当前页面URL
			extractedForm.Action = baseURL
		}
		
		// 添加表单到结果中，即使没有字段
		forms = append(forms, extractedForm)
	})
	
	return forms
}

// extractURLsFromHeaders 从HTTP响应头中提取URL
func (s *StaticCrawlerImpl) extractURLsFromHeaders(r *colly.Response) []string {
	urls := make([]string, 0)
	
	// 1. Location头（重定向）
	if location := r.Headers.Get("Location"); location != "" {
		absoluteURL := r.Request.AbsoluteURL(location)
		if absoluteURL != "" {
			urls = append(urls, absoluteURL)
		}
	}
	
	// 2. Link头（分页、预加载等）
	if linkHeader := r.Headers.Get("Link"); linkHeader != "" {
		// 解析Link头: </api/next>; rel="next"
		linkPattern := regexp.MustCompile(`<([^>]+)>`)
		matches := linkPattern.FindAllStringSubmatch(linkHeader, -1)
		for _, match := range matches {
			if len(match) > 1 {
				absoluteURL := r.Request.AbsoluteURL(match[1])
				if absoluteURL != "" {
					urls = append(urls, absoluteURL)
				}
			}
		}
	}
	
	// 3. Content-Location头
	if contentLoc := r.Headers.Get("Content-Location"); contentLoc != "" {
		absoluteURL := r.Request.AbsoluteURL(contentLoc)
		if absoluteURL != "" {
			urls = append(urls, absoluteURL)
		}
	}
	
	// 4. Refresh头
	if refresh := r.Headers.Get("Refresh"); refresh != "" {
		// 格式: "5; url=/home.php"
		parts := strings.Split(refresh, ";")
		for _, part := range parts {
			if strings.Contains(strings.ToLower(part), "url=") {
				urlPart := strings.TrimSpace(strings.SplitN(part, "=", 2)[1])
				urlPart = strings.Trim(urlPart, " '\"")
				absoluteURL := r.Request.AbsoluteURL(urlPart)
				if absoluteURL != "" {
					urls = append(urls, absoluteURL)
				}
			}
		}
	}
	
	return urls
}

// extractURLsFromInlineScripts 从内联JavaScript中提取URL
func (s *StaticCrawlerImpl) extractURLsFromInlineScripts(htmlContent, baseURL string) []string {
	urls := make([]string, 0)
	seen := make(map[string]bool)
	filteredCount := 0 // 🆕 v3.5: 统计过滤的URL数量
	
	// 1. 提取<script>标签内容
	scriptPattern := regexp.MustCompile(`(?i)<script[^>]*>([\s\S]*?)</script>`)
	scripts := scriptPattern.FindAllStringSubmatch(htmlContent, -1)
	
	for _, script := range scripts {
		if len(script) > 1 {
			jsCode := script[1]
			extractedURLs := s.extractURLsFromJSCode(jsCode)
			for _, u := range extractedURLs {
				if !seen[u] {
					seen[u] = true
					
					// 🆕 v3.5: 使用URL验证器过滤无效URL
					if s.urlValidator != nil && !s.urlValidator.IsValidBusinessURL(u) {
						filteredCount++
						continue
					}
					
					urls = append(urls, u)
				}
			}
		}
	}
	
	// 2. 提取事件处理器（onclick, onload等）
	eventAttrs := []string{"onclick", "onload", "onerror", "onsubmit", "onmouseover", 
	                       "onmouseout", "onfocus", "onblur", "onchange", "ondblclick"}
	
	for _, attr := range eventAttrs {
		pattern := fmt.Sprintf(`(?i)%s\s*=\s*["']([^"']+)["']`, attr)
		eventPattern := regexp.MustCompile(pattern)
		events := eventPattern.FindAllStringSubmatch(htmlContent, -1)
		
		for _, event := range events {
			if len(event) > 1 {
				handler := event[1]
				extractedURLs := s.extractURLsFromJSCode(handler)
				for _, u := range extractedURLs {
					if !seen[u] {
						seen[u] = true
						
						// 🆕 v3.5: 使用URL验证器过滤无效URL
						if s.urlValidator != nil && !s.urlValidator.IsValidBusinessURL(u) {
							filteredCount++
							continue
						}
						
						urls = append(urls, u)
					}
				}
			}
		}
	}
	
	// 🆕 v3.5: 输出过滤统计
	if filteredCount > 0 {
		fmt.Printf("    [URL过滤] 从JS中过滤了 %d 个无效URL\n", filteredCount)
	}
	
	return urls
}

// extractURLsFromJSCode 从JavaScript代码中提取URL（v4.0重写 - 使用专业提取器）
func (s *StaticCrawlerImpl) extractURLsFromJSCode(jsCode string) []string {
	// ✅ v4.0修复：使用专业的URL提取器
	extractor := NewURLExtractorFix()
	urls := extractor.ExtractFromJSCode(jsCode)
	
	// ✅ v4.0修复：应用质量过滤器
	if s.urlQualityFilter != nil {
		urls = s.urlQualityFilter.FilterURLs(urls)
	}
	
	// ✅ v4.0修复：应用URL验证器（双重保险）
	if s.urlValidator != nil {
		filtered := make([]string, 0, len(urls))
		for _, u := range urls {
			if s.urlValidator.IsValidBusinessURL(u) {
				filtered = append(filtered, u)
			}
		}
		return filtered
	}
	
	return urls
}

// extractURLsFromJSCodeLegacy 旧版JS代码URL提取（保留用于降级）
func (s *StaticCrawlerImpl) extractURLsFromJSCodeLegacy(jsCode string) []string {
	urls := make([]string, 0)
	seen := make(map[string]bool)
	
	// ⚠️ 只保留最严格的匹配模式
	patterns := []string{
		// AJAX和fetch（必须有明确的函数调用）
		`fetch\s*\(\s*['"]([^'"]+)['"]`,
		`\$\.ajax\s*\(\s*{[^}]*url\s*:\s*['"]([^'"]+)['"]`,
		`\$\.get\s*\(\s*['"]([^'"]+)['"]`,
		`\$\.post\s*\(\s*['"]([^'"]+)['"]`,
		`axios\.(get|post|put|delete|patch)\s*\(\s*['"]([^'"]+)['"]`,
		`xhr\.open\s*\(\s*['"](?:GET|POST)['"],\s*['"]([^'"]+)['"]`,
		
		// window.location（必须有明确的赋值）
		`window\.location\s*=\s*['"]([^'"]+)['"]`,
		`location\.href\s*=\s*['"]([^'"]+)['"]`,
		`window\.open\s*\(\s*['"]([^'"]+)['"]`,
		
		// API配置（只匹配特定变量名）
		`\b(?:apiUrl|baseURL|endpoint)\s*[:=]\s*['"]([^'"]+)['"]`,
		
		// API端点（必须在引号中且有/api/前缀）
		`['"]/(api/[a-zA-Z0-9_\-/]+)['"]`,
		`['"]/(v\d+/[a-zA-Z0-9_\-/]+)['"]`,
		
		// 文件路径（必须有明确的文件扩展名）
		`['"]([a-zA-Z0-9_\-/]+\.(?:php|jsp|asp|aspx|do|action|html|htm))['"]`,
	}
	
	// ⚠️ 降级逻辑已废弃，统一使用专业提取器
	// 这个函数保留只是为了向下兼容
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsCode, -1)
		
		for _, match := range matches {
			if len(match) >= 2 {
				url := match[len(match)-1]
				
				// 基础过滤
				if url == "" || url == "/" || url == "#" {
					continue
				}
				
				// 只保留HTTP相对路径或完整URL
				if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "http") {
					if !seen[url] {
						seen[url] = true
						urls = append(urls, url)
					}
				}
			}
		}
	}
	
	return urls
}

// extractURLsFromCSS 从CSS中提取URL
func (s *StaticCrawlerImpl) extractURLsFromCSS(htmlContent string) []string {
	urls := make([]string, 0)
	seen := make(map[string]bool)
	
	// 1. 提取<style>标签内容
	stylePattern := regexp.MustCompile(`(?i)<style[^>]*>([\s\S]*?)</style>`)
	styles := stylePattern.FindAllStringSubmatch(htmlContent, -1)
	
	cssContent := ""
	for _, style := range styles {
		if len(style) > 1 {
			cssContent += style[1] + "\n"
		}
	}
	
	// 2. 提取style属性
	styleAttrPattern := regexp.MustCompile(`(?i)style\s*=\s*["']([^"']+)["']`)
	styleAttrs := styleAttrPattern.FindAllStringSubmatch(htmlContent, -1)
	for _, attr := range styleAttrs {
		if len(attr) > 1 {
			cssContent += attr[1] + "\n"
		}
	}
	
	if cssContent == "" {
		return urls
	}
	
	// CSS URL提取模式
	patterns := []string{
		`url\s*\(\s*['"]?([^'")]+)['"]?\s*\)`,  // url()
		`@import\s+url\s*\(\s*['"]?([^'")]+)['"]?\s*\)`, // @import url()
		`@import\s+['"]([^'"]+)['"]`, // @import "..."
		`src\s*:\s*url\s*\(\s*['"]?([^'")]+)['"]?\s*\)`, // @font-face src
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(cssContent, -1)
		
		for _, match := range matches {
			if len(match) >= 2 {
				url := strings.TrimSpace(match[len(match)-1])
				
				// 过滤data:和javascript:
				if !strings.HasPrefix(url, "data:") && 
					!strings.HasPrefix(url, "javascript:") &&
					url != "" && !seen[url] {
					seen[url] = true
					urls = append(urls, url)
				}
			}
		}
	}
	
	return urls
}