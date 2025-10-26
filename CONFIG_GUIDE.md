# GogoSpider 配置指南 v3.0

## 📋 目录

- [概述](#概述)
- [优先级说明](#优先级说明)
- [命令行参数](#命令行参数)
- [配置文件详解](#配置文件详解)
- [预设场景](#预设场景)
- [敏感信息检测](#敏感信息检测)
- [黑名单配置](#黑名单配置)
- [批量扫描](#批量扫描)
- [常见问题](#常见问题)

---

## 概述

GogoSpider v3.0 采用了全新的配置架构：
- **命令行参数**：仅保留 15 个最常用参数，简洁易用
- **配置文件**：所有细节配置放在 JSON 文件中，方便管理
- **预设场景**：提供 5 种常用场景的配置模板

---

## 优先级说明

配置项的优先级（从高到低）：

```
命令行参数 > 配置文件 > 默认值
```

### 示例

```bash
# 命令行指定深度为 3，配置文件为 5，最终使用 3
./spider -url https://example.com -depth 3 -config config.json
```

### 作用域优先级

Scope 控制的过滤优先级（从高到低）：

```
exclude_regex > exclude_domains > exclude_paths > 
include_regex > include_domains > include_paths
```

这意味着：
1. 如果 URL 匹配 `exclude_regex`，直接拒绝
2. 如果 URL 在 `exclude_domains` 中，直接拒绝
3. 通过排除检查后，再检查包含规则

---

## 命令行参数

### 核心参数（必需）

| 参数 | 说明 | 示例 |
|------|------|------|
| `-url` | 目标 URL（必需） | `-url https://example.com` |

### 基础参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-config` | - | 配置文件路径 |
| `-depth` | 3 | 最大爬取深度 |
| `-max-pages` | 100 | 最大页面数 |
| `-workers` | 10 | 并发数 |

### 模式参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-mode` | smart | 爬取模式：static/dynamic/smart |
| `-preset` | - | 使用预设场景配置 |

### 输出参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-output` | ./ | 输出目录 |
| `-json` | false | 启用 JSON 输出 |
| `-quiet` | false | 静默模式 |

### 高级参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-proxy` | - | 代理服务器 |
| `-allow-subdomains` | false | 允许子域名 |
| `-batch-file` | - | 批量扫描文件 |

### 工具参数

| 参数 | 说明 |
|------|------|
| `-version` | 显示版本信息 |
| `-help` | 显示帮助信息 |

---

## 配置文件详解

### 完整配置文件

使用 `example_config_optimized.json` 作为模板。

### 黑名单配置

```json
"blacklist_settings": {
  "enabled": true,
  "domains": [
    "*.gov.cn",      // 政府网站
    "*.edu.cn",      // 教育网站
    "*.mil.cn",      // 军事网站
    "*.bank.com"     // 银行网站
  ],
  "domain_patterns": [
    "*bank*",        // 包含 bank 的域名
    "*payment*",     // 包含 payment 的域名
    "*admin.gov*"    // 政府管理域名
  ],
  "strict_mode": true
}
```

**说明：**
- `enabled`: 是否启用黑名单
- `domains`: 精确域名匹配（支持通配符 `*`）
- `domain_patterns`: 模糊匹配模式
- `strict_mode`: 
  - `true`: 匹配到黑名单直接拒绝
  - `false`: 匹配到只记录警告但继续爬取

### 批量扫描配置

```json
"batch_scan_settings": {
  "enabled": true,
  "input_file": "targets.txt",
  "concurrency": 5,
  "output_dir": "./batch_results",
  "per_target_timeout": 3600,
  "continue_on_error": true,
  "save_individual_reports": true,
  "save_summary_report": true
}
```

**或通过命令行：**

```bash
./spider -batch-file targets.txt -config config.json
```

**targets.txt 格式：**
```
https://example1.com
https://example2.com
https://example3.com
```

### 作用域配置

```json
"scope_settings": {
  "enabled": true,
  
  // 包含规则
  "include_domains": ["*.example.com"],
  "include_paths": ["/api/*", "/admin/*"],
  "include_regex": "",
  
  // 排除规则（优先级更高）
  "exclude_domains": ["cdn.example.com"],
  "exclude_paths": ["/*.jpg", "/*.png"],
  "exclude_regex": "\\.(jpg|png|css)$",
  
  // 其他限制
  "allow_subdomains": true,
  "stay_in_domain": true,
  "max_url_length": 2048,
  "max_params": 20
}
```

**优先级规则：**
1. 首先检查 `exclude_regex`
2. 然后检查 `exclude_domains`
3. 然后检查 `exclude_paths`
4. 最后检查 `include_*` 规则

**命令行参数与配置文件的关系：**
- 命令行 `-url` 指定的域名会自动添加到 `include_domains`
- 命令行 `-allow-subdomains` 会覆盖配置文件的 `allow_subdomains`

### 敏感信息检测配置

```json
"sensitive_detection_settings": {
  "enabled": true,
  "scan_response_body": true,
  "scan_response_headers": true,
  "min_severity": "LOW",
  "rules_file": "./sensitive_rules_config.json",
  "output_file": "",
  "realtime_output": true
}
```

**规则文件路径：**
- 支持相对路径：`./sensitive_rules_config.json`
- 支持绝对路径：`/path/to/rules.json`
- 默认使用：`./sensitive_rules_config.json`

**如何自定义规则：**
1. 复制 `sensitive_rules_config.json`
2. 修改或添加规则
3. 在配置文件中指定 `rules_file` 路径

---

## 预设场景

我们提供了 5 种常用场景的预设配置：

### 1. 快速扫描 (quick_scan)

**适用场景：** 初步侦查、快速测试、时间紧急

```bash
./spider -url https://example.com -preset quick_scan
```

**特点：**
- 深度：3 层
- 最大页面：200
- 只启用静态爬虫
- 较高相似度阈值（90%）
- 只检测中高危敏感信息

### 2. 深度扫描 (deep_scan)

**适用场景：** 安全测试、全面审计、API发现、漏洞挖掘

```bash
./spider -url https://example.com -preset deep_scan
```

**特点：**
- 深度：8 层
- 最大页面：5000
- 启用所有爬虫功能
- 包含外部数据源（Wayback）
- 检测所有级别敏感信息

### 3. API 发现 (api_discovery)

**适用场景：** API测试、接口文档生成、后端接口发现

```bash
./spider -url https://example.com -preset api_discovery
```

**特点：**
- 只关注 API 路径（/api/*, /v1/*, etc.）
- 排除静态资源
- 高业务价值过滤
- 适合生成 OpenAPI 文档

### 4. 批量扫描 (batch_scan)

**适用场景：** 多目标扫描、资产发现、批量测试

```bash
./spider -batch-file targets.txt -preset batch_scan
```

**特点：**
- 并发 5 个目标
- 中等深度和页面限制
- 自动保存每个目标的报告
- 生成汇总报告

### 5. 隐蔽扫描 (stealth_scan)

**适用场景：** 敏感目标、需要隐蔽、避免触发 WAF/IDS

```bash
./spider -url https://example.com -preset stealth_scan
```

**特点：**
- 低速率：5 req/s
- 随机延迟：1-3 秒
- 多个 User-Agent 轮换
- 不启用动态爬虫（避免 Chrome 特征）

---

## 敏感信息检测

### 检测规则

默认规则文件：`sensitive_rules_config.json`

**检测类别：**

1. **云存储密钥（HIGH）**
   - AWS S3 Access Key
   - 阿里云 OSS
   - 腾讯云 COS
   - 七牛云、华为云、百度云等

2. **第三方登录授权（HIGH）**
   - 微信 AppID/AppSecret
   - 支付宝 AppID/私钥
   - QQ、微博、抖音、钉钉等

3. **账号密码（HIGH）**
   - 管理员密码
   - 数据库密码
   - Redis 密码
   - 用户名密码组合

4. **数据库连接（HIGH）**
   - MySQL/PostgreSQL/MongoDB 连接串

5. **密钥和 Token（HIGH/MEDIUM）**
   - SSH 私钥
   - JWT Token
   - GitHub Token
   - Slack Token

6. **个人信息（LOW/MEDIUM）**
   - 中国手机号
   - 身份证号
   - 邮箱地址
   - 内网 IP

### 自定义规则

在配置文件中添加自定义规则：

```json
"sensitive_detection_settings": {
  "enable_custom_patterns": true,
  "custom_patterns": [
    {
      "name": "自定义API密钥",
      "pattern": "myapi_key_[a-zA-Z0-9]{32}",
      "severity": "HIGH",
      "mask": true
    }
  ]
}
```

### 输出格式

**控制台实时输出：**
```
[敏感信息] HIGH - AWS S3 Access Key
  URL: https://example.com/config.js
  位置: Response Body
  值: AKIA****************XXXX (已脱敏)
```

**JSON 报告：**
```json
{
  "scan_time": "2025-01-01 12:00:00",
  "target_domain": "example.com",
  "statistics": {
    "total_findings": 15,
    "high_severity": 8,
    "medium_severity": 5,
    "low_severity": 2
  },
  "findings": [...]
}
```

---

## 黑名单配置

### 为什么需要黑名单？

防止误爬以下类型的网站：
- 政府网站（*.gov.cn, *.gov）
- 教育机构（*.edu.cn, *.edu）
- 军事网站（*.mil.cn, *.mil）
- 金融机构（*bank*, *payment*）
- 司法机构（*police*, *court*）

### 配置方式

**方法 1：配置文件**

```json
"blacklist_settings": {
  "enabled": true,
  "domains": [
    "*.gov.cn",
    "*.edu.cn",
    "example-blocked.com"
  ],
  "domain_patterns": [
    "*bank*",
    "*payment*"
  ],
  "strict_mode": true
}
```

**方法 2：扩展默认黑名单**

所有预设场景都包含基础黑名单，你可以在此基础上添加：

```json
"domains": [
  "*.gov.cn",      // 默认
  "*.edu.cn",      // 默认
  "*.mil.cn",      // 默认
  "mycompany.com"  // 你添加的
]
```

### 匹配规则

**精确匹配：**
```json
"domains": ["example.com"]  // 只匹配 example.com
```

**通配符匹配：**
```json
"domains": ["*.example.com"]  // 匹配 api.example.com, www.example.com
```

**模糊匹配：**
```json
"domain_patterns": ["*bank*"]  // 匹配 mybank.com, bank-api.com
```

### 严格模式

- **strict_mode = true**：匹配到直接拒绝，不会爬取
- **strict_mode = false**：匹配到记录警告，但继续爬取

---

## 批量扫描

### 使用方法

**1. 准备目标文件 (targets.txt)：**
```
https://example1.com
https://example2.com
https://example3.com
```

**2. 运行批量扫描：**

```bash
# 方法1：使用预设配置
./spider -batch-file targets.txt -preset batch_scan

# 方法2：使用自定义配置
./spider -batch-file targets.txt -config my_batch_config.json

# 方法3：命令行指定并发数
./spider -batch-file targets.txt -batch-concurrency 10 -config config.json
```

### 输出结构

```
batch_results/
├── example1_com/
│   ├── urls.jsonl
│   ├── sensitive_info.json
│   └── report.json
├── example2_com/
│   ├── urls.jsonl
│   ├── sensitive_info.json
│   └── report.json
└── summary.json
```

### 批量扫描配置

```json
"batch_scan_settings": {
  "concurrency": 5,              // 同时扫描5个目标
  "per_target_timeout": 3600,    // 每个目标最多1小时
  "continue_on_error": true,     // 某个失败不影响其他
  "save_individual_reports": true,  // 每个目标单独报告
  "save_summary_report": true    // 生成汇总报告
}
```

### 汇总报告示例

```json
{
  "total_targets": 10,
  "successful": 8,
  "failed": 2,
  "total_urls": 5432,
  "total_sensitive_findings": 23,
  "targets": [
    {
      "url": "https://example1.com",
      "status": "success",
      "urls_found": 1234,
      "sensitive_findings": 5,
      "duration": 456
    }
  ]
}
```

---

## 常见问题

### 1. 命令行参数和配置文件都指定了，哪个生效？

**答：** 命令行参数优先级更高。

```bash
# 配置文件中 depth=5，但命令行指定 depth=3，最终使用 3
./spider -url https://example.com -depth 3 -config config.json
```

### 2. 如何指定敏感信息规则文件？

**答：** 在配置文件中指定 `rules_file`：

```json
"sensitive_detection_settings": {
  "rules_file": "./my_custom_rules.json"
}
```

### 3. 批量扫描时，每个目标使用相同配置吗？

**答：** 是的。批量扫描时：
- 所有目标使用相同的配置文件
- 可以通过命令行参数覆盖部分配置
- 每个目标的结果单独保存

### 4. 作用域限制的优先级是什么？

**答：** 
1. exclude_regex（最高）
2. exclude_domains
3. exclude_paths
4. include_regex
5. include_domains
6. include_paths（最低）

**排除规则优先于包含规则。**

### 5. 黑名单和作用域限制的关系？

**答：** 
- **黑名单** 优先级最高，会在最早阶段拦截
- **作用域限制** 在黑名单检查之后执行
- 执行顺序：黑名单 -> 作用域限制 -> 去重 -> 爬取

### 6. 如何禁用某个功能？

**答：** 在配置文件中设置 `enabled: false`：

```json
"sensitive_detection_settings": {
  "enabled": false  // 禁用敏感信息检测
},
"blacklist_settings": {
  "enabled": false  // 禁用黑名单
}
```

### 7. 预设场景可以修改吗？

**答：** 可以。两种方式：
1. 直接修改 `config_presets/` 目录下的文件
2. 使用预设作为基础，通过命令行覆盖部分参数

```bash
# 使用 quick_scan 但修改深度
./spider -url https://example.com -preset quick_scan -depth 5
```

### 8. 如何查看当前使用的配置？

**答：** 使用 debug 日志级别：

```bash
./spider -url https://example.com -config config.json -log-level debug
```

程序会输出合并后的最终配置。

---

## 快速开始示例

### 示例 1：基础扫描

```bash
./spider -url https://example.com
```

### 示例 2：使用预设配置

```bash
./spider -url https://example.com -preset deep_scan
```

### 示例 3：使用自定义配置

```bash
./spider -url https://example.com -config my_config.json
```

### 示例 4：批量扫描

```bash
./spider -batch-file targets.txt -preset batch_scan
```

### 示例 5：隐蔽扫描 + 代理

```bash
./spider -url https://example.com -preset stealth_scan -proxy http://127.0.0.1:8080
```

---

## 更多帮助

- 查看所有命令行参数：`./spider -help`
- 查看版本信息：`./spider -version`
- 查看示例配置：`cat example_config_optimized.json`
- 查看预设配置：`ls config_presets/`

---

**文档版本：** v3.0  
**最后更新：** 2025-10-26

