# API智能分析 - 使用指南

> **Spider Ultimate v2.5 新特性**  
> 自动分析REST/GraphQL/gRPC API，生成OpenAPI/Swagger文档

---

## 📋 目录

1. [功能概述](#功能概述)
2. [快速开始](#快速开始)
3. [REST API分析](#rest-api分析)
4. [GraphQL分析](#graphql分析)
5. [OpenAPI文档生成](#openapi文档生成)
6. [实战案例](#实战案例)
7. [集成到爬虫](#集成到爬虫)

---

## 功能概述

### 核心能力

| 功能 | 说明 | 支持度 |
|------|------|--------|
| **REST API分析** | 自动探测方法、参数、响应 | ✅ 完整 |
| **GraphQL分析** | Introspection查询、Schema提取 | ✅ 完整 |
| **gRPC支持** | Proto文件推断 | 🔄 计划中 |
| **OpenAPI生成** | 生成OpenAPI 3.0文档 | ✅ 完整 |
| **Swagger集成** | 可导入Swagger Editor | ✅ 完整 |
| **Markdown文档** | 生成可读文档 | ✅ 完整 |
| **Postman Collection** | 生成Postman导入文件 | 🔄 计划中 |

### 分析维度

**REST API**:
- ✅ 支持的HTTP方法 (GET/POST/PUT/DELETE/PATCH)
- ✅ URL参数（query/path/header/body）
- ✅ 请求/响应格式
- ✅ 状态码
- ✅ 认证要求
- ✅ 速率限制
- ✅ 响应Schema
- ✅ API版本
- ✅ 是否废弃

**GraphQL**:
- ✅ Schema结构
- ✅ Types/Queries/Mutations
- ✅ 字段和参数
- ✅ 指令（Directives）
- ✅ SDL生成

---

## 快速开始

### 安装依赖

确保你的项目中已经包含了API分析模块：

```bash
# 项目结构
spider-golang/
├── core/
│   ├── api_analyzer.go          # REST API分析器
│   ├── openapi_generator.go     # OpenAPI生成器
│   ├── graphql_analyzer.go      # GraphQL分析器
│   └── ...
└── examples/
    └── api_analysis_demo.go     # 演示程序
```

### 最简示例

```go
package main

import (
    "fmt"
    "spider-golang/core"
)

func main() {
    // 1. 创建分析器
    analyzer := core.NewAPIAnalyzer("api.example.com")
    
    // 2. 分析API端点
    endpoint, err := analyzer.AnalyzeEndpoint("https://api.example.com/users")
    if err != nil {
        panic(err)
    }
    
    // 3. 打印结果
    fmt.Printf("方法: %v\n", endpoint.Methods)
    fmt.Printf("参数: %d 个\n", len(endpoint.Parameters))
}
```

运行演示：

```bash
go run examples/api_analysis_demo.go
```

---

## REST API分析

### 基础用法

```go
// 创建分析器
analyzer := core.NewAPIAnalyzer("api.github.com")

// 可选：设置认证
analyzer.SetAuthentication("Bearer ghp_xxxxxxxxxxxxx")

// 分析单个端点
endpoint, err := analyzer.AnalyzeEndpoint("https://api.github.com/users/octocat")
if err != nil {
    log.Fatal(err)
}

// 查看结果
fmt.Printf("API类型: %s\n", endpoint.APIType)
fmt.Printf("支持方法: %v\n", endpoint.Methods)
fmt.Printf("需要认证: %v\n", endpoint.RequiresAuth)

// 查看参数
for _, param := range endpoint.Parameters {
    fmt.Printf("参数: %s (%s) - %s\n", 
        param.Name, param.Type, param.In)
}

// 查看响应Schema
if endpoint.ResponseSchema != nil {
    schema, _ := json.MarshalIndent(endpoint.ResponseSchema, "", "  ")
    fmt.Printf("响应Schema:\n%s\n", schema)
}

// 查看示例
for _, example := range endpoint.Examples {
    fmt.Printf("请求示例: %s %s\n", example.Method, example.URL)
    fmt.Printf("响应状态: %d\n", example.ResponseStatus)
    fmt.Printf("响应体: %s\n", example.ResponseBody)
}
```

### 批量分析

```go
analyzer := core.NewAPIAnalyzer("api.example.com")

// API端点列表
endpoints := []string{
    "https://api.example.com/users",
    "https://api.example.com/users/1",
    "https://api.example.com/posts",
    "https://api.example.com/posts/1/comments",
    "https://api.example.com/categories",
}

// 批量分析
for i, url := range endpoints {
    fmt.Printf("[%d/%d] 分析: %s\n", i+1, len(endpoints), url)
    
    endpoint, err := analyzer.AnalyzeEndpoint(url)
    if err != nil {
        log.Printf("分析失败: %v", err)
        continue
    }
    
    fmt.Printf("  ✓ 方法: %v\n", endpoint.Methods)
    fmt.Printf("  ✓ 参数: %d 个\n", len(endpoint.Parameters))
}

// 获取所有端点
allEndpoints := analyzer.GetAllEndpoints()
fmt.Printf("\n总共分析了 %d 个端点\n", len(allEndpoints))

// 统计信息
stats := analyzer.GetStatistics()
fmt.Printf("成功率: %.1f%%\n", stats["success_rate"])
```

### 高级功能

#### 1. 参数类型推断

分析器会自动推断参数类型：

```go
// 从响应JSON推断参数
{
    "id": 123,              // → 推断为 integer
    "name": "John",         // → 推断为 string
    "email": "j@a.com",     // → 推断为 string (format: email)
    "active": true,         // → 推断为 boolean
    "tags": ["a", "b"],     // → 推断为 array
    "profile": {...}        // → 推断为 object
}
```

#### 2. 认证检测

自动检测认证要求：

```go
endpoint, _ := analyzer.AnalyzeEndpoint("https://api.example.com/admin/users")

if endpoint.RequiresAuth {
    fmt.Println("此端点需要认证")
    
    // 查看认证错误响应
    for _, errResp := range endpoint.ErrorResponses {
        if errResp.StatusCode == 401 {
            fmt.Printf("401错误: %s\n", errResp.Message)
        }
    }
}
```

#### 3. 速率限制检测

自动提取速率限制信息：

```go
if endpoint.RateLimit != nil {
    fmt.Printf("速率限制: %d 请求/%s\n", 
        endpoint.RateLimit.Limit, 
        endpoint.RateLimit.Window)
    fmt.Printf("剩余: %d\n", endpoint.RateLimit.Remaining)
    fmt.Printf("重置时间: %d\n", endpoint.RateLimit.Reset)
}
```

---

## GraphQL分析

### 基础用法

```go
// 创建GraphQL分析器
analyzer := core.NewGraphQLAnalyzer("https://api.example.com/graphql")

// 可选：设置认证
analyzer.SetAuthentication("Bearer your-token")

// 执行Introspection查询
schema, err := analyzer.Analyze()
if err != nil {
    log.Fatal(err)
}

// 查看Schema
fmt.Printf("类型: %d\n", len(schema.Types))
fmt.Printf("查询: %d\n", len(schema.Queries))
fmt.Printf("变更: %d\n", len(schema.Mutations))

// 查看可用查询
for _, query := range schema.Queries {
    fmt.Printf("查询: %s → %s\n", query.Name, query.Type)
    
    // 查看参数
    if len(query.Args) > 0 {
        fmt.Println("  参数:")
        for _, arg := range query.Args {
            required := ""
            if strings.HasSuffix(arg.Type, "!") {
                required = " (必需)"
            }
            fmt.Printf("    - %s: %s%s\n", arg.Name, arg.Type, required)
        }
    }
}
```

### 生成SDL

```go
// 生成Schema Definition Language
sdl, err := analyzer.GenerateSDL()
if err != nil {
    log.Fatal(err)
}

fmt.Println(sdl)

// 输出示例:
/*
type User {
  """User ID"""
  id: ID!
  
  """User name"""
  name: String!
  
  """User email"""
  email: String
  
  """User posts"""
  posts(limit: Int): [Post]
}

type Query {
  """Get user by ID"""
  user(id: ID!): User
  
  """Get all users"""
  users(limit: Int, offset: Int): [User]
}

type Mutation {
  """Create a new user"""
  createUser(name: String!, email: String!): User
}
*/
```

### 导出Schema

```go
// 导出为.graphql文件
err := analyzer.ExportToFile("schema.graphql")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Schema已导出到: schema.graphql")
```

### 实际案例：GitHub GraphQL API

```go
// 分析GitHub GraphQL API
analyzer := core.NewGraphQLAnalyzer("https://api.github.com/graphql")
analyzer.SetAuthentication("Bearer ghp_xxxxxxxxxxxxx")

schema, err := analyzer.Analyze()
if err != nil {
    log.Fatal(err)
}

// 查看仓库相关查询
for _, query := range schema.Queries {
    if strings.Contains(strings.ToLower(query.Name), "repository") {
        fmt.Printf("查询: %s\n", query.Name)
        fmt.Printf("  返回类型: %s\n", query.Type)
        fmt.Printf("  参数:\n")
        for _, arg := range query.Args {
            fmt.Printf("    - %s: %s\n", arg.Name, arg.Type)
        }
    }
}

// 导出完整Schema
analyzer.ExportToFile("github_schema.graphql")
```

---

## OpenAPI文档生成

### 生成OpenAPI 3.0文档

```go
// 1. 先分析API
analyzer := core.NewAPIAnalyzer("api.example.com")

endpoints := []string{
    "https://api.example.com/users",
    "https://api.example.com/users/{id}",
    "https://api.example.com/posts",
}

for _, url := range endpoints {
    analyzer.AnalyzeEndpoint(url)
}

// 2. 创建OpenAPI生成器
generator := core.NewOpenAPIGenerator(analyzer)

// 3. 生成OpenAPI文档
spec, err := generator.Generate()
if err != nil {
    log.Fatal(err)
}

// 4. 导出为JSON
err = generator.ExportToFile("openapi.json")
if err != nil {
    log.Fatal(err)
}

fmt.Println("✅ OpenAPI文档已生成: openapi.json")
```

### 查看OpenAPI文档

**方法1: Swagger Editor (在线)**

1. 打开 https://editor.swagger.io/
2. File → Import File
3. 选择 `openapi.json`
4. 即可看到可视化的API文档

**方法2: Swagger UI (本地)**

```bash
# 使用Docker运行Swagger UI
docker run -p 8080:8080 \
  -e SWAGGER_JSON=/openapi.json \
  -v $(pwd)/openapi.json:/openapi.json \
  swaggerapi/swagger-ui

# 访问 http://localhost:8080
```

**方法3: VS Code插件**

1. 安装插件: Swagger Viewer
2. 右键 `openapi.json` → Preview Swagger

### OpenAPI文档结构

生成的文档包含：

```json
{
  "openapi": "3.0.0",
  "info": {
    "title": "api.example.com API",
    "description": "由Spider Ultimate自动生成",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "https://api.example.com",
      "description": "Production server"
    }
  ],
  "paths": {
    "/users": {
      "get": {
        "summary": "GET /users",
        "operationId": "getUsers",
        "parameters": [...],
        "responses": {
          "200": {
            "description": "Successful response",
            "content": {
              "application/json": {
                "schema": {...}
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {...},
    "securitySchemes": {...}
  }
}
```

### 生成Markdown文档

```go
generator := core.NewOpenAPIGenerator(analyzer)

// 生成Markdown
markdown, err := generator.GenerateMarkdownDoc()
if err != nil {
    log.Fatal(err)
}

// 保存到文件
ioutil.WriteFile("API_Documentation.md", []byte(markdown), 0644)

fmt.Println("✅ Markdown文档已生成: API_Documentation.md")
```

Markdown文档包含：

- 📊 统计总览
- 📑 目录
- 📝 每个端点的详细说明
- 💻 cURL示例
- 📄 响应示例
- ⚡ 速率限制信息

---

## 实战案例

### 案例1: 分析未知API

**场景**: 拿到一个API地址，完全不知道它的功能

```go
package main

import (
    "fmt"
    "log"
    "spider-golang/core"
)

func main() {
    targetAPI := "https://api.unknown-service.com"
    
    // 1. 创建分析器
    analyzer := core.NewAPIAnalyzer("api.unknown-service.com")
    
    // 2. 先尝试常见端点
    commonEndpoints := []string{
        targetAPI + "/",
        targetAPI + "/api",
        targetAPI + "/v1",
        targetAPI + "/users",
        targetAPI + "/docs",
        targetAPI + "/swagger.json",
        targetAPI + "/openapi.json",
    }
    
    discoveredEndpoints := make([]string, 0)
    
    fmt.Println("探测常见端点...")
    for _, url := range commonEndpoints {
        endpoint, err := analyzer.AnalyzeEndpoint(url)
        if err == nil && len(endpoint.Methods) > 0 {
            discoveredEndpoints = append(discoveredEndpoints, url)
            fmt.Printf("✓ 发现: %s (%v)\n", url, endpoint.Methods)
        }
    }
    
    // 3. 生成文档
    if len(discoveredEndpoints) > 0 {
        generator := core.NewOpenAPIGenerator(analyzer)
        generator.ExportToFile("discovered_api.json")
        
        markdown, _ := generator.GenerateMarkdownDoc()
        ioutil.WriteFile("discovered_api.md", []byte(markdown), 0644)
        
        fmt.Printf("\n✅ 发现 %d 个端点\n", len(discoveredEndpoints))
        fmt.Println("✅ 文档已生成:")
        fmt.Println("   - discovered_api.json (OpenAPI)")
        fmt.Println("   - discovered_api.md (Markdown)")
    } else {
        fmt.Println("❌ 未发现可用端点")
    }
}
```

### 案例2: API版本对比

**场景**: 对比新旧API版本的差异

```go
func compareAPIVersions(v1URL, v2URL string) {
    fmt.Println("对比API版本...")
    
    // 分析v1
    analyzerV1 := core.NewAPIAnalyzer("api.example.com")
    analyzerV1.AnalyzeEndpoint(v1URL)
    endpointsV1 := analyzerV1.GetAllEndpoints()
    
    // 分析v2
    analyzerV2 := core.NewAPIAnalyzer("api.example.com")
    analyzerV2.AnalyzeEndpoint(v2URL)
    endpointsV2 := analyzerV2.GetAllEndpoints()
    
    // 对比
    fmt.Printf("\nv1: %d 个端点\n", len(endpointsV1))
    fmt.Printf("v2: %d 个端点\n", len(endpointsV2))
    
    // 查找新增端点
    v1Map := make(map[string]bool)
    for _, ep := range endpointsV1 {
        v1Map[ep.URL] = true
    }
    
    newEndpoints := make([]string, 0)
    for _, ep := range endpointsV2 {
        if !v1Map[ep.URL] {
            newEndpoints = append(newEndpoints, ep.URL)
        }
    }
    
    if len(newEndpoints) > 0 {
        fmt.Println("\n🆕 新增端点:")
        for _, url := range newEndpoints {
            fmt.Printf("  + %s\n", url)
        }
    }
}
```

### 案例3: 渗透测试准备

**场景**: 为渗透测试准备API信息

```go
func prepareForPentest(targetAPI string) {
    analyzer := core.NewAPIAnalyzer(strings.TrimPrefix(targetAPI, "https://"))
    
    // 1. 分析所有可能的端点
    endpoints := discoverEndpoints(targetAPI)
    
    // 2. 记录关键信息
    report := make(map[string]interface{})
    report["target"] = targetAPI
    report["scan_time"] = time.Now().Format("2006-01-02 15:04:05")
    report["endpoints"] = make([]map[string]interface{}, 0)
    
    for _, url := range endpoints {
        endpoint, err := analyzer.AnalyzeEndpoint(url)
        if err != nil {
            continue
        }
        
        epInfo := map[string]interface{}{
            "url":          endpoint.URL,
            "methods":      endpoint.Methods,
            "requires_auth": endpoint.RequiresAuth,
            "parameters":   len(endpoint.Parameters),
        }
        
        // 标记高风险端点
        if isHighRisk(endpoint) {
            epInfo["risk"] = "HIGH"
            epInfo["reason"] = getRiskReason(endpoint)
        }
        
        report["endpoints"] = append(report["endpoints"].([]map[string]interface{}), epInfo)
    }
    
    // 3. 导出报告
    reportJSON, _ := json.MarshalIndent(report, "", "  ")
    ioutil.WriteFile("pentest_prep.json", reportJSON, 0644)
    
    fmt.Println("✅ 渗透测试准备报告已生成: pentest_prep.json")
}

func isHighRisk(endpoint *core.APIEndpoint) bool {
    url := strings.ToLower(endpoint.URL)
    
    // 高风险关键词
    keywords := []string{"admin", "delete", "exec", "upload", "password", "auth", "token"}
    for _, kw := range keywords {
        if strings.Contains(url, kw) {
            return true
        }
    }
    
    // DELETE方法
    for _, method := range endpoint.Methods {
        if method == "DELETE" {
            return true
        }
    }
    
    return false
}
```

---

## 集成到爬虫

### 在Spider中使用API分析

修改 `core/spider.go`，添加API分析功能：

```go
// Spider结构体中添加
type Spider struct {
    // ... 现有字段
    apiAnalyzer    *APIAnalyzer        // API分析器
    graphqlAnalyzer *GraphQLAnalyzer   // GraphQL分析器
}

// 在NewSpider中初始化
func NewSpider(cfg *config.Config) *Spider {
    spider := &Spider{
        // ... 现有初始化
        apiAnalyzer:    NewAPIAnalyzer(cfg.TargetURL),
        graphqlAnalyzer: nil, // 按需创建
    }
    return spider
}

// 添加方法：分析发现的API
func (s *Spider) analyzeAPIs() {
    fmt.Println("\n开始分析发现的API端点...")
    
    // 收集所有API端点
    apiEndpoints := make([]string, 0)
    
    s.mutex.Lock()
    for _, result := range s.results {
        for _, api := range result.APIs {
            apiEndpoints = append(apiEndpoints, api)
        }
    }
    s.mutex.Unlock()
    
    if len(apiEndpoints) == 0 {
        fmt.Println("未发现API端点")
        return
    }
    
    fmt.Printf("发现 %d 个API端点，开始分析...\n", len(apiEndpoints))
    
    // 分析每个端点
    for i, url := range apiEndpoints {
        fmt.Printf("[%d/%d] 分析: %s\n", i+1, len(apiEndpoints), url)
        
        // 检测API类型
        if strings.Contains(strings.ToLower(url), "graphql") {
            // GraphQL API
            if s.graphqlAnalyzer == nil {
                s.graphqlAnalyzer = NewGraphQLAnalyzer(url)
            }
            s.graphqlAnalyzer.Analyze()
        } else {
            // REST API
            s.apiAnalyzer.AnalyzeEndpoint(url)
        }
    }
    
    // 生成文档
    s.generateAPIDocumentation()
}

// 生成API文档
func (s *Spider) generateAPIDocumentation() {
    // REST API文档
    if len(s.apiAnalyzer.GetAllEndpoints()) > 0 {
        generator := NewOpenAPIGenerator(s.apiAnalyzer)
        
        // OpenAPI JSON
        generator.ExportToFile("api_openapi.json")
        fmt.Println("✅ OpenAPI文档: api_openapi.json")
        
        // Markdown
        markdown, _ := generator.GenerateMarkdownDoc()
        ioutil.WriteFile("api_documentation.md", []byte(markdown), 0644)
        fmt.Println("✅ Markdown文档: api_documentation.md")
    }
    
    // GraphQL Schema
    if s.graphqlAnalyzer != nil && s.graphqlAnalyzer.schema != nil {
        s.graphqlAnalyzer.ExportToFile("graphql_schema.graphql")
        fmt.Println("✅ GraphQL Schema: graphql_schema.graphql")
    }
}
```

### 在爬取完成后调用

```go
// 在Spider.Start()方法末尾添加
func (s *Spider) Start(targetURL string) error {
    // ... 现有爬取逻辑
    
    // 新增：分析API
    s.analyzeAPIs()
    
    return nil
}
```

### 命令行参数

添加API分析开关：

```go
// cmd/spider/main.go
var (
    // ... 现有参数
    analyzeAPI = flag.Bool("analyze-api", false, "是否分析API端点")
)

// 在main函数中
if *analyzeAPI {
    spider.analyzeAPIs()
}
```

使用：

```bash
# 启用API分析
spider-golang -url http://api.example.com -analyze-api

# 生成的文件:
# - api_openapi.json      (OpenAPI 3.0文档)
# - api_documentation.md  (Markdown文档)
# - graphql_schema.graphql (GraphQL Schema)
```

---

## 最佳实践

### 1. 设置合理的超时

```go
analyzer := core.NewAPIAnalyzer("api.example.com")
analyzer.client.Timeout = 30 * time.Second // 根据网络情况调整
```

### 2. 处理速率限制

```go
for i, url := range endpoints {
    endpoint, err := analyzer.AnalyzeEndpoint(url)
    
    // 检查速率限制
    if endpoint.RateLimit != nil && endpoint.RateLimit.Remaining < 10 {
        fmt.Println("接近速率限制，等待...")
        time.Sleep(time.Minute)
    }
    
    // 添加延迟
    if i < len(endpoints)-1 {
        time.Sleep(time.Second)
    }
}
```

### 3. 缓存分析结果

```go
// 保存分析结果
data, _ := json.Marshal(analyzer.GetAllEndpoints())
ioutil.WriteFile("api_cache.json", data, 0644)

// 下次加载缓存
data, _ := ioutil.ReadFile("api_cache.json")
var cachedEndpoints []*core.APIEndpoint
json.Unmarshal(data, &cachedEndpoints)
```

### 4. 错误处理

```go
endpoint, err := analyzer.AnalyzeEndpoint(url)
if err != nil {
    // 记录错误但继续
    log.Printf("分析 %s 失败: %v", url, err)
    continue
}

// 检查响应
if len(endpoint.Methods) == 0 {
    log.Printf("警告: %s 没有支持的方法", url)
}

if endpoint.RequiresAuth && analyzer.authHeader == "" {
    log.Printf("提示: %s 需要认证", url)
}
```

---

## 常见问题

### Q1: 如何处理需要认证的API？

```go
// 设置认证头
analyzer.SetAuthentication("Bearer your-token-here")

// 或在每个请求中添加
analyzer.SetAuthentication("Basic " + base64.StdEncoding.EncodeToString(
    []byte("username:password")))
```

### Q2: GraphQL Introspection被禁用怎么办？

某些生产环境会禁用Introspection。解决方案：

1. 查找公开的Schema文件
2. 使用开发环境的端点
3. 通过文档手工构建Schema

### Q3: 如何提高分析准确度？

1. **多次请求验证**：对同一端点发送多次请求
2. **参数变化测试**：测试不同的参数值
3. **结合文档**：如果有API文档，作为参考
4. **手工修正**：分析完成后手工审查和修正

### Q4: 分析速度太慢？

优化方法：

1. **并发分析**：
```go
var wg sync.WaitGroup
for _, url := range endpoints {
    wg.Add(1)
    go func(u string) {
        defer wg.Done()
        analyzer.AnalyzeEndpoint(u)
    }(url)
}
wg.Wait()
```

2. **跳过详细测试**：只进行基础分析

### Q5: 如何分析大量端点？

```go
// 批量分析优化
func analyzeInBatches(endpoints []string, batchSize int) {
    for i := 0; i < len(endpoints); i += batchSize {
        end := i + batchSize
        if end > len(endpoints) {
            end = len(endpoints)
        }
        
        batch := endpoints[i:end]
        fmt.Printf("分析批次 %d-%d/%d\n", i+1, end, len(endpoints))
        
        // 分析批次
        for _, url := range batch {
            analyzer.AnalyzeEndpoint(url)
        }
        
        // 批次间休息
        time.Sleep(5 * time.Second)
    }
}
```

---

## 总结

API智能分析功能可以：

✅ **节省时间** - 自动分析API，无需手工测试  
✅ **提高准确性** - 系统化探测，不遗漏细节  
✅ **生成文档** - 自动生成标准化文档  
✅ **辅助测试** - 为渗透测试提供详细信息  
✅ **版本管理** - 跟踪API变化

---

**下一步**: 尝试运行 `examples/api_analysis_demo.go`，开始你的API分析之旅！

```bash
go run examples/api_analysis_demo.go
```

🚀 **Happy API Analyzing!**

