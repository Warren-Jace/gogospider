# URL过滤管理器 - 集成指南

## 📋 目录

1. [架构概述](#架构概述)
2. [快速开始](#快速开始)
3. [预设模式](#预设模式)
4. [自定义配置](#自定义配置)
5. [集成到Spider](#集成到spider)
6. [调试和诊断](#调试和诊断)
7. [性能优化](#性能优化)

---

## 架构概述

### 核心组件

```
URLFilterManager (管理器)
    ↓
过滤器管道（按优先级执行）
    ├─ 1. BasicFormatFilter     (优先级 10) - 基础格式验证
    ├─ 2. BlacklistFilter       (优先级 20) - 黑名单过滤
    ├─ 3. ScopeFilter           (优先级 30) - 域名作用域控制
    ├─ 4. TypeClassifierFilter  (优先级 40) - URL类型分类
    └─ 5. BusinessValueFilter   (优先级 50) - 业务价值评估
```

### 设计原则

✅ **单一入口** - 所有URL过滤通过一个管理器  
✅ **职责分离** - 每个过滤器只负责一个维度  
✅ **管道模式** - 过滤器按顺序组成管道  
✅ **可配置** - 统一的配置接口  
✅ **可观测** - 完整的过滤链路追踪  
✅ **可扩展** - 易于添加新的过滤器  

---

## 快速开始

### 1. 使用预设配置（推荐）

```go
import "spider-golang/core"

// 创建过滤管理器（平衡模式）
manager := core.NewURLFilterManagerWithPreset(
    core.PresetBalanced, 
    "example.com",
)

// 过滤URL
result := manager.Filter("https://example.com/api/users", nil)

if result.Allowed && result.Action == core.FilterAllow {
    // 允许爬取
    fmt.Printf("✓ URL通过: %s (分数: %.1f)\n", url, result.Score)
} else {
    // 拒绝或降级
    fmt.Printf("✗ URL被过滤: %s\n", result.Reason)
}
```

### 2. 简化接口

```go
// 只需要判断是否爬取
if manager.ShouldCrawl("https://example.com/page") {
    // 爬取该URL
    crawl(url)
}
```

### 3. 批量过滤

```go
urls := []string{
    "https://example.com/page1",
    "https://example.com/page2",
    "https://example.com/api/users",
}

results := manager.FilterBatch(urls, nil)

for url, result := range results {
    if result.Allowed {
        fmt.Printf("✓ %s\n", url)
    } else {
        fmt.Printf("✗ %s: %s\n", url, result.Reason)
    }
}
```

---

## 预设模式

### 1. 平衡模式（Balanced）⭐ 推荐

适用场景：**通用爬虫、中型网站**

```go
manager := core.NewURLFilterManagerWithPreset(core.PresetBalanced, "example.com")
```

**特点：**
- ✅ 外部链接：降级（记录但不爬取）
- ✅ 静态资源：降级（记录但不爬取）
- ✅ JS文件：允许（需要分析）
- ✅ CSS文件：降级
- ✅ 最低业务分数：30分

**适用于：**
- 一般性网站爬取
- 需要平衡覆盖率和效率
- 大部分场景的默认选择

---

### 2. 严格模式（Strict）

适用场景：**大型网站、资源有限**

```go
manager := core.NewURLFilterManagerWithPreset(core.PresetStrict, "example.com")
```

**特点：**
- ❌ 外部链接：拒绝
- ❌ 静态资源：拒绝
- ✅ JS文件：允许
- ❌ CSS文件：拒绝
- ✅ 最低业务分数：40分（更高）

**适用于：**
- 目标明确的爬取任务
- 需要减少爬取量
- 只关心高价值URL

---

### 3. 宽松模式（Loose）

适用场景：**新网站探索、测试**

```go
manager := core.NewURLFilterManagerWithPreset(core.PresetLoose, "example.com")
```

**特点：**
- ✅ 外部链接：降级
- ✅ 静态资源：降级
- ✅ JS文件：允许
- ✅ CSS文件：允许
- ✅ 最低业务分数：20分（很低）
- ⚠️ 黑名单：禁用

**适用于：**
- 探索未知网站
- 测试和调试
- 需要最大覆盖率

---

### 4. API专用模式（API Only）

适用场景：**API端点发现**

```go
manager := core.NewURLFilterManagerWithPreset(core.PresetAPIOnly, "api.example.com")
```

**特点：**
- ✅ 只保留API端点（/api/, /rest/, /v1/, etc）
- ❌ 非API URL全部拒绝
- ❌ 外部链接：拒绝
- ❌ 静态资源：拒绝

**适用于：**
- API端点收集
- API安全测试
- 微服务发现

---

### 5. 深度扫描模式（Deep Scan）

适用场景：**安全审计、完整扫描**

```go
manager := core.NewURLFilterManagerWithPreset(core.PresetDeepScan, "example.com")
```

**特点：**
- ✅ 启用链路追踪
- ✅ 详细日志
- ✅ 不启用早停（完整评估）
- ✅ 更宽松的阈值
- ✅ 最低业务分数：15分

**适用于：**
- 安全审计
- 完整性检查
- 调试和诊断

---

## 自定义配置

### 使用构建器模式

```go
manager := core.NewFilterManagerBuilder("example.com").
    WithMode(core.FilterModeBalanced).
    WithCaching(true, 10000).
    WithEarlyStop(true).
    WithTrace(false, 100).
    AddBasicFormat().
    AddBlacklist().
    AddScope(core.ScopeFilterConfig{
        AllowSubdomains:    true,
        AllowHTTP:          true,
        AllowHTTPS:         true,
        ExternalLinkAction: core.FilterDegrade,
    }).
    AddTypeClassifier(core.TypeClassifierConfig{
        StaticResourceAction: core.FilterDegrade,
        JSFileAction:         core.FilterAllow,
        CSSFileAction:        core.FilterDegrade,
    }).
    AddBusinessValue(30.0, 70.0).
    Build()
```

### 动态调整

```go
// 启用/禁用特定过滤器
manager.EnableFilter("Blacklist")
manager.DisableFilter("BusinessValue")

// 切换模式
manager.SetMode(core.FilterModeStrict)
```

---

## 集成到Spider

### 第1步：在Spider结构中添加过滤管理器

```go
// core/spider.go

type Spider struct {
    // ... 其他字段 ...
    
    // 新增：统一的URL过滤管理器
    filterManager *URLFilterManager
    
    // ... 其他字段 ...
}
```

### 第2步：初始化过滤管理器

```go
// core/spider.go - NewSpider()

func NewSpider(cfg *config.Config) *Spider {
    spider := &Spider{
        // ... 其他初始化 ...
    }
    
    // 初始化过滤管理器（根据配置选择预设）
    preset := PresetBalanced // 默认
    if cfg.FilterSettings.Mode == "strict" {
        preset = PresetStrict
    } else if cfg.FilterSettings.Mode == "loose" {
        preset = PresetLoose
    }
    
    spider.filterManager = NewURLFilterManagerWithPreset(
        preset,
        cfg.TargetURL,
    )
    
    return spider
}
```

### 第3步：替换现有过滤逻辑

#### 原有代码（分散的过滤）

```go
// core/spider.go - collectLinksForLayer()

for _, link := range allLinks {
    // 登录墙检测
    if s.loginWallDetector != nil {
        shouldSkip, reason := s.loginWallDetector.ShouldSkipURL(link)
        if shouldSkip {
            continue
        }
    }
    
    // 扩展名检查
    if s.scopeController != nil {
        shouldRequest, reason := s.scopeController.ShouldRequestURL(link)
        if !shouldRequest {
            continue
        }
    }
    
    // 分层去重
    if s.layeredDedup != nil {
        shouldProcess, urlType, reason := s.layeredDedup.ShouldProcess(link, "GET")
        if !shouldProcess {
            continue
        }
    }
    
    // 业务过滤
    if s.businessFilter != nil {
        shouldCrawl, reason, score := s.businessFilter.ShouldCrawlURL(link)
        if !shouldCrawl {
            continue
        }
    }
    
    // ... 更多检查 ...
    
    tasksToSubmit = append(tasksToSubmit, link)
}
```

#### 新代码（统一过滤）

```go
// core/spider.go - collectLinksForLayer()

for _, link := range allLinks {
    // 统一的过滤入口
    result := s.filterManager.Filter(link, map[string]interface{}{
        "depth":       depth,
        "method":      "GET",
        "source_type": "html",
    })
    
    // 处理结果
    switch result.Action {
    case FilterAllow:
        // 允许爬取
        tasksToSubmit = append(tasksToSubmit, link)
        
    case FilterDegrade:
        // 降级处理（记录但不爬取）
        s.RecordDegradedURL(link, result.Reason)
        
    case FilterReject:
        // 拒绝（跳过）
        continue
    }
}
```

### 第4步：添加配置

```go
// config/config.go

type Config struct {
    // ... 其他字段 ...
    
    // 新增：过滤器设置
    FilterSettings FilterSettings `json:"filter_settings"`
}

type FilterSettings struct {
    Mode            string  `json:"mode"`              // strict/balanced/loose
    Preset          string  `json:"preset"`            // strict/balanced/loose/api_only/deep_scan
    EnableTrace     bool    `json:"enable_trace"`      // 启用链路追踪
    MinBusinessScore float64 `json:"min_business_score"` // 最低业务分数
}
```

### 第5步：配置文件示例

```json
{
  "target_url": "https://example.com",
  "filter_settings": {
    "preset": "balanced",
    "enable_trace": false,
    "min_business_score": 30.0
  },
  "depth_settings": {
    "max_depth": 3
  }
}
```

---

## 调试和诊断

### 1. 查看统计信息

```go
// 打印统计报告
manager.PrintStatistics()
```

**输出示例：**
```
╔════════════════════════════════════════════════════════════════╗
║              URL过滤管理器 - 统计报告                         ║
╠════════════════════════════════════════════════════════════════╣
║ 模式: balanced | 启用: true  | 早停: true                    ║
╠════════════════════════════════════════════════════════════════╣
║ 总处理:   1000        | 平均耗时: 123µs                       ║
║ 允许:     700          (70.0%)                                 ║
║ 拒绝:     200          (20.0%)                                 ║
║ 降级:     100          (10.0%)                                 ║
╠════════════════════════════════════════════════════════════════╣
║ 过滤器详情                                                     ║
╠════════════════════════════════════════════════════════════════╣
║ • BasicFormat                                                  ║
║   检查: 1000      | 拒绝: 50        (5.0%)  |    10µs         ║
║ • Blacklist                                                    ║
║   检查: 950       | 拒绝: 100       (10.5%) |    15µs         ║
║ • Scope                                                        ║
║   检查: 850       | 拒绝: 50        (5.9%)  |    20µs         ║
╚════════════════════════════════════════════════════════════════╝
```

### 2. 解释特定URL

```go
// 详细解释为什么URL被过滤
explanation := manager.ExplainURL("https://example.com/test.jpg")
fmt.Println(explanation)
```

**输出示例：**
```
═══════════════════════════════════════════════════════════════
URL: https://example.com/test.jpg
最终结果: 静态资源（.jpg） (降级)
处理时间: 156µs
执行过滤器数: 4
═══════════════════════════════════════════════════════════════
过滤链路:
  1. [✓] BasicFormat
     动作: 允许
     原因: 基础格式检查通过
     耗时: 12µs
  2. [✓] Blacklist
     动作: 允许
     原因: 黑名单检查通过
     耗时: 18µs
  3. [✓] Scope
     动作: 允许
     原因: 目标域名
     评分: 100.0
     耗时: 25µs
  4. [✗] TypeClassifier
     动作: 降级
     原因: 静态资源（.jpg）
     评分: 20.0
     耗时: 101µs
═══════════════════════════════════════════════════════════════
```

### 3. 启用链路追踪

```go
// 启用追踪（用于调试）
manager := core.NewFilterManagerBuilder("example.com").
    WithTrace(true, 200).  // 启用追踪，缓冲区200条
    // ... 其他配置 ...
    Build()

// 获取最近的追踪记录
traces := manager.GetRecentTraces(10)
for _, trace := range traces {
    fmt.Printf("URL: %s, 结果: %s, 耗时: %v\n", 
        trace.URL, trace.Result.Reason, trace.Duration)
}
```

---

## 性能优化

### 1. 启用结果缓存

```go
manager := core.NewFilterManagerBuilder("example.com").
    WithCaching(true, 20000).  // 缓存2万条结果
    Build()
```

### 2. 启用早停

```go
manager := core.NewFilterManagerBuilder("example.com").
    WithEarlyStop(true).  // 第一个拒绝就停止
    Build()
```

### 3. 禁用不需要的过滤器

```go
// 禁用业务价值评估（提升性能）
manager.DisableFilter("BusinessValue")
```

### 4. 性能对比

| 配置 | 平均耗时 | 说明 |
|-----|---------|------|
| 全部启用 + 无缓存 | ~150µs | 基准 |
| 全部启用 + 缓存 | ~80µs | 缓存命中时 |
| 启用早停 | ~60µs | 拒绝时快速返回 |
| 只启用基础过滤器 | ~30µs | 最快 |

---

## 迁移指南

### 从旧架构迁移

#### 步骤1：保留旧代码（向后兼容）

```go
// 在Spider中同时保留新旧两套
type Spider struct {
    // 新：统一过滤管理器
    filterManager *URLFilterManager
    
    // 旧：保留用于向后兼容
    urlValidator      URLValidatorInterface
    scopeController   *ScopeController
    businessFilter    *BusinessAwareURLFilter
    // ...
}
```

#### 步骤2：添加开关

```go
// config/config.go
type Config struct {
    UseNewFilterManager bool `json:"use_new_filter_manager"` // 新增开关
    // ...
}
```

#### 步骤3：条件使用

```go
// core/spider.go
func (s *Spider) collectLinksForLayer(depth int) []string {
    // ...
    
    for _, link := range allLinks {
        if s.config.UseNewFilterManager && s.filterManager != nil {
            // 使用新的过滤管理器
            result := s.filterManager.Filter(link, ctx)
            if !result.Allowed || result.Action != FilterAllow {
                continue
            }
        } else {
            // 使用旧的过滤逻辑
            if !s.urlValidator.IsValidBusinessURL(link) {
                continue
            }
            // ... 其他旧逻辑 ...
        }
        
        tasksToSubmit = append(tasksToSubmit, link)
    }
    
    return tasksToSubmit
}
```

#### 步骤4：逐步迁移

1. **阶段1：并行运行**（2周）
   - 同时运行新旧两套
   - 对比结果差异
   - 调整配置

2. **阶段2：默认新系统**（2周）
   - 默认使用新系统
   - 保留旧系统作为回退

3. **阶段3：移除旧代码**（1周）
   - 完全移除旧的过滤逻辑
   - 清理死代码

---

## 常见问题

### Q1：如何添加自定义过滤器？

```go
// 实现URLFilter接口
type MyCustomFilter struct {
    enabled bool
    // ...
}

func (f *MyCustomFilter) Name() string { return "MyCustom" }
func (f *MyCustomFilter) Priority() int { return 35 } // 设置优先级
func (f *MyCustomFilter) Filter(rawURL string, ctx *FilterContext) FilterResult {
    // 自定义逻辑
    if /* 条件 */ {
        return FilterResult{
            Allowed: false,
            Action:  FilterReject,
            Reason:  "自定义原因",
        }
    }
    return FilterResult{Allowed: true, Action: FilterAllow}
}
// ... 实现其他接口方法 ...

// 注册到管理器
manager.RegisterFilter(&MyCustomFilter{enabled: true})
```

### Q2：如何调整业务价值评分算法？

修改`core/url_filters.go`中的`calculateScore`方法。

### Q3：性能瓶颈在哪里？

使用追踪查看每个过滤器的耗时，通常是：
1. URL解析（可缓存）
2. 业务价值计算（可禁用）
3. 正则匹配（优化模式）

---

## 最佳实践

1. ✅ **生产环境使用平衡模式**
2. ✅ **启用缓存和早停**
3. ✅ **定期查看统计报告，调整配置**
4. ✅ **新网站先用宽松模式探索**
5. ✅ **调试时启用链路追踪**
6. ⚠️ **避免在循环中创建新的管理器**
7. ⚠️ **大规模爬取时注意缓存大小**

---

## 总结

**新架构的优势：**

✅ 统一的过滤入口  
✅ 清晰的职责分离  
✅ 灵活的配置系统  
✅ 强大的调试能力  
✅ 易于扩展  
✅ 性能优化  

**推荐使用流程：**

1. 开始：使用`PresetBalanced`
2. 调试：启用`ExplainURL`查看过滤原因
3. 优化：根据统计报告调整配置
4. 生产：禁用追踪，启用缓存

---

**文档版本：** v1.0  
**最后更新：** 2025-10-28

