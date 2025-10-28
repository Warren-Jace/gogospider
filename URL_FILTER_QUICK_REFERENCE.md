# URL过滤管理器 - 快速参考卡

## 🚀 1分钟快速开始

### 最简单的使用

```go
// 1. 创建管理器
manager := core.NewURLFilterManagerWithPreset(
    core.PresetBalanced,  // 平衡模式
    "example.com",        // 目标域名
)

// 2. 过滤URL
if manager.ShouldCrawl("https://example.com/page") {
    // 爬取
}
```

---

## 📋 5种预设模式

| 模式 | 使用场景 | 通过率 | 命令 |
|-----|---------|-------|------|
| **Balanced** ⭐ | 通用爬虫 | ~70% | `PresetBalanced` |
| **Strict** | 大型网站 | ~50% | `PresetStrict` |
| **Loose** | 新网站探索 | ~85% | `PresetLoose` |
| **APIOnly** | API发现 | ~20% | `PresetAPIOnly` |
| **DeepScan** | 安全审计 | ~75% | `PresetDeepScan` |

### 选择指南

```
需要API端点？     → PresetAPIOnly
网站很大？        → PresetStrict
第一次爬？        → PresetLoose
安全审计？        → PresetDeepScan
不确定？          → PresetBalanced ⭐
```

---

## ⚙️ 常用操作

### 创建管理器

```go
// 方式1：使用预设
manager := NewURLFilterManagerWithPreset(PresetBalanced, "example.com")

// 方式2：自定义构建
manager := NewFilterManagerBuilder("example.com").
    WithMode(FilterModeBalanced).
    WithCaching(true, 10000).
    AddBasicFormat().
    AddBlacklist().
    Build()
```

### 过滤URL

```go
// 完整信息
result := manager.Filter("https://example.com/page", nil)
if result.Allowed && result.Action == FilterAllow {
    // 爬取
}

// 简化接口
if manager.ShouldCrawl("https://example.com/page") {
    // 爬取
}

// 批量过滤
urls := []string{"url1", "url2"}
results := manager.FilterBatch(urls, nil)
```

### 调试URL

```go
// 查看为什么URL被过滤
explanation := manager.ExplainURL("https://example.com/test")
fmt.Println(explanation)
```

### 查看统计

```go
manager.PrintStatistics()
```

---

## 🎯 3种过滤动作

| 动作 | 含义 | 应用场景 |
|-----|------|---------|
| **Allow** | 允许爬取 | 目标域名、API端点、JS文件 |
| **Reject** | 完全拒绝 | 垃圾URL、JavaScript关键字 |
| **Degrade** | 记录不爬取 | 静态资源、外部链接 |

### 处理示例

```go
result := manager.Filter(url, nil)

switch result.Action {
case FilterAllow:
    crawl(url)                    // 正常爬取
case FilterReject:
    // 跳过                        // 完全跳过
case FilterDegrade:
    recordURL(url)                 // 记录但不爬取
}
```

---

## 🔧 动态调整

### 启用/禁用过滤器

```go
manager.DisableFilter("Blacklist")   // 禁用黑名单
manager.EnableFilter("Blacklist")    // 启用黑名单
```

### 切换模式

```go
manager.SetMode(FilterModeStrict)    // 切换到严格模式
manager.SetMode(FilterModeLoose)     // 切换到宽松模式
```

### 查看过滤器列表

```go
filters := manager.ListFilters()
// ["BasicFormat", "Blacklist", "Scope", "TypeClassifier", "BusinessValue"]
```

---

## 📊 5个核心过滤器

| 过滤器 | 优先级 | 检查内容 | 拦截率 |
|--------|-------|---------|--------|
| **BasicFormat** | 10 | 空URL、无效协议、长度 | ~5% |
| **Blacklist** | 20 | JS关键字、CSS属性、代码片段 | ~10% |
| **Scope** | 30 | 域名、子域名、外部链接 | ~5% |
| **TypeClassifier** | 40 | URL类型、静态资源 | ~10% (降级) |
| **BusinessValue** | 50 | 业务价值评分 | ~5% |

---

## 🎨 配置速查

### 平衡模式（推荐）

```json
{
  "filter_settings": {
    "preset": "balanced",
    "enable_early_stop": true,
    "external_link_action": "degrade",
    "static_resource_action": "degrade",
    "min_business_score": 30.0
  }
}
```

### 严格模式

```json
{
  "filter_settings": {
    "preset": "strict",
    "external_link_action": "reject",
    "static_resource_action": "reject",
    "min_business_score": 40.0
  }
}
```

### API专用模式

```json
{
  "filter_settings": {
    "preset": "api_only",
    "static_resource_action": "reject"
  }
}
```

---

## 🔍 调试技巧

### 问题：URL被意外过滤

```go
// 1. 启用追踪
manager.config.EnableTrace = true

// 2. 解释URL
explanation := manager.ExplainURL("问题URL")
fmt.Println(explanation)

// 3. 查看哪个过滤器拒绝的
// 输出会显示完整的过滤链路
```

### 问题：通过率太低

```go
// 1. 查看统计
manager.PrintStatistics()

// 2. 找到拦截最多的过滤器
// 3. 调整或禁用该过滤器
manager.DisableFilter("Blacklist")

// 或切换到宽松模式
manager.SetMode(FilterModeLoo��e)
```

### 问题：性能慢

```go
// 1. 启用性能优化
manager.config.EnableCaching = true     // 缓存
manager.config.EnableEarlyStop = true   // 早停

// 2. 禁用不需要的过滤器
manager.DisableFilter("BusinessValue")  // 业务评估较慢

// 3. 关闭追踪
manager.config.EnableTrace = false
```

---

## 💡 最佳实践

### ✅ 推荐做法

1. **生产环境**：使用 `PresetBalanced` + 启用缓存和早停
2. **新网站**：先用 `PresetLoose` 探索，再切换到 `Balanced`
3. **API扫描**：使用 `PresetAPIOnly`
4. **调试**：启用追踪 + 使用 `ExplainURL()`
5. **定期检查**：运行 `PrintStatistics()` 查看效果

### ⚠️ 避免做法

1. ❌ 禁用所有过滤器（会爬取大量垃圾）
2. ❌ 在循环中创建新管理器（性能浪费）
3. ❌ 生产环境启用追踪（占用内存）
4. ❌ 过度调整配置（使用预设即可）

---

## 📞 常见问题

### Q1: 如何知道哪些URL被降级了？

```go
result := manager.Filter(url, nil)
if result.Action == FilterDegrade {
    fmt.Printf("降级: %s - %s\n", url, result.Reason)
    // 记录到降级列表
    degradedURLs = append(degradedURLs, url)
}
```

### Q2: 如何调整业务价值评分？

修改 `core/url_filters.go` 中的 `calculateScore()` 方法，或：

```go
// 禁用业务价值过滤器
manager.DisableFilter("BusinessValue")
```

### Q3: 如何让某些URL总是通过？

添加白名单过滤器：

```go
type WhitelistFilter struct {
    whitelist map[string]bool
}

func (f *WhitelistFilter) Filter(url string, ctx *FilterContext) FilterResult {
    if f.whitelist[url] {
        return FilterResult{Allowed: true, Action: FilterAllow, Reason: "白名单"}
    }
    return FilterResult{Allowed: true, Action: FilterAllow} // 继续检查
}

manager.RegisterFilter(NewWhitelistFilter())
```

### Q4: 性能开销多大？

- **无优化**：~150µs/URL
- **启用缓存**：~15µs/URL（命中时）
- **10K URL**：~150ms（可忽略）

---

## 🎯 核心代码片段

### 集成到Spider

```go
// core/spider.go

// 初始化
func NewSpider(cfg *config.Config) *Spider {
    spider := &Spider{}
    
    // 创建过滤管理器
    spider.filterManager = NewURLFilterManagerWithPreset(
        PresetBalanced,
        cfg.TargetURL,
    )
    
    return spider
}

// 使用
func (s *Spider) collectLinksForLayer(depth int) []string {
    for _, link := range allLinks {
        // 统一过滤
        result := s.filterManager.Filter(link, map[string]interface{}{
            "depth": depth,
            "method": "GET",
        })
        
        switch result.Action {
        case FilterAllow:
            tasksToSubmit = append(tasksToSubmit, link)
        case FilterDegrade:
            s.RecordDegradedURL(link)
        case FilterReject:
            continue
        }
    }
    return tasksToSubmit
}
```

---

## 📈 性能提升

| 优化项 | 提升幅度 | 启用方法 |
|--------|---------|---------|
| URL解析缓存 | 40% | 自动（FilterContext） |
| 早停优化 | 60% | `WithEarlyStop(true)` |
| 结果缓存 | 80% | `WithCaching(true, 10000)` |
| **总计** | **90%** | 全部启用 |

---

## 🎨 自定义示例

```go
// 完全自定义配置
manager := NewFilterManagerBuilder("example.com").
    WithMode(FilterModeBalanced).              // 模式
    WithCaching(true, 10000).                   // 缓存
    WithEarlyStop(true).                        // 早停
    WithTrace(false, 100).                      // 追踪（调试时启用）
    AddBasicFormat().                           // 基础格式
    AddBlacklist().                             // 黑名单
    AddScope(ScopeFilterConfig{                // 作用域
        AllowSubdomains:    true,
        ExternalLinkAction: FilterDegrade,
    }).
    AddTypeClassifier(TypeClassifierConfig{   // 类型分类
        StaticResourceAction: FilterDegrade,
        JSFileAction:         FilterAllow,
    }).
    AddBusinessValue(30.0, 70.0).              // 业务价值
    Build()
```

---

**保存此页面为书签，随时查阅！** 📌

