package core

import (
	"fmt"
	"regexp"
	"strings"
)

// POSTRequestDetector POST请求检测器
// 用于从JavaScript代码中识别POST请求模式
type POSTRequestDetector struct {
	patterns []*POSTPattern
}

// POSTPattern POST请求模式
type POSTPattern struct {
	name        string
	pattern     *regexp.Regexp
	urlIndex    int  // URL在匹配组中的索引
	methodIndex int  // 方法在匹配组中的索引（如果有）
}

// DetectedPOSTRequest 检测到的POST请求
type DetectedPOSTRequest struct {
	URL         string
	Method      string
	Parameters  map[string]string
	ContentType string
	Source      string // 来源：ajax, fetch, axios, form等
}

// NewPOSTRequestDetector 创建POST请求检测器
func NewPOSTRequestDetector() *POSTRequestDetector {
	detector := &POSTRequestDetector{
		patterns: make([]*POSTPattern, 0),
	}
	
	detector.initPatterns()
	
	return detector
}

// initPatterns 初始化检测模式
func (d *POSTRequestDetector) initPatterns() {
	// 1. jQuery $.ajax POST请求
	d.addPattern("jquery-ajax-post", 
		`\$\.ajax\s*\(\s*\{\s*[^}]*type\s*:\s*['"]POST['"][^}]*url\s*:\s*['"]([^'"]+)['"]`,
		1, -1)
	
	d.addPattern("jquery-ajax-post-alt", 
		`\$\.ajax\s*\(\s*\{\s*[^}]*url\s*:\s*['"]([^'"]+)['"][^}]*type\s*:\s*['"]POST['"]`,
		1, -1)
	
	// 2. jQuery $.post
	d.addPattern("jquery-post", 
		`\$\.post\s*\(\s*['"]([^'"]+)['"]`,
		1, -1)
	
	// 3. axios POST请求
	d.addPattern("axios-post", 
		`axios\.post\s*\(\s*['"]([^'"]+)['"]`,
		1, -1)
	
	d.addPattern("axios-method", 
		`axios\s*\(\s*\{\s*[^}]*method\s*:\s*['"]POST['"][^}]*url\s*:\s*['"]([^'"]+)['"]`,
		1, -1)
	
	d.addPattern("axios-method-alt", 
		`axios\s*\(\s*\{\s*[^}]*url\s*:\s*['"]([^'"]+)['"][^}]*method\s*:\s*['"]POST['"]`,
		1, -1)
	
	// 4. fetch POST请求
	d.addPattern("fetch-post", 
		`fetch\s*\(\s*['"]([^'"]+)['"][^)]*method\s*:\s*['"]POST['"]`,
		1, -1)
	
	d.addPattern("fetch-post-alt", 
		`fetch\s*\(\s*['"]([^'"]+)['"][^)]*\{\s*[^}]*method\s*:\s*['"]POST['"]`,
		1, -1)
	
	// 5. XMLHttpRequest POST
	d.addPattern("xhr-post", 
		`(?:xhr|xmlhttp)\.open\s*\(\s*['"]POST['"],\s*['"]([^'"]+)['"]`,
		1, -1)
	
	// 6. 表单提交
	d.addPattern("form-action-post", 
		`<form[^>]*method\s*=\s*['"]POST['"][^>]*action\s*=\s*['"]([^'"]+)['"]`,
		1, -1)
	
	d.addPattern("form-action-post-alt", 
		`<form[^>]*action\s*=\s*['"]([^'"]+)['"][^>]*method\s*=\s*['"]POST['"]`,
		1, -1)
	
	// 7. 动态表单提交
	d.addPattern("form-submit", 
		`form\.action\s*=\s*['"]([^'"]+)['"][^;]*form\.method\s*=\s*['"]POST['"]`,
		1, -1)
	
	// 8. 其他HTTP库
	d.addPattern("request-post", 
		`request\.post\s*\(\s*['"]([^'"]+)['"]`,
		1, -1)
	
	// 9. 通用HTTP方法检测
	d.addPattern("http-post-method", 
		`(post|POST)\s*\(\s*['"]([^'"]+)['"]`,
		2, -1)
}

// addPattern 添加检测模式
func (d *POSTRequestDetector) addPattern(name, pattern string, urlIndex, methodIndex int) {
	compiled := regexp.MustCompile(pattern)
	d.patterns = append(d.patterns, &POSTPattern{
		name:        name,
		pattern:     compiled,
		urlIndex:    urlIndex,
		methodIndex: methodIndex,
	})
}

// DetectFromHTML 从HTML内容中检测POST请求
func (d *POSTRequestDetector) DetectFromHTML(htmlContent, baseURL string) []*DetectedPOSTRequest {
	requests := make([]*DetectedPOSTRequest, 0)
	seen := make(map[string]bool)
	
	// 1. 检测HTML表单
	formRequests := d.detectHTMLForms(htmlContent, baseURL)
	for _, req := range formRequests {
		key := req.URL + "|" + req.Method
		if !seen[key] {
			seen[key] = true
			requests = append(requests, req)
		}
	}
	
	// 2. 检测内联JavaScript
	scriptRequests := d.detectInlineScripts(htmlContent, baseURL)
	for _, req := range scriptRequests {
		key := req.URL + "|" + req.Method
		if !seen[key] {
			seen[key] = true
			requests = append(requests, req)
		}
	}
	
	return requests
}

// DetectFromJS 从JavaScript代码中检测POST请求
func (d *POSTRequestDetector) DetectFromJS(jsCode, baseURL string) []*DetectedPOSTRequest {
	requests := make([]*DetectedPOSTRequest, 0)
	seen := make(map[string]bool)
	
	for _, pattern := range d.patterns {
		matches := pattern.pattern.FindAllStringSubmatch(jsCode, -1)
		
		for _, match := range matches {
			if len(match) > pattern.urlIndex {
				url := match[pattern.urlIndex]
				
				// 跳过无效URL
				if url == "" || url == "/" || url == "#" {
					continue
				}
				
				// 构建完整URL
				fullURL := url
				if !strings.HasPrefix(url, "http") {
					fullURL = resolveURLSimple(baseURL, url)
				}
				
				// 去重
				key := fullURL + "|POST"
				if seen[key] {
					continue
				}
				seen[key] = true
				
				// 创建请求对象
				req := &DetectedPOSTRequest{
					URL:    fullURL,
					Method: "POST",
					Source: pattern.name,
				}
				
				// 尝试提取参数
				req.Parameters = d.extractParametersNearby(jsCode, match[0])
				
				requests = append(requests, req)
			}
		}
	}
	
	return requests
}

// detectHTMLForms 检测HTML表单
func (d *POSTRequestDetector) detectHTMLForms(htmlContent, baseURL string) []*DetectedPOSTRequest {
	requests := make([]*DetectedPOSTRequest, 0)
	
	// 匹配POST表单
	formPattern := regexp.MustCompile(`(?i)<form[^>]*>[\s\S]*?</form>`)
	forms := formPattern.FindAllString(htmlContent, -1)
	
	for _, form := range forms {
		// 检查是否为POST方法
		methodMatch := regexp.MustCompile(`(?i)method\s*=\s*['"]?(POST|post)['"]?`)
		if !methodMatch.MatchString(form) {
			continue
		}
		
		// 提取action
		actionPattern := regexp.MustCompile(`(?i)action\s*=\s*['"]([^'"]+)['"]`)
		actionMatch := actionPattern.FindStringSubmatch(form)
		
		var action string
		if len(actionMatch) > 1 {
			action = actionMatch[1]
		} else {
			action = baseURL // 默认提交到当前页面
		}
		
		// 构建完整URL
		fullURL := action
		if !strings.HasPrefix(action, "http") {
			fullURL = resolveURLSimple(baseURL, action)
		}
		
		// 提取表单字段
		parameters := d.extractFormFields(form)
		
		// 检测enctype
		contentType := "application/x-www-form-urlencoded"
		enctypePattern := regexp.MustCompile(`(?i)enctype\s*=\s*['"]([^'"]+)['"]`)
		enctypeMatch := enctypePattern.FindStringSubmatch(form)
		if len(enctypeMatch) > 1 {
			contentType = enctypeMatch[1]
		}
		
		req := &DetectedPOSTRequest{
			URL:         fullURL,
			Method:      "POST",
			Parameters:  parameters,
			ContentType: contentType,
			Source:      "html-form",
		}
		
		requests = append(requests, req)
	}
	
	return requests
}

// detectInlineScripts 检测内联脚本中的POST请求
func (d *POSTRequestDetector) detectInlineScripts(htmlContent, baseURL string) []*DetectedPOSTRequest {
	requests := make([]*DetectedPOSTRequest, 0)
	
	// 提取<script>标签
	scriptPattern := regexp.MustCompile(`(?i)<script[^>]*>([\s\S]*?)</script>`)
	scripts := scriptPattern.FindAllStringSubmatch(htmlContent, -1)
	
	for _, script := range scripts {
		if len(script) > 1 {
			jsCode := script[1]
			scriptRequests := d.DetectFromJS(jsCode, baseURL)
			requests = append(requests, scriptRequests...)
		}
	}
	
	return requests
}

// extractFormFields 提取表单字段
func (d *POSTRequestDetector) extractFormFields(formHTML string) map[string]string {
	parameters := make(map[string]string)
	
	// 匹配input, select, textarea字段
	fieldPattern := regexp.MustCompile(`(?i)<(input|select|textarea)[^>]*>`)
	fields := fieldPattern.FindAllString(formHTML, -1)
	
	for _, field := range fields {
		// 提取name
		namePattern := regexp.MustCompile(`(?i)name\s*=\s*['"]([^'"]+)['"]`)
		nameMatch := namePattern.FindStringSubmatch(field)
		
		if len(nameMatch) > 1 {
			name := nameMatch[1]
			
			// 提取value
			valuePattern := regexp.MustCompile(`(?i)value\s*=\s*['"]([^'"]+)['"]`)
			valueMatch := valuePattern.FindStringSubmatch(field)
			
			var value string
			if len(valueMatch) > 1 {
				value = valueMatch[1]
			} else {
				// 根据字段类型设置默认值
				typePattern := regexp.MustCompile(`(?i)type\s*=\s*['"]([^'"]+)['"]`)
				typeMatch := typePattern.FindStringSubmatch(field)
				
				if len(typeMatch) > 1 {
					fieldType := strings.ToLower(typeMatch[1])
					value = d.getDefaultValue(fieldType, name)
				} else {
					value = "test_value"
				}
			}
			
			parameters[name] = value
		}
	}
	
	return parameters
}

// extractParametersNearby 从代码上下文中提取参数
func (d *POSTRequestDetector) extractParametersNearby(jsCode, matchedText string) map[string]string {
	parameters := make(map[string]string)
	
	// 找到匹配文本的位置
	index := strings.Index(jsCode, matchedText)
	if index == -1 {
		return parameters
	}
	
	// 提取周围200个字符的上下文
	start := index - 200
	if start < 0 {
		start = 0
	}
	end := index + len(matchedText) + 200
	if end > len(jsCode) {
		end = len(jsCode)
	}
	
	context := jsCode[start:end]
	
	// 提取data对象或参数对象
	dataPattern := regexp.MustCompile(`data\s*:\s*\{([^}]+)\}`)
	dataMatch := dataPattern.FindStringSubmatch(context)
	
	if len(dataMatch) > 1 {
		// 解析参数
		paramsText := dataMatch[1]
		paramPattern := regexp.MustCompile(`['"]?(\w+)['"]?\s*:\s*['"]?([^'",}]+)['"]?`)
		paramMatches := paramPattern.FindAllStringSubmatch(paramsText, -1)
		
		for _, pm := range paramMatches {
			if len(pm) > 2 {
				key := strings.Trim(pm[1], `"' `)
				value := strings.Trim(pm[2], `"' `)
				parameters[key] = value
			}
		}
	}
	
	return parameters
}

// getDefaultValue 根据字段类型返回默认值
func (d *POSTRequestDetector) getDefaultValue(fieldType, fieldName string) string {
	fieldName = strings.ToLower(fieldName)
	
	switch {
	case strings.Contains(fieldName, "email"):
		return "test@example.com"
	case strings.Contains(fieldName, "user") || strings.Contains(fieldName, "username"):
		return "admin"
	case strings.Contains(fieldName, "pass") || strings.Contains(fieldName, "password"):
		return "password123"
	case strings.Contains(fieldName, "phone") || strings.Contains(fieldName, "mobile"):
		return "13800138000"
	case strings.Contains(fieldName, "code") || strings.Contains(fieldName, "captcha"):
		return "1234"
	case fieldType == "hidden":
		return "hidden_value"
	case fieldType == "password":
		return "password123"
	case fieldType == "email":
		return "test@example.com"
	case fieldType == "number":
		return "123"
	case fieldType == "tel":
		return "13800138000"
	default:
		return "test_value"
	}
}

// resolveURLSimple 简单的URL解析
func resolveURLSimple(baseURL, relativeURL string) string {
	// 如果是绝对URL，直接返回
	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}
	
	// 如果以/开头，是绝对路径
	if strings.HasPrefix(relativeURL, "/") {
		// 提取baseURL的scheme和host
		parts := strings.SplitN(baseURL, "://", 2)
		if len(parts) != 2 {
			return relativeURL
		}
		
		scheme := parts[0]
		hostAndPath := parts[1]
		host := strings.Split(hostAndPath, "/")[0]
		
		return fmt.Sprintf("%s://%s%s", scheme, host, relativeURL)
	}
	
	// 相对路径，拼接到baseURL
	baseURL = strings.TrimSuffix(baseURL, "/")
	return baseURL + "/" + relativeURL
}

// PrintReport 打印检测报告
func (d *POSTRequestDetector) PrintReport(requests []*DetectedPOSTRequest) {
	if len(requests) == 0 {
		fmt.Println("\n[POST检测] 未检测到POST请求")
		return
	}
	
	fmt.Printf("\n[POST检测] 检测到 %d 个POST请求:\n", len(requests))
	fmt.Println(strings.Repeat("=", 70))
	
	for i, req := range requests {
		fmt.Printf("\n%d. %s %s\n", i+1, req.Method, req.URL)
		fmt.Printf("   来源: %s\n", req.Source)
		
		if req.ContentType != "" {
			fmt.Printf("   Content-Type: %s\n", req.ContentType)
		}
		
		if len(req.Parameters) > 0 {
			fmt.Printf("   参数 (%d个):\n", len(req.Parameters))
			for k, v := range req.Parameters {
				if len(v) > 50 {
					v = v[:50] + "..."
				}
				fmt.Printf("     %s = %s\n", k, v)
			}
		}
	}
	
	fmt.Println(strings.Repeat("=", 70))
}

