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