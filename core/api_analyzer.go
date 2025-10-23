package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// APIAnalyzer API智能分析器
type APIAnalyzer struct {
	endpoints      map[string]*APIEndpoint
	client         *http.Client
	targetDomain   string
	userAgent      string
	authHeader     string
	
	// 统计信息
	totalRequests  int
	successCount   int
	failureCount   int
}

// APIEndpoint API端点详细信息
type APIEndpoint struct {
	URL             string                 `json:"url"`
	Methods         []string               `json:"methods"`
	Parameters      []APIParameter         `json:"parameters"`
	Headers         map[string]string      `json:"headers"`
	RequestBody     interface{}            `json:"request_body,omitempty"`
	ResponseBody    interface{}            `json:"response_body,omitempty"`
	StatusCodes     []int                  `json:"status_codes"`
	ContentType     string                 `json:"content_type"`
	RequiresAuth    bool                   `json:"requires_auth"`
	RateLimit       *RateLimit             `json:"rate_limit,omitempty"`
	Description     string                 `json:"description,omitempty"`
	Examples        []APIExample           `json:"examples"`
	ResponseSchema  map[string]interface{} `json:"response_schema,omitempty"`
	ErrorResponses  []ErrorResponse        `json:"error_responses,omitempty"`
	
	// 推断的信息
	APIType         string                 `json:"api_type"` // REST, GraphQL, gRPC, SOAP
	Version         string                 `json:"version,omitempty"`
	Deprecated      bool                   `json:"deprecated"`
}

// APIParameter API参数
type APIParameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // query, path, header, body
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Description string      `json:"description,omitempty"`
	Example     interface{} `json:"example,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Format      string      `json:"format,omitempty"` // date, email, uuid等
}

// APIExample API请求/响应示例
type APIExample struct {
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	RequestHeaders map[string]string `json:"request_headers,omitempty"`
	RequestBody    string            `json:"request_body,omitempty"`
	ResponseStatus int               `json:"response_status"`
	ResponseBody   string            `json:"response_body"`
	Description    string            `json:"description,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	StatusCode  int    `json:"status_code"`
	Message     string `json:"message"`
	Example     string `json:"example"`
}

// RateLimit 速率限制信息
type RateLimit struct {
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	Reset     int64  `json:"reset"`
	Window    string `json:"window"`
}

// NewAPIAnalyzer 创建API分析器
func NewAPIAnalyzer(targetDomain string) *APIAnalyzer {
	return &APIAnalyzer{
		endpoints:    make(map[string]*APIEndpoint),
		client:       &http.Client{Timeout: 30 * time.Second},
		targetDomain: targetDomain,
		userAgent:    "Spider-Ultimate-API-Analyzer/2.5",
	}
}

// SetAuthentication 设置认证信息
func (aa *APIAnalyzer) SetAuthentication(authHeader string) {
	aa.authHeader = authHeader
}

// AnalyzeEndpoint 分析单个API端点
func (aa *APIAnalyzer) AnalyzeEndpoint(url string) (*APIEndpoint, error) {
	fmt.Printf("[API分析] 开始分析: %s\n", url)
	
	endpoint := &APIEndpoint{
		URL:          url,
		Methods:      make([]string, 0),
		Parameters:   make([]APIParameter, 0),
		Headers:      make(map[string]string),
		StatusCodes:  make([]int, 0),
		Examples:     make([]APIExample, 0),
		ErrorResponses: make([]ErrorResponse, 0),
	}
	
	// 1. 检测API类型
	apiType := aa.detectAPIType(url)
	endpoint.APIType = apiType
	fmt.Printf("  [API类型] %s\n", apiType)
	
	// 2. 尝试OPTIONS方法（获取支持的方法）
	allowedMethods := aa.testOptionsMethod(url)
	if len(allowedMethods) > 0 {
		endpoint.Methods = allowedMethods
		fmt.Printf("  [支持方法] %v\n", allowedMethods)
	} else {
		// 如果OPTIONS不可用，逐个测试常见方法
		endpoint.Methods = aa.probeHTTPMethods(url)
		fmt.Printf("  [探测方法] %v\n", endpoint.Methods)
	}
	
	// 3. 分析每个方法
	for _, method := range endpoint.Methods {
		fmt.Printf("  [测试方法] %s\n", method)
		
		resp, body, err := aa.sendRequest(method, url, nil, "")
		if err != nil {
			fmt.Printf("    ⚠️  请求失败: %v\n", err)
			continue
		}
		
		// 记录状态码
		if !containsInt(endpoint.StatusCodes, resp.StatusCode) {
			endpoint.StatusCodes = append(endpoint.StatusCodes, resp.StatusCode)
		}
		
		// 记录Content-Type
		if endpoint.ContentType == "" {
			endpoint.ContentType = resp.Header.Get("Content-Type")
		}
		
		// 检测认证要求
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			endpoint.RequiresAuth = true
			fmt.Printf("    🔐 需要认证\n")
			
			// 记录错误响应
			endpoint.ErrorResponses = append(endpoint.ErrorResponses, ErrorResponse{
				StatusCode: resp.StatusCode,
				Message:    "Authentication required",
				Example:    body,
			})
			continue
		}
		
		// 提取速率限制信息
		if rateLimit := aa.extractRateLimit(resp); rateLimit != nil {
			endpoint.RateLimit = rateLimit
			fmt.Printf("    ⏱️  速率限制: %d/%s\n", rateLimit.Limit, rateLimit.Window)
		}
		
		// 解析响应体
		if strings.Contains(endpoint.ContentType, "application/json") {
			// JSON响应
			var jsonData interface{}
			if err := json.Unmarshal([]byte(body), &jsonData); err == nil {
				endpoint.ResponseBody = jsonData
				
				// 生成响应Schema
				endpoint.ResponseSchema = aa.generateJSONSchema(jsonData)
				
				// 推断请求参数
				inferredParams := aa.inferParametersFromResponse(jsonData)
				endpoint.Parameters = append(endpoint.Parameters, inferredParams...)
				
				fmt.Printf("    ✅ JSON响应 (推断 %d 个参数)\n", len(inferredParams))
			}
		} else if strings.Contains(endpoint.ContentType, "text/html") {
			fmt.Printf("    ℹ️  HTML响应 (可能不是API)\n")
		}
		
		// 创建示例
		example := APIExample{
			Method:         method,
			URL:            url,
			RequestHeaders: aa.getDefaultHeaders(),
			ResponseStatus: resp.StatusCode,
			ResponseBody:   truncateString(body, 500),
		}
		endpoint.Examples = append(endpoint.Examples, example)
	}
	
	// 4. 测试参数（如果有推断的参数）
	if len(endpoint.Parameters) > 0 {
		fmt.Printf("  [测试参数] 共 %d 个\n", len(endpoint.Parameters))
		endpoint.Parameters = aa.testParameters(url, endpoint.Methods, endpoint.Parameters)
	}
	
	// 5. 提取版本信息
	endpoint.Version = aa.extractVersion(url, endpoint.Headers)
	
	// 6. 检测是否废弃
	endpoint.Deprecated = aa.isDeprecated(endpoint)
	
	// 保存到缓存
	aa.endpoints[url] = endpoint
	
	fmt.Printf("  [分析完成] 方法:%d, 参数:%d, 状态码:%v\n",
		len(endpoint.Methods), len(endpoint.Parameters), endpoint.StatusCodes)
	
	return endpoint, nil
}

// detectAPIType 检测API类型
func (aa *APIAnalyzer) detectAPIType(url string) string {
	urlLower := strings.ToLower(url)
	
	// GraphQL
	if strings.Contains(urlLower, "graphql") {
		return "GraphQL"
	}
	
	// gRPC (通常在特定端口)
	if strings.Contains(urlLower, ":50051") || strings.Contains(urlLower, "grpc") {
		return "gRPC"
	}
	
	// SOAP
	if strings.Contains(urlLower, "wsdl") || strings.Contains(urlLower, "soap") {
		return "SOAP"
	}
	
	// JSON-RPC
	if strings.Contains(urlLower, "rpc") || strings.Contains(urlLower, "jsonrpc") {
		return "JSON-RPC"
	}
	
	// 默认REST
	return "REST"
}

// testOptionsMethod 测试OPTIONS方法
func (aa *APIAnalyzer) testOptionsMethod(url string) []string {
	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		return nil
	}
	
	req.Header.Set("User-Agent", aa.userAgent)
	
	resp, err := aa.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	// 从Allow头获取支持的方法
	allowHeader := resp.Header.Get("Allow")
	if allowHeader == "" {
		return nil
	}
	
	methods := strings.Split(allowHeader, ",")
	result := make([]string, 0)
	for _, method := range methods {
		method = strings.TrimSpace(method)
		if method != "" {
			result = append(result, strings.ToUpper(method))
		}
	}
	
	return result
}

// probeHTTPMethods 探测HTTP方法
func (aa *APIAnalyzer) probeHTTPMethods(url string) []string {
	testMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"}
	supportedMethods := make([]string, 0)
	
	for _, method := range testMethods {
		resp, _, err := aa.sendRequest(method, url, nil, "")
		if err != nil {
			continue
		}
		
		// 405 = Method Not Allowed, 404 = Not Found
		if resp.StatusCode != 405 && resp.StatusCode != 404 {
			supportedMethods = append(supportedMethods, method)
		}
	}
	
	return supportedMethods
}

// sendRequest 发送HTTP请求
func (aa *APIAnalyzer) sendRequest(method, url string, headers map[string]string, body string) (*http.Response, string, error) {
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	
	var req *http.Request
	var err error
	if reqBody != nil {
		req, err = http.NewRequest(method, url, reqBody)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	
	if err != nil {
		return nil, "", err
	}
	
	// 设置默认头
	req.Header.Set("User-Agent", aa.userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	
	// 设置认证头
	if aa.authHeader != "" {
		req.Header.Set("Authorization", aa.authHeader)
	}
	
	// 设置自定义头
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	// 如果有body，设置Content-Type
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	
	aa.totalRequests++
	
	resp, err := aa.client.Do(req)
	if err != nil {
		aa.failureCount++
		return nil, "", err
	}
	defer resp.Body.Close()
	
	aa.successCount++
	
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, "", err
	}
	
	return resp, string(bodyBytes), nil
}

// inferParametersFromResponse 从响应推断请求参数
func (aa *APIAnalyzer) inferParametersFromResponse(data interface{}) []APIParameter {
	params := make([]APIParameter, 0)
	
	// 如果是对象，遍历字段
	if obj, ok := data.(map[string]interface{}); ok {
		for key, value := range obj {
			param := APIParameter{
				Name: key,
				In:   "query", // 默认query参数
				Type: aa.inferType(value),
			}
			
			// 设置示例值
			param.Example = value
			
			// 推断是否必需（简单启发式）
			if strings.Contains(strings.ToLower(key), "id") {
				param.Required = true
				param.In = "path"
			}
			
			params = append(params, param)
		}
	}
	
	// 如果是数组，分析数组元素
	if arr, ok := data.([]interface{}); ok && len(arr) > 0 {
		if obj, ok := arr[0].(map[string]interface{}); ok {
			return aa.inferParametersFromResponse(obj)
		}
	}
	
	return params
}

// testParameters 测试参数有效性
func (aa *APIAnalyzer) testParameters(url string, methods []string, params []APIParameter) []APIParameter {
	validatedParams := make([]APIParameter, 0)
	
	for _, param := range params {
		// 只测试前3个参数（避免过多请求）
		if len(validatedParams) >= 3 {
			validatedParams = append(validatedParams, param)
			continue
		}
		
		// 构造测试URL
		testURL := url
		testValue := aa.generateTestValue(param.Type)
		
		if param.In == "query" {
			sep := "?"
			if strings.Contains(url, "?") {
				sep = "&"
			}
			testURL = fmt.Sprintf("%s%s%s=%v", url, sep, param.Name, testValue)
		}
		
		// 尝试GET请求
		method := "GET"
		if len(methods) > 0 {
			method = methods[0]
		}
		
		resp, _, err := aa.sendRequest(method, testURL, nil, "")
		if err == nil && resp.StatusCode == 200 {
			param.Required = false // 不报错说明不是必需的
			fmt.Printf("    ✓ 参数 %s 有效\n", param.Name)
		}
		
		validatedParams = append(validatedParams, param)
	}
	
	return validatedParams
}

// generateTestValue 生成测试值
func (aa *APIAnalyzer) generateTestValue(paramType string) interface{} {
	switch paramType {
	case "string":
		return "test"
	case "integer", "number":
		return 1
	case "boolean":
		return true
	case "array":
		return []interface{}{"test"}
	case "object":
		return map[string]interface{}{"key": "value"}
	default:
		return "test"
	}
}

// inferType 推断数据类型
func (aa *APIAnalyzer) inferType(value interface{}) string {
	if value == nil {
		return "null"
	}
	
	switch value.(type) {
	case string:
		return "string"
	case float64, int, int64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// generateJSONSchema 生成JSON Schema
func (aa *APIAnalyzer) generateJSONSchema(data interface{}) map[string]interface{} {
	schema := make(map[string]interface{})
	
	switch v := data.(type) {
	case map[string]interface{}:
		schema["type"] = "object"
		properties := make(map[string]interface{})
		
		for key, value := range v {
			properties[key] = aa.generateJSONSchema(value)
		}
		
		schema["properties"] = properties
		
	case []interface{}:
		schema["type"] = "array"
		if len(v) > 0 {
			schema["items"] = aa.generateJSONSchema(v[0])
		}
		
	case string:
		schema["type"] = "string"
		
	case float64, int, int64:
		schema["type"] = "number"
		
	case bool:
		schema["type"] = "boolean"
		
	case nil:
		schema["type"] = "null"
	}
	
	return schema
}

// extractRateLimit 提取速率限制信息
func (aa *APIAnalyzer) extractRateLimit(resp *http.Response) *RateLimit {
	limit := resp.Header.Get("X-RateLimit-Limit")
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	reset := resp.Header.Get("X-RateLimit-Reset")
	
	if limit == "" && remaining == "" {
		return nil
	}
	
	rateLimit := &RateLimit{Window: "hour"}
	
	if limit != "" {
		fmt.Sscanf(limit, "%d", &rateLimit.Limit)
	}
	if remaining != "" {
		fmt.Sscanf(remaining, "%d", &rateLimit.Remaining)
	}
	if reset != "" {
		fmt.Sscanf(reset, "%d", &rateLimit.Reset)
	}
	
	return rateLimit
}

// extractVersion 提取版本信息
func (aa *APIAnalyzer) extractVersion(url string, headers map[string]string) string {
	// 从URL提取
	versionPattern := regexp.MustCompile(`/v(\d+)(/|$)`)
	matches := versionPattern.FindStringSubmatch(url)
	if len(matches) > 1 {
		return "v" + matches[1]
	}
	
	// 从headers提取
	if version := headers["X-API-Version"]; version != "" {
		return version
	}
	
	return ""
}

// isDeprecated 检测是否废弃
func (aa *APIAnalyzer) isDeprecated(endpoint *APIEndpoint) bool {
	// 检查URL中的deprecated关键字
	if strings.Contains(strings.ToLower(endpoint.URL), "deprecated") {
		return true
	}
	
	// 检查响应头
	if deprecated := endpoint.Headers["X-Deprecated"]; deprecated != "" {
		return true
	}
	
	return false
}

// getDefaultHeaders 获取默认请求头
func (aa *APIAnalyzer) getDefaultHeaders() map[string]string {
	headers := map[string]string{
		"User-Agent": aa.userAgent,
		"Accept":     "application/json",
	}
	
	if aa.authHeader != "" {
		headers["Authorization"] = aa.authHeader
	}
	
	return headers
}

// GetAllEndpoints 获取所有端点
func (aa *APIAnalyzer) GetAllEndpoints() []*APIEndpoint {
	endpoints := make([]*APIEndpoint, 0, len(aa.endpoints))
	for _, endpoint := range aa.endpoints {
		endpoints = append(endpoints, endpoint)
	}
	
	// 按URL排序
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].URL < endpoints[j].URL
	})
	
	return endpoints
}

// GetStatistics 获取统计信息
func (aa *APIAnalyzer) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"total_endpoints": len(aa.endpoints),
		"total_requests":  aa.totalRequests,
		"success_count":   aa.successCount,
		"failure_count":   aa.failureCount,
		"success_rate":    float64(aa.successCount) / float64(aa.totalRequests) * 100,
	}
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

