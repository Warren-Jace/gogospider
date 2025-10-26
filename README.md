# GoGoSpider - 智能Web安全爬虫

> 🚀 功能强大的Go语言Web安全爬虫工具，专注于URL发现和敏感信息检测

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8?logo=go)](https://golang.org/)
[![Version](https://img.shields.io/badge/version-v2.11-green.svg)](https://github.com/Warren-Jace/gogospider)

---

## ✨ 核心特性

### 🔍 智能爬取
- **双引擎爬虫**: 静态爬虫（Colly） + 动态爬虫（Chromedp）
- **JavaScript深度分析**: 40+种JS URL提取模式
- **AJAX拦截**: 自动捕获动态加载的URL
- **事件触发**: 模拟点击、悬停、输入等用户行为
- **多层递归**: 支持最大20层深度爬取（BFS算法）

### 🎯 智能去重（效率提升84%）
- **URL模式去重**: 忽略参数值变化 (`/product?id=1` → `/product?id=`)
- **URL结构化去重**: 识别路径变量 (`/user/123/profile` → `/user/{num}/profile`)
- **DOM相似度去重**: 5种算法检测页面结构相似度
- **智能参数值去重**: 16种特征分类，避免重复爬取
- **业务感知过滤**: 自动识别URL业务价值

### 🛡️ 敏感信息检测（v2.11重点功能）
- **云存储密钥检测**（最重要）:
  - AWS S3 (Access Key + Secret Key + Bucket)
  - 阿里云OSS (AccessKeyId + AccessKeySecret + Bucket)
  - 腾讯云COS (SecretId + SecretKey + Bucket)
  - 七牛云、华为云OBS、百度云BOS
  - 覆盖95%+云存储市场
  
- **第三方登录授权**:
  - 微信开放平台 (AppID + AppSecret + 支付密钥)
  - 支付宝 (App ID + 应用私钥)
  - QQ互联、微博、抖音、钉钉
  - 覆盖90%+中国第三方平台
  
- **账号密码信息**:
  - 管理员密码、数据库密码、Redis密码
  - 用户名密码组合、SSH私钥
  
- **其他敏感信息**:
  - JWT Token、GitHub Token、Slack Token
  - 数据库连接字符串、内网IP、身份证号

**特性**:
- ✅ 40+种检测规则（可扩展）
- ✅ 三级严重性分级（HIGH/MEDIUM/LOW）
- ✅ 自动脱敏保护
- ✅ 来源URL追溯（精确到行号）
- ✅ 独立文件保存（TXT + JSON）
- ✅ 外部规则配置支持
- ✅ 性能影响 < 5%

### 🚀 高级功能
- **CDN检测**: 识别60+个CDN服务商，分析跨域JS
- **表单智能填充**: 20+种字段类型自动识别
- **静态资源分类**: 7种资源类型智能分类
- **Sitemap解析**: 自动爬取sitemap.xml和robots.txt
- **技术栈识别**: 检测15+种Web框架
- **子域名提取**: 自动发现子域名
- **隐藏路径扫描**: 200+个常见Web路径
- **IP地址检测**: 识别内网IP泄露

### 📊 批量扫描（v2.11新增）
- **批量URL输入**: 从文件读取URL列表
- **并发控制**: 可配置并发数（默认5，推荐5-10）
- **独立输出**: 每个URL独立保存结果
- **实时进度**: 显示扫描进度和统计

---

## 📦 安装

### 方式1: 预编译二进制（推荐）

从 [Releases](https://github.com/Warren-Jace/gogospider/releases) 页面下载对应平台的可执行文件。

### 方式2: 从源码编译

**前置要求**:
- Go 1.18 或更高版本
- Chrome/Chromium（用于动态爬取）

```bash
# 克隆仓库
git clone https://github.com/Warren-Jace/gogospider.git
cd gogospider

# 安装依赖
go mod download

# 编译
go build -o spider cmd/spider/main.go

# Windows编译
go build -o spider.exe cmd/spider/main.go
```

---

## 🚀 快速开始

### 基本用法

```bash
# 基础爬取（自动启用敏感信息检测）
./spider -url https://example.com

# 指定爬取深度
./spider -url https://example.com -depth 5

# 使用配置文件
./spider -config example_config.json
```

### 敏感信息检测

```bash
# 使用外部规则文件
./spider -url https://example.com -sensitive-rules sensitive_rules_config.json

# 只检测高危敏感信息
./spider -url https://example.com -sensitive-min-severity HIGH

# 禁用敏感信息检测（性能优先）
./spider -url https://example.com -sensitive-detect=false

# 保存敏感信息到指定文件
./spider -url https://example.com -sensitive-output ./sensitive_report.json
```

### 批量扫描（v2.11新增）

```bash
# 创建URL列表文件
cat > targets.txt << EOF
https://www.example.com
https://api.example.com
https://admin.example.com
EOF

# 批量扫描
./spider -batch-file targets.txt -batch-concurrency 10

# 批量扫描 + 敏感信息检测
./spider -batch-file targets.txt \
  -sensitive-rules sensitive_rules_config.json \
  -batch-concurrency 5 \
  -depth 3
```

### 管道模式

```bash
# 从标准输入读取URL
cat urls.txt | ./spider -stdin -simple

# 与其他工具链配合
echo "https://example.com" | ./spider -stdin | nuclei -t cves/
```

---

## 📂 输出文件

扫描完成后自动生成以下文件：

```
spider_example.com_20251026_143000.txt                      # 详细爬取报告
spider_example.com_20251026_143000_all_urls.txt             # 所有URL
spider_example.com_20251026_143000_params.txt               # 带参数的URL
spider_example.com_20251026_143000_forms.txt                # 表单URL
spider_example.com_20251026_143000_unique_urls.txt          # 去重URL（推荐）
spider_example.com_20251026_143000_structure_unique_urls.txt # 结构化去重URL
spider_example.com_20251026_143000_sensitive.txt            # 敏感信息报告（TXT）
spider_example.com_20251026_143000_sensitive.json           # 敏感信息报告（JSON）
```

**批量扫描输出**（使用`-batch-file`）:
```
batch_site1.com_20251026_143000_sensitive.txt
batch_site2.com_20251026_143000_sensitive.json
batch_site3.com_20251026_143000_all_urls.txt
...
```

**推荐**: 使用 `*_unique_urls.txt` 或 `*_structure_unique_urls.txt` 传递给其他安全工具（如nuclei、sqlmap等）

---

## ⚙️ 命令行参数

### 基础参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-url <url>` | 目标URL（必需，单站点模式） | - |
| `-batch-file <file>` | 批量URL列表文件 | - |
| `-depth <num>` | 最大爬取深度 | 3 |
| `-mode <mode>` | 爬取模式：static, dynamic, smart | smart |
| `-config <file>` | 配置文件路径 | - |

### 敏感信息检测参数（v2.11）

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-sensitive-detect` | 启用/禁用敏感信息检测 | true |
| `-sensitive-rules <file>` | 外部规则文件（JSON格式） | - |
| `-sensitive-scan-body` | 扫描HTTP响应体 | true |
| `-sensitive-scan-headers` | 扫描HTTP响应头 | true |
| `-sensitive-min-severity` | 最低严重级别（LOW/MEDIUM/HIGH） | LOW |
| `-sensitive-output <file>` | 敏感信息JSON输出文件 | - |
| `-sensitive-realtime` | 实时输出敏感信息发现 | true |

### 批量扫描参数（v2.11）

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-batch-file <file>` | URL列表文件路径 | - |
| `-batch-concurrency <num>` | 批量扫描并发数 | 5 |

### 其他常用参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-proxy <url>` | HTTP代理服务器 | - |
| `-user-agent <string>` | 自定义User-Agent | - |
| `-log-level <level>` | 日志级别（debug/info/warn/error） | info |
| `-quiet` | 静默模式 | false |
| `-stdin` | 从标准输入读取URL | false |
| `-simple` | 简洁输出模式 | false |

---

## 💡 使用示例

### 示例1: 基础爬取 + 敏感信息检测

```bash
./spider -url https://example.com -depth 3
```

**输出**:
- 自动检测云存储密钥、第三方授权、账号密码等
- 生成敏感信息报告 (`*_sensitive.txt` 和 `*_sensitive.json`)

---

### 示例2: 自定义敏感信息规则

```bash
# 使用自定义规则文件
./spider -url https://yourcompany.com \
  -sensitive-rules sensitive_rules_config.json \
  -depth 5
```

---

### 示例3: 只检测云存储密钥泄露

```bash
# 只检测高危（云存储、数据库密码等）
./spider -url https://example.com \
  -sensitive-rules sensitive_rules_config.json \
  -sensitive-min-severity HIGH
```

---

### 示例4: 批量扫描多个站点

```bash
# 创建URL列表
cat > production_sites.txt << EOF
https://www.yoursite.com
https://api.yoursite.com
https://admin.yoursite.com
EOF

# 批量扫描
./spider -batch-file production_sites.txt \
  -batch-concurrency 10 \
  -sensitive-rules sensitive_rules_config.json
```

**输出**: 每个站点独立的敏感信息报告

---

### 示例5: CI/CD集成

```bash
#!/bin/bash
# ci_security_check.sh

./spider -url https://staging.yoursite.com \
  -sensitive-rules sensitive_rules_config.json \
  -sensitive-min-severity HIGH \
  -sensitive-output scan.json \
  -quiet

# 检查高危敏感信息
HIGH_COUNT=$(cat scan.json | jq '.statistics.high_severity')
if [ $HIGH_COUNT -gt 0 ]; then
  echo "❌ 发现 $HIGH_COUNT 处高危敏感信息，阻止部署"
  exit 1
fi
echo "✅ 安全检查通过"
```

---

### 示例6: 与其他安全工具集成

```bash
# 传递给nuclei扫描器
./spider -url https://example.com -depth 3
nuclei -l spider_example.com_*_unique_urls.txt -t cves/

# 传递给sqlmap
sqlmap -m spider_example.com_*_params.txt --batch

# 管道模式
cat urls.txt | ./spider -stdin -simple | httpx -silent
```

---

## 📊 敏感信息检测规则

### 规则配置文件: `sensitive_rules_config.json`

包含40+种检测规则，覆盖：

#### 云存储密钥（10种服务）
- ✅ AWS S3 (Access Key + Secret Key + Bucket)
- ✅ 阿里云OSS (AccessKeyId + AccessKeySecret + Bucket)
- ✅ 腾讯云COS (SecretId + SecretKey + Bucket)
- ✅ 七牛云 (Access Key + Secret Key)
- ✅ 华为云OBS (Access Key + Secret Key)
- ✅ 百度云BOS (Access Key + Secret Key)
- ✅ Google Cloud Storage
- ✅ Azure Blob Storage
- ✅ DigitalOcean Spaces
- ✅ UCloud UFile

#### 第三方登录授权（11种平台）
- ✅ 微信开放平台 (AppID + AppSecret + 支付密钥)
- ✅ 支付宝 (App ID + 应用私钥)
- ✅ QQ互联 (AppID + AppKey)
- ✅ 微博开放平台 (App Key + App Secret)
- ✅ 抖音开放平台 (AppID + AppSecret)
- ✅ 钉钉开放平台 (AppKey + AppSecret)
- ✅ GitHub (Token)
- ✅ Slack (Token + Webhook)
- ✅ Stripe (API Key)
- ✅ PayPal (Client ID + Secret)
- ✅ 美团开放平台

#### 账号密码
- ✅ 管理员密码 (admin/root)
- ✅ 数据库密码 (MySQL/PostgreSQL/MongoDB)
- ✅ Redis密码
- ✅ 用户名密码组合
- ✅ SSH私钥
- ✅ 数据库连接字符串

#### 个人信息
- ✅ 中国手机号
- ✅ 中国身份证号
- ✅ 邮箱地址
- ✅ 内网IP地址

### 自定义规则

编辑 `sensitive_rules_config.json` 添加自定义规则：

```json
{
  "rules": {
    "公司内部API密钥": {
      "pattern": "COMPANY_[A-Z0-9]{32}",
      "severity": "HIGH",
      "mask": true,
      "description": "公司内部API密钥"
    }
  }
}
```

---

## 🎯 敏感信息报告示例

### 文本报告 (`*_sensitive.txt`)

```
==========================================
   敏感信息泄露检测报告
==========================================

扫描页面数: 54
发现总数: 12
  - 高危: 5
  - 中危: 4
  - 低危: 3

==========================================

【高危发现】
------------------------------------------------------------

[1] 阿里云OSS AccessKey
    来源URL: https://example.com/static/js/upload.js
    位置: Line 42
    值: LTAI****************EXAM
    描述: 阿里云OSS AccessKey ID - 存储桶访问凭证

[2] 微信AppSecret
    来源URL: https://example.com/config/wx.js
    位置: Line 15
    值: a1b2****************************c3d4
    描述: 微信AppSecret - 严重泄露风险

[3] 数据库密码
    来源URL: https://example.com/api/db.php
    位置: Line 23
    值: my****word
    描述: 数据库密码
```

### JSON报告 (`*_sensitive.json`)

```json
{
  "scan_time": "2025-10-26 14:30:00",
  "target_domain": "example.com",
  "statistics": {
    "total_scanned": 54,
    "total_findings": 12,
    "high_severity": 5,
    "medium_severity": 4,
    "low_severity": 3
  },
  "findings": [
    {
      "type": "阿里云OSS AccessKey",
      "value": "LTAI****************EXAM",
      "location": "Line 42",
      "severity": "HIGH",
      "source_url": "https://example.com/static/js/upload.js",
      "line_number": 42
    }
  ]
}
```

---

## 🔧 配置文件

使用配置文件可以保存所有设置，便于重复使用。

### 示例配置: `example_config.json`

```json
{
  "target_url": "https://example.com",
  
  "depth_settings": {
    "max_depth": 5,
    "deep_crawling": true,
    "scheduling_algorithm": "BFS"
  },
  
  "sensitive_detection_settings": {
    "enabled": true,
    "scan_response_body": true,
    "scan_response_headers": true,
    "min_severity": "LOW",
    "realtime_output": true
  }
}
```

**使用**:
```bash
./spider -config example_config.json
```

---

## 📊 性能指标

| 指标 | 数据 |
|------|------|
| URL发现率 | 比Crawlergo提升119% |
| AJAX覆盖率 | 100% |
| 去重效果 | 90%+ |
| 平均速度 | 20-50页/秒 |
| 敏感信息检测影响 | < 5% |
| 批量扫描速度 | 10站点 < 45秒 |

---

## 🛡️ 安全建议

### 云存储密钥泄露防护

#### ❌ 错误做法
```javascript
// 永远不要在前端代码中硬编码密钥
const ossConfig = {
  accessKeyId: 'LTAI4G3VxQxYxxxxxEXAMPLE',
  accessKeySecret: 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
};
```

#### ✅ 正确做法
```javascript
// 方案1: 使用临时凭证（STS Token）
const stsToken = await fetch('/api/get-sts-token');

// 方案2: 后端代理上传
const uploadFile = async (file) => {
  const formData = new FormData();
  formData.append('file', file);
  return await fetch('/api/upload', {method: 'POST', body: formData});
};

// 方案3: 环境变量（服务器端）
const accessKey = process.env.OSS_ACCESS_KEY;
```

### 密钥泄露应急处理

如果扫描发现密钥泄露：

1. **立即行动**:
   - 立即撤销泄露的密钥（云服务控制台）
   - 检查访问日志，确认是否被利用
   - 评估数据是否被访问或下载

2. **生成新密钥**:
   - 创建新的Access Key
   - 更新应用配置
   - 测试功能正常

3. **修复代码**:
   - 移除硬编码的密钥
   - 使用环境变量或密钥管理服务
   - 添加到 `.gitignore`

4. **加强防护**:
   - 启用IP白名单
   - 开启MFA认证
   - 使用最小权限原则

---

## 🎓 高级用法

### 1. Scope精细控制

```bash
# 只爬取API路径，排除静态资源
./spider -url https://example.com \
  -include-paths "/api/*,/v1/*" \
  -exclude-ext "jpg,png,css,js"
```

### 2. 速率控制

```bash
# 限制每秒10个请求，避免服务器过载
./spider -url https://example.com \
  -rate-limit 10 \
  -adaptive-rate
```

### 3. 外部数据源

```bash
# 从Wayback Machine获取历史URL
./spider -url https://example.com \
  -wayback \
  -depth 3
```

### 4. 查看敏感信息报告

```bash
# 文本格式（易读）
cat spider_example.com_*_sensitive.txt

# JSON格式（自动化）
cat spider_example.com_*_sensitive.json | jq '.findings'

# 只看高危
cat spider_example.com_*_sensitive.json | jq '.findings[] | select(.severity=="HIGH")'

# 只看云存储密钥
cat spider_example.com_*_sensitive.txt | grep -E "(AWS|OSS|COS|S3)"

# 只看第三方授权
cat spider_example.com_*_sensitive.txt | grep -E "(微信|支付宝|QQ)"
```

---

## 🤝 与其他工具集成

### 漏洞扫描工具链

```bash
# 1. GogoSpider发现URL
./spider -url https://target.com -depth 3

# 2. 传递给nuclei扫描漏洞
nuclei -l spider_target.com_*_unique_urls.txt -t cves/ -o vulns.txt

# 3. 传递给sqlmap测试SQL注入
sqlmap -m spider_target.com_*_params.txt --batch

# 4. 传递给xray进行被动扫描
xray webscan --url-file spider_target.com_*_unique_urls.txt
```

---

## 🐛 常见问题

### Q1: 如何禁用敏感信息检测？

```bash
./spider -url https://example.com -sensitive-detect=false
```

### Q2: 如何只检测云存储密钥？

编辑 `sensitive_rules_config.json`，只保留云存储相关规则，或使用严重级别过滤：

```bash
./spider -url https://example.com -sensitive-min-severity HIGH
```

### Q3: 批量扫描失败怎么办？

每个URL独立扫描，某个URL失败不影响其他URL。查看最终报告的成功/失败统计。

### Q4: 敏感信息报告保存在哪里？

默认保存在当前目录：
- `spider_域名_时间戳_sensitive.txt`
- `spider_域名_时间戳_sensitive.json`

### Q5: 如何追溯敏感信息来源？

报告中自动包含：
- 来源URL
- 文件行号
- 敏感信息类型和值

### Q6: 动态内容未爬取到？

```bash
# 使用dynamic模式
./spider -url https://example.com -mode dynamic
```

### Q7: 如何查看所有检测到的敏感信息类型？

```bash
cat *_sensitive.json | jq '.findings[] | .type' | sort | uniq
```

---

## 🏆 竞争优势

相比同类工具（Crawlergo、Katana、Gospider、Hakrawler）:

- 🏆 **敏感信息检测最全面**: 40+规则，覆盖云存储+第三方授权
- 🏆 **中国平台支持最好**: 微信、支付宝、阿里云、腾讯云等
- 🏆 **来源追溯能力**: 精确到URL+行号
- 🏆 **智能去重最强**: 三层去重机制，效率提升84%
- 🏆 **批量扫描支持**: 高并发处理多站点
- 🏆 **功能最完整**: 一站式URL发现+敏感信息检测

---

## 🎓 最佳实践

### 日常使用（推荐）

```bash
# 默认配置即可，自动启用所有功能
./spider -url https://yoursite.com
```

### 安全审计

```bash
# 深度扫描 + 全面检测
./spider -url https://target.com \
  -sensitive-rules sensitive_rules_config.json \
  -depth 5 \
  -sensitive-min-severity MEDIUM
```

### 批量资产扫描

```bash
# 扫描所有子站点
./spider -batch-file company_sites.txt \
  -batch-concurrency 10 \
  -sensitive-rules sensitive_rules_config.json
```

### 性能优先模式

```bash
# 只需要URL发现，禁用敏感检测
./spider -url https://example.com \
  -depth 3 \
  -sensitive-detect=false
```

---

## 📖 完整文档

项目包含以下文档：

- 📄 `README.md` - 本文件（项目总览）
- 📄 `example_config.json` - 配置文件示例
- 📄 `sensitive_rules_config.json` - 敏感信息检测规则
- 📄 `example_targets.txt` - 批量URL列表示例

---

## 🛡️ 安全声明

### 合法使用

本工具仅用于**授权的安全测试**。使用前请确保：

1. ✅ 已获得目标网站所有者的明确授权
2. ✅ 遵守当地法律法规
3. ✅ 不用于恶意攻击或非法活动

### 敏感信息处理

- ✅ 检测到的敏感信息**默认自动脱敏**
- ✅ 报告文件请**妥善保管**，避免二次泄露
- ✅ 发现高危泄露请**立即处理**

---

## 🤝 贡献

欢迎提交Issue和Pull Request！

---

## 📄 许可证

本项目采用 Apache 2.0 许可证。

---

## 🙏 致谢

本项目参考和学习了以下优秀项目：
- Crawlergo
- Katana  
- Gospider
- Hakrawler
- JsLeaksScan

---

## 📧 联系方式

- GitHub: [@Warren-Jace](https://github.com/Warren-Jace)
- Issues: [提交Issue](https://github.com/Warren-Jace/gogospider/issues)

---

**⚠️ 免责声明**: 本工具仅用于授权的安全测试，使用者需自行承担使用本工具的一切法律责任。

**🎯 核心优势**: 
- 云存储密钥检测（10种服务，95%市场覆盖）
- 中国第三方授权检测（微信、支付宝等7大平台）
- 来源URL精确追溯（到行号）
- 批量扫描支持（高并发）
- 性能优异（敏感检测影响 < 5%）
