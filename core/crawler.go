package core

import (
	"net/url"
	"spider-golang/config"
)

// Result 爬取结果
type Result struct {
	URL         string
	StatusCode  int
	ContentType string
	Links       []string
	Assets      []string
	Forms       []Form
	APIs        []string
	
	// POST请求数据
	POSTRequests []POSTRequest // POST请求列表（包含完整参数）
	
	// 用于高级检测
	HTMLContent string            // HTML内容（用于技术栈和敏感信息检测）
	Headers     map[string]string // HTTP响应头
	
	// DOM相似度检测
	IsSimilar    bool   // 是否与已爬取的页面相似
	SimilarToURL string // 相似的页面URL
}

// POSTRequest POST请求数据
type POSTRequest struct {
	URL          string            // POST请求的URL
	Method       string            // 请求方法（POST/PUT/PATCH等）
	Parameters   map[string]string // POST参数（key-value）
	Body         string            // 完整的请求体
	ContentType  string            // Content-Type（application/x-www-form-urlencoded, multipart/form-data等）
	Response     *POSTResponse     // POST请求的响应（如果已提交）
	FromForm     bool              // 是否来自表单
	FormAction   string            // 原始表单action
}

// Form 表单信息
type Form struct {
	Action string
	Method string
	Fields []FormField
}

// FormField 表单字段
type FormField struct {
	Name     string
	Type     string
	Value    string
	Required bool
}

// POSTResponse POST请求的响应
type POSTResponse struct {
	StatusCode  int               // 响应状态码
	Headers     map[string]string // 响应头
	Body        string            // 响应体
	NewURLs     []string          // 从响应中发现的新URL
	RedirectURL string            // 重定向URL（如果有）
}

// Crawler 爬虫接口
type Crawler interface {
	// Crawl 执行爬取
	Crawl(url *url.URL) (*Result, error)
	
	// Configure 配置爬虫
	Configure(config *config.Config)
	
	// Stop 停止爬取
	Stop()
}

// StaticCrawler 静态爬虫接口
type StaticCrawler interface {
	Crawler
	
	// ParseHTML 解析HTML内容
	ParseHTML(htmlContent string, baseURL *url.URL) (*Result, error)
}

// DynamicCrawler 动态爬虫接口
type DynamicCrawler interface {
	Crawler
	
	// ExecuteJS 执行JavaScript
	ExecuteJS(script string) (interface{}, error)
}