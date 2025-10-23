package core

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

// HiddenPathDiscovery 隐藏路径发现器
type HiddenPathDiscovery struct {
	client    *http.Client
	baseURL   string
	userAgent string
	mutex     sync.Mutex
	results   []string
}

// NewHiddenPathDiscovery 创建隐藏路径发现器
func NewHiddenPathDiscovery(baseURL, userAgent string) *HiddenPathDiscovery {
	return &HiddenPathDiscovery{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:   baseURL,
		userAgent: userAgent,
		results:   make([]string, 0),
	}
}

// DiscoverAllHiddenPaths 发现所有隐藏路径
func (hpd *HiddenPathDiscovery) DiscoverAllHiddenPaths() []string {
	var wg sync.WaitGroup
	
	// 同时运行多个发现任务
	wg.Add(6)
	
	go func() {
		defer wg.Done()
		hpd.discoverFromRobotsTxt()
	}()
	
	go func() {
		defer wg.Done()
		hpd.discoverFromSitemap()
	}()
	
	go func() {
		defer wg.Done()
		hpd.discoverBackupFiles()
	}()
	
	go func() {
		defer wg.Done()
		hpd.discoverConfigFiles()
	}()
	
	go func() {
		defer wg.Done()
		hpd.discoverAdminPaths()
	}()
	
	go func() {
		defer wg.Done()
		hpd.discoverCommonFiles()
	}()
	
	wg.Wait()
	
	return hpd.GetResults()
}

// discoverFromRobotsTxt 从robots.txt发现路径
func (hpd *HiddenPathDiscovery) discoverFromRobotsTxt() {
	robotsURL := hpd.resolveURL("/robots.txt")
	content := hpd.fetchContent(robotsURL)
	
	if content != "" {
		hpd.addResult(fmt.Sprintf("ROBOTS: %s", robotsURL))
		
		// 解析robots.txt中的路径
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Disallow:") || strings.HasPrefix(line, "Allow:") {
				path := strings.TrimSpace(strings.Split(line, ":")[1])
				if path != "/" && path != "" {
					fullURL := hpd.resolveURL(path)
					hpd.addResult(fmt.Sprintf("ROBOTS_PATH: %s", fullURL))
				}
			}
			if strings.HasPrefix(line, "Sitemap:") {
				sitemapURL := strings.TrimSpace(strings.Split(line, ":")[1])
				if strings.HasPrefix(sitemapURL, "http") {
					hpd.addResult(fmt.Sprintf("SITEMAP: %s", sitemapURL))
				}
			}
		}
	}
}

// discoverFromSitemap 从sitemap发现路径
func (hpd *HiddenPathDiscovery) discoverFromSitemap() {
	sitemapURLs := []string{
		"/sitemap.xml",
		"/sitemap_index.xml",
		"/sitemap.txt",
		"/sitemaps.xml",
		"/sitemap/sitemap.xml",
	}
	
	for _, sitemapPath := range sitemapURLs {
		sitemapURL := hpd.resolveURL(sitemapPath)
		content := hpd.fetchContent(sitemapURL)
		
		if content != "" {
			hpd.addResult(fmt.Sprintf("SITEMAP: %s", sitemapURL))
			
			// 从sitemap提取URL
			urlRegex := regexp.MustCompile(`<loc>(.*?)</loc>`)
			matches := urlRegex.FindAllStringSubmatch(content, -1)
			
			for _, match := range matches {
				if len(match) > 1 {
					discoveredURL := match[1]
					hpd.addResult(fmt.Sprintf("SITEMAP_URL: %s", discoveredURL))
				}
			}
			
			// 查找其他sitemap引用
			sitemapRegex := regexp.MustCompile(`<sitemap>\s*<loc>(.*?)</loc>`)
			sitemapMatches := sitemapRegex.FindAllStringSubmatch(content, -1)
			
			for _, match := range sitemapMatches {
				if len(match) > 1 {
					nestedSitemapURL := match[1]
					hpd.addResult(fmt.Sprintf("NESTED_SITEMAP: %s", nestedSitemapURL))
				}
			}
		}
	}
}

// discoverBackupFiles 发现备份文件
func (hpd *HiddenPathDiscovery) discoverBackupFiles() {
	parsedURL, err := url.Parse(hpd.baseURL)
	if err != nil {
		return
	}
	
	basePath := parsedURL.Path
	if basePath == "" || basePath == "/" {
		basePath = "/index"
	}
	
	// 备份文件扩展名
	backupExtensions := []string{
		".bak", ".backup", ".old", ".orig", ".save", ".tmp", ".swp",
		".copy", ".1", ".2", ".3", ".~", ".backup~", ".old~",
		".tar", ".tar.gz", ".zip", ".rar", ".7z",
	}
	
	// 常见的备份文件名
	backupFiles := []string{
		"/backup.zip", "/backup.tar", "/backup.tar.gz", "/site.zip",
		"/www.zip", "/web.zip", "/database.sql", "/db.sql", "/dump.sql",
		"/backup.sql", "/site.sql", "/config.bak", "/config.backup",
		"/.htaccess.bak", "/.htaccess.old", "/web.config.bak",
		"/application.properties.bak", "/settings.py.bak",
	}
	
	// 测试当前路径的备份版本
	for _, ext := range backupExtensions {
		testURL := hpd.baseURL + ext
		if hpd.checkPath(testURL) {
			hpd.addResult(fmt.Sprintf("BACKUP_FILE: %s", testURL))
		}
		
		// 测试去掉扩展名后的备份版本
		if strings.Contains(basePath, ".") {
			nameWithoutExt := strings.TrimSuffix(basePath, path.Ext(basePath))
			testURL := hpd.resolveURL(nameWithoutExt + ext)
			if hpd.checkPath(testURL) {
				hpd.addResult(fmt.Sprintf("BACKUP_FILE: %s", testURL))
			}
		}
	}
	
	// 测试常见备份文件
	for _, backupFile := range backupFiles {
		testURL := hpd.resolveURL(backupFile)
		if hpd.checkPath(testURL) {
			hpd.addResult(fmt.Sprintf("BACKUP_FILE: %s", testURL))
		}
	}
}

// discoverConfigFiles 发现配置文件
func (hpd *HiddenPathDiscovery) discoverConfigFiles() {
	configFiles := []string{
		// Web服务器配置
		"/.htaccess", "/.htpasswd", "/web.config", "/httpd.conf",
		"/apache2.conf", "/nginx.conf", "/.server", "/.apache",
		
		// 应用配置文件
		"/config.php", "/config.inc.php", "/config.py", "/settings.py",
		"/config.json", "/config.xml", "/config.yml", "/config.yaml",
		"/application.properties", "/application.yml", "/application.yaml",
		"/app.config", "/web.xml", "/struts.xml", "/spring.xml",
		
		// 数据库配置
		"/database.php", "/db.php", "/database.yml", "/database.json",
		"/connection.php", "/connect.php", "/mysql.php", "/pgsql.php",
		
		// 框架配置文件
		"/.env", "/.env.local", "/.env.production", "/.env.development",
		"/composer.json", "/package.json", "/requirements.txt", "/Gemfile",
		"/pom.xml", "/build.xml", "/gulpfile.js", "/webpack.config.js",
		
		// 版本控制
		"/.git/config", "/.svn/entries", "/.hg/hgrc", "/CVS/Entries",
		"/.gitignore", "/.gitconfig", "/.git/HEAD", "/.git/index",
		
		// IDE和编辑器文件
		"/.vscode/settings.json", "/.idea/workspace.xml", "/nbproject/project.xml",
		"/.project", "/.classpath", "/.settings", "/.metadata",
		
		// 系统文件
		"/readme.txt", "/README.md", "/CHANGELOG", "/LICENSE", "/VERSION",
		"/install.txt", "/INSTALL", "/UPGRADE", "/TODO", "/NOTICE",
		
		// 调试和测试文件
		"/phpinfo.php", "/info.php", "/test.php", "/debug.php",
		"/status.php", "/health.php", "/ping.php", "/check.php",
	}
	
	for _, configFile := range configFiles {
		testURL := hpd.resolveURL(configFile)
		if hpd.checkPath(testURL) {
			hpd.addResult(fmt.Sprintf("CONFIG_FILE: %s", testURL))
		}
	}
}

// discoverAdminPaths 发现管理路径
func (hpd *HiddenPathDiscovery) discoverAdminPaths() {
	adminPaths := []string{
		// 通用管理路径
		"/admin", "/admin/", "/admin/login", "/admin/index.php",
		"/administrator", "/administrator/", "/administration",
		"/manage", "/manager", "/management", "/control", "/panel",
		
		// CMS管理路径
		"/wp-admin", "/wp-admin/", "/wp-login.php", "/wp-config.php",
		"/drupal/admin", "/joomla/administrator", "/magento/admin",
		"/typo3/", "/ghost/admin", "/craft/admin",
		
		// 框架管理路径
		"/laravel/admin", "/symfony/admin", "/django/admin",
		"/rails/admin", "/spring/admin", "/struts/admin",
		
		// 服务管理路径
		"/phpmyadmin", "/phpMyAdmin", "/pma", "/mysql", "/database",
		"/cpanel", "/plesk", "/webmin", "/directadmin",
		
		// API管理路径
		"/api/admin", "/api/v1/admin", "/api/v2/admin", "/rest/admin",
		"/graphql/admin", "/swagger", "/swagger-ui", "/api-docs",
		
		// 监控和统计
		"/stats", "/statistics", "/analytics", "/metrics", "/monitor",
		"/status", "/health", "/info", "/debug", "/logs",
		
		// 备份和工具
		"/backup", "/backups", "/tools", "/utilities", "/scripts",
		"/cron", "/jobs", "/tasks", "/queue", "/cache",
		
		// 特殊路径
		"/hidden", "/secret", "/private", "/internal", "/restricted",
		"/dev", "/test", "/demo", "/staging", "/beta",
		
		// 文件管理
		"/filemanager", "/files", "/uploads", "/media", "/assets",
		"/documents", "/downloads", "/storage", "/data",
		
		// 系统目录
		"/system", "/includes", "/inc", "/lib", "/libs", "/vendor",
		"/node_modules", "/bower_components", "/.well-known",
	}
	
	for _, adminPath := range adminPaths {
		testURL := hpd.resolveURL(adminPath)
		if hpd.checkPath(testURL) {
			hpd.addResult(fmt.Sprintf("ADMIN_PATH: %s", testURL))
		}
	}
}

// discoverCommonFiles 发现常见文件
func (hpd *HiddenPathDiscovery) discoverCommonFiles() {
	commonFiles := []string{
		// 常见页面
		"/login", "/logout", "/signin", "/signup", "/register",
		"/contact", "/about", "/help", "/support", "/faq",
		"/search", "/profile", "/account", "/settings", "/preferences",
		
		// 功能页面
		"/upload", "/download", "/export", "/import", "/backup",
		"/reset", "/forgot", "/recover", "/activate", "/verify",
		
		// API端点
		"/api", "/api/", "/api/v1", "/api/v2", "/rest", "/graphql",
		"/json", "/xml", "/rss", "/feed", "/sitemap", "/robots",
		
		// 安全相关
		"/security", "/auth", "/oauth", "/sso", "/ldap", "/saml",
		"/cert", "/ssl", "/tls", "/keys", "/tokens", "/sessions",
		
		// 开发相关
		"/dev", "/development", "/staging", "/production", "/test",
		"/debug", "/trace", "/log", "/logs", "/error", "/errors",
		
		// 数据相关
		"/data", "/database", "/db", "/sql", "/query", "/report",
		"/export", "/csv", "/excel", "/pdf", "/json", "/xml",
		
		// 多媒体
		"/images", "/img", "/pics", "/photos", "/gallery", "/media",
		"/videos", "/audio", "/docs", "/files", "/attachments",
		
		// 特殊功能
		"/proxy", "/redirect", "/forward", "/bounce", "/link",
		"/short", "/url", "/goto", "/exit", "/out", "/external",
	}
	
	for _, commonFile := range commonFiles {
		testURL := hpd.resolveURL(commonFile)
		if hpd.checkPath(testURL) {
			hpd.addResult(fmt.Sprintf("COMMON_PATH: %s", testURL))
		}
	}
}

// checkPath 检查路径是否存在
func (hpd *HiddenPathDiscovery) checkPath(testURL string) bool {
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return false
	}
	
	if hpd.userAgent != "" {
		req.Header.Set("User-Agent", hpd.userAgent)
	}
	
	resp, err := hpd.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// 检查状态码，200, 301, 302, 403 都表示路径存在
	return resp.StatusCode == 200 || resp.StatusCode == 301 || 
		   resp.StatusCode == 302 || resp.StatusCode == 403 || 
		   resp.StatusCode == 401 || resp.StatusCode == 500
}

// fetchContent 获取URL内容
func (hpd *HiddenPathDiscovery) fetchContent(url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	
	if hpd.userAgent != "" {
		req.Header.Set("User-Agent", hpd.userAgent)
	}
	
	resp, err := hpd.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return ""
	}
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	
	return string(body)
}

// resolveURL 解析相对URL为绝对URL
func (hpd *HiddenPathDiscovery) resolveURL(relativePath string) string {
	if strings.HasPrefix(relativePath, "http") {
		return relativePath
	}
	
	// 解析baseURL，提取scheme和host
	parsedBase, err := url.Parse(hpd.baseURL)
	if err != nil {
		// 如果解析失败，使用简单拼接（向后兼容）
		baseURL := strings.TrimSuffix(hpd.baseURL, "/")
		if !strings.HasPrefix(relativePath, "/") {
			relativePath = "/" + relativePath
		}
		return baseURL + relativePath
	}
	
	// 使用scheme和host构建基础URL
	baseURL := parsedBase.Scheme + "://" + parsedBase.Host
	
	// 确保relativePath以/开头
	if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}
	
	return baseURL + relativePath
}

// addResult 添加发现结果
func (hpd *HiddenPathDiscovery) addResult(result string) {
	hpd.mutex.Lock()
	defer hpd.mutex.Unlock()
	
	// 检查是否已存在
	for _, existing := range hpd.results {
		if existing == result {
			return
		}
	}
	
	hpd.results = append(hpd.results, result)
}

// GetResults 获取所有发现结果
func (hpd *HiddenPathDiscovery) GetResults() []string {
	hpd.mutex.Lock()
	defer hpd.mutex.Unlock()
	
	// 返回副本
	results := make([]string, len(hpd.results))
	copy(results, hpd.results)
	
	return results
}

// DiscoverJSEndpoints 从JavaScript文件中发现端点
func (hpd *HiddenPathDiscovery) DiscoverJSEndpoints(jsContent string) []string {
	endpoints := make([]string, 0)
	
	// API端点模式
	patterns := []string{
		// URL字符串
		`['"]([^'"]*(?:/api/|/API/|/rest/|/graphql/|/v\d+/)[^'"]*)['"]`,
		// 路径字符串
		`['"]([^'"]*\.(?:php|asp|aspx|jsp|do|action|cfm|cgi|pl|py)[^'"]*)['"]`,
		// AJAX调用
		`\.(?:get|post|put|delete|ajax)\s*\(\s*['"]([^'"]+)['"]`,
		// fetch调用
		`fetch\s*\(\s*['"]([^'"]+)['"]`,
		// XMLHttpRequest
		`\.open\s*\(\s*['"][^'"]*['"]\s*,\s*['"]([^'"]+)['"]`,
		// 相对路径
		`['"]([^'"]*(?:\?|&)[^'"]*)['"]`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsContent, -1)
		
		for _, match := range matches {
			if len(match) > 1 {
				endpoint := match[1]
				// 过滤掉明显不是端点的内容
				if hpd.isValidEndpoint(endpoint) {
					fullURL := hpd.resolveURL(endpoint)
					endpoints = append(endpoints, fullURL)
				}
			}
		}
	}
	
	return endpoints
}

// isValidEndpoint 检查是否为有效端点
func (hpd *HiddenPathDiscovery) isValidEndpoint(endpoint string) bool {
	// 过滤条件
	if len(endpoint) < 3 || len(endpoint) > 200 {
		return false
	}
	
	// 不能包含特殊字符
	invalidChars := []string{" ", "\t", "\n", "javascript:", "mailto:", "tel:", "#"}
	for _, char := range invalidChars {
		if strings.Contains(endpoint, char) {
			return false
		}
	}
	
	// 必须是路径格式或包含特定模式
	return strings.HasPrefix(endpoint, "/") || 
		   strings.Contains(endpoint, "api") ||
		   strings.Contains(endpoint, ".php") ||
		   strings.Contains(endpoint, ".asp") ||
		   strings.Contains(endpoint, "?") ||
		   strings.Contains(endpoint, "=")
}
