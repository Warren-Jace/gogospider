# GogoSpider 程序运行逻辑示意图

## 1. 程序架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        GogoSpider v3.4                          │
│                   智能Web安全爬虫系统                            │
└─────────────────────────────────────────────────────────────────┘

                              ▼
                    ┌──────────────────┐
                    │   cmd/spider     │
                    │   main.go        │
                    │  (程序入口)       │
                    └────────┬─────────┘
                             │
                             ▼
        ┌────────────────────┴────────────────────┐
        │          配置加载与验证                  │
        │  • config.json (配置文件)               │
        │  • 命令行参数解析                        │
        │  • 参数验证与合并                        │
        └────────────────────┬────────────────────┘
                             │
                             ▼
        ┌────────────────────────────────────────┐
        │         创建Spider实例                  │
        │     (core/spider.go)                   │
        │  • 初始化所有核心组件                    │
        │  • 配置各个子系统                        │
        └────────────────────┬───────────────────┘
                             │
                             ▼
        ┌────────────────────────────────────────┐
        │          Spider.Start()                │
        │         开始爬取流程                     │
        └────────────────────┬───────────────────┘
                             │
                             ▼
```

## 2. 核心组件架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Spider (协调器)                           │
│                    core/spider.go                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │ 静态爬虫      │  │ 动态爬虫      │  │ 被动爬虫      │        │
│  │StaticCrawler│  │DynamicCrawler│  │PassiveCrawler│        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
│                                                                 │
│  ┌──────────────────────────────────────────────────┐          │
│  │           内容分析器                              │          │
│  │  • JSAnalyzer (JS分析)                           │          │
│  │  • CSSAnalyzer (CSS分析)                         │          │
│  │  • APIAnalyzer (API推测)                         │          │
│  │  • FormFiller (表单填充)                         │          │
│  └──────────────────────────────────────────────────┘          │
│                                                                 │
│  ┌──────────────────────────────────────────────────┐          │
│  │           URL处理与去重                           │          │
│  │  • URLFilterManager (URL过滤管理)                │          │
│  │  • LayeredDeduplicator (分层去重)                │          │
│  │  • URLPatternDedup (模式去重)                    │          │
│  │  • URLStructureDedup (结构去重)                  │          │
│  │  • URLCanonicalizer (URL规范化)                 │          │
│  └──────────────────────────────────────────────────┘          │
│                                                                 │
│  ┌──────────────────────────────────────────────────┐          │
│  │           智能检测与过滤                          │          │
│  │  • SensitiveInfoManager (敏感信息检测)           │          │
│  │  • BusinessFilter (业务感知过滤)                 │          │
│  │  • LoginWallDetector (登录墙检测)                │          │
│  │  • RedirectManager (重定向管理)                  │          │
│  │  • POSTDetector (POST请求检测)                   │          │
│  └──────────────────────────────────────────────────┘          │
│                                                                 │
│  ┌──────────────────────────────────────────────────┐          │
│  │           辅助功能                                │          │
│  │  • CookieManager (Cookie管理)                    │          │
│  │  • RateLimiter (速率控制)                        │          │
│  │  • WorkerPool (并发控制)                         │          │
│  │  • AdaptiveLearner (自适应学习)                  │          │
│  └──────────────────────────────────────────────────┘          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 3. 主要工作流程

### 3.1 爬取流程

```
                    开始爬取
                       ▼
         ┌─────────────────────────┐
         │  1. 初始化URL队列        │
         │     (目标URL入队)        │
         └────────────┬────────────┘
                      │
                      ▼
         ┌─────────────────────────┐
         │  2. URL过滤与验证        │
         │   • URLFilterManager    │
         │   • 黑名单检查           │
         │   • 范围检查             │
         └────────────┬────────────┘
                      │
                      ▼
         ┌─────────────────────────┐
         │  3. URL去重              │
         │   • 分层去重策略         │
         │   • 模式去重             │
         │   • 结构去重             │
         └────────────┬────────────┘
                      │
                      ▼
         ┌─────────────────────────┐
         │  4. 选择爬虫引擎         │
         │   • 静态页面 → Static   │
         │   • 动态页面 → Dynamic  │
         └────────────┬────────────┘
                      │
         ┌────────────┴────────────┐
         ▼                         ▼
    ┌─────────┐            ┌──────────┐
    │Static   │            │Dynamic   │
    │Crawler  │            │Crawler   │
    │(Colly)  │            │(Chromedp)│
    └────┬────┘            └─────┬────┘
         │                       │
         └───────────┬───────────┘
                     ▼
         ┌─────────────────────────┐
         │  5. 页面内容提取         │
         │   • HTML解析            │
         │   • 链接提取             │
         │   • 表单识别             │
         │   • AJAX拦截            │
         └────────────┬────────────┘
                      │
                      ▼
         ┌─────────────────────────┐
         │  6. JS/CSS分析          │
         │   • JS中的URL提取       │
         │   • API端点推测         │
         │   • 跨域JS分析          │
         └────────────┬────────────┘
                      │
                      ▼
         ┌─────────────────────────┐
         │  7. 敏感信息检测         │
         │   • 响应体扫描           │
         │   • 响应头扫描           │
         │   • 40+种规则匹配       │
         └────────────┬────────────┘
                      │
                      ▼
         ┌─────────────────────────┐
         │  8. 新URL入队            │
         │   • 深度+1              │
         │   • 优先级计算           │
         │   • 重复检查             │
         └────────────┬────────────┘
                      │
                      ▼
         ┌─────────────────────────┐
         │  9. 判断是否继续         │
         │   • 达到最大深度?       │
         │   • 达到最大页数?       │
         │   • 队列为空?           │
         └────────────┬────────────┘
                      │
         ┌────────────┴────────────┐
         ▼                         ▼
        继续                      结束
         │                         │
         └───返回步骤2              ▼
                          ┌──────────────┐
                          │ 生成输出结果  │
                          └──────────────┘
```

### 3.2 URL处理流程

```
      新URL发现
         │
         ▼
┌─────────────────┐
│ 1. URL规范化     │
│  • IDN处理      │
│  • 去重斜杠     │
│  • 移除追踪参数 │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 2. URL过滤      │
│  • 静态资源     │
│  • 黑名单       │
│  • 域名范围     │
│  • 文件扩展名   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 3. 多层去重     │
│  L1: 完全匹配   │
│  L2: 参数去重   │
│  L3: 结构去重   │
│  L4: 模式去重   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 4. 质量评分     │
│  • 业务价值     │
│  • 深度惩罚     │
│  • API加权      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 5. 入队或丢弃   │
└─────────────────┘
```

## 4. 数据流向图

```
┌────────────┐
│  目标URL    │
└─────┬──────┘
      │
      ▼
┌──────────────────┐
│  URL队列          │
│  (优先级队列)     │
└─────┬────────────┘
      │
      ▼
┌──────────────────┐      ┌──────────────────┐
│  爬虫引擎         │ ───→ │  HTTP请求         │
│  (Static/Dynamic)│ ←─── │  (响应数据)       │
└─────┬────────────┘      └──────────────────┘
      │
      ▼
┌──────────────────┐
│  内容提取         │
│  • 链接          │
│  • 表单          │
│  • API端点       │
│  • JS/CSS       │
└─────┬────────────┘
      │
      ▼
┌──────────────────┐
│  智能分析         │
│  • JS分析        │
│  • 敏感信息检测   │
│  • 技术栈识别     │
└─────┬────────────┘
      │
      ├──────────────────┐
      │                  │
      ▼                  ▼
┌──────────┐      ┌──────────┐
│ 新URL    │      │ 结果数据  │
│ (入队)   │      │ (保存)   │
└──────────┘      └──────────┘
```

## 5. 关键数据结构

### 5.1 核心结构体

```go
// Spider - 主协调器
type Spider struct {
    config              *config.Config
    staticCrawler       StaticCrawler      // 静态爬虫
    dynamicCrawler      DynamicCrawler     // 动态爬虫
    jsAnalyzer          *JSAnalyzer        // JS分析器
    urlFilterManager    *URLFilterManager  // URL过滤管理器
    layeredDedup        *LayeredDeduplicator // 分层去重器
    sensitiveManager    *SensitiveInfoManager // 敏感信息管理器
    // ... 更多组件
}

// Result - 爬取结果
type Result struct {
    URL         string
    StatusCode  int
    Links       []string           // 发现的链接
    Forms       []Form             // 表单信息
    APIs        []string           // API端点
    POSTRequests []POSTRequest     // POST请求
    HTMLContent string             // HTML内容
    Headers     map[string]string  // HTTP头
}

// Config - 配置结构
type Config struct {
    TargetURL              string
    DepthSettings          DepthSettings
    StrategySettings       StrategySettings
    FilterSettings         FilterSettings
    SensitiveDetectionSettings SensitiveDetectionSettings
    // ... 更多配置
}
```

## 6. 主要功能模块

### 6.1 爬虫引擎

| 模块 | 文件 | 功能 |
|------|------|------|
| 静态爬虫 | `core/static_crawler.go` | 基于Colly的静态页面爬取 |
| 动态爬虫 | `core/dynamic_crawler.go` | 基于Chromedp的动态页面爬取 |
| 被动爬虫 | `core/passive_crawler.go` | Sitemap和robots.txt解析 |

### 6.2 内容分析

| 模块 | 文件 | 功能 |
|------|------|------|
| JS分析器 | `core/js_analyzer.go` | JavaScript URL提取和API推测 |
| CSS分析器 | `core/css_analyzer.go` | CSS中的URL提取 |
| 表单填充 | `core/smart_form_filler.go` | 智能表单识别和填充 |
| API分析 | `core/api_analyzer.go` | GraphQL和REST API分析 |

### 6.3 URL处理

| 模块 | 文件 | 功能 |
|------|------|------|
| URL过滤管理 | `core/url_filter_manager.go` | 统一URL过滤入口 |
| 分层去重 | `core/layered_deduplicator.go` | 4层智能去重策略 |
| 模式去重 | `core/url_pattern_dedup.go` | URL模式识别去重 |
| 结构去重 | `core/url_structure_dedup.go` | URL结构化去重 |
| URL规范化 | `core/url_canonicalizer.go` | URL标准化处理 |

### 6.4 智能检测

| 模块 | 文件 | 功能 |
|------|------|------|
| 敏感信息检测 | `core/sensitive_info_detector.go` | 40+种敏感信息规则 |
| 敏感信息管理 | `core/sensitive_info_manager.go` | 统一敏感信息管理 |
| 业务感知过滤 | `core/business_aware_filter.go` | 基于业务价值的URL过滤 |
| 登录墙检测 | `core/login_wall_detector.go` | 检测登录页面 |
| POST检测 | `core/post_request_detector.go` | POST请求智能检测 |

### 6.5 辅助功能

| 模块 | 文件 | 功能 |
|------|------|------|
| Cookie管理 | `core/cookie_manager.go` | Cookie加载和管理 |
| 速率控制 | `core/rate_limiter.go` | 请求速率限制 |
| 并发控制 | `core/worker_pool.go` | Worker池管理 |
| 自适应学习 | `core/adaptive_priority_learner.go` | 优先级自适应调整 |
| 重定向管理 | `core/redirect_manager.go` | 重定向链检测 |

## 7. 配置文件

### 7.1 主配置文件 (config.json)

```json
{
  "target_url": "https://example.com",
  "depth_settings": {
    "max_depth": 3,
    "deep_crawling": true,
    "scheduling_algorithm": "BFS"
  },
  "strategy_settings": {
    "enable_static_crawler": true,
    "enable_dynamic_crawler": true,
    "enable_js_analysis": true
  },
  "filter_settings": {
    "enabled": true,
    "static_resources_mode": "record_only"
  },
  "sensitive_detection_settings": {
    "enabled": true,
    "rules_file": "sensitive_rules_standard.json"
  }
}
```

### 7.2 敏感信息规则 (sensitive_rules_*.json)

- `sensitive_rules_minimal.json` - 10个核心规则
- `sensitive_rules_standard.json` - 40个标准规则
- `sensitive_rules_config.json` - 完整规则集

## 8. 输出文件

爬取完成后生成3个核心文件：

| 文件 | 说明 |
|------|------|
| `*_detail.txt` | 完整爬取数据（页面、链接、表单、API等） |
| `*_all_links.txt` | 所有发现的链接（包括域外、静态资源） |
| `*_in_scope.txt` | 范围内的有效链接（可直接用于测试） |
| `*_sensitive.txt` | 敏感信息报告（文本格式） |
| `*_sensitive.json` | 敏感信息报告（JSON格式） |

## 9. 性能优化策略

### 9.1 URL处理优化

- **URL解析缓存**: 缓存已解析的URL，避免重复解析
- **分片锁机制**: 降低并发竞争，提升多线程性能
- **混合去重**: 布隆过滤器 + 精确去重

### 9.2 爬取优化

- **静态资源过滤**: 只记录不请求，效率提升70%
- **优先级队列**: 高价值URL优先爬取
- **自适应学习**: 根据结果动态调整优先级

### 9.3 内存优化

- **字符串池**: 复用常见字符串，减少内存分配
- **分层去重**: 逐层过滤，减少内存占用

## 10. 核心算法

### 10.1 BFS调度算法

```
初始化: queue = [目标URL]
while queue not empty and not reach_limit:
    url = queue.pop()
    if url已访问: continue
    
    result = crawl(url)
    新URLs = extract_urls(result)
    
    for 新url in 新URLs:
        if 通过过滤 and 未去重:
            queue.push(新url)
```

### 10.2 分层去重算法

```
Level 1: 完全匹配
  if url in visited_set: 丢弃

Level 2: 忽略参数值
  pattern = url_to_pattern(url)  # /page?id=*
  if pattern in pattern_set: 丢弃

Level 3: 结构化去重
  struct = url_to_struct(url)    # /user/{id}/profile
  if struct in struct_set: 丢弃

Level 4: DOM相似度
  similarity = dom_similarity(url)
  if similarity > threshold: 丢弃
```

## 11. 典型使用场景

### 场景1: 快速扫描
```bash
spider -url https://example.com -depth 3
```

### 场景2: API发现
```bash
spider -url https://example.com -include-paths "/api/*,/v1/*"
```

### 场景3: 敏感信息检测
```bash
spider -url https://example.com -sensitive-rules sensitive_rules_standard.json
```

### 场景4: 批量扫描
```bash
spider -batch-file targets.txt -batch-concurrency 10
```

---

## 总结

GogoSpider是一个功能强大的智能Web安全爬虫，采用模块化设计，支持：

- ✅ **双引擎爬虫**: 静态+动态
- ✅ **智能去重**: 4层去重策略
- ✅ **敏感信息检测**: 40+种规则
- ✅ **高性能**: 多种优化策略
- ✅ **易扩展**: 组件化设计

核心优势在于**智能URL处理**、**敏感信息检测**和**高效的去重机制**。

