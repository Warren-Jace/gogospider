# URL模式+DOM相似度去重功能说明

## 版本
v4.5 - 2025年11月5日

## 功能概述

**URL模式+DOM相似度去重**是一种更精准的去重策略，解决了单纯依赖URL相似度判断可能导致的误判问题。

### 核心思路

1. **URL模式分组**：根据URL模式（忽略参数值）进行分组
2. **采样验证**：对于每个URL模式，先访问N次（默认3次）
3. **DOM相似度计算**：计算这N次访问的DOM结构相似度
4. **智能判断**：如果DOM相似度高（默认≥85%），说明这个URL模式下的页面确实相似，后续相同模式的URL直接跳过
5. **避免误判**：单纯URL相似但内容不同的页面会被正常爬取

## 为什么需要这个功能？

### 问题场景

传统的URL相似度去重可能会出现以下问题：

```
例1：URL相似但内容不同
http://example.com/product?id=1  -> 显示产品A
http://example.com/product?id=2  -> 显示产品B
http://example.com/product?id=3  -> 显示产品C
```

如果单纯使用URL模式去重，会认为这三个URL模式相同（都是`http://example.com/product?id=`），
可能只爬取第一个就跳过后面的，导致遗漏产品B和C的信息。

### 解决方案

使用**URL模式+DOM相似度**去重：

1. 第一次访问 `id=1`，记录DOM结构
2. 第二次访问 `id=2`，记录DOM结构，与第一次比较
3. 第三次访问 `id=3`，记录DOM结构，与前两次比较
4. 计算三次访问的DOM平均相似度
5. 如果相似度 < 85%（默认阈值），说明内容确实不同，继续爬取
6. 如果相似度 ≥ 85%，说明是相似页面（如列表翻页），后续跳过

## 配置说明

### 在 config.json 中配置

```json
{
  "deduplication_settings": {
    "enable_url_pattern_dom_dedup": true,     // 启用URL模式+DOM去重
    "url_pattern_dom_sample_count": 3,        // 采样次数（默认3次）
    "url_pattern_dom_threshold": 0.85         // DOM相似度阈值（默认85%）
  }
}
```

### 配置项说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enable_url_pattern_dom_dedup` | bool | `true` | 是否启用URL模式+DOM去重 |
| `url_pattern_dom_sample_count` | int | `3` | 采样次数，建议3-5次 |
| `url_pattern_dom_threshold` | float64 | `0.85` | DOM相似度阈值（0-1之间） |

## 工作流程

### 流程图

```
URL进入
  ↓
提取URL模式（忽略参数值）
  ↓
是否是新模式？
  ├─ 是 → 创建新分组，开始采样（允许爬取）
  └─ 否 → 检查分组状态
           ├─ 采样中 → 继续采样（允许爬取）
           └─ 已验证 → 检查相似度
                        ├─ 相似 → 跳过爬取 ⛔
                        └─ 不同 → 允许爬取 ✅
```

### 详细步骤

1. **URL提交前检查**
   - 调用 `ShouldCrawl(url)` 检查是否应该爬取
   - 返回值：(是否爬取, 原因, 是否需要DOM分析)

2. **爬取页面**
   - 如果允许爬取，发起HTTP请求
   - 获取HTML内容

3. **记录DOM签名**
   - 调用 `RecordDOMSignature(url, htmlContent)` 记录DOM结构
   - 提取DOM特征：标签序列、标签分布、结构深度等

4. **验证相似度**
   - 当采样次数达到阈值（如3次）时
   - 自动计算所有采样页面的DOM相似度
   - 两两比较，计算平均值

5. **后续处理**
   - 相似模式：标记为"已验证-相似"，后续URL直接跳过
   - 不同模式：标记为"已验证-不同"，后续URL正常爬取

## DOM相似度算法

### 多维度计算

组合以下多个维度计算相似度：

1. **结构哈希**（快速判断）
   - 基于关键结构特征生成MD5哈希
   - 完全相同返回100%相似度

2. **SimHash相似度**（内容指纹）
   - 基于汉明距离计算
   - 适合大规模文本相似度比较

3. **结构特征相似度**
   - DOM深度
   - 节点总数
   - 链接数量
   - 表单数量
   - 输入框数量

4. **标签分布相似度**（余弦相似度）
   - 计算各种HTML标签的分布向量
   - 使用余弦相似度比较

### 综合评分

最终相似度 = (SimHash相似度 + 结构特征相似度 + 标签分布相似度) / 3

## 使用示例

### 场景1：列表翻页（会被去重）

```
http://example.com/products?page=1  -> DOM相似度: 92%
http://example.com/products?page=2  -> DOM相似度: 94%
http://example.com/products?page=3  -> DOM相似度: 93%

平均相似度: 93% ≥ 85% → 判定为相似页面
结果：page=4、page=5 等后续页面会被跳过
```

### 场景2：不同产品详情（不会被去重）

```
http://example.com/product?id=1  -> 产品A详情
http://example.com/product?id=2  -> 产品B详情
http://example.com/product?id=3  -> 产品C详情

DOM相似度: 65% < 85% → 判定为不同内容
结果：所有产品详情页都会被爬取
```

### 场景3：混合情况

```
http://api.com/v1/users?page=1   -> 用户列表第1页
http://api.com/v1/users?page=2   -> 用户列表第2页  
http://api.com/v1/users?page=3   -> 用户列表第3页
→ 平均相似度 95% → 后续page=4,5...跳过

http://api.com/v1/products?id=1  -> 产品详情A
http://api.com/v1/products?id=2  -> 产品详情B
http://api.com/v1/products?id=3  -> 产品详情C
→ 平均相似度 62% → 后续id=4,5...继续爬取
```

## 报告输出

### 统计报告

爬取完成后会显示详细统计：

```
================================================================================
            URL模式+DOM相似度去重报告
================================================================================

【总体统计】
  处理URL总数:       150
  唯一URL模式:       25
  正在采样的模式:     2
  已验证的模式:       23
    - 相似模式:       8 (34.8%)
    - 不同模式:       15 (65.2%)
  采样的URL数:       69
  跳过的URL数:       81
  去重率:           54.0%

【相似模式详情】（Top 10）
────────────────────────────────────────────────────────────────────────────

1. 模式: http://example.com/list?page=
   平均DOM相似度: 94.3%
   采样URL数: 3
   跳过URL数: 47
   首次URL: http://example.com/list?page=1
   验证时间: 2025-11-05 15:30:25
   采样示例:
     [1] http://example.com/list?page=1
     [2] http://example.com/list?page=2
     [3] http://example.com/list?page=3

2. 模式: http://example.com/search?q=&page=
   平均DOM相似度: 89.7%
   采样URL数: 3
   跳过URL数: 22
   ...

【内容不同的模式】（说明：这些URL模式相似但内容不同，都会保留）
────────────────────────────────────────────────────────────────────────────

1. 模式: http://example.com/product?id=
   平均DOM相似度: 58.2% (低于阈值85.0%，内容确实不同)
   采样URL数: 3
   首次URL: http://example.com/product?id=1

2. 模式: http://example.com/user?id=
   平均DOM相似度: 62.5% (低于阈值85.0%，内容确实不同)
   采样URL数: 3
   首次URL: http://example.com/user?id=100
```

## 性能影响

### 优势
- ✅ **更精准的去重**：避免误判，不会遗漏重要内容
- ✅ **自动学习**：无需手动配置哪些模式应该去重
- ✅ **节省资源**：对于列表翻页等相似页面，显著减少请求

### 开销
- ⚠️ **采样成本**：每个新模式需要3次（可配置）请求验证
- ⚠️ **计算开销**：需要提取和比较DOM结构

### 适用场景
- ✅ 大规模网站爬取（大量列表翻页）
- ✅ 电商网站（产品列表 vs 产品详情）
- ✅ 新闻网站（文章列表 vs 文章详情）
- ❌ 小型网站（URL总数 < 50，开销大于收益）

## 调优建议

### 1. 采样次数 (`url_pattern_dom_sample_count`)

```json
{
  "url_pattern_dom_sample_count": 3  // 默认值
}
```

- **小网站（<100 URLs）**：设置为 `2`，快速验证
- **中型网站（100-1000 URLs）**：保持 `3`（默认）
- **大型网站（>1000 URLs）**：设置为 `5`，更准确判断

### 2. 相似度阈值 (`url_pattern_dom_threshold`)

```json
{
  "url_pattern_dom_threshold": 0.85  // 默认85%
}
```

- **严格模式**：`0.90`（90%）- 只跳过高度相似的页面
- **平衡模式**：`0.85`（85%）- 默认值，适合大多数场景
- **宽松模式**：`0.75`（75%）- 更激进的去重，可能误判

### 3. 禁用功能

如果发现误判或不适合你的场景，可以禁用：

```json
{
  "enable_url_pattern_dom_dedup": false
}
```

## 与其他去重策略的关系

本功能与现有去重策略**协同工作**：

| 去重策略 | 作用时机 | 优先级 |
|---------|---------|-------|
| URL完全匹配 | 已访问URL直接跳过 | 1（最高） |
| **URL模式+DOM** | URL模式相似时验证DOM | **2** |
| 智能参数去重 | 参数值相似度判断 | 3 |
| 业务感知过滤 | 低价值URL过滤 | 4 |
| DOM相似度 | 全局DOM相似度检查 | 5（最低） |

## 实现细节

### 核心文件

- `core/url_pattern_dom_dedup.go` - 主要实现
- `core/dom_similarity.go` - DOM相似度算法
- `core/spider.go` - 集成到爬虫流程
- `config/config.go` - 配置定义

### 关键数据结构

```go
type URLPatternWithDOMDeduplicator struct {
    patternGroups map[string]*PatternGroup  // URL模式分组
    domDetector   *DOMSimilarityDetector    // DOM检测器
    sampleCount   int                       // 采样次数
    domSimilarityThreshold float64          // 相似度阈值
}

type PatternGroup struct {
    Pattern       string              // URL模式
    SampleURLs    []string            // 采样URL
    DOMSignatures []*DOMSignature     // DOM签名
    SampleCount   int                 // 当前采样数
    IsVerified    bool                // 是否已验证
    IsSimilar     bool                // 是否相似
    AvgSimilarity float64             // 平均相似度
    SkippedCount  int                 // 跳过次数
}
```

## FAQ

### Q1: 会不会遗漏重要页面？
A: 不会。只有在**多次验证确认DOM高度相似**后才会跳过。对于内容不同的页面（如产品详情），DOM相似度会很低，会被正常爬取。

### Q2: 采样次数应该设置多少？
A: 默认3次适合大多数场景。如果网站页面变化大，可以设置为5次；如果追求速度，可以设置为2次。

### Q3: 如何判断是否误判？
A: 查看报告中的"内容不同的模式"部分，如果DOM相似度 < 阈值但实际内容相似，说明阈值设置过高，建议降低到0.75-0.80。

### Q4: 对爬取速度有影响吗？
A: 有一定影响。每个新模式需要3次采样请求，但之后可以跳过大量相似页面，总体上节省时间和资源。

### Q5: 可以单独使用这个功能吗？
A: 可以。将其他去重功能禁用，只启用 `enable_url_pattern_dom_dedup: true` 即可。

## 更新日志

### v4.5 (2025-11-05)
- ✨ 新增URL模式+DOM相似度去重功能
- ✨ 采样验证机制，避免误判
- ✨ 详细的去重报告输出
- ✨ 可配置采样次数和相似度阈值

## 技术支持

如有问题或建议，请：
1. 查看日志输出（启用 `-log-level debug`）
2. 查看去重报告
3. 调整配置参数
4. 提交Issue反馈

---

**作者**: GogoSpider Team  
**日期**: 2025年11月5日  
**版本**: v4.5

