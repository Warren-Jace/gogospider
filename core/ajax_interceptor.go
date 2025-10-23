package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// AjaxInterceptor AJAX请求拦截器
type AjaxInterceptor struct {
	interceptedURLs []string
	mutex           sync.Mutex
	targetDomain    string
}

// NewAjaxInterceptor 创建AJAX拦截器
func NewAjaxInterceptor(targetDomain string) *AjaxInterceptor {
	return &AjaxInterceptor{
		interceptedURLs: make([]string, 0),
		targetDomain:    targetDomain,
	}
}

// StartListening 开始监听网络请求
func (ai *AjaxInterceptor) StartListening(ctx context.Context) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventRequestWillBeSent:
			// 记录所有请求（检查是否为AJAX）
			url := ev.Request.URL
			method := ev.Request.Method
			
			// 检查是否可能是AJAX请求
			if ai.isPotentialAjaxURL(url, method, ev.Request.Headers) {
				ai.addURL(url)
			}
		case *network.EventResponseReceived:
			// 也记录响应中的URL（如果看起来像API）
			url := ev.Response.URL
			if ai.isPotentialAjaxURL(url, "", nil) {
				ai.addURL(url)
			}
		}
	})
}

// isPotentialAjaxURL 判断URL是否可能是AJAX请求（增强版）
func (ai *AjaxInterceptor) isPotentialAjaxURL(url, method string, headers map[string]interface{}) bool {
	urlLower := strings.ToLower(url)
	
	// 排除静态资源
	staticExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".css", ".js", ".ico", ".svg", ".woff", ".woff2", ".ttf", ".eot"}
	for _, ext := range staticExtensions {
		if strings.HasSuffix(urlLower, ext) {
			return false
		}
	}
	
	// 1. 检查URL模式（扩展关键词列表）
	ajaxKeywords := []string{
		"/ajax/", "/api/", "/rest/", "/graphql/",
		".json", ".xml", "showxml", "getxml",
		"/v1/", "/v2/", "/v3/",
		"comment", "product", "listproduct", "showimage",
		"artists", "categories", "titles",
		"?ajax=", "&ajax=",
	}
	
	for _, keyword := range ajaxKeywords {
		if strings.Contains(urlLower, keyword) {
			return true
		}
	}
	
	// 2. 检查PHP动态页面（带参数）
	if strings.Contains(urlLower, ".php?") {
		return true
	}
	
	// 3. 检查请求头
	if headers != nil {
		// XMLHttpRequest 标识
		if xhrHeader, ok := headers["X-Requested-With"]; ok {
			if xhrStr, ok := xhrHeader.(string); ok && xhrStr == "XMLHttpRequest" {
				return true
			}
		}
		
		// Fetch API 特征 - Accept头包含json或xml
		if acceptHeader, ok := headers["Accept"]; ok {
			if acceptStr, ok := acceptHeader.(string); ok {
				acceptStrLower := strings.ToLower(acceptStr)
				if strings.Contains(acceptStrLower, "application/json") ||
					strings.Contains(acceptStrLower, "text/xml") ||
					strings.Contains(acceptStrLower, "application/xml") ||
					strings.Contains(acceptStrLower, "application/javascript") {
					return true
				}
			}
		}
		
		// Content-Type检查
		if ctHeader, ok := headers["Content-Type"]; ok {
			if ctStr, ok := ctHeader.(string); ok {
				ctStrLower := strings.ToLower(ctStr)
				if strings.Contains(ctStrLower, "application/json") ||
					strings.Contains(ctStrLower, "text/xml") ||
					strings.Contains(ctStrLower, "application/xml") ||
					strings.Contains(ctStrLower, "application/x-www-form-urlencoded") {
					return true
				}
			}
		}
	}
	
	// 4. POST/PUT/DELETE/PATCH 请求很可能是AJAX
	if method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH" {
		return true
	}
	
	return false
}

// addURL 添加URL（线程安全）
func (ai *AjaxInterceptor) addURL(url string) {
	ai.mutex.Lock()
	defer ai.mutex.Unlock()
	
	// 过滤非目标域名的URL
	if ai.targetDomain != "" && !strings.Contains(url, ai.targetDomain) {
		return
	}
	
	// 去重
	for _, existingURL := range ai.interceptedURLs {
		if existingURL == url {
			return
		}
	}
	
	ai.interceptedURLs = append(ai.interceptedURLs, url)
	fmt.Printf("  [AJAX拦截] 发现AJAX请求: %s\n", url)
}

// GetInterceptedURLs 获取拦截的URL
func (ai *AjaxInterceptor) GetInterceptedURLs() []string {
	ai.mutex.Lock()
	defer ai.mutex.Unlock()
	
	// 返回副本
	result := make([]string, len(ai.interceptedURLs))
	copy(result, ai.interceptedURLs)
	return result
}

// GetStatistics 获取统计信息
func (ai *AjaxInterceptor) GetStatistics() map[string]int {
	ai.mutex.Lock()
	defer ai.mutex.Unlock()
	
	return map[string]int{
		"total_ajax_requests": len(ai.interceptedURLs),
	}
}

// Clear 清空拦截记录
func (ai *AjaxInterceptor) Clear() {
	ai.mutex.Lock()
	defer ai.mutex.Unlock()
	
	ai.interceptedURLs = make([]string, 0)
}

