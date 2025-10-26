package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// CookieManager Cookie管理器
type CookieManager struct {
	cookies []*http.Cookie
}

// NewCookieManager 创建Cookie管理器
func NewCookieManager() *CookieManager {
	return &CookieManager{
		cookies: make([]*http.Cookie, 0),
	}
}

// LoadFromFile 从文件加载Cookie
// 支持多种格式：Netscape格式、JSON格式、简单键值对格式
func (cm *CookieManager) LoadFromFile(filename string) error {
	// 读取文件
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取Cookie文件失败: %v", err)
	}
	
	content := string(data)
	
	// 尝试解析JSON格式
	if strings.HasPrefix(strings.TrimSpace(content), "{") {
		return cm.loadFromJSON(content)
	}
	
	// 尝试解析Netscape格式
	if strings.Contains(content, "# Netscape HTTP Cookie File") || strings.Contains(content, "\t") {
		return cm.loadFromNetscape(content)
	}
	
	// 尝试简单键值对格式
	return cm.loadFromSimple(content)
}

// loadFromJSON 从JSON格式加载Cookie
// 格式：{"cookie_name": "cookie_value", ...}
func (cm *CookieManager) loadFromJSON(content string) error {
	var cookieMap map[string]string
	if err := json.Unmarshal([]byte(content), &cookieMap); err != nil {
		return fmt.Errorf("解析JSON Cookie失败: %v", err)
	}
	
	for name, value := range cookieMap {
		cookie := &http.Cookie{
			Name:  name,
			Value: value,
		}
		cm.cookies = append(cm.cookies, cookie)
	}
	
	fmt.Printf("[Cookie] 从JSON加载了 %d 个Cookie\n", len(cm.cookies))
	return nil
}

// loadFromNetscape 从Netscape格式加载Cookie
// 格式：domain	flag	path	secure	expiration	name	value
func (cm *CookieManager) loadFromNetscape(content string) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	count := 0
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// 跳过注释和空行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// 解析字段（使用tab分隔）
		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}
		
		domain := fields[0]
		path := fields[2]
		secureStr := fields[3]
		expirationStr := fields[4]
		name := fields[5]
		value := fields[6]
		
		// 创建Cookie
		cookie := &http.Cookie{
			Name:   name,
			Value:  value,
			Path:   path,
			Domain: domain,
		}
		
		// 设置Secure标志
		if secureStr == "TRUE" {
			cookie.Secure = true
		}
		
		// 设置过期时间
		if expiration, err := strconv.ParseInt(expirationStr, 10, 64); err == nil {
			cookie.Expires = time.Unix(expiration, 0)
		}
		
		cm.cookies = append(cm.cookies, cookie)
		count++
	}
	
	fmt.Printf("[Cookie] 从Netscape格式加载了 %d 个Cookie\n", count)
	return scanner.Err()
}

// loadFromSimple 从简单格式加载Cookie
// 格式：name=value; name2=value2
// 或每行一个：name=value
func (cm *CookieManager) loadFromSimple(content string) error {
	content = strings.TrimSpace(content)
	
	var pairs []string
	
	// 检查是否是单行分号分隔格式
	if strings.Contains(content, ";") && !strings.Contains(content, "\n") {
		pairs = strings.Split(content, ";")
	} else {
		// 每行一个
		pairs = strings.Split(content, "\n")
	}
	
	count := 0
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// 移除可能的引号
		value = strings.Trim(value, "\"'")
		
		cookie := &http.Cookie{
			Name:  name,
			Value: value,
		}
		
		cm.cookies = append(cm.cookies, cookie)
		count++
	}
	
	fmt.Printf("[Cookie] 从简单格式加载了 %d 个Cookie\n", count)
	return nil
}

// LoadFromString 从字符串加载Cookie（支持命令行传入）
// 格式：name1=value1; name2=value2
func (cm *CookieManager) LoadFromString(cookieString string) error {
	if cookieString == "" {
		return nil
	}
	
	pairs := strings.Split(cookieString, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		cookie := &http.Cookie{
			Name:  strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		}
		
		cm.cookies = append(cm.cookies, cookie)
	}
	
	fmt.Printf("[Cookie] 从字符串加载了 %d 个Cookie\n", len(cm.cookies))
	return nil
}

// ApplyToRequest 将Cookie应用到HTTP请求
func (cm *CookieManager) ApplyToRequest(req *http.Request) {
	if len(cm.cookies) == 0 {
		return
	}
	
	// 将所有Cookie添加到请求
	for _, cookie := range cm.cookies {
		req.AddCookie(cookie)
	}
}

// ApplyToURL 为指定URL设置Cookie Jar
func (cm *CookieManager) ApplyToURL(targetURL string) ([]*http.Cookie, error) {
	if len(cm.cookies) == 0 {
		return nil, nil
	}
	
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	
	// 过滤适用于该URL的Cookie
	var applicableCookies []*http.Cookie
	for _, cookie := range cm.cookies {
		// 检查域名是否匹配
		if cookie.Domain == "" || 
		   strings.HasSuffix(parsedURL.Host, cookie.Domain) ||
		   parsedURL.Host == cookie.Domain {
			applicableCookies = append(applicableCookies, cookie)
		}
	}
	
	return applicableCookies, nil
}

// GetCookies 获取所有Cookie
func (cm *CookieManager) GetCookies() []*http.Cookie {
	return cm.cookies
}

// GetCookieCount 获取Cookie数量
func (cm *CookieManager) GetCookieCount() int {
	return len(cm.cookies)
}

// GetCookieHeader 获取Cookie头字符串
func (cm *CookieManager) GetCookieHeader() string {
	if len(cm.cookies) == 0 {
		return ""
	}
	
	var parts []string
	for _, cookie := range cm.cookies {
		parts = append(parts, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}
	
	return strings.Join(parts, "; ")
}

// PrintSummary 打印Cookie摘要
func (cm *CookieManager) PrintSummary() {
	if len(cm.cookies) == 0 {
		fmt.Println("[Cookie] 未加载任何Cookie")
		return
	}
	
	fmt.Printf("[Cookie] 已加载 %d 个Cookie:\n", len(cm.cookies))
	for i, cookie := range cm.cookies {
		if i < 5 { // 只显示前5个
			maskedValue := cookie.Value
			if len(maskedValue) > 20 {
				maskedValue = maskedValue[:10] + "..." + maskedValue[len(maskedValue)-10:]
			}
			fmt.Printf("  - %s = %s\n", cookie.Name, maskedValue)
		}
	}
	if len(cm.cookies) > 5 {
		fmt.Printf("  ... 还有 %d 个Cookie\n", len(cm.cookies)-5)
	}
}

