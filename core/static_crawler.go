package core

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path"
	"path/filepath"
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
}

// 保存响应数据包到文件
func (s *StaticCrawlerImpl) saveResponseToFile(url string, body []byte, contentType string) error {
	// 创建responses目录
	dir := filepath.Join(".", "responses")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// 生成文件名（使用URL的MD5哈希）
	hash := fmt.Sprintf("%x", md5.Sum([]byte(url)))
	
	// 根据内容类型确定文件扩展名
	var ext string
	switch {
	case strings.Contains(contentType, "text/html"):
		ext = ".html"
	case strings.Contains(contentType, "application/javascript") || strings.Contains(contentType, "text/javascript"):
		ext = ".js"
	case strings.Contains(contentType, "text/css"):
		ext = ".css"
	case strings.Contains(contentType, "application/json"):
		ext = ".json"
	case strings.Contains(contentType, "image/"):
		ext = ".bin" // 默认二进制格式
	default:
		ext = ".txt"
	}
	
	filename := filepath.Join(dir, hash+ext)
	
	// 写入文件
	return os.WriteFile(filename, body, 0644)
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

// Crawl 执行爬取
func (s *StaticCrawlerImpl) Crawl(startURL *url.URL) (*Result, error) {
	result := &Result{
		URL:    startURL.String(),
		Links:  make([]string, 0),
		Assets: make([]string, 0),
		Forms:  make([]Form, 0),
		APIs:   make([]string, 0),
	}
	
	// 设置请求前回调，实现User-Agent轮换和域名范围检查
	s.collector.OnRequest(func(r *colly.Request) {
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
	})
	
	// 设置HTML回调
	s.collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// 验证链接格式，避免处理javascript:和mailto:等非HTTP链接
		if !IsValidURL(link) {
			return
		}
		absoluteURL := e.Request.AbsoluteURL(link)
		if absoluteURL != "" {
			// 检查是否为重复链接
			if !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
				result.Links = append(result.Links, absoluteURL)
			}
		}
	})
	
	// 设置资源回调
	s.collector.OnHTML("link[href], script[src], img[src]", func(e *colly.HTMLElement) {
		var assetURL string
		if e.Name == "link" {
			assetURL = e.Attr("href")
		} else {
			assetURL = e.Attr("src")
		}
		absoluteURL := e.Request.AbsoluteURL(assetURL)
		if absoluteURL != "" {
			// 检查是否为重复资源
			if !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
				result.Assets = append(result.Assets, absoluteURL)
			}
		}
	})
	
	// 设置表单回调
	s.collector.OnHTML("form", func(e *colly.HTMLElement) {
		action := e.Attr("action")
		method := e.Attr("method")
		
		// 收集表单字段
		fields := make([]FormField, 0)
		e.ForEach("input, select, textarea", func(_ int, el *colly.HTMLElement) {
			field := FormField{
				Name:  el.Attr("name"),
				Type:  el.Attr("type"),
				Value: el.Attr("value"),
			}
			fields = append(fields, field)
		})
		
		formData := Form{
			Action: action,
			Method: method,
			Fields: fields,
		}
		
		// 提取表单参数
		params, err := s.paramHandler.ExtractParams(formData.Action)
		if err == nil && len(params) > 0 {
			// 这里可以进一步处理参数，但为简化我们只检查是否有参数
			result.Forms = append(result.Forms, formData)
		}
	})
	
	// 设置API端点回调
	s.collector.OnHTML("script[src]", func(e *colly.HTMLElement) {
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
	s.collector.OnResponse(func(r *colly.Response) {
		result.StatusCode = r.StatusCode
		result.ContentType = r.Headers.Get("Content-Type")
		
		// 保存HTML内容和Headers供高级检测使用
		result.HTMLContent = string(r.Body)
		result.Headers = make(map[string]string)
		for key, values := range *r.Headers {
			if len(values) > 0 {
				result.Headers[key] = values[0]
			}
		}
		
		// 保存响应数据包
		if err := s.saveResponseToFile(r.Request.URL.String(), r.Body, result.ContentType); err != nil {
			fmt.Printf("保存响应数据包失败 %s: %v\n", r.Request.URL.String(), err)
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
	s.collector.OnError(func(r *colly.Response, err error) {
		fmt.Printf("请求错误 %s: %v\n", r.Request.URL, err)
	})
	
	// 开始爬取
	err := s.collector.Visit(startURL.String())
	if err != nil {
		return nil, fmt.Errorf("访问URL失败 %s: %v", startURL.String(), err)
	}
	
	// 等待所有请求完成
	s.collector.Wait()
	
	return result, nil
}

// Stop 停止爬取
func (s *StaticCrawlerImpl) Stop() {
	// 等待所有请求完成
	s.collector.Wait()
}

// resolveURL 将相对URL转换为绝对URL
func resolveURL(baseURL *url.URL, relativeURL string) string {
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
	
	// 提取链接
	doc.Find("a[href]").Each(func(i int, selection *goquery.Selection) {
		link := selection.AttrOr("href", "")
		// 验证链接格式，避免处理javascript:和mailto:等非HTTP链接
		if !IsValidURL(link) {
			return
		}
		
		// 转换为绝对URL
		absoluteURL := resolveURL(baseURL, link)
		if absoluteURL != "" {
			// 检查是否为重复链接
			if !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
				result.Links = append(result.Links, absoluteURL)
			}
		}
	})
	
	// 提取资源链接
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
			// 检查是否为重复资源
			if !s.duplicateHandler.IsDuplicateURL(absoluteURL) {
				result.Assets = append(result.Assets, absoluteURL)
			}
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