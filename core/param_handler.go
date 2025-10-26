package core

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// ParamHandler URL参数处理器
type ParamHandler struct {
	mutex sync.Mutex
	// 安全参数模式
	securityParams []string
	// 常见的危险参数
	dangerousParams []string
	// 文件包含参数
	fileInclusionParams []string
}

// NewParamHandler 创建参数处理器实例
func NewParamHandler() *ParamHandler {
	return &ParamHandler{
		securityParams: []string{
			"debug", "test", "admin", "password", "token", "auth", "session", "key",
			"secret", "config", "settings", "backup", "tmp", "temp", "dev", "development",
			"prod", "production", "staging", "internal", "private", "hidden", "beta",
			"alpha", "demo", "example", "sample", "preview", "draft",
		},
		dangerousParams: []string{
			"file", "path", "dir", "folder", "page", "url", "link", "src", "include",
			"require", "load", "read", "open", "download", "upload", "exec", "cmd",
			"command", "system", "shell", "eval", "code", "script", "function", "method",
			"class", "module", "plugin", "extension", "callback", "redirect", "return",
		},
		fileInclusionParams: []string{
			"file", "filename", "filepath", "path", "page", "template", "skin", "theme",
			"layout", "view", "inc", "include", "require", "load", "import", "read",
			"open", "get", "fetch", "pull", "download", "cat", "type", "show", "display",
		},
	}
}

// MergeParams 将识别到的参数自动拼接到对应URL中
func (p *ParamHandler) MergeParams(baseURL string, params []string) ([]string, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	// 获取现有查询参数
	queryParams := parsedURL.Query()

	// 将新参数添加到查询参数中
	// 这里简化处理，实际应用中可能需要更复杂的逻辑
	for i, param := range params {
		// 创建参数名（使用param_前缀避免冲突）
		paramName := "param_" + string(rune(i+'a'))
		if i >= 26 {
			paramName = "param_" + string(rune(i%26+'a')) + string(rune(i/26+'a'))
		}

		// 添加参数值
		queryParams.Add(paramName, param)
	}

	// 更新URL的查询参数
	parsedURL.RawQuery = queryParams.Encode()

	// 返回处理后的URL
	return []string{parsedURL.String()}, nil
}

// NormalizeURL 规范化URL，移除无关参数
func (p *ParamHandler) NormalizeURL(targetURL string) (string, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}

	// 获取查询参数
	queryParams := parsedURL.Query()

	// 移除常见的会话参数
	sessionParams := []string{"jsessionid", "phpsessid", "asp.net_sessionid", "sid"}
	for _, param := range sessionParams {
		queryParams.Del(param)
	}

	// 更新URL
	parsedURL.RawQuery = queryParams.Encode()

	return parsedURL.String(), nil
}

// CompareURLs 比较两个URL是否相似（忽略参数差异）
func (p *ParamHandler) CompareURLs(url1, url2 string) (bool, error) {
	parsedURL1, err := url.Parse(url1)
	if err != nil {
		return false, err
	}

	parsedURL2, err := url.Parse(url2)
	if err != nil {
		return false, err
	}

	// 比较协议、主机和路径
	if parsedURL1.Scheme != parsedURL2.Scheme ||
		parsedURL1.Host != parsedURL2.Host ||
		parsedURL1.Path != parsedURL2.Path {
		return false, nil
	}

	// 可以添加更复杂的比较逻辑
	// 例如比较参数结构而不是具体值

	return true, nil
}

// ExtractPathPatterns 从URL列表中提取路径模式
func (p *ParamHandler) ExtractPathPatterns(urls []string) []string {
	patterns := make([]string, 0)

	// 统计路径段出现频率
	pathSegments := make(map[string]int)

	for _, urlString := range urls {
		parsedURL, err := url.Parse(urlString)
		if err != nil {
			continue
		}

		// 分割路径
		segments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		for _, segment := range segments {
			// 如果段看起来像ID（数字），则归类为参数化段
			if isNumericString(segment) {
				pathSegments["{id}"]++
			} else {
				pathSegments[segment]++
			}
		}
	}

	// 构建模式
	// 这里简化实现，实际应用中可以使用更复杂的模式识别算法
	for pattern := range pathSegments {
		patterns = append(patterns, pattern)
	}

	return patterns
}

// ExtractParams 从URL中提取参数
func (p *ParamHandler) ExtractParams(targetURL string) (map[string][]string, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	// 返回查询参数
	return parsedURL.Query(), nil
}

// parseInt 安全地将字符串转换为整数
func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// GenerateParamVariations 生成参数变体
func (ph *ParamHandler) GenerateParamVariations(baseURL string) []string {
	variations := make([]string, 0)

	// 解析基础URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return variations
	}

	// 获取查询参数
	params := parsedURL.Query()

	// 如果没有参数，直接返回原URL
	if len(params) == 0 {
		return []string{baseURL}
	}

	// 生成不同的参数组合
	// 1. 原始参数
	variations = append(variations, baseURL)

	// 2. 添加常见参数变体
	commonParams := []string{"id", "page", "category", "product", "user", "token"}

	for _, paramName := range commonParams {
		// 检查是否已存在该参数
		if _, exists := params[paramName]; !exists {
			// 添加参数变体
			newParams := url.Values{}
			for k, v := range params {
				newParams[k] = v
			}
			newParams.Set(paramName, "1") // 添加默认值

			// 构造新URL
			newURL := *parsedURL
			newURL.RawQuery = newParams.Encode()
			variations = append(variations, newURL.String())

			// 优化: 每个参数名只测试一个默认值，避免URL爆炸
			// 如需测试多个值，请使用专门的Fuzzer工具
			// commonValues := []string{"2", "admin", "test", "debug", "123"}
			// for _, value := range commonValues {
			// 	newParams.Set(paramName, value)
			// 	newURL.RawQuery = newParams.Encode()
			// 	variations = append(variations, newURL.String())
			// }
		}
	}

	// === 移除：HPP (HTTP Parameter Pollution) 变体 ===
	// HTTP参数污染测试属于攻击性测试，不适合纯爬虫工具
	// 如需测试，请使用专业的安全测试工具

	// 4. 特定于目标站点的参数变体
	// cart.php 相关参数
	if strings.Contains(baseURL, "cart.php") {
		cartParams := []string{"price", "addcart"}
		for _, paramName := range cartParams {
			if _, exists := params[paramName]; !exists {
				newParams := url.Values{}
				for k, v := range params {
					newParams[k] = v
				}
				// 为购物车添加典型的参数值
				if paramName == "price" {
					newParams.Set(paramName, "199")
				} else if paramName == "addcart" {
					newParams.Set(paramName, "1")
				}

				newURL := *parsedURL
				newURL.RawQuery = newParams.Encode()
				variations = append(variations, newURL.String())
			}
		}
	}

	// 5. showimage.php 相关参数
	if strings.Contains(baseURL, "showimage.php") {
		imageParams := []string{"file", "size"}
		for _, paramName := range imageParams {
			if _, exists := params[paramName]; !exists {
				newParams := url.Values{}
				for k, v := range params {
					newParams[k] = v
				}
				// 为图片显示添加典型的参数值
				if paramName == "file" {
					newParams.Set(paramName, "./pictures/1.jpg")
				} else if paramName == "size" {
					newParams.Set(paramName, "160")
				}

				newURL := *parsedURL
				newURL.RawQuery = newParams.Encode()
				variations = append(variations, newURL.String())
			}
		}
	}

	// 6. 移除部分参数的变体
	// 只有当原始URL有多个参数时才生成
	if len(params) > 1 {
		paramNames := make([]string, 0, len(params))
		for paramName := range params {
			paramNames = append(paramNames, paramName)
		}

		// 为每个参数生成一个移除了该参数的变体
		for _, paramNameToRemove := range paramNames {
			newParams := url.Values{}
			for k, v := range params {
				if k != paramNameToRemove {
					newParams[k] = v
				}
			}

			newURL := *parsedURL
			newURL.RawQuery = newParams.Encode()
			variations = append(variations, newURL.String())
		}
	}

	// 去重
	uniqueVariations := make([]string, 0)
	seen := make(map[string]bool)
	for _, variation := range variations {
		if !seen[variation] {
			seen[variation] = true
			uniqueVariations = append(uniqueVariations, variation)
		}
	}

	return uniqueVariations
}

// DiscoverParametersFromMultipleSources 从多个来源发现参数
func (ph *ParamHandler) DiscoverParametersFromMultipleSources(htmlContent, jsContent, headers string) []string {
	allParams := make(map[string]bool)

	// 从HTML表单中发现参数
	htmlParams := ph.extractParamsFromHTML(htmlContent)
	for _, param := range htmlParams {
		allParams[param] = true
	}

	// 从JavaScript中发现参数
	jsParams := ph.extractParamsFromJS(jsContent)
	for _, param := range jsParams {
		allParams[param] = true
	}

	// 从HTTP响应头中发现参数
	headerParams := ph.extractParamsFromHeaders(headers)
	for _, param := range headerParams {
		allParams[param] = true
	}

	// 从HTML注释中发现参数
	commentParams := ph.extractParamsFromComments(htmlContent)
	for _, param := range commentParams {
		allParams[param] = true
	}

	// 转换为slice
	result := make([]string, 0, len(allParams))
	for param := range allParams {
		result = append(result, param)
	}

	return result
}

// extractParamsFromHTML 从HTML内容中提取参数
func (ph *ParamHandler) extractParamsFromHTML(htmlContent string) []string {
	params := make([]string, 0)

	// 匹配input, select, textarea的name属性
	patterns := []string{
		`<input[^>]+name\s*=\s*['"]([\w\[\]]+)['"][^>]*>`,
		`<select[^>]+name\s*=\s*['"]([\w\[\]]+)['"][^>]*>`,
		`<textarea[^>]+name\s*=\s*['"]([\w\[\]]+)['"][^>]*>`,
		// 隐藏域参数
		`<input[^>]+type\s*=\s*['"]hidden['"][^>]+name\s*=\s*['"]([\w\[\]]+)['"][^>]*>`,
		// data-* 属性中的参数
		`data-[\w-]+\s*=\s*['"]([^'"]+)['"]`,
		// URL中的参数引用
		`[\?&]([\w\[\]]+)=`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(htmlContent, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				params = append(params, match[1])
			}
		}
	}

	return params
}

// extractParamsFromJS 从JavaScript内容中提取参数
func (ph *ParamHandler) extractParamsFromJS(jsContent string) []string {
	params := make([]string, 0)

	patterns := []string{
		// 对象属性
		`['"]([^'"]+)['"]\s*:\s*['"][^'"]*['"]`,
		// 变量赋值
		`(?:var|let|const)\s+([\w$]+)\s*=`,
		// 函数参数
		`function\s+\w+\s*\(\s*([\w,\s]+)\s*\)`,
		// API调用中的参数
		`[\?&]([\w\[\]]+)=`,
		// POST数据中的参数
		`['"]([\w\[\]]+)['"]:\s*[^,}]+`,
		// FormData参数
		`(?:append|set)\s*\(\s*['"]([^'"]+)['"]`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsContent, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				// 过滤掉JavaScript关键字和常见函数名
				if !isJavaScriptKeyword(match[1]) {
					params = append(params, match[1])
				}
			}
		}
	}

	return params
}

// extractParamsFromHeaders 从HTTP响应头中提取参数
func (ph *ParamHandler) extractParamsFromHeaders(headers string) []string {
	params := make([]string, 0)

	patterns := []string{
		// Set-Cookie中的参数
		`Set-Cookie:\s*([^=]+)=`,
		// X-* 自定义头中的参数名
		`X-([\w-]+):`,
		// Location重定向中的参数
		`Location:.*[\?&]([\w\[\]]+)=`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(headers, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				params = append(params, match[1])
			}
		}
	}

	return params
}

// extractParamsFromComments 从HTML注释中提取参数
func (ph *ParamHandler) extractParamsFromComments(htmlContent string) []string {
	params := make([]string, 0)

	// 匹配HTML注释
	commentPattern := `<!--(.*?)-->`
	re := regexp.MustCompile(commentPattern)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			comment := match[1]
			// 在注释中查找参数模式
			paramPatterns := []string{
				`[\?&]([\w\[\]]+)=`,
				`['"]([^'"]+)['"]\s*:\s*`,
				`param\s*:\s*['"]([^'"]+)['"]`,
				`parameter\s*:\s*['"]([^'"]+)['"]`,
				`field\s*:\s*['"]([^'"]+)['"]`,
			}

			for _, pattern := range paramPatterns {
				paramRe := regexp.MustCompile(pattern)
				paramMatches := paramRe.FindAllStringSubmatch(comment, -1)
				for _, paramMatch := range paramMatches {
					if len(paramMatch) > 1 && paramMatch[1] != "" {
						params = append(params, paramMatch[1])
					}
				}
			}
		}
	}

	return params
}

// AnalyzeParameterSecurity 分析参数的安全风险
func (ph *ParamHandler) AnalyzeParameterSecurity(paramName string) (string, int) {
	paramLower := strings.ToLower(paramName)

	// 检查是否为危险参数
	for _, dangerous := range ph.dangerousParams {
		if strings.Contains(paramLower, dangerous) {
			return fmt.Sprintf("DANGEROUS: 可能存在%s相关漏洞", dangerous), 3
		}
	}

	// 检查是否为文件包含参数
	for _, fileParam := range ph.fileInclusionParams {
		if strings.Contains(paramLower, fileParam) {
			return "FILE_INCLUSION: 可能存在文件包含漏洞", 3
		}
	}

	// 检查是否为安全相关参数
	for _, secParam := range ph.securityParams {
		if strings.Contains(paramLower, secParam) {
			return fmt.Sprintf("SECURITY: %s相关参数，需要重点测试", secParam), 2
		}
	}

	// 检查SQL注入风险参数
	sqlParams := []string{"id", "user", "product", "category", "search", "query", "name", "email"}
	for _, sqlParam := range sqlParams {
		if strings.Contains(paramLower, sqlParam) {
			return "SQL_INJECTION: 可能存在SQL注入漏洞", 2
		}
	}

	// 检查XSS风险参数
	xssParams := []string{"message", "comment", "content", "text", "description", "title", "subject"}
	for _, xssParam := range xssParams {
		if strings.Contains(paramLower, xssParam) {
			return "XSS: 可能存在跨站脚本漏洞", 2
		}
	}

	return "INFO: 常规参数", 1
}

// GenerateSecurityTestVariations 生成安全测试参数变体
// ⚠️ 已弃用：作为纯爬虫工具，不应包含攻击性payload
// 该函数保留但不再被调用，如需安全测试请使用专业工具（sqlmap、Burp等）
func (ph *ParamHandler) GenerateSecurityTestVariations(baseURL string) []string {
	// 作为纯爬虫工具，返回空列表
	// 如需安全测试，请使用：
	// - sqlmap: SQL注入检测
	// - Burp Suite: 综合漏洞扫描
	// - XSStrike: XSS检测
	// - Nuclei: 漏洞扫描框架
	return []string{}
}

// GenerateParameterFuzzList 生成参数模糊测试列表
func (ph *ParamHandler) GenerateParameterFuzzList(targetURL string) []string {
	fuzzList := make([]string, 0)

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fuzzList
	}

	// 基础URL（无参数）
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path

	// 常见参数名列表
	commonParams := []string{
		// 通用参数
		"id", "page", "limit", "offset", "sort", "order", "search", "q", "query",
		"filter", "category", "type", "status", "action", "method", "format",

		// 用户相关
		"user", "username", "userid", "uid", "email", "password", "pass", "pwd",
		"token", "auth", "session", "key", "api_key", "access_token",

		// 文件相关
		"file", "filename", "path", "dir", "folder", "upload", "download",
		"image", "img", "pic", "photo", "document", "doc", "pdf",

		// 数据库相关
		"table", "column", "field", "record", "row", "data", "value",
		"insert", "update", "delete", "select", "where", "join",

		// 系统相关
		"cmd", "command", "exec", "system", "shell", "script", "function",
		"class", "method", "module", "plugin", "extension", "callback",

		// 调试相关
		"debug", "test", "dev", "development", "staging", "prod", "production",
		"admin", "administrator", "root", "config", "settings", "options",

		// 重定向相关
		"redirect", "return", "next", "continue", "url", "link", "ref", "referer",
		"target", "destination", "forward", "back", "home", "exit",

		// 特殊功能
		"preview", "view", "show", "display", "print", "export", "import",
		"backup", "restore", "reset", "clear", "clean", "flush", "cache",
	}

	// 为每个参数生成测试URL
	for _, param := range commonParams {
		// 基本测试值
		testValues := []string{"1", "test", "admin", "../", "null", "true", "false"}

		for _, value := range testValues {
			testURL := fmt.Sprintf("%s?%s=%s", baseURL, param, value)
			fuzzList = append(fuzzList, testURL)
		}

		// 空值测试
		fuzzList = append(fuzzList, fmt.Sprintf("%s?%s=", baseURL, param))

		// 数组参数测试
		fuzzList = append(fuzzList, fmt.Sprintf("%s?%s[]=1", baseURL, param))
		fuzzList = append(fuzzList, fmt.Sprintf("%s?%s[0]=1", baseURL, param))
	}

	return fuzzList
}

// isJavaScriptKeyword 检查是否为JavaScript关键字
func isJavaScriptKeyword(word string) bool {
	keywords := map[string]bool{
		"function": true, "var": true, "let": true, "const": true, "if": true,
		"else": true, "for": true, "while": true, "do": true, "switch": true,
		"case": true, "default": true, "break": true, "continue": true, "return": true,
		"try": true, "catch": true, "finally": true, "throw": true, "new": true,
		"this": true, "typeof": true, "instanceof": true, "delete": true, "void": true,
		"true": true, "false": true, "null": true, "undefined": true, "NaN": true,
		"Infinity": true, "Array": true, "Object": true, "String": true, "Number": true,
		"Boolean": true, "Date": true, "Math": true, "JSON": true, "console": true,
		"window": true, "document": true, "location": true, "history": true,
		"length": true, "prototype": true, "constructor": true, "toString": true,
		"valueOf": true, "hasOwnProperty": true, "isPrototypeOf": true,
	}

	_, isKeyword := keywords[strings.ToLower(word)]
	return isKeyword
}

// GeneratePOSTVariations 生成POST请求参数变体（用于爬虫测试，不含攻击payload）
// ⚠️ 已修改：移除了所有攻击性payload，只保留正常的参数变体
func (ph *ParamHandler) GeneratePOSTVariations(postReq POSTRequest) []POSTRequest {
	variations := make([]POSTRequest, 0)

	// 添加原始请求
	variations = append(variations, postReq)

	// 如果没有参数，返回空变体，让调用者决定是否进行爆破
	if len(postReq.Parameters) == 0 {
		return variations
	}

	// === 移除：SQL注入、XSS、命令注入等攻击性payload ===
	// 作为纯爬虫工具，不应包含攻击性测试
	// 如需安全测试，请导出URL和表单后使用专业工具

	// === 保留：正常的参数变体（用于爬虫测试） ===

	// 1. 参数值变化（使用正常值）
	normalValues := []string{"1", "2", "test", "admin", "true", "false"}
	for paramName := range postReq.Parameters {
		for _, value := range normalValues {
			newReq := ph.clonePOSTRequest(postReq)
			newReq.Parameters[paramName] = value
			newReq.Body = ph.buildPOSTBody(newReq.Parameters)
			variations = append(variations, newReq)
		}
	}

	// 2. 空值测试（正常的边界测试）
	for paramName := range postReq.Parameters {
		newReq := ph.clonePOSTRequest(postReq)
		newReq.Parameters[paramName] = ""
		newReq.Body = ph.buildPOSTBody(newReq.Parameters)
		variations = append(variations, newReq)
	}

	// 3. 数组参数测试（正常的格式测试）
	for paramName, paramValue := range postReq.Parameters {
		newReq := ph.clonePOSTRequest(postReq)
		newReq.Parameters[paramName+"[]"] = paramValue
		delete(newReq.Parameters, paramName)
		newReq.Body = ph.buildPOSTBody(newReq.Parameters)
		variations = append(variations, newReq)
	}

	return variations
}

// ExtractPOSTParameters 从POSTRequest中提取参数信息（用于分析）
func (ph *ParamHandler) ExtractPOSTParameters(postReq POSTRequest) []map[string]interface{} {
	params := make([]map[string]interface{}, 0)

	for name, value := range postReq.Parameters {
		paramInfo := map[string]interface{}{
			"name":         name,
			"value":        value,
			"value_length": len(value),
			"from_form":    postReq.FromForm,
		}

		// 安全分析
		risk, level := ph.AnalyzeParameterSecurity(name)
		paramInfo["security_risk"] = risk
		paramInfo["risk_level"] = level

		// 值类型分析
		if _, err := strconv.Atoi(value); err == nil {
			paramInfo["value_type"] = "number"
		} else if matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, value); matched {
			paramInfo["value_type"] = "email"
		} else if matched, _ := regexp.MatchString(`^https?://`, value); matched {
			paramInfo["value_type"] = "url"
		} else {
			paramInfo["value_type"] = "text"
		}

		params = append(params, paramInfo)
	}

	return params
}

// clonePOSTRequest 克隆POST请求
func (ph *ParamHandler) clonePOSTRequest(req POSTRequest) POSTRequest {
	newReq := POSTRequest{
		URL:         req.URL,
		Method:      req.Method,
		Parameters:  make(map[string]string),
		Body:        req.Body,
		ContentType: req.ContentType,
		FromForm:    req.FromForm,
		FormAction:  req.FormAction,
	}

	// 深拷贝参数
	for k, v := range req.Parameters {
		newReq.Parameters[k] = v
	}

	return newReq
}

// buildPOSTBody 构建POST请求体
func (ph *ParamHandler) buildPOSTBody(parameters map[string]string) string {
	values := url.Values{}
	for key, value := range parameters {
		values.Add(key, value)
	}
	return values.Encode()
}

// GeneratePOSTParameterFuzzList 生成POST参数爆破列表（用于无参数表单）
func (ph *ParamHandler) GeneratePOSTParameterFuzzList(baseURL string) []POSTRequest {
	fuzzList := make([]POSTRequest, 0)

	// 常见POST参数组合（按场景分类）
	postParamCombinations := []map[string]string{
		// === 认证/登录场景 ===
		{"username": "admin", "password": "admin123"},
		{"username": "test", "password": "test123"},
		{"user": "admin", "pass": "admin123"},
		{"email": "admin@test.com", "password": "admin123"},
		{"login": "admin", "pwd": "admin123"},
		{"account": "admin", "password": "admin123"},
		{"uname": "admin", "upass": "admin123"},

		// === 用户信息场景 ===
		{"username": "testuser", "email": "test@example.com", "password": "Test@123"},
		{"name": "Test User", "email": "test@example.com", "phone": "13800138000"},
		{"firstname": "Test", "lastname": "User", "email": "test@example.com"},

		// === 搜索场景 ===
		{"search": "test", "q": "admin"},
		{"query": "test", "type": "all"},
		{"keyword": "admin", "category": "1"},
		{"s": "test"},

		// === 数据操作场景 ===
		{"id": "1", "action": "update"},
		{"id": "1", "action": "delete"},
		{"userid": "1", "operation": "edit"},
		{"item_id": "1", "quantity": "1"},

		// === 文件操作场景 ===
		{"file": "test.txt", "action": "read"},
		{"filename": "../../../etc/passwd"},
		{"path": "/tmp/test"},
		{"upload": "test.php"},

		// === 评论/留言场景 ===
		{"comment": "test comment", "author": "Test User"},
		{"message": "test message", "name": "Test"},
		{"content": "test content", "title": "Test Title"},
		{"text": "test text", "user": "admin"},

		// === API测试场景 ===
		{"api_key": "test123", "action": "list"},
		{"token": "abc123def456", "method": "get"},
		{"auth": "Bearer test123", "resource": "users"},
		{"key": "test", "secret": "secret123"},

		// === 系统/调试场景 ===
		{"debug": "1", "show_errors": "1"},
		{"test": "1", "verbose": "1"},
		{"dev": "1", "trace": "1"},
		{"admin": "1", "mode": "debug"},

		// === 单参数测试 ===
		{"id": "1"},
		{"page": "1"},
		{"user": "admin"},
		{"action": "test"},
		{"cmd": "whoami"},
		{"file": "index.php"},
		{"data": "test"},
		{"value": "1"},
		{"key": "test"},
		{"token": "abc123"},
		{"session": "test123"},
		{"redirect": "/admin"},
		{"url": "http://evil.com"},
		{"callback": "alert(1)"},

		// === 常见字段名组合 ===
		{"username": "admin"},
		{"password": "admin123"},
		{"email": "test@example.com"},
		{"name": "Test"},
		{"phone": "13800138000"},
		{"address": "Test Address"},
		{"title": "Test Title"},
		{"description": "Test Description"},
		{"content": "Test Content"},
		{"message": "Test Message"},
		{"comment": "Test Comment"},
	}

	// 为每个参数组合生成POST请求
	for _, params := range postParamCombinations {
		body := ph.buildPOSTBody(params)

		postReq := POSTRequest{
			URL:         baseURL,
			Method:      "POST",
			Parameters:  params,
			Body:        body,
			ContentType: "application/x-www-form-urlencoded",
			FromForm:    false,
			FormAction:  baseURL,
		}

		fuzzList = append(fuzzList, postReq)
	}

	return fuzzList
}
