package core

import (
	"net"
	"net/url"
	"regexp"
)

// IPDetector IP地址检测器
type IPDetector struct {
	ipv4Pattern *regexp.Regexp
	ipv6Pattern *regexp.Regexp
}

// NewIPDetector 创建IP检测器
func NewIPDetector() *IPDetector {
	return &IPDetector{
		// IPv4正则表达式
		ipv4Pattern: regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}(:\d+)?$`),
		// IPv6正则表达式（简化版）
		ipv6Pattern: regexp.MustCompile(`^\[?([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}\]?(:\d+)?$`),
	}
}

// IsIPBasedURL 检查URL是否使用IP地址作为主机
func (ipd *IPDetector) IsIPBasedURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	
	host := parsedURL.Hostname()
	if host == "" {
		return false
	}
	
	// 检查是否为IP地址（使用Go标准库）
	ip := net.ParseIP(host)
	if ip != nil {
		return true
	}
	
	// 额外检查：使用正则表达式（处理一些边界情况）
	if ipd.ipv4Pattern.MatchString(host) || ipd.ipv6Pattern.MatchString(host) {
		return true
	}
	
	return false
}

// IsPrivateIP 检查是否为内网IP
func (ipd *IPDetector) IsPrivateIP(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	
	host := parsedURL.Hostname()
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	
	// 检查是否为私有IP地址
	privateIPBlocks := []string{
		"10.0.0.0/8",     // RFC1918 - Class A
		"172.16.0.0/12",  // RFC1918 - Class B
		"192.168.0.0/16", // RFC1918 - Class C
		"127.0.0.0/8",    // Loopback
		"169.254.0.0/16", // Link-local
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 private (Unique Local Addresses)
		"fe80::/10",      // IPv6 link-local
	}
	
	for _, block := range privateIPBlocks {
		_, ipNet, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}
	
	return false
}

// IsPublicIP 检查是否为公网IP
func (ipd *IPDetector) IsPublicIP(rawURL string) bool {
	if !ipd.IsIPBasedURL(rawURL) {
		return false
	}
	return !ipd.IsPrivateIP(rawURL)
}

// GetIPType 获取IP类型
func (ipd *IPDetector) GetIPType(rawURL string) string {
	if !ipd.IsIPBasedURL(rawURL) {
		return "NOT_IP"
	}
	
	if ipd.IsPrivateIP(rawURL) {
		return "PRIVATE_IP"
	}
	
	return "PUBLIC_IP"
}

// ClassifyIPLinks 分类IP链接
func (ipd *IPDetector) ClassifyIPLinks(urls []string) map[string][]string {
	result := make(map[string][]string)
	result["private_ip"] = make([]string, 0)
	result["public_ip"] = make([]string, 0)
	result["non_ip"] = make([]string, 0)
	
	// 用于去重
	seenPrivate := make(map[string]bool)
	seenPublic := make(map[string]bool)
	
	for _, urlStr := range urls {
		if ipd.IsIPBasedURL(urlStr) {
			if ipd.IsPrivateIP(urlStr) {
				if !seenPrivate[urlStr] {
					seenPrivate[urlStr] = true
					result["private_ip"] = append(result["private_ip"], urlStr)
				}
			} else {
				if !seenPublic[urlStr] {
					seenPublic[urlStr] = true
					result["public_ip"] = append(result["public_ip"], urlStr)
				}
			}
		} else {
			result["non_ip"] = append(result["non_ip"], urlStr)
		}
	}
	
	return result
}

// ExtractIPFromURL 从URL中提取IP地址
func (ipd *IPDetector) ExtractIPFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	
	host := parsedURL.Hostname()
	ip := net.ParseIP(host)
	if ip != nil {
		return host
	}
	
	return ""
}

// GetIPStatistics 获取IP链接统计信息
func (ipd *IPDetector) GetIPStatistics(classifiedIPs map[string][]string) map[string]int {
	stats := make(map[string]int)
	
	stats["private_count"] = len(classifiedIPs["private_ip"])
	stats["public_count"] = len(classifiedIPs["public_ip"])
	stats["total_count"] = stats["private_count"] + stats["public_count"]
	
	return stats
}

// HasPrivateIPLeak 检查是否存在内网IP泄露
func (ipd *IPDetector) HasPrivateIPLeak(urls []string) bool {
	for _, url := range urls {
		if ipd.IsIPBasedURL(url) && ipd.IsPrivateIP(url) {
			return true
		}
	}
	return false
}

