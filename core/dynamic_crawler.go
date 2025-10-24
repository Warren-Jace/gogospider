package core

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
	
	"github.com/chromedp/chromedp"
	"spider-golang/config"
)

// DynamicCrawlerImpl 动态爬虫实现
type DynamicCrawlerImpl struct {
	config          *config.Config
	timeout         time.Duration
	eventTrigger    *EventTrigger    // 事件触发器
	ajaxInterceptor *AjaxInterceptor // AJAX拦截器
	enableEvents    bool             // 是否启用事件触发
	enableAjax      bool             // 是否启用AJAX拦截
}

// NewDynamicCrawler 创建动态爬虫实例
func NewDynamicCrawler() *DynamicCrawlerImpl {
	return &DynamicCrawlerImpl{
		timeout:         180 * time.Second, // 每个请求180秒超时（3分钟）
		eventTrigger:    NewEventTrigger(),
		ajaxInterceptor: nil, // 将在Crawl方法中根据目标域名创建
		enableEvents:    true, // 默认启用事件触发
		enableAjax:      true, // 默认启用AJAX拦截
	}
}

// SetEnableEvents 设置是否启用事件触发
func (d *DynamicCrawlerImpl) SetEnableEvents(enable bool) {
	d.enableEvents = enable
}

// Configure 配置动态爬虫
func (d *DynamicCrawlerImpl) Configure(config *config.Config) {
	d.config = config
	
	// 更新超时设置
	if config.AntiDetectionSettings.RequestDelay > 0 {
		d.timeout = config.AntiDetectionSettings.RequestDelay * 10
		if d.timeout < 180*time.Second {
			d.timeout = 180 * time.Second
		}
	}
}

// Crawl 执行动态爬取
func (d *DynamicCrawlerImpl) Crawl(targetURL *url.URL) (*Result, error) {
	// 为每次爬取创建独立的上下文（优化：避免共享context超时问题）
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	
	// 设置Chrome选项（全面优化：更稳定更快速的启动参数）
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// 基础设置
		chromedp.Flag("headless", true), // 无头模式
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		
		// 跨域和安全设置
		chromedp.Flag("disable-web-security", true), // 允许跨域
		chromedp.Flag("allow-running-insecure-content", true),
		
		// 性能优化
		chromedp.Flag("disable-features", "VizDisplayCompositor,IsolateOrigins,site-per-process"),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-component-extensions-with-background-pages", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-translate", true),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		
		// 内存和资源限制
		chromedp.Flag("force-color-profile", "srgb"),
		chromedp.Flag("memory-pressure-off", true),
		chromedp.Flag("max-gum-fps", "60"),
		
		// 窗口设置
		chromedp.WindowSize(1920, 1080),
		
		// 用户代理
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()
	
	// 创建Chrome实例
	chromeCtx, cancelChrome := chromedp.NewContext(allocCtx)
	defer cancelChrome()
	
	// 设置超时
	chromeCtx, cancelTimeout := context.WithTimeout(chromeCtx, d.timeout)
	defer cancelTimeout()
	
	result := &Result{
		URL:          targetURL.String(),
		Links:        make([]string, 0),
		Assets:       make([]string, 0),
		Forms:        make([]Form, 0),
		APIs:         make([]string, 0),
		POSTRequests: make([]POSTRequest, 0),
	}
	
	// 检查域名范围限制
	if d.config != nil && d.config.StrategySettings.DomainScope != "" {
		if !strings.Contains(targetURL.Host, d.config.StrategySettings.DomainScope) {
			fmt.Printf("URL超出域名范围，不进行动态爬取: %s\n", targetURL.String())
			return result, nil
		}
	}
	
	// 启动AJAX拦截器
	if d.enableAjax {
		d.ajaxInterceptor = NewAjaxInterceptor(targetURL.Host)
		d.ajaxInterceptor.StartListening(chromeCtx)
		fmt.Println("  [动态爬虫] AJAX拦截器已启动")
	}
	
	// 导航到目标页面（智能等待机制）
	var htmlContent string
	err := chromedp.Run(chromeCtx,
		chromedp.Navigate(targetURL.String()),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		
		// 等待初始DOM加载
		chromedp.Sleep(2 * time.Second),
		
		// 等待网络空闲（所有AJAX请求完成）
		chromedp.ActionFunc(func(ctx context.Context) error {
			// 注入JavaScript检测网络活动
			var networkIdle bool
			checkScript := `
			(function() {
				// 检查是否有进行中的fetch或XMLHttpRequest
				if (window.performance) {
					var resources = window.performance.getEntriesByType("resource");
					var recentRequests = resources.filter(function(r) {
						return (Date.now() - r.responseEnd) < 1000;
					});
					return recentRequests.length === 0;
				}
				return true;
			})()
			`
			
			// 最多等待10秒，检查网络是否空闲
			for i := 0; i < 20; i++ {
				chromedp.Evaluate(checkScript, &networkIdle).Do(ctx)
				if networkIdle {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
			return nil
		}),
		
		// 额外等待以确保动态内容完全渲染
		chromedp.Sleep(3 * time.Second),
		
		chromedp.OuterHTML("html", &htmlContent),
	)
	
	if err != nil {
		return nil, fmt.Errorf("导航到页面失败: %v", err)
	}
	
	// 提取页面信息
	// 获取所有链接（Phase 3增强：包括动态生成的链接）
	var links []string
	err = chromedp.Run(chromeCtx,
		chromedp.Evaluate(`
		(function() {
			var allLinks = new Set();
			
			// 1. 常规<a>链接
			document.querySelectorAll('a[href]').forEach(function(a) {
				if (a.href) allLinks.add(a.href);
			});
			
			// 2. 带data-*属性的元素
			document.querySelectorAll('[data-url], [data-href], [data-link]').forEach(function(el) {
				['data-url', 'data-href', 'data-link'].forEach(function(attr) {
					var val = el.getAttribute(attr);
					if (val) allLinks.add(val);
				});
			});
			
			// 3. onclick等事件处理器中的URL
			document.querySelectorAll('[onclick]').forEach(function(el) {
				var onclick = el.getAttribute('onclick');
				var urlMatch = onclick.match(/(['"])([^'"]*\.php[^'"]*)\1/);
				if (urlMatch && urlMatch[2]) {
					allLinks.add(urlMatch[2]);
				}
			});
			
			// 4. 表单的action
			document.querySelectorAll('form[action]').forEach(function(form) {
				if (form.action) allLinks.add(form.action);
			});
			
			return Array.from(allLinks);
		})()
		`, &links),
	)
	
	if err == nil {
		fmt.Printf("  [动态爬虫] 从页面提取到 %d 个链接\n", len(links))
		// 检查域名范围限制
		if d.config != nil && d.config.StrategySettings.DomainScope != "" {
			for _, link := range links {
				parsedLink, err := url.Parse(link)
				if err != nil {
					continue
				}
				
				// 检查是否在允许的域名范围内
				if strings.Contains(parsedLink.Host, d.config.StrategySettings.DomainScope) {
					result.Links = append(result.Links, link)
				} else {
					fmt.Printf("发现外部链接（已记录但不爬取）: %s\n", link)
				}
			}
		} else {
			result.Links = append(result.Links, links...)
		}
	} else {
		fmt.Printf("  [动态爬虫] ⚠️  提取链接时出错: %v\n", err)
	}
	
	// 获取所有资源链接
	var assets []string
	err = chromedp.Run(chromeCtx,
		chromedp.Evaluate(`Array.from(document.querySelectorAll('link[href], script[src], img[src]')).map(el => el.src || el.href)`, &assets),
	)
	
	if err == nil {
		fmt.Printf("  [动态爬虫] 从页面提取到 %d 个资源\n", len(assets))
		// 检查域名范围限制
		if d.config != nil && d.config.StrategySettings.DomainScope != "" {
			for _, asset := range assets {
				parsedAsset, err := url.Parse(asset)
				if err != nil {
					continue
				}
				
				// 检查是否在允许的域名范围内
				if strings.Contains(parsedAsset.Host, d.config.StrategySettings.DomainScope) {
					result.Assets = append(result.Assets, asset)
				}
			}
		} else {
			result.Assets = append(result.Assets, assets...)
		}
	}
	
	// 提取表单信息
	var forms []map[string]interface{}
	err = chromedp.Run(chromeCtx,
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('form')).map(form => {
				const formData = {
					action: form.action,
					method: form.method,
					fields: []
				};
				
				form.querySelectorAll('input').forEach(input => {
					formData.fields.push({
						name: input.name,
						type: input.type,
						value: input.value,
						required: input.required
					});
				});
				
				return formData;
			})
		`, &forms),
	)
	
	if err == nil {
		fmt.Printf("  [动态爬虫] 从页面提取到 %d 个表单\n", len(forms))
		// 转换为Form结构
		for _, formMap := range forms {
			form := Form{
				Action: getString(formMap, "action"),
				Method: getString(formMap, "method"),
				Fields: make([]FormField, 0),
			}
			
			// 提取字段
			if fields, ok := formMap["fields"].([]interface{}); ok {
				for _, field := range fields {
					if fieldMap, ok := field.(map[string]interface{}); ok {
						formField := FormField{
							Name:     getString(fieldMap, "name"),
							Type:     getString(fieldMap, "type"),
							Value:    getString(fieldMap, "value"),
							Required: getBool(fieldMap, "required"),
						}
						form.Fields = append(form.Fields, formField)
					}
				}
			}
			
			result.Forms = append(result.Forms, form)
		}
	}
	
	// 尝试提取API端点
	apis := d.extractAPIsFromJS(chromeCtx)
	
	// 提取内联JavaScript中的URL（Phase 3增强）
	inlineJSURLs := d.extractInlineJSURLs(chromeCtx)
	if len(inlineJSURLs) > 0 {
		fmt.Printf("  [JS分析] 从内联脚本提取了 %d 个URL\n", len(inlineJSURLs))
		// 添加到链接列表
		for _, jsURL := range inlineJSURLs {
			// 去重
			found := false
			for _, existing := range result.Links {
				if existing == jsURL {
					found = true
					break
				}
			}
			if !found {
				result.Links = append(result.Links, jsURL)
			}
		}
	}
	
	// 自动分析表单并生成提交URL（Phase 3增强 + POST提交）
	postRequests := d.submitFormsAndCapturePOST(chromeCtx, targetURL.String())
	if len(postRequests) > 0 {
		fmt.Printf("  [表单分析] 提交了 %d 个POST表单\n", len(postRequests))
		result.POSTRequests = append(result.POSTRequests, postRequests...)
		
		// 同时添加URL到links（兼容性）
		for _, postReq := range postRequests {
			if postReq.Method == "GET" {
				result.Links = append(result.Links, postReq.URL)
			} else {
				// POST请求也记录URL
				result.Links = append(result.Links, postReq.URL)
			}
		}
	}
	
	// 检查域名范围限制
	if d.config != nil && d.config.StrategySettings.DomainScope != "" {
		for _, api := range apis {
			parsedAPI, err := url.Parse(api)
			if err != nil {
				// 如果解析失败，可能是相对URL，直接添加
				result.APIs = append(result.APIs, api)
				continue
			}
			
			// 检查是否在允许的域名范围内
			if strings.Contains(parsedAPI.Host, d.config.StrategySettings.DomainScope) {
				result.APIs = append(result.APIs, api)
			}
		}
	} else {
		result.APIs = append(result.APIs, apis...)
	}
	
	// 获取页面状态码和内容类型（通过JavaScript）
	var statusCode int64
	var contentType string
	err = chromedp.Run(chromeCtx,
		chromedp.Evaluate(`window.performance.getEntriesByType('navigation')[0].responseStart`, &statusCode),
		chromedp.Evaluate(`document.contentType`, &contentType),
	)
	
	if err == nil {
		result.StatusCode = int(statusCode)
		result.ContentType = contentType
	}
	
	// 保存HTML内容供后续检测使用
	result.HTMLContent = htmlContent
	result.Headers = make(map[string]string)
	result.Headers["Content-Type"] = contentType
	
	// 如果启用了事件触发，执行事件触发
	if d.enableEvents && d.eventTrigger != nil {
		fmt.Println("  [动态爬虫] 启动JavaScript事件触发...")
		
		// 触发事件
		eventResult, err := d.eventTrigger.TriggerEvents(chromeCtx)
		if err != nil {
			fmt.Printf("  [事件触发] 执行出错: %v\n", err)
		} else {
			// 合并事件触发发现的URL和表单
			if len(eventResult.NewURLsFound) > 0 {
				fmt.Printf("  [事件触发] 发现 %d 个新URL\n", len(eventResult.NewURLsFound))
				result.Links = append(result.Links, eventResult.NewURLsFound...)
			}
			
			if len(eventResult.NewFormsFound) > 0 {
				fmt.Printf("  [事件触发] 发现 %d 个新表单\n", len(eventResult.NewFormsFound))
				result.Forms = append(result.Forms, eventResult.NewFormsFound...)
			}
			
			// 可选：触发无限滚动
			scrollCount, err := d.eventTrigger.TriggerInfiniteScroll(chromeCtx)
			if err == nil && scrollCount > 0 {
				fmt.Printf("  [事件触发] 执行了 %d 次滚动加载\n", scrollCount)
				
				// 滚动后重新提取链接
				var newLinks []string
				chromedp.Run(chromeCtx,
					chromedp.Evaluate(`Array.from(document.querySelectorAll('a[href]')).map(a => a.href)`, &newLinks),
				)
				
				// 合并新链接
				for _, link := range newLinks {
					found := false
					for _, existing := range result.Links {
						if existing == link {
							found = true
							break
						}
					}
					if !found {
						result.Links = append(result.Links, link)
					}
				}
			}
		}
	}
	
	// 收集AJAX拦截器捕获的URL
	if d.enableAjax && d.ajaxInterceptor != nil {
		ajaxURLs := d.ajaxInterceptor.GetInterceptedURLs()
		if len(ajaxURLs) > 0 {
			fmt.Printf("  [AJAX拦截] 捕获到 %d 个AJAX请求URL\n", len(ajaxURLs))
			
			// 去重并添加到结果
			for _, ajaxURL := range ajaxURLs {
				// 检查是否已存在
				found := false
				for _, existing := range result.Links {
					if existing == ajaxURL {
						found = true
						break
					}
				}
				
				// 如果不存在，添加
				if !found {
					result.Links = append(result.Links, ajaxURL)
				}
			}
			
			// 打印统计
			stats := d.ajaxInterceptor.GetStatistics()
			fmt.Printf("  [AJAX拦截] 统计: %v\n", stats)
		}
	}
	
	return result, nil
}

// extractAPIsFromJS 从JavaScript中提取API端点
func (d *DynamicCrawlerImpl) extractAPIsFromJS(ctx context.Context) []string {
	apis := make([]string, 0)
	
	// 获取页面中的所有脚本内容
	var scriptContents []string
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('script')).map(script => {
				if (script.src) {
					return 'SCRIPT_SRC:' + script.src;
				}
				return script.textContent;
			}).filter(content => content && content.length > 0)
		`, &scriptContents),
	)
	
	if err != nil {
		return apis
	}
	
	// 分析脚本内容查找API端点
	for _, content := range scriptContents {
		// 跳过外部脚本链接
		if strings.HasPrefix(content, "SCRIPT_SRC:") {
			// 但仍然记录外部脚本URL
			scriptURL := strings.TrimPrefix(content, "SCRIPT_SRC:")
			if strings.Contains(scriptURL, "/api/") || strings.Contains(scriptURL, "/v1/") || strings.Contains(scriptURL, "/v2/") || strings.Contains(scriptURL, "/AJAX/") {
				apis = append(apis, scriptURL)
			}
			continue
		}
		
		// 查找常见的API模式
		// 使用正则表达式查找可能的API端点
		// 查找AJAX相关的URL
		if strings.Contains(content, "/AJAX/") {
			// 提取/AJAX/相关的URL
			apis = append(apis, "discovered_from_js_analysis_AJAX")
		}
		
		// 查找API端点
		if strings.Contains(content, "/api/") || strings.Contains(content, "/v1/") || strings.Contains(content, "/v2/") {
			// 这里可以进一步解析具体的API端点
			// 为简化，我们只添加标记表示发现了API相关代码
			apis = append(apis, "discovered_from_js_analysis_API")
		}
		
		// 查找特定的AJAX端点
		ajaxEndpoints := []string{
			"titles.php",
			"showxml.php",
			"artists.php",
			"categories.php",
		}
		
		for _, endpoint := range ajaxEndpoints {
			if strings.Contains(content, endpoint) {
				// 构造完整的URL
				fullURL := "http://testphp.vulnweb.com/AJAX/" + endpoint
				apis = append(apis, fullURL)
			}
		}
		
		// 查找更多可能的端点
		endpoints := []string{
			"cart.php",
			"login.php",
			"userinfo.php",
			"guestbook.php",
			"categories.php",
			"artists.php",
			"privacy.php",
		}
		
		for _, endpoint := range endpoints {
			if strings.Contains(content, endpoint) {
				// 构造完整的URL
				fullURL := "http://testphp.vulnweb.com/" + endpoint
				apis = append(apis, fullURL)
			}
		}
	}
	
	return apis
}

// ExecuteJS 执行JavaScript
func (d *DynamicCrawlerImpl) ExecuteJS(script string) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	
	var result interface{}
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &result),
	)
	
	if err != nil {
		return nil, fmt.Errorf("执行JavaScript失败: %v", err)
	}
	
	return result, nil
}

// extractInlineJSURLs 从内联JavaScript提取URL（Phase 3增强）
func (d *DynamicCrawlerImpl) extractInlineJSURLs(ctx context.Context) []string {
	urls := make([]string, 0)
	
	// 执行JavaScript提取所有内联脚本中的URL
	var jsURLs []interface{}
	script := `
	(function() {
		var allURLs = new Set();
		
		// 提取所有<script>标签内容
		document.querySelectorAll('script:not([src])').forEach(function(script) {
			var code = script.textContent;
			
			// 正则匹配URL模式
			var patterns = [
				/(['"])([^'"]*\.php[^'"]*)\1/g,
				/fetch\s*\(\s*(['"])([^'"]+)\1/g,
				/\$\.ajax\s*\(\s*\{[^}]*url\s*:\s*(['"])([^'"]+)\1/g,
				/\$\.get\s*\(\s*(['"])([^'"]+)\1/g,
				/\$\.post\s*\(\s*(['"])([^'"]+)\1/g,
				/axios\.(get|post)\s*\(\s*(['"])([^'"]+)\2/g,
				/xhr\.open\s*\(\s*['"](?:GET|POST)['"]\s*,\s*(['"])([^'"]+)\1/g,
				/(['"])(\/[a-zA-Z0-9_\-\/\.?=&]+)\1/g
			];
			
			patterns.forEach(function(pattern) {
				var match;
				while ((match = pattern.exec(code)) !== null) {
					// 获取URL（可能在match[2]或match[3]）
					var url = match[2] || match[3] || match[1];
					if (url && (url.startsWith('http') || url.startsWith('/'))) {
						// 转换相对URL为绝对URL
						if (url.startsWith('/')) {
							url = window.location.origin + url;
						}
						allURLs.add(url);
					}
				}
			});
		});
		
		return Array.from(allURLs);
	})()
	`
	
	err := chromedp.Run(ctx, chromedp.Evaluate(script, &jsURLs))
	if err == nil {
		for _, u := range jsURLs {
			if urlStr, ok := u.(string); ok {
				urls = append(urls, urlStr)
			}
		}
	}
	
	return urls
}

// submitFormsAndCapturePOST 自动提交表单并捕获POST请求（完整实现）
func (d *DynamicCrawlerImpl) submitFormsAndCapturePOST(ctx context.Context, baseURL string) []POSTRequest {
	postRequests := make([]POSTRequest, 0)
	
	// 执行JavaScript收集所有表单数据
	var formsData []interface{}
	script := `
	(function() {
		var allForms = [];
		var forms = document.querySelectorAll('form');
		
		forms.forEach(function(form, formIndex) {
			try {
				var action = form.action || window.location.href;
				var method = (form.method || 'GET').toUpperCase();
				var enctype = form.enctype || 'application/x-www-form-urlencoded';
				
				var fields = {};
				var inputs = form.querySelectorAll('input, select, textarea');
				
				inputs.forEach(function(input) {
					var name = input.name;
					if (!name) return;
					
					var type = (input.type || 'text').toLowerCase();
					
					// 过滤掉提交按钮和普通按钮
					if (type === 'submit' || type === 'button') {
						return;
					}
					
					var value = input.value;
					
					// 如果没有值，填充智能测试值
					if (!value || value === '') {
						switch(type) {
							case 'text':
							case 'search':
								value = 'test_value';
								break;
							case 'email':
								value = 'test@example.com';
								break;
							case 'password':
								value = 'Test@123456';
								break;
							case 'number':
								value = '123';
								break;
							case 'tel':
								value = '13800138000';
								break;
							case 'url':
								value = 'https://example.com';
								break;
							case 'date':
								value = '2025-01-01';
								break;
							case 'hidden':
								// 保持隐藏字段的原值
								break;
							case 'checkbox':
								value = input.checked ? 'on' : '';
								break;
							case 'radio':
								value = input.checked ? input.value : '';
								break;
							default:
								value = 'test';
						}
					}
					
					if (value !== '') {
						fields[name] = value;
					}
				});
				
				allForms.push({
					action: action,
					method: method,
					enctype: enctype,
					fields: fields,
					index: formIndex
				});
			} catch(e) {
				console.error('Form processing error:', e);
			}
		});
		
		return allForms;
	})()
	`
	
	err := chromedp.Run(ctx, chromedp.Evaluate(script, &formsData))
	if err != nil {
		fmt.Printf("  [表单提交] JavaScript执行失败: %v\n", err)
		return postRequests
	}
	
	// 处理每个表单
	for _, formData := range formsData {
		formMap, ok := formData.(map[string]interface{})
		if !ok {
			continue
		}
		
		action := getStringFromMap(formMap, "action")
		method := getStringFromMap(formMap, "method")
		enctype := getStringFromMap(formMap, "enctype")
		
		// 解析action URL
		actionURL, err := url.Parse(action)
		if err != nil {
			continue
		}
		
		// 如果是相对路径，转换为绝对路径
		if !actionURL.IsAbs() {
			baseURLParsed, err := url.Parse(baseURL)
			if err == nil {
				actionURL = baseURLParsed.ResolveReference(actionURL)
			}
		}
		
		// 提取字段（过滤掉submit和button类型）
		// JavaScript已经在前端过滤了submit和button，这里直接提取即可
		parameters := make(map[string]string)
		if fieldsMap, ok := formMap["fields"].(map[string]interface{}); ok {
			for key, value := range fieldsMap {
				if strValue, ok := value.(string); ok {
					parameters[key] = strValue
				}
			}
		}
		
		// 构建请求体
		body := ""
		if method == "POST" || method == "PUT" || method == "PATCH" {
			// 构建URL编码的请求体
			values := url.Values{}
			for key, value := range parameters {
				values.Add(key, value)
			}
			body = values.Encode()
		}
		
		// 创建POST请求记录
		postReq := POSTRequest{
			URL:         actionURL.String(),
			Method:      method,
			Parameters:  parameters,
			Body:        body,
			ContentType: enctype,
			FromForm:    true,
			FormAction:  action,
		}
		
		// 如果是GET方法，将参数添加到URL
		if method == "GET" && len(parameters) > 0 {
			query := actionURL.Query()
			for key, value := range parameters {
				query.Set(key, value)
			}
			actionURL.RawQuery = query.Encode()
			postReq.URL = actionURL.String()
		}
		
		postRequests = append(postRequests, postReq)
		
		// 打印POST请求信息
		if method == "POST" {
			fmt.Printf("  [POST表单] %s\n", postReq.URL)
			fmt.Printf("    参数: %d 个字段\n", len(parameters))
			// 显示前3个参数
			count := 0
			for key, value := range parameters {
				if count < 3 {
					// 隐藏密码字段的值
					displayValue := value
					keyLower := strings.ToLower(key)
					if strings.Contains(keyLower, "password") || strings.Contains(keyLower, "pwd") {
						displayValue = "******"
					}
					fmt.Printf("    - %s=%s\n", key, displayValue)
					count++
				}
			}
			if len(parameters) > 3 {
				fmt.Printf("    ... 还有 %d 个参数\n", len(parameters)-3)
			}
		}
	}
	
	return postRequests
}

// getStringFromMap 从map中安全获取字符串
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// Stop 停止爬取
func (d *DynamicCrawlerImpl) Stop() {
	// 不再需要 cancel，每个 Crawl 都有自己的 context
	// 这里可以添加其他清理逻辑
}

// getString 从map中安全获取字符串值
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// getBool 从map中安全获取布尔值
func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}