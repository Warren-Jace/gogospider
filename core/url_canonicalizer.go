package core

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/idna"
)

// URLCanonicalizer URL规范化器
// 用于将URL转换为标准形式，便于去重和比较
type URLCanonicalizer struct {
	normalizeProtocol    bool // http->https
	stripDefaultPort     bool // 移除:80/:443
	lowercaseDomain      bool // 域名小写
	sortQueryParams      bool // 参数排序
	removeTrackingParams bool // 移除tracking参数

	trackingParams map[string]bool
}

// NewURLCanonicalizer 创建URL规范化器
func NewURLCanonicalizer() *URLCanonicalizer {
	c := &URLCanonicalizer{
		normalizeProtocol:    false, // 保持原协议
		stripDefaultPort:     true,
		lowercaseDomain:      true,
		sortQueryParams:      true,
		removeTrackingParams: true,
		trackingParams:       make(map[string]bool),
	}

	// 初始化常见tracking参数
	trackingList := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term",
		"gclid", "fbclid", "msclkid", "mc_cid", "mc_eid",
		"_ga", "_gid", "_gac", "fbadid",
		"ref", "referrer", "source",
		"campaign_id", "ad_id", "adgroup_id",
	}
	for _, p := range trackingList {
		c.trackingParams[p] = true
	}

	return c
}

// CanonicalizeURL 规范化URL
func (c *URLCanonicalizer) CanonicalizeURL(rawURL string) (string, error) {
	// 1. URL解析
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// 2. 处理协议
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme == "" {
		scheme = "http" // 默认协议
	}
	if c.normalizeProtocol && scheme == "http" {
		scheme = "https"
	}

	// 3. 处理域名（IDN->Punycode + 小写）
	host := parsedURL.Host

	// 3.1 分离主机和端口
	hostPart, port := splitHostPort(host)

	// 3.2 IDN域名转Punycode
	if needsPunycode(hostPart) {
		punycoded, err := idna.ToASCII(hostPart)
		if err == nil {
			hostPart = punycoded
		}
	}

	// 3.3 域名小写
	if c.lowercaseDomain {
		hostPart = strings.ToLower(hostPart)
	}

	// 3.4 移除默认端口
	if c.stripDefaultPort {
		if (scheme == "http" && port == "80") ||
			(scheme == "https" && port == "443") {
			port = ""
		}
	}

	// 3.5 重组host
	if port != "" {
		host = hostPart + ":" + port
	} else {
		host = hostPart
	}

	// 4. 处理路径
	pathStr := parsedURL.Path

	// 4.1 规范化路径（去除.和..、重复斜杠）
	pathStr = path.Clean(pathStr)

	// 4.2 确保路径以/开头（如果非空）
	if pathStr != "" && !strings.HasPrefix(pathStr, "/") {
		pathStr = "/" + pathStr
	}

	// 4.3 Percent-decoding规范化
	pathStr = normalizePercentEncoding(pathStr)

	// 5. 处理查询参数
	query := parsedURL.Query()

	// 5.1 移除tracking参数
	if c.removeTrackingParams {
		for param := range query {
			if c.trackingParams[strings.ToLower(param)] {
				query.Del(param)
			}
		}
	}

	// 5.2 参数排序
	var queryStr string
	if c.sortQueryParams {
		queryStr = sortQueryString(query)
	} else {
		queryStr = query.Encode()
	}

	// 6. 重组URL
	result := scheme + "://" + host + pathStr
	if queryStr != "" {
		result += "?" + queryStr
	}
	// 注意：通常忽略fragment (#hash)，因为对服务器无影响

	return result, nil
}

// splitHostPort 分离主机名和端口
func splitHostPort(hostport string) (host, port string) {
	// 处理IPv6地址: [::1]:8080
	if strings.HasPrefix(hostport, "[") {
		if idx := strings.LastIndex(hostport, "]:"); idx != -1 {
			return hostport[:idx+1], hostport[idx+2:]
		}
		return hostport, ""
	}

	// 普通域名: example.com:8080
	if idx := strings.LastIndex(hostport, ":"); idx != -1 {
		return hostport[:idx], hostport[idx+1:]
	}

	return hostport, ""
}

// needsPunycode 检查是否需要Punycode编码（包含非ASCII字符）
func needsPunycode(host string) bool {
	for _, r := range host {
		if r > 127 {
			return true
		}
	}
	return false
}

// normalizePercentEncoding 规范化百分号编码
func normalizePercentEncoding(pathStr string) string {
	// 解码可以安全解码的字符（unreserved characters）
	// RFC 3986: A-Z a-z 0-9 - _ . ~
	unreserved := regexp.MustCompile(`%([2-7][0-9A-F])`)

	decoded := unreserved.ReplaceAllStringFunc(pathStr, func(encoded string) string {
		// 提取十六进制数
		hex := encoded[1:]
		var char byte
		_, err := fmt.Sscanf(hex, "%x", &char)
		if err != nil {
			return encoded
		}

		// 判断是否为unreserved字符
		if (char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.' || char == '~' {
			return string(char)
		}

		// 保留编码，但统一为大写
		return "%" + strings.ToUpper(hex)
	})

	// 统一其他百分号编码为大写
	return regexp.MustCompile(`%[0-9a-f]{2}`).ReplaceAllStringFunc(decoded,
		func(s string) string {
			return strings.ToUpper(s)
		})
}

// sortQueryString 对查询参数排序
func sortQueryString(query url.Values) string {
	if len(query) == 0 {
		return ""
	}

	// 提取所有键并排序
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建排序后的查询字符串
	var parts []string
	for _, k := range keys {
		// 对同一键的多个值也排序
		values := query[k]
		sort.Strings(values)

		for _, v := range values {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}

	return strings.Join(parts, "&")
}

// AddTrackingParam 添加自定义tracking参数
func (c *URLCanonicalizer) AddTrackingParam(param string) {
	c.trackingParams[strings.ToLower(param)] = true
}

// SetNormalizeProtocol 设置是否标准化协议（http->https）
func (c *URLCanonicalizer) SetNormalizeProtocol(enable bool) {
	c.normalizeProtocol = enable
}

// CanonicalizeURLSimple 简化版本（只返回结果，忽略错误）
func CanonicalizeURLSimple(rawURL string) string {
	c := NewURLCanonicalizer()
	result, err := c.CanonicalizeURL(rawURL)
	if err != nil {
		return rawURL // 出错则返回原URL
	}
	return result
}

