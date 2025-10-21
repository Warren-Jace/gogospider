package core

import (
	"net/http"
	"regexp"
	"strings"
)

// TechInfo 技术信息
type TechInfo struct {
	Name       string   // 技术名称
	Version    string   // 版本
	Category   string   // 分类
	Confidence int      // 置信度 (0-100)
	Evidence   []string // 证据
}

// TechStackDetector 技术栈检测器
type TechStackDetector struct {
	detectionRules map[string]*DetectionRule
}

// DetectionRule 检测规则
type DetectionRule struct {
	Name     string
	Category string
	
	// HTTP头检测
	Headers map[string]*regexp.Regexp
	
	// HTML内容检测
	HTMLPatterns []*regexp.Regexp
	
	// Meta标签检测
	MetaPatterns map[string]*regexp.Regexp
	
	// Cookie检测
	Cookies []string
	
	// JavaScript检测
	JSPatterns []*regexp.Regexp
	
	// URL路径检测
	PathPatterns []*regexp.Regexp
	
	// 版本提取
	VersionPattern *regexp.Regexp
}

// NewTechStackDetector 创建技术栈检测器
func NewTechStackDetector() *TechStackDetector {
	tsd := &TechStackDetector{
		detectionRules: make(map[string]*DetectionRule),
	}
	
	tsd.initializeRules()
	
	return tsd
}

// initializeRules 初始化检测规则
func (tsd *TechStackDetector) initializeRules() {
	// === 前端框架 ===
	
	// React
	tsd.addRule(&DetectionRule{
		Name:     "React",
		Category: "前端框架",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`<div[^>]+id=["\']root["\']`),
			regexp.MustCompile(`react\.js`),
			regexp.MustCompile(`react-dom\.js`),
			regexp.MustCompile(`data-reactroot`),
			regexp.MustCompile(`__REACT_DEVTOOLS_GLOBAL_HOOK__`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`React\.createElement`),
			regexp.MustCompile(`ReactDOM\.render`),
		},
		VersionPattern: regexp.MustCompile(`react@([\d.]+)`),
	})
	
	// Vue.js
	tsd.addRule(&DetectionRule{
		Name:     "Vue.js",
		Category: "前端框架",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`vue\.js`),
			regexp.MustCompile(`v-bind`),
			regexp.MustCompile(`v-model`),
			regexp.MustCompile(`v-if`),
			regexp.MustCompile(`v-for`),
			regexp.MustCompile(`@click`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`new Vue\(`),
			regexp.MustCompile(`Vue\.component`),
			regexp.MustCompile(`createApp\(`),
		},
		VersionPattern: regexp.MustCompile(`vue@([\d.]+)`),
	})
	
	// Angular
	tsd.addRule(&DetectionRule{
		Name:     "Angular",
		Category: "前端框架",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`ng-app`),
			regexp.MustCompile(`ng-controller`),
			regexp.MustCompile(`ng-model`),
			regexp.MustCompile(`angular\.js`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`angular\.module`),
		},
		VersionPattern: regexp.MustCompile(`angular@([\d.]+)`),
	})
	
	// jQuery
	tsd.addRule(&DetectionRule{
		Name:     "jQuery",
		Category: "JavaScript库",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`jquery\.js`),
			regexp.MustCompile(`jquery\.min\.js`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\$\(document\)\.ready`),
			regexp.MustCompile(`jQuery\.`),
		},
		VersionPattern: regexp.MustCompile(`jquery[/-]([\d.]+)`),
	})
	
	// === 后端框架/CMS ===
	
	// WordPress
	tsd.addRule(&DetectionRule{
		Name:     "WordPress",
		Category: "CMS",
		Headers: map[string]*regexp.Regexp{
			"X-Powered-By": regexp.MustCompile(`WordPress`),
		},
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`/wp-content/`),
			regexp.MustCompile(`/wp-includes/`),
			regexp.MustCompile(`wp-json`),
			regexp.MustCompile(`wordpress`),
		},
		PathPatterns: []*regexp.Regexp{
			regexp.MustCompile(`/wp-admin/`),
			regexp.MustCompile(`/wp-login\.php`),
		},
		MetaPatterns: map[string]*regexp.Regexp{
			"generator": regexp.MustCompile(`WordPress ([\d.]+)`),
		},
		VersionPattern: regexp.MustCompile(`WordPress ([\d.]+)`),
	})
	
	// Laravel
	tsd.addRule(&DetectionRule{
		Name:     "Laravel",
		Category: "PHP框架",
		Headers: map[string]*regexp.Regexp{
			"Set-Cookie": regexp.MustCompile(`laravel_session`),
		},
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`csrf-token`),
			regexp.MustCompile(`laravel`),
		},
		Cookies: []string{"laravel_session", "XSRF-TOKEN"},
	})
	
	// Django
	tsd.addRule(&DetectionRule{
		Name:     "Django",
		Category: "Python框架",
		Headers: map[string]*regexp.Regexp{
			"Set-Cookie": regexp.MustCompile(`csrftoken`),
		},
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`csrfmiddlewaretoken`),
		},
		Cookies: []string{"csrftoken", "sessionid"},
	})
	
	// Spring Boot
	tsd.addRule(&DetectionRule{
		Name:     "Spring Boot",
		Category: "Java框架",
		Headers: map[string]*regexp.Regexp{
			"X-Application-Context": regexp.MustCompile(`.+`),
		},
		Cookies: []string{"JSESSIONID"},
		PathPatterns: []*regexp.Regexp{
			regexp.MustCompile(`/actuator/`),
		},
	})
	
	// === Web服务器 ===
	
	// Nginx
	tsd.addRule(&DetectionRule{
		Name:     "Nginx",
		Category: "Web服务器",
		Headers: map[string]*regexp.Regexp{
			"Server": regexp.MustCompile(`nginx`),
		},
		VersionPattern: regexp.MustCompile(`nginx/([\d.]+)`),
	})
	
	// Apache
	tsd.addRule(&DetectionRule{
		Name:     "Apache",
		Category: "Web服务器",
		Headers: map[string]*regexp.Regexp{
			"Server": regexp.MustCompile(`Apache`),
		},
		VersionPattern: regexp.MustCompile(`Apache/([\d.]+)`),
	})
	
	// IIS
	tsd.addRule(&DetectionRule{
		Name:     "Microsoft IIS",
		Category: "Web服务器",
		Headers: map[string]*regexp.Regexp{
			"Server": regexp.MustCompile(`Microsoft-IIS`),
		},
		VersionPattern: regexp.MustCompile(`Microsoft-IIS/([\d.]+)`),
	})
	
	// === 编程语言 ===
	
	// PHP
	tsd.addRule(&DetectionRule{
		Name:     "PHP",
		Category: "编程语言",
		Headers: map[string]*regexp.Regexp{
			"X-Powered-By": regexp.MustCompile(`PHP`),
		},
		PathPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\.php`),
		},
		Cookies: []string{"PHPSESSID"},
		VersionPattern: regexp.MustCompile(`PHP/([\d.]+)`),
	})
	
	// ASP.NET
	tsd.addRule(&DetectionRule{
		Name:     "ASP.NET",
		Category: "编程语言",
		Headers: map[string]*regexp.Regexp{
			"X-Powered-By":     regexp.MustCompile(`ASP\.NET`),
			"X-AspNet-Version": regexp.MustCompile(`.+`),
		},
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`__VIEWSTATE`),
		},
		Cookies: []string{"ASP.NET_SessionId"},
		VersionPattern: regexp.MustCompile(`ASP\.NET ([\d.]+)`),
	})
	
	// Node.js
	tsd.addRule(&DetectionRule{
		Name:     "Node.js",
		Category: "编程语言",
		Headers: map[string]*regexp.Regexp{
			"X-Powered-By": regexp.MustCompile(`Express`),
		},
	})
	
	// === CDN/云服务 ===
	
	// Cloudflare
	tsd.addRule(&DetectionRule{
		Name:     "Cloudflare",
		Category: "CDN",
		Headers: map[string]*regexp.Regexp{
			"Server":         regexp.MustCompile(`cloudflare`),
			"CF-RAY":         regexp.MustCompile(`.+`),
			"CF-Cache-Status": regexp.MustCompile(`.+`),
		},
		Cookies: []string{"__cfduid"},
	})
	
	// 阿里云
	tsd.addRule(&DetectionRule{
		Name:     "阿里云CDN",
		Category: "CDN",
		Headers: map[string]*regexp.Regexp{
			"Ali-Swift-Global-Savetime": regexp.MustCompile(`.+`),
			"Eagleid":                   regexp.MustCompile(`.+`),
		},
	})
	
	// 腾讯云
	tsd.addRule(&DetectionRule{
		Name:     "腾讯云CDN",
		Category: "CDN",
		Headers: map[string]*regexp.Regexp{
			"X-NWS-LOG-UUID": regexp.MustCompile(`.+`),
		},
	})
	
	// === 安全组件 ===
	
	// WAF检测
	tsd.addRule(&DetectionRule{
		Name:     "WAF (Web应用防火墙)",
		Category: "安全组件",
		Headers: map[string]*regexp.Regexp{
			"X-WAF-Event": regexp.MustCompile(`.+`),
		},
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`您的请求被拦截`),
			regexp.MustCompile(`Request blocked`),
		},
	})
	
	// === 更多前端框架 ===
	
	// Bootstrap
	tsd.addRule(&DetectionRule{
		Name:     "Bootstrap",
		Category: "CSS框架",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`bootstrap\.css`),
			regexp.MustCompile(`bootstrap\.min\.css`),
			regexp.MustCompile(`class="[^"]*\bcontainer\b[^"]*"`),
			regexp.MustCompile(`class="[^"]*\brow\b[^"]*"`),
			regexp.MustCompile(`class="[^"]*\bcol-\w+-\d+\b[^"]*"`),
		},
		VersionPattern: regexp.MustCompile(`bootstrap[/-]v?([\d.]+)`),
	})
	
	// Tailwind CSS
	tsd.addRule(&DetectionRule{
		Name:     "Tailwind CSS",
		Category: "CSS框架",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`tailwind\.css`),
			regexp.MustCompile(`class="[^"]*\b(flex|grid|hidden|block|inline)\b[^"]*"`),
		},
	})
	
	// Next.js
	tsd.addRule(&DetectionRule{
		Name:     "Next.js",
		Category: "前端框架",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`__NEXT_DATA__`),
			regexp.MustCompile(`_next/static`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`__NEXT_DATA__`),
		},
		PathPatterns: []*regexp.Regexp{
			regexp.MustCompile(`/_next/`),
		},
	})
	
	// Nuxt.js
	tsd.addRule(&DetectionRule{
		Name:     "Nuxt.js",
		Category: "前端框架",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`__NUXT__`),
			regexp.MustCompile(`_nuxt/`),
		},
		PathPatterns: []*regexp.Regexp{
			regexp.MustCompile(`/_nuxt/`),
		},
	})
	
	// Svelte
	tsd.addRule(&DetectionRule{
		Name:     "Svelte",
		Category: "前端框架",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`svelte`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`SvelteComponent`),
		},
	})
	
	// === 更多CMS系统 ===
	
	// Joomla
	tsd.addRule(&DetectionRule{
		Name:     "Joomla",
		Category: "CMS",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`/components/com_`),
			regexp.MustCompile(`Joomla!`),
		},
		MetaPatterns: map[string]*regexp.Regexp{
			"generator": regexp.MustCompile(`Joomla! - ([\d.]+)`),
		},
		VersionPattern: regexp.MustCompile(`Joomla! - ([\d.]+)`),
	})
	
	// Drupal
	tsd.addRule(&DetectionRule{
		Name:     "Drupal",
		Category: "CMS",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`Drupal`),
			regexp.MustCompile(`/sites/default/files`),
		},
		MetaPatterns: map[string]*regexp.Regexp{
			"generator": regexp.MustCompile(`Drupal ([\d.]+)`),
		},
		Headers: map[string]*regexp.Regexp{
			"X-Drupal-Cache": regexp.MustCompile(`.+`),
			"X-Generator":    regexp.MustCompile(`Drupal`),
		},
		VersionPattern: regexp.MustCompile(`Drupal ([\d.]+)`),
	})
	
	// Magento
	tsd.addRule(&DetectionRule{
		Name:     "Magento",
		Category: "电商平台",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`Mage\.Cookies`),
			regexp.MustCompile(`/skin/frontend/`),
		},
		Cookies: []string{"frontend"},
	})
	
	// Shopify
	tsd.addRule(&DetectionRule{
		Name:     "Shopify",
		Category: "电商平台",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`cdn\.shopify\.com`),
			regexp.MustCompile(`Shopify\.`),
		},
		Headers: map[string]*regexp.Regexp{
			"X-ShopId": regexp.MustCompile(`.+`),
		},
	})
	
	// === 更多后端框架 ===
	
	// Express.js
	tsd.addRule(&DetectionRule{
		Name:     "Express",
		Category: "Node.js框架",
		Headers: map[string]*regexp.Regexp{
			"X-Powered-By": regexp.MustCompile(`Express`),
		},
	})
	
	// Flask
	tsd.addRule(&DetectionRule{
		Name:     "Flask",
		Category: "Python框架",
		Headers: map[string]*regexp.Regexp{
			"Server": regexp.MustCompile(`Werkzeug`),
		},
		Cookies: []string{"session"},
	})
	
	// Ruby on Rails
	tsd.addRule(&DetectionRule{
		Name:     "Ruby on Rails",
		Category: "Ruby框架",
		Cookies: []string{"_rails_session"},
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`csrf-token`),
		},
	})
	
	// Gin (Go)
	tsd.addRule(&DetectionRule{
		Name:     "Gin",
		Category: "Go框架",
		Headers: map[string]*regexp.Regexp{
			"X-Powered-By": regexp.MustCompile(`Gin`),
		},
	})
	
	// Koa
	tsd.addRule(&DetectionRule{
		Name:     "Koa",
		Category: "Node.js框架",
		Headers: map[string]*regexp.Regexp{
			"X-Powered-By": regexp.MustCompile(`Koa`),
		},
	})
	
	// ThinkPHP
	tsd.addRule(&DetectionRule{
		Name:     "ThinkPHP",
		Category: "PHP框架",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`ThinkPHP`),
		},
		Cookies: []string{"thinkphp_show_page_trace"},
	})
	
	// Yii
	tsd.addRule(&DetectionRule{
		Name:     "Yii",
		Category: "PHP框架",
		Cookies: []string{"YII_CSRF_TOKEN"},
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`Yii Framework`),
		},
	})
	
	// CodeIgniter
	tsd.addRule(&DetectionRule{
		Name:     "CodeIgniter",
		Category: "PHP框架",
		Cookies: []string{"ci_session"},
	})
	
	// === 数据库相关 ===
	
	// MySQL
	tsd.addRule(&DetectionRule{
		Name:     "MySQL",
		Category: "数据库",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`MySQL`),
		},
	})
	
	// MongoDB
	tsd.addRule(&DetectionRule{
		Name:     "MongoDB",
		Category: "数据库",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`MongoDB`),
		},
	})
	
	// Redis
	tsd.addRule(&DetectionRule{
		Name:     "Redis",
		Category: "缓存",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`Redis`),
		},
	})
	
	// === JavaScript库 ===
	
	// Axios
	tsd.addRule(&DetectionRule{
		Name:     "Axios",
		Category: "JavaScript库",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`axios\.js`),
			regexp.MustCompile(`axios\.min\.js`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`axios\.`),
		},
	})
	
	// Lodash
	tsd.addRule(&DetectionRule{
		Name:     "Lodash",
		Category: "JavaScript库",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`lodash\.js`),
			regexp.MustCompile(`lodash\.min\.js`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`_\.`),
		},
	})
	
	// Moment.js
	tsd.addRule(&DetectionRule{
		Name:     "Moment.js",
		Category: "JavaScript库",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`moment\.js`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`moment\(`),
		},
	})
	
	// Chart.js
	tsd.addRule(&DetectionRule{
		Name:     "Chart.js",
		Category: "JavaScript库",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`chart\.js`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`new Chart\(`),
		},
	})
	
	// === 更多CDN ===
	
	// jsDelivr
	tsd.addRule(&DetectionRule{
		Name:     "jsDelivr CDN",
		Category: "CDN",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`cdn\.jsdelivr\.net`),
		},
	})
	
	// unpkg
	tsd.addRule(&DetectionRule{
		Name:     "unpkg CDN",
		Category: "CDN",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`unpkg\.com`),
		},
	})
	
	// cdnjs
	tsd.addRule(&DetectionRule{
		Name:     "cdnjs",
		Category: "CDN",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`cdnjs\.cloudflare\.com`),
		},
	})
	
	// Google Hosted Libraries
	tsd.addRule(&DetectionRule{
		Name:     "Google Hosted Libraries",
		Category: "CDN",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`ajax\.googleapis\.com`),
		},
	})
	
	// === 分析和监控 ===
	
	// Google Analytics
	tsd.addRule(&DetectionRule{
		Name:     "Google Analytics",
		Category: "分析工具",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`google-analytics\.com/analytics\.js`),
			regexp.MustCompile(`googletagmanager\.com/gtag`),
			regexp.MustCompile(`ga\('create'`),
		},
		JSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`ga\(`),
			regexp.MustCompile(`gtag\(`),
		},
	})
	
	// 百度统计
	tsd.addRule(&DetectionRule{
		Name:     "百度统计",
		Category: "分析工具",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`hm\.baidu\.com`),
		},
	})
	
	// 腾讯分析
	tsd.addRule(&DetectionRule{
		Name:     "腾讯分析",
		Category: "分析工具",
		HTMLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`tajs\.qq\.com`),
		},
	})
	
	// === 容器和部署 ===
	
	// Docker
	tsd.addRule(&DetectionRule{
		Name:     "Docker",
		Category: "容器技术",
		Headers: map[string]*regexp.Regexp{
			"X-Powered-By": regexp.MustCompile(`Docker`),
		},
	})
	
	// Kubernetes
	tsd.addRule(&DetectionRule{
		Name:     "Kubernetes",
		Category: "容器编排",
		Headers: map[string]*regexp.Regexp{
			"X-Kubernetes": regexp.MustCompile(`.+`),
		},
	})
	
	// Vercel
	tsd.addRule(&DetectionRule{
		Name:     "Vercel",
		Category: "部署平台",
		Headers: map[string]*regexp.Regexp{
			"X-Vercel-Id": regexp.MustCompile(`.+`),
			"Server":      regexp.MustCompile(`Vercel`),
		},
	})
	
	// Netlify
	tsd.addRule(&DetectionRule{
		Name:     "Netlify",
		Category: "部署平台",
		Headers: map[string]*regexp.Regexp{
			"X-NF-Request-ID": regexp.MustCompile(`.+`),
			"Server":          regexp.MustCompile(`Netlify`),
		},
	})
}

// addRule 添加检测规则
func (tsd *TechStackDetector) addRule(rule *DetectionRule) {
	tsd.detectionRules[rule.Name] = rule
}

// Detect 检测技术栈
func (tsd *TechStackDetector) Detect(response *http.Response, htmlContent string) []*TechInfo {
	techs := make([]*TechInfo, 0)
	detected := make(map[string]bool)
	
	for _, rule := range tsd.detectionRules {
		techInfo := &TechInfo{
			Name:     rule.Name,
			Category: rule.Category,
			Evidence: make([]string, 0),
		}
		
		confidence := 0
		
		// 检测HTTP头
		if len(rule.Headers) > 0 {
			for header, pattern := range rule.Headers {
				if value := response.Header.Get(header); value != "" {
					if pattern.MatchString(value) {
						confidence += 30
						techInfo.Evidence = append(techInfo.Evidence, 
							"Header: "+header+"="+value)
						
						// 提取版本
						if rule.VersionPattern != nil {
							if matches := rule.VersionPattern.FindStringSubmatch(value); len(matches) > 1 {
								techInfo.Version = matches[1]
							}
						}
					}
				}
			}
		}
		
		// 检测HTML内容
		if len(rule.HTMLPatterns) > 0 {
			for _, pattern := range rule.HTMLPatterns {
				if pattern.MatchString(htmlContent) {
					confidence += 20
					techInfo.Evidence = append(techInfo.Evidence, 
						"HTML: 匹配到特征模式")
					
					// 提取版本
					if rule.VersionPattern != nil && techInfo.Version == "" {
						if matches := rule.VersionPattern.FindStringSubmatch(htmlContent); len(matches) > 1 {
							techInfo.Version = matches[1]
						}
					}
				}
			}
		}
		
		// 检测Meta标签
		if len(rule.MetaPatterns) > 0 {
			for metaName, pattern := range rule.MetaPatterns {
				metaRegex := regexp.MustCompile(`<meta[^>]+name=["\']` + metaName + `["\'][^>]+content=["\']([^"\']+)["\']`)
				if matches := metaRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
					if pattern.MatchString(matches[1]) {
						confidence += 40
						techInfo.Evidence = append(techInfo.Evidence, 
							"Meta: "+metaName+"="+matches[1])
						
						// 提取版本
						if rule.VersionPattern != nil && techInfo.Version == "" {
							if versionMatches := rule.VersionPattern.FindStringSubmatch(matches[1]); len(versionMatches) > 1 {
								techInfo.Version = versionMatches[1]
							}
						}
					}
				}
			}
		}
		
		// 检测Cookie
		if len(rule.Cookies) > 0 {
			cookies := response.Cookies()
			for _, cookie := range cookies {
				for _, expectedCookie := range rule.Cookies {
					if strings.Contains(strings.ToLower(cookie.Name), strings.ToLower(expectedCookie)) {
						confidence += 25
						techInfo.Evidence = append(techInfo.Evidence, 
							"Cookie: "+cookie.Name)
					}
				}
			}
		}
		
		// 检测JavaScript
		if len(rule.JSPatterns) > 0 {
			for _, pattern := range rule.JSPatterns {
				if pattern.MatchString(htmlContent) {
					confidence += 30
					techInfo.Evidence = append(techInfo.Evidence, 
						"JavaScript: 匹配到特征代码")
				}
			}
		}
		
		// 检测URL路径
		if len(rule.PathPatterns) > 0 && response.Request != nil {
			for _, pattern := range rule.PathPatterns {
				if pattern.MatchString(response.Request.URL.String()) {
					confidence += 20
					techInfo.Evidence = append(techInfo.Evidence, 
						"Path: 匹配到特征路径")
				}
			}
		}
		
		// 如果置信度大于30，认为检测到该技术
		if confidence >= 30 {
			techInfo.Confidence = confidence
			if confidence > 100 {
				techInfo.Confidence = 100
			}
			
			if !detected[rule.Name] {
				detected[rule.Name] = true
				techs = append(techs, techInfo)
			}
		}
	}
	
	return techs
}

// DetectFromContent 从HTML内容和Headers检测
func (tsd *TechStackDetector) DetectFromContent(htmlContent string, headers map[string]string) []*TechInfo {
	techs := make([]*TechInfo, 0)
	detected := make(map[string]bool)
	
	for _, rule := range tsd.detectionRules {
		confidence := 0
		techInfo := &TechInfo{
			Name:     rule.Name,
			Category: rule.Category,
			Evidence: make([]string, 0),
		}
		
		// 检测HTTP头
		if len(rule.Headers) > 0 && headers != nil {
			for headerName, pattern := range rule.Headers {
				if value, exists := headers[headerName]; exists && value != "" {
					if pattern.MatchString(value) {
						confidence += 30
						techInfo.Evidence = append(techInfo.Evidence, "Header: "+headerName+"="+value)
						
						// 提取版本
						if rule.VersionPattern != nil && techInfo.Version == "" {
							if matches := rule.VersionPattern.FindStringSubmatch(value); len(matches) > 1 {
								techInfo.Version = matches[1]
							}
						}
					}
				}
			}
		}
		
		// 检测HTML模式
		if len(rule.HTMLPatterns) > 0 {
			for _, pattern := range rule.HTMLPatterns {
				if pattern.MatchString(htmlContent) {
					confidence += 20
					techInfo.Evidence = append(techInfo.Evidence, "HTML特征匹配")
					
					// 提取版本
					if rule.VersionPattern != nil && techInfo.Version == "" {
						if matches := rule.VersionPattern.FindStringSubmatch(htmlContent); len(matches) > 1 {
							techInfo.Version = matches[1]
						}
					}
				}
			}
		}
		
		// 检测JS模式
		if len(rule.JSPatterns) > 0 {
			for _, pattern := range rule.JSPatterns {
				if pattern.MatchString(htmlContent) {
					confidence += 30
					techInfo.Evidence = append(techInfo.Evidence, "JavaScript特征匹配")
				}
			}
		}
		
		// 检测Meta标签
		if len(rule.MetaPatterns) > 0 {
			for metaName, pattern := range rule.MetaPatterns {
				metaRegex := regexp.MustCompile(`<meta[^>]+name=["\']` + metaName + `["\'][^>]+content=["\']([^"\']+)["\']`)
				if matches := metaRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
					if pattern.MatchString(matches[1]) {
						confidence += 40
						techInfo.Evidence = append(techInfo.Evidence, "Meta: "+metaName)
						
						if rule.VersionPattern != nil && techInfo.Version == "" {
							if versionMatches := rule.VersionPattern.FindStringSubmatch(matches[1]); len(versionMatches) > 1 {
								techInfo.Version = versionMatches[1]
							}
						}
					}
				}
			}
		}
		
		// 检测Cookies（从Headers中的Set-Cookie）
		if len(rule.Cookies) > 0 && headers != nil {
			if setCookie, exists := headers["Set-Cookie"]; exists {
				for _, expectedCookie := range rule.Cookies {
					if strings.Contains(strings.ToLower(setCookie), strings.ToLower(expectedCookie)) {
						confidence += 25
						techInfo.Evidence = append(techInfo.Evidence, "Cookie: "+expectedCookie)
					}
				}
			}
		}
		
		if confidence >= 30 && !detected[rule.Name] {
			techInfo.Confidence = confidence
			if techInfo.Confidence > 100 {
				techInfo.Confidence = 100
			}
			detected[rule.Name] = true
			techs = append(techs, techInfo)
		}
	}
	
	return techs
}

// DetectFromHTML 仅从HTML内容检测（保持向后兼容）
func (tsd *TechStackDetector) DetectFromHTML(htmlContent string) []*TechInfo {
	return tsd.DetectFromContent(htmlContent, nil)
}

// FormatTechStack 格式化技术栈信息
func (tsd *TechStackDetector) FormatTechStack(techs []*TechInfo) string {
	if len(techs) == 0 {
		return "未检测到已知技术栈"
	}
	
	var result strings.Builder
	
	// 按分类分组
	categories := make(map[string][]*TechInfo)
	for _, tech := range techs {
		categories[tech.Category] = append(categories[tech.Category], tech)
	}
	
	for category, techList := range categories {
		result.WriteString(category + ":\n")
		for _, tech := range techList {
			if tech.Version != "" {
				result.WriteString("  - " + tech.Name + " " + tech.Version + 
					" (置信度:" + string(rune(tech.Confidence)) + "%)\n")
			} else {
				result.WriteString("  - " + tech.Name + 
					" (置信度:" + string(rune(tech.Confidence)) + "%)\n")
			}
		}
	}
	
	return result.String()
}

// GetTechStackSummary 获取技术栈摘要
func (tsd *TechStackDetector) GetTechStackSummary(techs []*TechInfo) []string {
	summary := make([]string, 0)
	
	for _, tech := range techs {
		if tech.Version != "" {
			summary = append(summary, tech.Name+" "+tech.Version)
		} else {
			summary = append(summary, tech.Name)
		}
	}
	
	return summary
}

// AddCustomRule 添加自定义检测规则
func (tsd *TechStackDetector) AddCustomRule(rule *DetectionRule) {
	tsd.detectionRules[rule.Name] = rule
}

