package core

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// SitemapCrawler sitemap爬取器
type SitemapCrawler struct {
	client  *http.Client
	timeout time.Duration
}

// SitemapURL sitemap中的URL条目
type SitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}

// URLSet sitemap的urlset结构
type URLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []SitemapURL `xml:"url"`
}

// SitemapIndex sitemap索引
type SitemapIndex struct {
	XMLName  xml.Name  `xml:"sitemapindex"`
	Sitemaps []Sitemap `xml:"sitemap"`
}

// Sitemap sitemap条目
type Sitemap struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// RobotsInfo robots.txt信息
type RobotsInfo struct {
	DisallowPaths []string // Disallow路径
	AllowPaths    []string // Allow路径
	SitemapURLs   []string // Sitemap URL
}

// NewSitemapCrawler 创建sitemap爬取器
func NewSitemapCrawler() *SitemapCrawler {
	return &SitemapCrawler{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// 最多跟随5个重定向
				if len(via) >= 5 {
					return fmt.Errorf("stopped after 5 redirects")
				}
				return nil
			},
		},
		timeout: 10 * time.Second,
	}
}

// CrawlSitemap 爬取sitemap.xml
func (sc *SitemapCrawler) CrawlSitemap(baseURL string) []string {
	allURLs := make([]string, 0)
	seen := make(map[string]bool)
	
	// 常见的sitemap路径
	sitemapPaths := []string{
		"/sitemap.xml",
		"/sitemap_index.xml",
		"/sitemap.php",
		"/sitemap/",
		"/sitemap/sitemap.xml",
		"/sitemaps.xml",
	}
	
	for _, path := range sitemapPaths {
		sitemapURL := strings.TrimSuffix(baseURL, "/") + path
		
		if seen[sitemapURL] {
			continue
		}
		seen[sitemapURL] = true
		
		urls := sc.fetchSitemap(sitemapURL)
		allURLs = append(allURLs, urls...)
	}
	
	return allURLs
}

// fetchSitemap 获取单个sitemap
func (sc *SitemapCrawler) fetchSitemap(sitemapURL string) []string {
	urls := make([]string, 0)
	
	resp, err := sc.client.Get(sitemapURL)
	if err != nil || resp.StatusCode != 200 {
		return urls
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return urls
	}
	
	// 尝试解析为URLSet
	var urlset URLSet
	if err := xml.Unmarshal(body, &urlset); err == nil && len(urlset.URLs) > 0 {
		for _, url := range urlset.URLs {
			if url.Loc != "" {
				urls = append(urls, url.Loc)
			}
		}
		return urls
	}
	
	// 尝试解析为SitemapIndex
	var index SitemapIndex
	if err := xml.Unmarshal(body, &index); err == nil && len(index.Sitemaps) > 0 {
		// 递归获取子sitemap
		for _, sitemap := range index.Sitemaps {
			if sitemap.Loc != "" {
				subURLs := sc.fetchSitemap(sitemap.Loc)
				urls = append(urls, subURLs...)
			}
		}
		return urls
	}
	
	return urls
}

// CrawlRobotsTxt 爬取robots.txt
func (sc *SitemapCrawler) CrawlRobotsTxt(baseURL string) *RobotsInfo {
	info := &RobotsInfo{
		DisallowPaths: make([]string, 0),
		AllowPaths:    make([]string, 0),
		SitemapURLs:   make([]string, 0),
	}
	
	robotsURL := strings.TrimSuffix(baseURL, "/") + "/robots.txt"
	
	resp, err := sc.client.Get(robotsURL)
	if err != nil || resp.StatusCode != 200 {
		return info
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return info
	}
	
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Disallow: /admin/ - 渗透测试的关键目标！
		if strings.HasPrefix(line, "Disallow:") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "Disallow:"))
			if path != "" && path != "/" && path != "/*" {
				// 构造完整URL
				fullURL := strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(path, "/")
				info.DisallowPaths = append(info.DisallowPaths, fullURL)
			}
		}
		
		// Allow: /public/
		if strings.HasPrefix(line, "Allow:") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "Allow:"))
			if path != "" && path != "/" {
				fullURL := strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(path, "/")
				info.AllowPaths = append(info.AllowPaths, fullURL)
			}
		}
		
		// Sitemap: http://example.com/sitemap.xml
		if strings.HasPrefix(line, "Sitemap:") {
			sitemapURL := strings.TrimSpace(strings.TrimPrefix(line, "Sitemap:"))
			if sitemapURL != "" {
				info.SitemapURLs = append(info.SitemapURLs, sitemapURL)
			}
		}
	}
	
	return info
}

// GetAllURLs 获取sitemap和robots.txt中的所有URL
func (sc *SitemapCrawler) GetAllURLs(baseURL string) ([]string, *RobotsInfo) {
	allURLs := make([]string, 0)
	
	// 1. 爬取sitemap.xml
	sitemapURLs := sc.CrawlSitemap(baseURL)
	allURLs = append(allURLs, sitemapURLs...)
	
	// 2. 爬取robots.txt
	robotsInfo := sc.CrawlRobotsTxt(baseURL)
	
	// 从robots.txt中的sitemap继续爬取
	for _, sitemapURL := range robotsInfo.SitemapURLs {
		urls := sc.fetchSitemap(sitemapURL)
		allURLs = append(allURLs, urls...)
	}
	
	// Disallow路径是渗透测试的重点
	allURLs = append(allURLs, robotsInfo.DisallowPaths...)
	allURLs = append(allURLs, robotsInfo.AllowPaths...)
	
	return allURLs, robotsInfo
}

