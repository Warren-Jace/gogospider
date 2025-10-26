package core

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// RedirectManager 重定向管理器
type RedirectManager struct {
	maxRedirects          int                    // 最大重定向次数
	redirectChains        map[string][]string    // 重定向链记录
	authRedirects         map[string]string      // 认证重定向记录 (原始URL -> 登录URL)
	totalRedirects        int                    // 总重定向次数
	authRedirectCount     int                    // 认证重定向次数
	followRedirect        bool                   // 是否跟随重定向
	detectAuthRedirect    bool                   // 是否检测认证重定向
	stopOnAuthRedirect    bool                   // 遇到认证重定向时是否停止
}

// RedirectInfo 重定向信息
type RedirectInfo struct {
	OriginalURL    string   // 原始URL
	FinalURL       string   // 最终URL
	RedirectChain  []string // 重定向链
	IsAuthRedirect bool     // 是否是认证重定向
	StatusCode     int      // 重定向状态码
}

// NewRedirectManager 创建重定向管理器
func NewRedirectManager() *RedirectManager {
	return &RedirectManager{
		maxRedirects:       10,
		redirectChains:     make(map[string][]string),
		authRedirects:      make(map[string]string),
		totalRedirects:     0,
		authRedirectCount:  0,
		followRedirect:     true,  // 默认跟随重定向
		detectAuthRedirect: true,  // 默认检测认证重定向
		stopOnAuthRedirect: false, // 默认不停止（只警告）
	}
}

// SetFollowRedirect 设置是否跟随重定向
func (rm *RedirectManager) SetFollowRedirect(follow bool) {
	rm.followRedirect = follow
}

// SetStopOnAuthRedirect 设置遇到认证重定向时是否停止
func (rm *RedirectManager) SetStopOnAuthRedirect(stop bool) {
	rm.stopOnAuthRedirect = stop
}

// IsAuthRedirectURL 检查URL是否是认证/登录相关的重定向
func (rm *RedirectManager) IsAuthRedirectURL(urlStr string) bool {
	urlLower := strings.ToLower(urlStr)
	
	// 检查常见的登录/认证URL模式
	authPatterns := []string{
		"/login",
		"/signin",
		"/auth/",
		"/sso/",
		"/oauth/",
		"/authenticate",
		"/account/login",
		"/user/login",
		"/passport/",
		"login.php",
		"signin.php",
		"auth.php",
		"sso.php",
	}
	
	for _, pattern := range authPatterns {
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}
	
	// 检查域名是否是认证域名
	authDomains := []string{
		"auth.",
		"login.",
		"signin.",
		"sso.",
		"passport.",
		"account.",
		"oauth.",
	}
	
	parsedURL, err := url.Parse(urlStr)
	if err == nil {
		hostLower := strings.ToLower(parsedURL.Host)
		for _, domain := range authDomains {
			if strings.HasPrefix(hostLower, domain) {
				return true
			}
		}
	}
	
	return false
}

// CheckRedirect 检查重定向（实现http.Client的CheckRedirect函数签名）
func (rm *RedirectManager) CheckRedirect(req *http.Request, via []*http.Request) error {
	// 记录重定向
	rm.totalRedirects++
	
	if len(via) >= rm.maxRedirects {
		return fmt.Errorf("重定向次数超过限制：%d", rm.maxRedirects)
	}
	
	// 如果不跟随重定向
	if !rm.followRedirect {
		return http.ErrUseLastResponse
	}
	
	// 检测认证重定向
	if rm.detectAuthRedirect && rm.IsAuthRedirectURL(req.URL.String()) {
		rm.authRedirectCount++
		
		// 记录认证重定向
		if len(via) > 0 {
			originalURL := via[0].URL.String()
			rm.authRedirects[originalURL] = req.URL.String()
			
			// 打印警告
			fmt.Printf("\n⚠️  [认证重定向] 检测到认证重定向！\n")
			fmt.Printf("   原始URL: %s\n", originalURL)
			fmt.Printf("   重定向到: %s\n", req.URL.String())
			fmt.Printf("   💡 提示: 该网站需要登录，建议使用Cookie认证\n\n")
		}
		
		// 如果设置了遇到认证重定向就停止
		if rm.stopOnAuthRedirect {
			return http.ErrUseLastResponse // 不跟随认证重定向
		}
	}
	
	// 记录重定向链
	if len(via) > 0 {
		originalURL := via[0].URL.String()
		chain := make([]string, 0, len(via)+1)
		for _, r := range via {
			chain = append(chain, r.URL.String())
		}
		chain = append(chain, req.URL.String())
		rm.redirectChains[originalURL] = chain
	}
	
	return nil // 允许跟随重定向
}

// RecordRedirect 记录重定向（手动记录方式）
func (rm *RedirectManager) RecordRedirect(originalURL, targetURL string, statusCode int) *RedirectInfo {
	rm.totalRedirects++
	
	isAuth := rm.IsAuthRedirectURL(targetURL)
	if isAuth {
		rm.authRedirectCount++
		rm.authRedirects[originalURL] = targetURL
	}
	
	return &RedirectInfo{
		OriginalURL:    originalURL,
		FinalURL:       targetURL,
		RedirectChain:  []string{originalURL, targetURL},
		IsAuthRedirect: isAuth,
		StatusCode:     statusCode,
	}
}

// GetAuthRedirects 获取所有认证重定向
func (rm *RedirectManager) GetAuthRedirects() map[string]string {
	return rm.authRedirects
}

// GetStatistics 获取统计信息
func (rm *RedirectManager) GetStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["total_redirects"] = rm.totalRedirects
	stats["auth_redirects"] = rm.authRedirectCount
	stats["normal_redirects"] = rm.totalRedirects - rm.authRedirectCount
	stats["unique_auth_targets"] = len(rm.authRedirects)
	
	if rm.totalRedirects > 0 {
		authRatio := float64(rm.authRedirectCount) / float64(rm.totalRedirects)
		stats["auth_redirect_ratio"] = authRatio
		stats["auth_redirect_percent"] = authRatio * 100
	} else {
		stats["auth_redirect_ratio"] = 0.0
		stats["auth_redirect_percent"] = 0.0
	}
	
	return stats
}

// PrintReport 打印重定向报告
func (rm *RedirectManager) PrintReport() {
	if rm.totalRedirects == 0 {
		return
	}
	
	stats := rm.GetStatistics()
	
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔄 重定向检测报告")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("总重定向次数: %d 个\n", rm.totalRedirects)
	fmt.Printf("认证重定向: %d 个 (%.1f%%)\n", rm.authRedirectCount, stats["auth_redirect_percent"].(float64))
	fmt.Printf("正常重定向: %d 个\n", rm.totalRedirects-rm.authRedirectCount)
	
	if rm.authRedirectCount > 0 {
		fmt.Println()
		fmt.Println("检测到的认证重定向：")
		count := 0
		for originalURL, loginURL := range rm.authRedirects {
			count++
			if count <= 5 {
				fmt.Printf("  %s\n    → %s\n", originalURL, loginURL)
			}
		}
		if len(rm.authRedirects) > 5 {
			fmt.Printf("  ... 还有 %d 个认证重定向\n", len(rm.authRedirects)-5)
		}
		
		fmt.Println()
		fmt.Println("⚠️  建议：")
		fmt.Println("  网站存在认证重定向，以下URL需要登录才能访问：")
		for originalURL := range rm.authRedirects {
			fmt.Printf("    - %s\n", originalURL)
			break // 只显示第一个作为示例
		}
		fmt.Println()
		fmt.Println("💡 解决方案：")
		fmt.Println("  1. 使用Cookie认证：")
		fmt.Println("     spider.exe -url <target> -cookie-file cookies.json")
		fmt.Println("  2. 禁止跟随认证重定向：")
		fmt.Println("     在配置文件中设置 StopOnAuthRedirect: true")
		fmt.Println("  3. 查看详细说明：Cookie使用指南.md")
	} else {
		fmt.Println()
		fmt.Println("✅ 所有重定向均为正常跳转，未发现认证重定向")
	}
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// ShouldFollowRedirect 判断是否应该跟随重定向
func (rm *RedirectManager) ShouldFollowRedirect(targetURL string) (bool, string) {
	if !rm.followRedirect {
		return false, "重定向跟随已禁用"
	}
	
	if rm.detectAuthRedirect && rm.stopOnAuthRedirect && rm.IsAuthRedirectURL(targetURL) {
		return false, "检测到认证重定向，已配置为不跟随"
	}
	
	return true, ""
}

// GetRedirectChain 获取URL的重定向链
func (rm *RedirectManager) GetRedirectChain(originalURL string) []string {
	if chain, exists := rm.redirectChains[originalURL]; exists {
		return chain
	}
	return []string{originalURL}
}

// HasAuthRedirect 检查是否有认证重定向
func (rm *RedirectManager) HasAuthRedirect() bool {
	return rm.authRedirectCount > 0
}

// GetAuthRedirectRatio 获取认证重定向比例
func (rm *RedirectManager) GetAuthRedirectRatio() float64 {
	if rm.totalRedirects == 0 {
		return 0.0
	}
	return float64(rm.authRedirectCount) / float64(rm.totalRedirects)
}

