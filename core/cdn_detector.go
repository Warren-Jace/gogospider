package core

import (
	"strings"
)

// CDNDetector CDN检测器
type CDNDetector struct {
	knownCDNs       []string
	cdnKeywords     []string
	userWhitelist   []string
}

// NewCDNDetector 创建CDN检测器
func NewCDNDetector() *CDNDetector {
	return &CDNDetector{
		knownCDNs:     getKnownCDNList(),
		cdnKeywords:   getCDNKeywords(),
		userWhitelist: make([]string, 0),
	}
}

// getKnownCDNList 获取已知CDN列表（国内外主流CDN）
func getKnownCDNList() []string {
	return []string{
		// === 国际主流CDN ===
		"cloudflare.com",
		"cloudflare.net",
		"cdnjs.cloudflare.com",
		"akamai.net",
		"akamaihd.net",
		"fastly.net",
		"jsdelivr.net",
		"unpkg.com",
		"npmcdn.com",
		"googleapis.com",
		"gstatic.com",
		"bootstrap.com",
		"bootstrapcdn.com",
		"jquery.com",
		"ajax.googleapis.com",
		
		// AWS CloudFront
		"cloudfront.net",
		"amazonaws.com",
		"awsstatic.com",
		
		// Azure CDN
		"azureedge.net",
		"azure.com",
		
		// === 中国CDN - 阿里云 ===
		"aliyun.com",
		"aliyuncs.com",
		"alicdn.com",
		"tbcdn.cn",
		"taobaocdn.com",
		"tmall.com",
		"aliapp.com",
		"alidns.com",
		"alipay.com",
		"alipayobjects.com",
		
		// === 中国CDN - 腾讯云 ===
		"myqcloud.com",
		"qcloud.com",
		"tencent.com",
		"tencentcs.com",
		"qq.com",
		"gtimg.com",  // 腾讯图片CDN
		"qpic.cn",    // QQ图片CDN
		"url.cn",     // 腾讯短链CDN
		"wxs.qq.com", // 微信静态资源
		
		// === 中国CDN - 百度云 ===
		"bcebos.com",
		"baidupcs.com",
		"bdstatic.com",
		"bdimg.com",
		"baidu.com",
		"bcehost.com",
		"bcecdn.com",
		
		// === 中国CDN - 华为云 ===
		"huaweicloud.com",
		"myhuaweicloud.com",
		"hwcloudcdn.com",
		
		// === 中国CDN - 七牛云 ===
		"qiniu.com",
		"qiniucdn.com",
		"qnssl.com",
		"qbox.me",
		
		// === 中国CDN - 又拍云 ===
		"upyun.com",
		"upaiyun.com",
		"aicdn.com",
		
		// === 中国CDN - 网宿科技 ===
		"wscdns.com",
		"wangsu.com",
		"chinanetcenter.com",
		
		// === 中国CDN - 金山云 ===
		"ksyun.com",
		"ksyuncs.com",
		"kingsoft.com",
		
		// === 中国CDN - UCloud ===
		"ucloud.cn",
		"ucloud.com.cn",
		"ufileos.com",
		
		// === 其他国内CDN ===
		"bootcss.com",    // BootCDN
		"staticfile.org", // 七牛云存储
		"cdnjs.net",      // 国内CDN镜像
		"360.cn",         // 360 CDN
		"360safe.com",
		"netease.com",    // 网易云
		"163.com",
		"126.net",
		"sinajs.cn",      // 新浪CDN
		"sinaimg.cn",
		"sina.com.cn",
	}
}

// getCDNKeywords 获取CDN关键字（域名模式）
func getCDNKeywords() []string {
	return []string{
		"cdn.",
		"static.",
		"assets.",
		"asset.",
		"img.",
		"image.",
		"images.",
		"pic.",
		"pictures.",
		"media.",
		"resource.",
		"resources.",
		"file.",
		"files.",
		"upload.",
		"uploads.",
		"storage.",
		"oss.",      // 对象存储
		"cos.",      // 腾讯云对象存储
		"obs.",      // 华为云对象存储
		"s3.",       // AWS S3
		"blob.",     // Azure Blob
		"cache.",
		"public.",
		"dist.",
		"js.",
		"css.",
		"fonts.",
		"video.",
		"videos.",
	}
}

// IsCDN 判断域名是否为CDN
func (cd *CDNDetector) IsCDN(domain string) bool {
	domain = strings.ToLower(domain)
	
	// 1. 检查用户自定义白名单
	for _, whitelist := range cd.userWhitelist {
		if strings.Contains(domain, strings.ToLower(whitelist)) {
			return true
		}
	}
	
	// 2. 检查已知CDN列表
	for _, cdn := range cd.knownCDNs {
		if strings.Contains(domain, cdn) {
			return true
		}
	}
	
	// 3. 检查CDN关键字（前缀模式）
	for _, keyword := range cd.cdnKeywords {
		if strings.HasPrefix(domain, keyword) {
			return true
		}
		// 检查子域名
		if strings.Contains(domain, "."+keyword) {
			return true
		}
	}
	
	return false
}

// IsSameBaseDomain 判断两个域名是否同源（主域名相同）
func (cd *CDNDetector) IsSameBaseDomain(domain1, domain2 string) bool {
	domain1 = strings.ToLower(domain1)
	domain2 = strings.ToLower(domain2)
	
	// 去除端口号
	domain1 = strings.Split(domain1, ":")[0]
	domain2 = strings.Split(domain2, ":")[0]
	
	// 完全相同
	if domain1 == domain2 {
		return true
	}
	
	// 提取主域名（最后两段）
	parts1 := strings.Split(domain1, ".")
	parts2 := strings.Split(domain2, ".")
	
	if len(parts1) >= 2 && len(parts2) >= 2 {
		// 获取主域名
		base1 := parts1[len(parts1)-2] + "." + parts1[len(parts1)-1]
		base2 := parts2[len(parts2)-2] + "." + parts2[len(parts2)-1]
		
		// 特殊处理中国二级域名 (com.cn, net.cn等)
		if len(parts1) >= 3 && (parts1[len(parts1)-1] == "cn" || parts1[len(parts1)-1] == "jp") {
			if parts1[len(parts1)-2] == "com" || parts1[len(parts1)-2] == "net" || 
			   parts1[len(parts1)-2] == "org" || parts1[len(parts1)-2] == "gov" {
				base1 = parts1[len(parts1)-3] + "." + parts1[len(parts1)-2] + "." + parts1[len(parts1)-1]
			}
		}
		
		if len(parts2) >= 3 && (parts2[len(parts2)-1] == "cn" || parts2[len(parts2)-1] == "jp") {
			if parts2[len(parts2)-2] == "com" || parts2[len(parts2)-2] == "net" || 
			   parts2[len(parts2)-2] == "org" || parts2[len(parts2)-2] == "gov" {
				base2 = parts2[len(parts2)-3] + "." + parts2[len(parts2)-2] + "." + parts2[len(parts2)-1]
			}
		}
		
		return base1 == base2
	}
	
	return false
}

// AddToWhitelist 添加自定义CDN域名到白名单
func (cd *CDNDetector) AddToWhitelist(domain string) {
	cd.userWhitelist = append(cd.userWhitelist, domain)
}

// SetWhitelist 设置白名单（替换现有白名单）
func (cd *CDNDetector) SetWhitelist(domains []string) {
	cd.userWhitelist = domains
}

// GetWhitelist 获取当前白名单
func (cd *CDNDetector) GetWhitelist() []string {
	return cd.userWhitelist
}

// IsStaticResource 判断是否为静态资源
func IsStaticResource(url string) bool {
	staticExtensions := []string{
		".js", ".css", ".json",
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico", ".webp",
		".woff", ".woff2", ".ttf", ".eot", ".otf",
		".mp4", ".webm", ".ogg",
		".pdf", ".zip", ".rar",
	}
	
	urlLower := strings.ToLower(url)
	
	for _, ext := range staticExtensions {
		if strings.Contains(urlLower, ext) {
			return true
		}
	}
	
	return false
}

// GetCDNInfo 获取CDN信息（用于日志）
func (cd *CDNDetector) GetCDNInfo(domain string) string {
	domain = strings.ToLower(domain)
	
	// 识别具体的CDN提供商
	cdnProviders := map[string]string{
		"aliyun":      "阿里云CDN",
		"alicdn":      "阿里云CDN",
		"aliyuncs":    "阿里云",
		"myqcloud":    "腾讯云CDN",
		"qcloud":      "腾讯云",
		"tencent":     "腾讯云",
		"bdstatic":    "百度云CDN",
		"bcebos":      "百度云",
		"baidu":       "百度",
		"qiniu":       "七牛云CDN",
		"upyun":       "又拍云CDN",
		"huaweicloud": "华为云CDN",
		"ksyun":       "金山云CDN",
		"cloudflare":  "Cloudflare",
		"akamai":      "Akamai",
		"fastly":      "Fastly",
		"cloudfront":  "AWS CloudFront",
		"azureedge":   "Azure CDN",
		"jsdelivr":    "jsDelivr",
		"bootcss":     "BootCDN",
		"staticfile":  "Staticfile CDN",
	}
	
	for key, name := range cdnProviders {
		if strings.Contains(domain, key) {
			return name
		}
	}
	
	return "未知CDN"
}

