package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// GraphQLAnalyzer GraphQL API分析器
type GraphQLAnalyzer struct {
	endpoint   string
	client     *http.Client
	authHeader string
	schema     *GraphQLSchema
}

// GraphQLSchema GraphQL Schema
type GraphQLSchema struct {
	Types       []GraphQLType      `json:"types"`
	Queries     []GraphQLField     `json:"queries"`
	Mutations   []GraphQLField     `json:"mutations"`
	Directives  []GraphQLDirective `json:"directives"`
}

// GraphQLType GraphQL类型
type GraphQLType struct {
	Kind        string         `json:"kind"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Fields      []GraphQLField `json:"fields,omitempty"`
	InputFields []GraphQLField `json:"inputFields,omitempty"`
	EnumValues  []string       `json:"enumValues,omitempty"`
}

// GraphQLField GraphQL字段
type GraphQLField struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Type        string              `json:"type"`
	Args        []GraphQLArgument   `json:"args,omitempty"`
}

// GraphQLArgument GraphQL参数
type GraphQLArgument struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultValue,omitempty"`
}

// GraphQLDirective GraphQL指令
type GraphQLDirective struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Locations   []string `json:"locations"`
}

// NewGraphQLAnalyzer 创建GraphQL分析器
func NewGraphQLAnalyzer(endpoint string) *GraphQLAnalyzer {
	return &GraphQLAnalyzer{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// SetAuthentication 设置认证
func (ga *GraphQLAnalyzer) SetAuthentication(authHeader string) {
	ga.authHeader = authHeader
}

// Analyze 分析GraphQL端点
func (ga *GraphQLAnalyzer) Analyze() (*GraphQLSchema, error) {
	fmt.Printf("[GraphQL分析] 开始分析: %s\n", ga.endpoint)
	
	// 1. 通过Introspection查询获取Schema
	schema, err := ga.introspect()
	if err != nil {
		return nil, fmt.Errorf("introspection查询失败: %v", err)
	}
	
	ga.schema = schema
	
	fmt.Printf("  [Schema] 类型:%d, 查询:%d, 变更:%d\n",
		len(schema.Types), len(schema.Queries), len(schema.Mutations))
	
	return schema, nil
}

// introspect 执行Introspection查询
func (ga *GraphQLAnalyzer) introspect() (*GraphQLSchema, error) {
	// GraphQL Introspection查询
	query := `
	{
		__schema {
			types {
				kind
				name
				description
				fields {
					name
					description
					type {
						name
						kind
						ofType {
							name
							kind
						}
					}
					args {
						name
						description
						type {
							name
							kind
						}
						defaultValue
					}
				}
				inputFields {
					name
					description
					type {
						name
						kind
					}
					defaultValue
				}
				enumValues {
					name
					description
				}
			}
			queryType {
				name
				fields {
					name
					description
					type {
						name
						kind
					}
					args {
						name
						description
						type {
							name
							kind
						}
					}
				}
			}
			mutationType {
				name
				fields {
					name
					description
					type {
						name
						kind
					}
					args {
						name
						description
						type {
							name
							kind
						}
					}
				}
			}
			directives {
				name
				description
				locations
			}
		}
	}
	`
	
	// 发送查询
	body := map[string]string{
		"query": query,
	}
	
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequest("POST", ga.endpoint, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	if ga.authHeader != "" {
		req.Header.Set("Authorization", ga.authHeader)
	}
	
	resp, err := ga.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	
	// 检查错误
	if errors, ok := result["errors"]; ok {
		return nil, fmt.Errorf("GraphQL错误: %v", errors)
	}
	
	// 提取schema
	schema := ga.parseIntrospectionResult(result)
	
	return schema, nil
}

// parseIntrospectionResult 解析Introspection结果
func (ga *GraphQLAnalyzer) parseIntrospectionResult(result map[string]interface{}) *GraphQLSchema {
	schema := &GraphQLSchema{
		Types:      make([]GraphQLType, 0),
		Queries:    make([]GraphQLField, 0),
		Mutations:  make([]GraphQLField, 0),
		Directives: make([]GraphQLDirective, 0),
	}
	
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return schema
	}
	
	schemaData, ok := data["__schema"].(map[string]interface{})
	if !ok {
		return schema
	}
	
	// 解析类型
	if types, ok := schemaData["types"].([]interface{}); ok {
		for _, t := range types {
			if typeMap, ok := t.(map[string]interface{}); ok {
				gqlType := ga.parseType(typeMap)
				
				// 跳过内置类型
				if !strings.HasPrefix(gqlType.Name, "__") {
					schema.Types = append(schema.Types, gqlType)
				}
			}
		}
	}
	
	// 解析查询
	if queryType, ok := schemaData["queryType"].(map[string]interface{}); ok {
		if fields, ok := queryType["fields"].([]interface{}); ok {
			for _, f := range fields {
				if fieldMap, ok := f.(map[string]interface{}); ok {
					schema.Queries = append(schema.Queries, ga.parseField(fieldMap))
				}
			}
		}
	}
	
	// 解析变更
	if mutationType, ok := schemaData["mutationType"].(map[string]interface{}); ok {
		if fields, ok := mutationType["fields"].([]interface{}); ok {
			for _, f := range fields {
				if fieldMap, ok := f.(map[string]interface{}); ok {
					schema.Mutations = append(schema.Mutations, ga.parseField(fieldMap))
				}
			}
		}
	}
	
	// 解析指令
	if directives, ok := schemaData["directives"].([]interface{}); ok {
		for _, d := range directives {
			if dirMap, ok := d.(map[string]interface{}); ok {
				directive := GraphQLDirective{
					Name:        getStringValue(dirMap, "name"),
					Description: getStringValue(dirMap, "description"),
					Locations:   make([]string, 0),
				}
				
				if locations, ok := dirMap["locations"].([]interface{}); ok {
					for _, loc := range locations {
						if locStr, ok := loc.(string); ok {
							directive.Locations = append(directive.Locations, locStr)
						}
					}
				}
				
				schema.Directives = append(schema.Directives, directive)
			}
		}
	}
	
	return schema
}

// parseType 解析类型
func (ga *GraphQLAnalyzer) parseType(typeMap map[string]interface{}) GraphQLType {
	gqlType := GraphQLType{
		Kind:        getStringValue(typeMap, "kind"),
		Name:        getStringValue(typeMap, "name"),
		Description: getStringValue(typeMap, "description"),
		Fields:      make([]GraphQLField, 0),
		InputFields: make([]GraphQLField, 0),
		EnumValues:  make([]string, 0),
	}
	
	// 解析字段
	if fields, ok := typeMap["fields"].([]interface{}); ok {
		for _, f := range fields {
			if fieldMap, ok := f.(map[string]interface{}); ok {
				gqlType.Fields = append(gqlType.Fields, ga.parseField(fieldMap))
			}
		}
	}
	
	// 解析输入字段
	if inputFields, ok := typeMap["inputFields"].([]interface{}); ok {
		for _, f := range inputFields {
			if fieldMap, ok := f.(map[string]interface{}); ok {
				gqlType.InputFields = append(gqlType.InputFields, ga.parseField(fieldMap))
			}
		}
	}
	
	// 解析枚举值
	if enumValues, ok := typeMap["enumValues"].([]interface{}); ok {
		for _, ev := range enumValues {
			if evMap, ok := ev.(map[string]interface{}); ok {
				gqlType.EnumValues = append(gqlType.EnumValues, getStringValue(evMap, "name"))
			}
		}
	}
	
	return gqlType
}

// parseField 解析字段
func (ga *GraphQLAnalyzer) parseField(fieldMap map[string]interface{}) GraphQLField {
	field := GraphQLField{
		Name:        getStringValue(fieldMap, "name"),
		Description: getStringValue(fieldMap, "description"),
		Type:        ga.parseTypeRef(fieldMap["type"]),
		Args:        make([]GraphQLArgument, 0),
	}
	
	// 解析参数
	if args, ok := fieldMap["args"].([]interface{}); ok {
		for _, a := range args {
			if argMap, ok := a.(map[string]interface{}); ok {
				arg := GraphQLArgument{
					Name:         getStringValue(argMap, "name"),
					Description:  getStringValue(argMap, "description"),
					Type:         ga.parseTypeRef(argMap["type"]),
					DefaultValue: getStringValue(argMap, "defaultValue"),
				}
				field.Args = append(field.Args, arg)
			}
		}
	}
	
	return field
}

// parseTypeRef 解析类型引用
func (ga *GraphQLAnalyzer) parseTypeRef(typeRef interface{}) string {
	if typeRef == nil {
		return ""
	}
	
	typeMap, ok := typeRef.(map[string]interface{})
	if !ok {
		return ""
	}
	
	kind := getStringValue(typeMap, "kind")
	name := getStringValue(typeMap, "name")
	
	switch kind {
	case "NON_NULL":
		// 非空类型
		if ofType := typeMap["ofType"]; ofType != nil {
			return ga.parseTypeRef(ofType) + "!"
		}
	case "LIST":
		// 列表类型
		if ofType := typeMap["ofType"]; ofType != nil {
			return "[" + ga.parseTypeRef(ofType) + "]"
		}
	default:
		return name
	}
	
	return ""
}

// GenerateSDL 生成SDL (Schema Definition Language)
func (ga *GraphQLAnalyzer) GenerateSDL() (string, error) {
	if ga.schema == nil {
		return "", fmt.Errorf("schema未初始化，请先调用Analyze()")
	}
	
	var sb strings.Builder
	
	// 生成类型定义
	sb.WriteString("# Types\n\n")
	for _, t := range ga.schema.Types {
		// 跳过标量类型
		if t.Kind == "SCALAR" {
			continue
		}
		
		sb.WriteString(fmt.Sprintf("type %s {\n", t.Name))
		if t.Description != "" {
			sb.WriteString(fmt.Sprintf("  \"\"\"%s\"\"\"\n", t.Description))
		}
		
		for _, field := range t.Fields {
			if field.Description != "" {
				sb.WriteString(fmt.Sprintf("  \"\"\"%s\"\"\"\n", field.Description))
			}
			
			// 字段定义
			if len(field.Args) > 0 {
				args := make([]string, 0)
				for _, arg := range field.Args {
					argDef := fmt.Sprintf("%s: %s", arg.Name, arg.Type)
					if arg.DefaultValue != "" {
						argDef += " = " + arg.DefaultValue
					}
					args = append(args, argDef)
				}
				sb.WriteString(fmt.Sprintf("  %s(%s): %s\n", field.Name, strings.Join(args, ", "), field.Type))
			} else {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", field.Name, field.Type))
			}
		}
		
		sb.WriteString("}\n\n")
	}
	
	// 生成Query
	if len(ga.schema.Queries) > 0 {
		sb.WriteString("type Query {\n")
		for _, query := range ga.schema.Queries {
			if query.Description != "" {
				sb.WriteString(fmt.Sprintf("  \"\"\"%s\"\"\"\n", query.Description))
			}
			
			if len(query.Args) > 0 {
				args := make([]string, 0)
				for _, arg := range query.Args {
					args = append(args, fmt.Sprintf("%s: %s", arg.Name, arg.Type))
				}
				sb.WriteString(fmt.Sprintf("  %s(%s): %s\n", query.Name, strings.Join(args, ", "), query.Type))
			} else {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", query.Name, query.Type))
			}
		}
		sb.WriteString("}\n\n")
	}
	
	// 生成Mutation
	if len(ga.schema.Mutations) > 0 {
		sb.WriteString("type Mutation {\n")
		for _, mutation := range ga.schema.Mutations {
			if mutation.Description != "" {
				sb.WriteString(fmt.Sprintf("  \"\"\"%s\"\"\"\n", mutation.Description))
			}
			
			if len(mutation.Args) > 0 {
				args := make([]string, 0)
				for _, arg := range mutation.Args {
					args = append(args, fmt.Sprintf("%s: %s", arg.Name, arg.Type))
				}
				sb.WriteString(fmt.Sprintf("  %s(%s): %s\n", mutation.Name, strings.Join(args, ", "), mutation.Type))
			} else {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", mutation.Name, mutation.Type))
			}
		}
		sb.WriteString("}\n\n")
	}
	
	return sb.String(), nil
}

// ExportToFile 导出到文件
func (ga *GraphQLAnalyzer) ExportToFile(filename string) error {
	sdl, err := ga.GenerateSDL()
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(filename, []byte(sdl), 0644)
}

// getStringValue 安全获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

