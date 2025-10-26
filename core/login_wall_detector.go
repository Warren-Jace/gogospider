package core

import (
	"fmt"
	"net/url"
	"strings"
)

// LoginWallDetector 登录墙检测器
type LoginWallDetector struct {
	loginURLPatterns   []string // 登录URL模式
	loginPageSignals   []string // 登录页面信号
	detectedLoginURLs  map[string]int // 检测到的登录URL及其出现次数
	totalLoginPages    int             // 总登录页面数
	totalPages         int             // 总页面数
	warningThreshold   float64         // 登录页面占比警告阈值
}

// NewLoginWallDetector 创建登录墙检测器
func NewLoginWallDetector() *LoginWallDetector {
	return &LoginWallDetector{
		loginURLPatterns: []string{
			"/login",
			"/signin",
			"/auth/",
			"/account/login",
			"/user/login",
			"/sso/",
			"/oauth/",
			"login.php",
			"signin.php",
			"auth.php",
		},
		loginPageSignals: []string{
			"登录",
			"登陆",
			"用户登录",
			"账号登录",
			"sign in",
			"log in",
			"user login",
			"account login",
			"authentication required",
			"please login",
			"login required",
			"password",
			"username",
		},
		detectedLoginURLs: make(map[string]int),
		totalLoginPages:   0,
		totalPages:        0,
		warningThreshold:  0.5, // 50%登录页面时触发警告
	}
}

// IsLoginURL 检查URL是否是登录URL
func (lwd *LoginWallDetector) IsLoginURL(urlStr string) bool {
	urlLower := strings.ToLower(urlStr)
	
	// 检查URL模式
	for _, pattern := range lwd.loginURLPatterns {
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}
	
	return false
}

// IsLoginPage 检查HTML内容是否是登录页面
func (lwd *LoginWallDetector) IsLoginPage(htmlContent string) bool {
	htmlLower := strings.ToLower(htmlContent)
	
	matchCount := 0
	for _, signal := range lwd.loginPageSignals {
		if strings.Contains(htmlLower, strings.ToLower(signal)) {
			matchCount++
		}
	}
	
	// 如果匹配到3个以上信号，认为是登录页面
	return matchCount >= 3
}

// RecordPage 记录页面
func (lwd *LoginWallDetector) RecordPage(urlStr string, htmlContent string) {
	lwd.totalPages++
	
	isLogin := lwd.IsLoginURL(urlStr) || lwd.IsLoginPage(htmlContent)
	
	if isLogin {
		lwd.totalLoginPages++
		
		// 提取基础URL（去除参数）
		baseURL := lwd.extractBaseURL(urlStr)
		lwd.detectedLoginURLs[baseURL]++
	}
}

// extractBaseURL 提取基础URL（去除参数）
func (lwd *LoginWallDetector) extractBaseURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	
	// 只保留scheme + host + path
	baseURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)
	return baseURL
}

// ShouldWarn 是否应该发出警告
func (lwd *LoginWallDetector) ShouldWarn() bool {
	if lwd.totalPages == 0 {
		return false
	}
	
	ratio := float64(lwd.totalLoginPages) / float64(lwd.totalPages)
	return ratio >= lwd.warningThreshold
}

// GetStatistics 获取统计信息
func (lwd *LoginWallDetector) GetStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["total_pages"] = lwd.totalPages
	stats["login_pages"] = lwd.totalLoginPages
	stats["normal_pages"] = lwd.totalPages - lwd.totalLoginPages
	
	if lwd.totalPages > 0 {
		ratio := float64(lwd.totalLoginPages) / float64(lwd.totalPages)
		stats["login_ratio"] = ratio
		stats["login_ratio_percent"] = ratio * 100
	} else {
		stats["login_ratio"] = 0.0
		stats["login_ratio_percent"] = 0.0
	}
	
	stats["detected_login_urls"] = lwd.detectedLoginURLs
	stats["unique_login_urls"] = len(lwd.detectedLoginURLs)
	
	return stats
}

// PrintWarning 打印警告信息
func (lwd *LoginWallDetector) PrintWarning() {
	if !lwd.ShouldWarn() {
		return
	}
	
	stats := lwd.GetStatistics()
	loginRatio := stats["login_ratio_percent"].(float64)
	
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("⚠️  警告：检测到登录墙！")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("总爬取页面: %d 个\n", lwd.totalPages)
	fmt.Printf("登录页面: %d 个 (%.1f%%)\n", lwd.totalLoginPages, loginRatio)
	fmt.Printf("正常页面: %d 个\n", lwd.totalPages-lwd.totalLoginPages)
	fmt.Println()
	
	fmt.Println("检测到的登录URL：")
	count := 0
	for loginURL, occurrences := range lwd.detectedLoginURLs {
		count++
		if count <= 5 {
			fmt.Printf("  - %s (出现 %d 次)\n", loginURL, occurrences)
		}
	}
	if len(lwd.detectedLoginURLs) > 5 {
		fmt.Printf("  ... 还有 %d 个登录URL\n", len(lwd.detectedLoginURLs)-5)
	}
	
	fmt.Println()
	fmt.Println("📌 原因分析：")
	fmt.Println("  网站需要登录才能访问，爬虫无法获取登录后的内容。")
	fmt.Println()
	fmt.Println("💡 解决方案：")
	fmt.Println("  1. 使用Cookie认证：")
	fmt.Println("     spider.exe -url <target> -cookie-file cookies.txt")
	fmt.Println()
	fmt.Println("  2. 使用Cookie字符串：")
	fmt.Println("     spider.exe -url <target> -cookie \"session_id=xxx; token=yyy\"")
	fmt.Println()
	fmt.Println("  3. 排除登录页面（如果只需要公开内容）：")
	fmt.Println("     在配置文件中设置：")
	fmt.Println("     \"ExcludePaths\": [\"/login*\", \"/auth/*\"]")
	fmt.Println()
	fmt.Println("📚 详细说明请查看：如何解决登录问题.md")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

// PrintSummary 打印摘要（在爬取结束时调用）
func (lwd *LoginWallDetector) PrintSummary() {
	if lwd.totalPages == 0 {
		return
	}
	
	stats := lwd.GetStatistics()
	
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔍 登录墙检测报告")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("总爬取页面: %d 个\n", lwd.totalPages)
	fmt.Printf("登录页面: %d 个 (%.1f%%)\n", lwd.totalLoginPages, stats["login_ratio_percent"].(float64))
	fmt.Printf("正常页面: %d 个\n", lwd.totalPages-lwd.totalLoginPages)
	fmt.Printf("唯一登录URL: %d 个\n", len(lwd.detectedLoginURLs))
	
	if lwd.ShouldWarn() {
		fmt.Println()
		fmt.Println("⚠️  警告：登录页面占比过高（>50%），建议使用Cookie认证")
		fmt.Println("   详细说明：如何解决登录问题.md")
	} else if lwd.totalLoginPages > 0 {
		fmt.Println()
		fmt.Println("ℹ️  发现少量登录页面，属于正常范围")
	} else {
		fmt.Println()
		fmt.Println("✅ 未发现登录墙，所有页面均可正常访问")
	}
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// ShouldSkipURL 是否应该跳过该URL（登录页面变体）
func (lwd *LoginWallDetector) ShouldSkipURL(urlStr string) (bool, string) {
	if !lwd.IsLoginURL(urlStr) {
		return false, ""
	}
	
	baseURL := lwd.extractBaseURL(urlStr)
	
	// 如果这个登录URL已经爬取过很多次，跳过
	if count, exists := lwd.detectedLoginURLs[baseURL]; exists && count > 3 {
		reason := fmt.Sprintf("登录页面变体（该登录URL已爬取%d次）", count)
		return true, reason
	}
	
	return false, ""
}

