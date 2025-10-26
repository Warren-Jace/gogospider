package core

import "strings"

// CommonPaths 200个最常见的Web应用路径（精选版）
// 原则：业务价值高、信息丰富、无恶意攻击内容
var CommonPaths = []string{
	// ========== 核心业务功能（40个）==========
	// 用户认证与账户管理
	"/login", "/login.php", "/login.html", "/signin", "/sign-in",
	"/logout", "/signout", "/register", "/signup", "/sign-up",
	"/account", "/profile", "/user", "/users", "/dashboard",
	"/settings", "/preferences", "/password", "/forgot-password", "/reset-password",
	
	// 核心业务页面
	"/index", "/home", "/main", "/welcome", "/portal",
	"/about", "/about-us", "/contact", "/contact-us", "/help",
	"/faq", "/support", "/feedback", "/search", "/sitemap",
	
	// 内容管理
	"/content", "/page", "/pages", "/post", "/posts",
	"/article", "/articles", "/news", "/blog", "/category",
	
	// ========== API接口（30个）==========
	// RESTful API
	"/api", "/api/v1", "/api/v2", "/api/v3",
	"/rest", "/rest/v1", "/rest/v2",
	"/graphql", "/graph", "/query",
	
	// 数据接口
	"/data", "/json", "/xml", "/ajax",
	"/service", "/services", "/endpoint", "/endpoints",
	
	// 用户API
	"/api/user", "/api/users", "/api/auth", "/api/login",
	"/api/profile", "/api/account", "/api/session",
	
	// 业务API
	"/api/product", "/api/products", "/api/order", "/api/orders",
	"/api/cart", "/api/payment", "/api/search",
	
	// ========== 管理后台（25个）==========
	"/admin", "/admin/", "/admin/index", "/admin/login",
	"/admin/dashboard", "/admin/home", "/admin/panel",
	"/administrator", "/administration", "/manage", "/manager",
	"/management", "/console", "/control", "/panel",
	"/backend", "/back-end", "/backoffice", "/back-office",
	
	// CMS后台
	"/wp-admin", "/wp-login.php", "/wordpress/wp-admin",
	"/cms", "/cms/admin", "/cpanel",
	
	// ========== 文件与资源（20个）==========
	"/upload", "/uploads", "/upload.php", "/uploader",
	"/download", "/downloads", "/file", "/files",
	"/media", "/images", "/img", "/pics", "/photos",
	"/videos", "/audio", "/documents", "/docs",
	"/attachments", "/assets", "/resources",
	
	// ========== 配置与系统（25个）==========
	// 配置文件
	"/config", "/configuration", "/setup", "/install",
	"/installation", "/wizard", "/init", "/initialize",
	
	// 环境文件
	"/.env", "/env", "/environment", "/config.php",
	"/config.json", "/config.xml", "/config.yml",
	"/settings.php", "/settings.json", "/app.config",
	
	// 系统信息
	"/info", "/phpinfo", "/phpinfo.php", "/server-info",
	"/status", "/health", "/ping", "/version",
	"/system", "/sysinfo", "/diagnostics",
	
	// ========== 安全相关（20个）==========
	"/auth", "/oauth", "/oauth2", "/sso",
	"/security", "/secure", "/ssl", "/certificate",
	"/token", "/tokens", "/session", "/sessions",
	"/key", "/keys", "/secret", "/secrets",
	"/verify", "/validation", "/captcha", "/2fa",
	
	// ========== 业务功能（30个）==========
	// 电商
	"/shop", "/store", "/product", "/products",
	"/cart", "/checkout", "/order", "/orders",
	"/payment", "/pay", "/invoice", "/invoices",
	"/wishlist", "/favorite", "/compare", "/review",
	
	// 社交
	"/message", "/messages", "/chat", "/inbox",
	"/notification", "/notifications", "/friend", "/friends",
	"/follow", "/followers", "/comment", "/comments",
	
	// 内容管理
	"/editor", "/edit", "/create", "/new",
	"/delete", "/remove", "/update", "/modify",
	
	// ========== 监控与日志（10个）==========
	"/monitor", "/monitoring", "/metrics", "/stats",
	"/statistics", "/analytics", "/report", "/reports",
	"/log", "/logs",
	
	// ========== 开发与测试（15个）==========
	"/dev", "/development", "/test", "/testing",
	"/debug", "/trace", "/demo", "/sample",
	"/staging", "/preview", "/beta", "/alpha",
	"/temp", "/tmp", "/cache",
	
	// ========== 文档与帮助（10个）==========
	"/doc", "/docs", "/documentation", "/manual",
	"/guide", "/tutorial", "/readme", "/changelog",
	"/license", "/terms",
	
	// ========== 特殊功能（15个）==========
	"/export", "/import", "/backup", "/restore",
	"/sync", "/webhook", "/callback", "/notify",
	"/schedule", "/cron", "/job", "/task",
	"/queue", "/worker", "/service-worker",
}

// GetCommonPathsByCategory 按类别获取路径
func GetCommonPathsByCategory() map[string][]string {
	return map[string][]string{
		"核心业务": {
			"/login", "/register", "/dashboard", "/home", "/profile",
			"/account", "/settings", "/search", "/about", "/contact",
		},
		"API接口": {
			"/api", "/api/v1", "/api/v2", "/graphql", "/rest",
			"/api/user", "/api/products", "/api/auth", "/data", "/json",
		},
		"管理后台": {
			"/admin", "/admin/login", "/administrator", "/manage", "/backend",
			"/wp-admin", "/cms", "/panel", "/console", "/control",
		},
		"文件管理": {
			"/upload", "/uploads", "/download", "/files", "/media",
			"/images", "/documents", "/attachments", "/assets", "/resources",
		},
		"系统配置": {
			"/.env", "/config", "/settings", "/phpinfo.php", "/status",
			"/health", "/info", "/version", "/system", "/setup",
		},
		"安全认证": {
			"/auth", "/oauth", "/sso", "/token", "/session",
			"/security", "/verify", "/captcha", "/2fa", "/certificate",
		},
		"业务功能": {
			"/shop", "/cart", "/checkout", "/order", "/payment",
			"/message", "/notification", "/editor", "/create", "/export",
		},
		"监控日志": {
			"/monitor", "/metrics", "/stats", "/analytics", "/log",
			"/report", "/debug", "/trace", "/error", "/health",
		},
	}
}

// GetHighPriorityPaths 获取高优先级路径（最有价值的50个）
func GetHighPriorityPaths() []string {
	return []string{
		// 最高价值：管理后台
		"/admin", "/admin/", "/admin/login", "/administrator",
		"/wp-admin", "/phpmyadmin", "/cpanel",
		
		// 高价值：API接口
		"/api", "/api/v1", "/api/v2", "/graphql", "/rest",
		"/api/admin", "/api/user", "/swagger",
		
		// 高价值：认证相关
		"/login", "/signin", "/register", "/auth", "/oauth",
		"/token", "/session", "/sso",
		
		// 高价值：配置文件
		"/.env", "/config", "/config.php", "/phpinfo.php",
		"/web.config", "/.git/config",
		
		// 高价值：敏感信息
		"/backup", "/database", "/db", "/data", "/export",
		"/log", "/logs", "/debug", "/info",
		
		// 高价值：文件操作
		"/upload", "/download", "/file", "/files",
		"/filemanager", "/editor",
		
		// 高价值：系统功能
		"/system", "/console", "/panel", "/dashboard",
		"/status", "/health", "/monitor",
	}
}

// PathPriority 路径优先级
type PathPriority struct {
	Path     string
	Priority int // 1-10，10最高
	Category string
	Reason   string
}

// GetPathPriorities 获取所有路径的优先级
func GetPathPriorities() []PathPriority {
	priorities := []PathPriority{
		// 优先级10：极高价值（管理后台）
		{"/admin", 10, "管理后台", "系统管理入口"},
		{"/admin/", 10, "管理后台", "系统管理入口"},
		{"/administrator", 10, "管理后台", "管理员面板"},
		{"/phpmyadmin", 10, "管理后台", "数据库管理"},
		{"/cpanel", 10, "管理后台", "控制面板"},
		
		// 优先级9：高价值（API和认证）
		{"/api", 9, "API接口", "API根路径"},
		{"/api/v1", 9, "API接口", "API版本1"},
		{"/graphql", 9, "API接口", "GraphQL接口"},
		{"/login", 9, "认证", "登录入口"},
		{"/auth", 9, "认证", "认证接口"},
		{"/.env", 9, "配置", "环境变量文件"},
		
		// 优先级8：重要（配置和数据）
		{"/config", 8, "配置", "配置文件"},
		{"/phpinfo.php", 8, "信息泄露", "PHP信息页"},
		{"/backup", 8, "数据", "备份文件"},
		{"/database", 8, "数据", "数据库"},
		{"/export", 8, "数据", "数据导出"},
		
		// 优先级7：中高（文件操作）
		{"/upload", 7, "文件", "文件上传"},
		{"/download", 7, "文件", "文件下载"},
		{"/files", 7, "文件", "文件管理"},
		{"/filemanager", 7, "文件", "文件管理器"},
		
		// 优先级6：中等（常见功能）
		{"/dashboard", 6, "功能", "仪表板"},
		{"/profile", 6, "功能", "用户资料"},
		{"/settings", 6, "功能", "设置"},
		{"/register", 6, "功能", "注册"},
		
		// 优先级5：一般（信息页面）
		{"/about", 5, "信息", "关于页面"},
		{"/contact", 5, "信息", "联系页面"},
		{"/help", 5, "信息", "帮助页面"},
		{"/faq", 5, "信息", "常见问题"},
		
		// 优先级4：较低（监控日志）
		{"/status", 4, "监控", "状态页"},
		{"/health", 4, "监控", "健康检查"},
		{"/monitor", 4, "监控", "监控页面"},
		{"/log", 4, "日志", "日志文件"},
		
		// 优先级3：低（开发环境）
		{"/dev", 3, "开发", "开发环境"},
		{"/test", 3, "开发", "测试环境"},
		{"/debug", 3, "开发", "调试页面"},
		
		// 优先级2：很低（静态资源）
		{"/images", 2, "资源", "图片目录"},
		{"/css", 2, "资源", "样式目录"},
		{"/js", 2, "资源", "脚本目录"},
		
		// 优先级1：最低（其他）
		{"/robots.txt", 1, "其他", "爬虫协议"},
		{"/sitemap.xml", 1, "其他", "站点地图"},
	}
	
	return priorities
}

// GetPathPriority 获取单个路径的优先级
func GetPathPriority(path string) int {
	// 高优先级关键词
	highPriorityKeywords := []string{
		"admin", "phpmyadmin", "cpanel", "login", "auth",
		"api", "graphql", ".env", "config", "backup",
		"database", "phpinfo",
	}
	
	// 中优先级关键词
	mediumPriorityKeywords := []string{
		"upload", "download", "file", "dashboard", "panel",
		"manage", "export", "import", "user", "account",
	}
	
	pathLower := strings.ToLower(path)
	
	// 检查高优先级
	for _, keyword := range highPriorityKeywords {
		if strings.Contains(pathLower, keyword) {
			return 9
		}
	}
	
	// 检查中优先级
	for _, keyword := range mediumPriorityKeywords {
		if strings.Contains(pathLower, keyword) {
			return 6
		}
	}
	
	// 默认低优先级
	return 3
}

