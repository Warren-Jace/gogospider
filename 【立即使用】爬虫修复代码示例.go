// +build ignore

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"

	"golang.org/x/net/idna"
)

// ============================================================================
// 1. URL规范化器（Canonicalizer）
// ============================================================================

type URLCanonicalizer struct {
	removeTrackingParams bool
	trackingParams       map[string]bool
}

func NewURLCanonicalizer() *URLCanonicalizer {
	c := &URLCanonicalizer{
		removeTrackingParams: true,
		trackingParams:       make(map[string]bool),
	}

	// 初始化tracking参数
	trackingList := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term",
		"gclid", "fbclid", "msclkid", "mc_cid", "mc_eid",
		"_ga", "_gid", "_gac", "fbadid", "ref", "referrer", "source",
	}
	for _, p := range trackingList {
		c.trackingParams[p] = true
	}

	return c
}

func (c *URLCanonicalizer) CanonicalizeURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// 1. 协议小写
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme == "" {
		scheme = "http"
	}

	// 2. 域名处理
	host := parsedURL.Host
	hostPart, port := splitHostPort(host)

	// IDN转Punycode
	if needsPunycode(hostPart) {
		if punycoded, err := idna.ToASCII(hostPart); err == nil {
			hostPart = punycoded
		}
	}
	hostPart = strings.ToLower(hostPart)

	// 移除默认端口
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}

	if port != "" {
		host = hostPart + ":" + port
	} else {
		host = hostPart
	}

	// 3. 路径规范化
	pathStr := path.Clean(parsedURL.Path)
	if pathStr != "" && !strings.HasPrefix(pathStr, "/") {
		pathStr = "/" + pathStr
	}

	// 4. 参数处理
	query := parsedURL.Query()

	// 移除tracking参数
	if c.removeTrackingParams {
		for param := range query {
			if c.trackingParams[strings.ToLower(param)] {
				query.Del(param)
			}
		}
	}

	// 参数排序
	queryStr := sortQueryString(query)

	// 5. 重组URL
	result := scheme + "://" + host + pathStr
	if queryStr != "" {
		result += "?" + queryStr
	}

	return result, nil
}

func splitHostPort(hostport string) (host, port string) {
	if idx := strings.LastIndex(hostport, ":"); idx != -1 {
		return hostport[:idx], hostport[idx+1:]
	}
	return hostport, ""
}

func needsPunycode(host string) bool {
	for _, r := range host {
		if r > 127 {
			return true
		}
	}
	return false
}

func sortQueryString(query url.Values) string {
	if len(query) == 0 {
		return ""
	}

	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		values := query[k]
		sort.Strings(values)
		for _, v := range values {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}

	return strings.Join(parts, "&")
}

// ============================================================================
// 2. 参数提取器（ExtractParams）
// ============================================================================

func extractParams(rawURL string) (map[string][]string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	return parsedURL.Query(), nil
}

// ============================================================================
// 3. 敏感参数检测器（IsSensitiveParam）
// ============================================================================

type ParamSensitivity struct {
	ParamName   string
	IsSensitive bool
	Severity    string // HIGH, MEDIUM, LOW
	Category    string // auth, sensitive, dangerous, sql
}

func isSensitiveParam(paramName string) *ParamSensitivity {
	paramLower := strings.ToLower(paramName)

	// 高危认证参数（精确匹配）
	highAuthParams := map[string]bool{
		"token": true, "access_token": true, "auth_token": true,
		"api_key": true, "apikey": true, "app_key": true,
		"password": true, "passwd": true, "pwd": true, "pass": true,
		"secret": true, "client_secret": true, "api_secret": true,
	}
	if highAuthParams[paramLower] {
		return &ParamSensitivity{
			ParamName:   paramName,
			IsSensitive: true,
			Severity:    "HIGH",
			Category:    "auth",
		}
	}

	// 中危敏感参数
	mediumParams := map[string]bool{
		"email": true, "e_mail": true, "mail": true,
		"phone": true, "telephone": true, "mobile": true,
		"session": true, "session_id": true, "sessionid": true,
		"cookie": true,
	}
	if mediumParams[paramLower] {
		return &ParamSensitivity{
			ParamName:   paramName,
			IsSensitive: true,
			Severity:    "MEDIUM",
			Category:    "sensitive",
		}
	}

	// 危险操作参数（精确匹配）
	dangerousParams := map[string]bool{
		"cmd": true, "command": true, "exec": true, "execute": true,
		"file": true, "filename": true, "filepath": true,
		"path": true, "dir": true, "directory": true,
	}
	if dangerousParams[paramLower] {
		return &ParamSensitivity{
			ParamName:   paramName,
			IsSensitive: true,
			Severity:    "HIGH",
			Category:    "dangerous",
		}
	}

	// SQL注入风险（精确匹配，避免误报video_id、valid等）
	sqlParams := map[string]bool{
		"id": true, "user_id": true, "uid": true, "account_id": true,
	}
	if sqlParams[paramLower] {
		return &ParamSensitivity{
			ParamName:   paramName,
			IsSensitive: true,
			Severity:    "LOW",
			Category:    "sql",
		}
	}

	return &ParamSensitivity{
		ParamName:   paramName,
		IsSensitive: false,
	}
}

// JWT Token检测
func isJWTToken(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}

	for _, part := range parts {
		if len(part) < 10 {
			return false
		}
	}

	return true
}

// ============================================================================
// 4. 去重器（Deduplicator）
// ============================================================================

type Deduplicator struct {
	seen          sync.Map
	canonicalizer *URLCanonicalizer
}

func NewDeduplicator() *Deduplicator {
	return &Deduplicator{
		canonicalizer: NewURLCanonicalizer(),
	}
}

func (d *Deduplicator) IsDuplicate(rawURL string) bool {
	// 1. 规范化URL
	canonical, err := d.canonicalizer.CanonicalizeURL(rawURL)
	if err != nil {
		canonical = rawURL
	}

	// 2. 计算指纹
	fingerprint := d.calculateFingerprint(canonical)

	// 3. 检查并设置（原子操作）
	_, loaded := d.seen.LoadOrStore(fingerprint, true)

	return loaded // true表示重复
}

func (d *Deduplicator) calculateFingerprint(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

// ============================================================================
// 5. 主函数 - 演示所有功能
// ============================================================================

func main() {
	fmt.Println("======================================")
	fmt.Println("爬虫修复代码示例")
	fmt.Println("======================================\n")

	// 测试1: URL规范化
	fmt.Println("【测试1】URL规范化")
	testURLs := []string{
		"HTTP://Example.COM:80/path?b=2&a=1&utm_source=google",
		"https://中文.com/路径",
		"http://example.com//api///users//",
		"https://example.com:443/page?gclid=abc123&id=1",
	}

	canonicalizer := NewURLCanonicalizer()
	for _, testURL := range testURLs {
		canonical, err := canonicalizer.CanonicalizeURL(testURL)
		if err != nil {
			fmt.Printf("❌ 错误: %v\n", err)
			continue
		}
		fmt.Printf("原URL: %s\n", testURL)
		fmt.Printf("规范: %s\n\n", canonical)
	}

	// 测试2: 参数提取
	fmt.Println("\n【测试2】参数提取")
	paramURL := "http://example.com/api?id=123&name=test&tags=go&tags=web"
	params, err := extractParams(paramURL)
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
	} else {
		fmt.Printf("URL: %s\n", paramURL)
		fmt.Printf("参数: %v\n", params)
	}

	// 测试3: 敏感参数检测
	fmt.Println("\n【测试3】敏感参数检测")
	testParams := []string{
		"token", "video_id", "valid", "id", "password",
		"email", "cmd", "user_id", "grid_id",
	}

	for _, param := range testParams {
		result := isSensitiveParam(param)
		if result.IsSensitive {
			fmt.Printf("参数: %-12s - ⚠️  敏感 [%s/%s]\n",
				param, result.Severity, result.Category)
		} else {
			fmt.Printf("参数: %-12s - ✅ 正常\n", param)
		}
	}

	// 测试4: JWT检测
	fmt.Println("\n【测试4】JWT Token检测")
	jwtToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	notJWT := "not-a-jwt-token"

	fmt.Printf("Token: %s...\n", jwtToken[:50])
	fmt.Printf("是JWT: %v\n\n", isJWTToken(jwtToken))

	fmt.Printf("Token: %s\n", notJWT)
	fmt.Printf("是JWT: %v\n", isJWTToken(notJWT))

	// 测试5: 并发去重
	fmt.Println("\n【测试5】并发去重")
	dedup := NewDeduplicator()

	dedupURLs := []string{
		"http://example.com/page1",
		"http://example.com/page2",
		"http://example.com/page1", // 重复
		"HTTP://EXAMPLE.COM/page1", // 大小写不同，规范化后重复
		"http://example.com:80/page1", // 默认端口，规范化后重复
	}

	for _, testURL := range dedupURLs {
		if dedup.IsDuplicate(testURL) {
			fmt.Printf("重复: %s\n", testURL)
		} else {
			fmt.Printf("新URL: %s\n", testURL)
		}
	}

	fmt.Println("\n======================================")
	fmt.Println("所有测试完成")
	fmt.Println("======================================")
}

// ============================================================================
// 附录：验证用的示例URL列表
// ============================================================================

/*
验证URL列表：

1. URL规范化测试
http://Example.COM:80/path                           # 域名大小写、默认端口
https://中文.com/路径                                   # IDN域名
http://example.com//api///users//                    # 重复斜杠

2. Tracking参数过滤测试
http://example.com/page?id=1&utm_source=google&utm_medium=cpc
http://example.com/page?id=1&gclid=abc123&fbclid=def456

3. 参数排序测试
http://example.com?z=3&a=1&m=2                       # 应排序为 ?a=1&m=2&z=3

4. 敏感参数测试
?token=abc123                                        # 高危
?video_id=123                                        # 正常（不应误报）
?password=secret                                     # 高危
?valid=true                                          # 正常（不应误报）

5. JWT检测测试
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U

6. 去重测试（应全部识别为同一URL）
http://example.com/page1
HTTP://EXAMPLE.COM/page1
http://example.com:80/page1
http://example.com/page1?utm_source=google
*/

// ============================================================================
// 使用说明
// ============================================================================

/*
安装依赖：
go get golang.org/x/net/idna

编译运行：
go run 【立即使用】爬虫修复代码示例.go

预期输出：
- URL规范化：显示规范化后的URL
- 参数提取：显示解析出的参数
- 敏感参数检测：标记敏感参数及严重度
- JWT检测：识别JWT token
- 去重：识别重复URL

集成到项目：
1. 将代码拆分到对应的文件：
   - url_canonicalizer.go
   - param_extractor.go
   - sensitive_detector.go
   - deduplicator.go

2. 在Spider中使用：
   canonicalizer := NewURLCanonicalizer()
   dedup := NewDeduplicator()

   for _, rawURL := range urls {
       if dedup.IsDuplicate(rawURL) {
           continue // 跳过重复URL
       }

       params, _ := extractParams(rawURL)
       for paramName := range params {
           sensitivity := isSensitiveParam(paramName)
           if sensitivity.IsSensitive {
               log.Printf("发现敏感参数: %s [%s]", paramName, sensitivity.Severity)
           }
       }
   }
*/

