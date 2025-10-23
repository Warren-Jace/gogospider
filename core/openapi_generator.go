package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

// OpenAPIGenerator OpenAPI/Swagger文档生成器
type OpenAPIGenerator struct {
	analyzer *APIAnalyzer
}

// OpenAPISpec OpenAPI 3.0规范
type OpenAPISpec struct {
	OpenAPI string                 `json:"openapi"`
	Info    OpenAPIInfo            `json:"info"`
	Servers []OpenAPIServer        `json:"servers,omitempty"`
	Paths   map[string]interface{} `json:"paths"`
	Components *OpenAPIComponents  `json:"components,omitempty"`
	Security []map[string][]string `json:"security,omitempty"`
}

// OpenAPIInfo 信息
type OpenAPIInfo struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Version     string  `json:"version"`
	Contact     *Contact `json:"contact,omitempty"`
}

// Contact 联系信息
type Contact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// OpenAPIServer 服务器
type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// OpenAPIComponents 组件
type OpenAPIComponents struct {
	Schemas         map[string]interface{} `json:"schemas,omitempty"`
	SecuritySchemes map[string]interface{} `json:"securitySchemes,omitempty"`
}

// NewOpenAPIGenerator 创建OpenAPI生成器
func NewOpenAPIGenerator(analyzer *APIAnalyzer) *OpenAPIGenerator {
	return &OpenAPIGenerator{
		analyzer: analyzer,
	}
}

// Generate 生成OpenAPI文档
func (og *OpenAPIGenerator) Generate() (*OpenAPISpec, error) {
	endpoints := og.analyzer.GetAllEndpoints()
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("没有可用的API端点")
	}
	
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       fmt.Sprintf("%s API", og.analyzer.targetDomain),
			Description: fmt.Sprintf("由Spider Ultimate自动生成的API文档\n生成时间: %s", time.Now().Format("2006-01-02 15:04:05")),
			Version:     "1.0.0",
		},
		Servers: []OpenAPIServer{
			{
				URL:         fmt.Sprintf("https://%s", og.analyzer.targetDomain),
				Description: "Production server",
			},
		},
		Paths: make(map[string]interface{}),
		Components: &OpenAPIComponents{
			Schemas:         make(map[string]interface{}),
			SecuritySchemes: make(map[string]interface{}),
		},
	}
	
	// 检测是否需要认证
	needsAuth := false
	for _, endpoint := range endpoints {
		if endpoint.RequiresAuth {
			needsAuth = true
			break
		}
	}
	
	// 添加安全方案
	if needsAuth {
		spec.Components.SecuritySchemes["bearerAuth"] = map[string]interface{}{
			"type":   "http",
			"scheme": "bearer",
			"bearerFormat": "JWT",
		}
		spec.Security = []map[string][]string{
			{"bearerAuth": []string{}},
		}
	}
	
	// 转换每个端点
	for _, endpoint := range endpoints {
		pathItem := og.convertEndpointToPathItem(endpoint)
		
		// 提取路径（移除query参数）
		path := strings.Split(endpoint.URL, "?")[0]
		path = strings.TrimPrefix(path, fmt.Sprintf("https://%s", og.analyzer.targetDomain))
		path = strings.TrimPrefix(path, fmt.Sprintf("http://%s", og.analyzer.targetDomain))
		
		if path == "" {
			path = "/"
		}
		
		spec.Paths[path] = pathItem
		
		// 添加响应Schema到components
		if endpoint.ResponseSchema != nil {
			schemaName := og.generateSchemaName(path)
			spec.Components.Schemas[schemaName] = endpoint.ResponseSchema
		}
	}
	
	return spec, nil
}

// convertEndpointToPathItem 转换端点为PathItem
func (og *OpenAPIGenerator) convertEndpointToPathItem(endpoint *APIEndpoint) map[string]interface{} {
	pathItem := make(map[string]interface{})
	
	// 为每个方法生成操作
	for _, method := range endpoint.Methods {
		operation := og.createOperation(endpoint, method)
		pathItem[strings.ToLower(method)] = operation
	}
	
	return pathItem
}

// createOperation 创建操作
func (og *OpenAPIGenerator) createOperation(endpoint *APIEndpoint, method string) map[string]interface{} {
	operation := map[string]interface{}{
		"summary":     fmt.Sprintf("%s %s", method, endpoint.URL),
		"description": endpoint.Description,
		"operationId": og.generateOperationID(endpoint.URL, method),
		"parameters":  og.convertParameters(endpoint.Parameters),
		"responses":   og.createResponses(endpoint),
	}
	
	// 如果需要认证
	if endpoint.RequiresAuth {
		operation["security"] = []map[string][]string{
			{"bearerAuth": []string{}},
		}
	}
	
	// 如果有请求体（POST/PUT/PATCH）
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if endpoint.RequestBody != nil {
			operation["requestBody"] = map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": endpoint.RequestBody,
					},
				},
			}
		}
	}
	
	// 添加标签（从URL提取）
	tags := og.extractTags(endpoint.URL)
	if len(tags) > 0 {
		operation["tags"] = tags
	}
	
	return operation
}

// convertParameters 转换参数
func (og *OpenAPIGenerator) convertParameters(params []APIParameter) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	
	for _, param := range params {
		p := map[string]interface{}{
			"name":        param.Name,
			"in":          param.In,
			"required":    param.Required,
			"description": param.Description,
			"schema": map[string]interface{}{
				"type": param.Type,
			},
		}
		
		// 添加示例
		if param.Example != nil {
			p["example"] = param.Example
		}
		
		// 添加枚举
		if len(param.Enum) > 0 {
			schema := p["schema"].(map[string]interface{})
			schema["enum"] = param.Enum
		}
		
		// 添加默认值
		if param.Default != nil {
			schema := p["schema"].(map[string]interface{})
			schema["default"] = param.Default
		}
		
		// 添加格式
		if param.Format != "" {
			schema := p["schema"].(map[string]interface{})
			schema["format"] = param.Format
		}
		
		result = append(result, p)
	}
	
	return result
}

// createResponses 创建响应
func (og *OpenAPIGenerator) createResponses(endpoint *APIEndpoint) map[string]interface{} {
	responses := make(map[string]interface{})
	
	// 成功响应
	for _, statusCode := range endpoint.StatusCodes {
		if statusCode >= 200 && statusCode < 300 {
			response := map[string]interface{}{
				"description": fmt.Sprintf("Successful response (status %d)", statusCode),
			}
			
			// 如果有响应体
			if endpoint.ResponseBody != nil || endpoint.ResponseSchema != nil {
				content := map[string]interface{}{
					endpoint.ContentType: map[string]interface{}{},
				}
				
				if endpoint.ResponseSchema != nil {
					content[endpoint.ContentType] = map[string]interface{}{
						"schema": endpoint.ResponseSchema,
					}
				}
				
				// 添加示例
				if len(endpoint.Examples) > 0 {
					for _, example := range endpoint.Examples {
						if example.ResponseStatus == statusCode {
							jsonContent := content[endpoint.ContentType].(map[string]interface{})
							jsonContent["example"] = example.ResponseBody
							break
						}
					}
				}
				
				response["content"] = content
			}
			
			responses[fmt.Sprintf("%d", statusCode)] = response
		}
	}
	
	// 错误响应
	for _, errResp := range endpoint.ErrorResponses {
		responses[fmt.Sprintf("%d", errResp.StatusCode)] = map[string]interface{}{
			"description": errResp.Message,
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"example": errResp.Example,
				},
			},
		}
	}
	
	// 默认响应（如果没有任何响应）
	if len(responses) == 0 {
		responses["200"] = map[string]interface{}{
			"description": "Successful response",
		}
	}
	
	return responses
}

// generateOperationID 生成操作ID
func (og *OpenAPIGenerator) generateOperationID(url, method string) string {
	// 从URL提取路径部分
	parts := strings.Split(url, "/")
	pathParts := make([]string, 0)
	
	for _, part := range parts {
		if part != "" && !strings.HasPrefix(part, "http") {
			// 移除query参数
			part = strings.Split(part, "?")[0]
			// 转换为驼峰命名
			pathParts = append(pathParts, strings.Title(part))
		}
	}
	
	// 组合：方法 + 路径
	operationID := strings.ToLower(method) + strings.Join(pathParts, "")
	
	// 清理特殊字符
	operationID = strings.ReplaceAll(operationID, "-", "")
	operationID = strings.ReplaceAll(operationID, "_", "")
	operationID = strings.ReplaceAll(operationID, ".", "")
	
	return operationID
}

// extractTags 从URL提取标签
func (og *OpenAPIGenerator) extractTags(url string) []string {
	// 从URL路径提取第一级目录作为标签
	parts := strings.Split(url, "/")
	tags := make([]string, 0)
	
	for i, part := range parts {
		if part != "" && !strings.HasPrefix(part, "http") && i < 5 {
			// 移除query参数
			part = strings.Split(part, "?")[0]
			
			// 过滤掉数字ID和常见模式
			if !isNumericString(part) && part != "v1" && part != "v2" && part != "api" {
				tags = append(tags, strings.Title(part))
			}
		}
	}
	
	if len(tags) == 0 {
		tags = append(tags, "Default")
	}
	
	return tags
}

// generateSchemaName 生成Schema名称
func (og *OpenAPIGenerator) generateSchemaName(path string) string {
	// 移除前导斜杠和尾随斜杠
	path = strings.Trim(path, "/")
	
	// 分割路径
	parts := strings.Split(path, "/")
	
	// 过滤并首字母大写
	nameParts := make([]string, 0)
	for _, part := range parts {
		if part != "" && !isNumericString(part) {
			nameParts = append(nameParts, strings.Title(part))
		}
	}
	
	// 组合
	if len(nameParts) == 0 {
		return "Response"
	}
	
	return strings.Join(nameParts, "") + "Response"
}

// ExportToFile 导出到文件
func (og *OpenAPIGenerator) ExportToFile(filename string) error {
	spec, err := og.Generate()
	if err != nil {
		return err
	}
	
	// 转换为JSON
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	
	// 写入文件
	return ioutil.WriteFile(filename, data, 0644)
}

// ExportToYAML 导出为YAML格式
func (og *OpenAPIGenerator) ExportToYAML(filename string) error {
	// TODO: 实现YAML导出
	// 需要引入yaml库: gopkg.in/yaml.v3
	return fmt.Errorf("YAML导出尚未实现，请使用JSON格式")
}

// GenerateMarkdownDoc 生成Markdown文档
func (og *OpenAPIGenerator) GenerateMarkdownDoc() (string, error) {
	endpoints := og.analyzer.GetAllEndpoints()
	if len(endpoints) == 0 {
		return "", fmt.Errorf("没有可用的API端点")
	}
	
	var sb strings.Builder
	
	// 标题
	sb.WriteString(fmt.Sprintf("# %s API Documentation\n\n", og.analyzer.targetDomain))
	sb.WriteString(fmt.Sprintf("**自动生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**总端点数**: %d\n\n", len(endpoints)))
	
	// 统计信息
	stats := og.analyzer.GetStatistics()
	sb.WriteString("## 统计信息\n\n")
	sb.WriteString(fmt.Sprintf("- 总请求数: %v\n", stats["total_requests"]))
	sb.WriteString(fmt.Sprintf("- 成功率: %.1f%%\n\n", stats["success_rate"]))
	
	// 目录
	sb.WriteString("## 目录\n\n")
	for i, endpoint := range endpoints {
		sb.WriteString(fmt.Sprintf("%d. [%s](#endpoint-%d)\n", i+1, endpoint.URL, i+1))
	}
	sb.WriteString("\n")
	
	// 详细文档
	sb.WriteString("## API端点详情\n\n")
	
	for i, endpoint := range endpoints {
		sb.WriteString(fmt.Sprintf("### <a name=\"endpoint-%d\"></a>%d. %s\n\n", i+1, i+1, endpoint.URL))
		
		// 基本信息
		sb.WriteString("**基本信息**\n\n")
		sb.WriteString(fmt.Sprintf("- **类型**: %s\n", endpoint.APIType))
		sb.WriteString(fmt.Sprintf("- **方法**: %s\n", strings.Join(endpoint.Methods, ", ")))
		sb.WriteString(fmt.Sprintf("- **Content-Type**: %s\n", endpoint.ContentType))
		
		if endpoint.Version != "" {
			sb.WriteString(fmt.Sprintf("- **版本**: %s\n", endpoint.Version))
		}
		
		if endpoint.RequiresAuth {
			sb.WriteString("- **认证**: 🔐 需要\n")
		}
		
		if endpoint.Deprecated {
			sb.WriteString("- **状态**: ⚠️ 已废弃\n")
		}
		
		sb.WriteString("\n")
		
		// 参数
		if len(endpoint.Parameters) > 0 {
			sb.WriteString("**参数**\n\n")
			sb.WriteString("| 名称 | 位置 | 类型 | 必需 | 说明 |\n")
			sb.WriteString("|------|------|------|------|------|\n")
			
			for _, param := range endpoint.Parameters {
				required := "否"
				if param.Required {
					required = "是"
				}
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
					param.Name, param.In, param.Type, required, param.Description))
			}
			sb.WriteString("\n")
		}
		
		// 响应状态码
		if len(endpoint.StatusCodes) > 0 {
			sb.WriteString("**响应状态码**\n\n")
			for _, code := range endpoint.StatusCodes {
				sb.WriteString(fmt.Sprintf("- `%d`\n", code))
			}
			sb.WriteString("\n")
		}
		
		// 示例
		if len(endpoint.Examples) > 0 {
			sb.WriteString("**请求示例**\n\n")
			example := endpoint.Examples[0]
			
			sb.WriteString("```bash\n")
			sb.WriteString(fmt.Sprintf("curl -X %s \\\n", example.Method))
			sb.WriteString(fmt.Sprintf("  '%s' \\\n", example.URL))
			
			for key, value := range example.RequestHeaders {
				sb.WriteString(fmt.Sprintf("  -H '%s: %s' \\\n", key, value))
			}
			
			if example.RequestBody != "" {
				sb.WriteString(fmt.Sprintf("  -d '%s'\n", example.RequestBody))
			}
			
			sb.WriteString("```\n\n")
			
			sb.WriteString("**响应示例**\n\n")
			sb.WriteString("```json\n")
			sb.WriteString(example.ResponseBody)
			sb.WriteString("\n```\n\n")
		}
		
		// 速率限制
		if endpoint.RateLimit != nil {
			sb.WriteString("**速率限制**\n\n")
			sb.WriteString(fmt.Sprintf("- 限制: %d 请求/%s\n", endpoint.RateLimit.Limit, endpoint.RateLimit.Window))
			sb.WriteString(fmt.Sprintf("- 剩余: %d\n\n", endpoint.RateLimit.Remaining))
		}
		
		sb.WriteString("---\n\n")
	}
	
	return sb.String(), nil
}

