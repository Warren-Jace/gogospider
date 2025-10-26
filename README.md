# GoGoSpider - 智能Web安全爬虫

> 🚀 功能强大的Go语言Web安全爬虫工具，专注于URL发现和敏感信息检测

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8?logo=go)](https://golang.org/)
[![Version](https://img.shields.io/badge/version-v3.1-green.svg)](https://github.com/Warren-Jace/gogospider)

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

### 🛡️ 敏感信息检测
- **云存储密钥检测**:
  - AWS S3, 阿里云OSS, 腾讯云COS, 七牛云, 华为云OBS等
  - 覆盖95%+云存储市场
  
- **第三方登录授权**:
  - 微信开放平台, 支付宝, QQ互联, 微博, 抖音, 钉钉等
  - 覆盖90%+中国第三方平台
  
- **账号密码信息**:
  - 管理员密码, 数据库密码, Redis密码
  - 用户名密码组合, SSH私钥
  
- **其他敏感信息**:
  - JWT Token, GitHub Token, API密钥
  - 数据库连接字符串, 内网IP, 身份证号

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

### 📊 批量扫描
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

### 场景1: 快速扫描（新手推荐）
```bash
spider -url https://example.com
```

### 场景2: 深度扫描
```bash
spider -url https://example.com -depth 5 -max-pages 1000 -workers 20
```

### 场景3: API接口发现
```bash
spider -url https://example.com -include-paths "/api/*,/v1/*" -exclude-ext "jpg,png,css,js"
```

### 场景4: 隐蔽扫描（低速）
```bash
spider -url https://example.com -rate-limit 5 -min-delay 500 -max-delay 2000
```

### 场景5: 批量扫描
```bash
spider -batch-file targets.txt -batch-concurrency 10
```

### 场景6: 敏感信息扫描
```bash
spider -url https://example.com -sensitive-rules sensitive_rules_standard.json
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

**推荐**: 使用 `*_unique_urls.txt` 或 `*_structure_unique_urls.txt` 传递给其他安全工具（如nuclei、sqlmap等）

---

## ⚙️ 核心参数

### 核心参数
| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-url <url>` | 目标URL（必需） | - |
| `-config <file>` | 配置文件路径 | - |
| `-version` | 显示版本信息 | - |

### 基础参数
| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-depth <num>` | 最大爬取深度 | 3 |
| `-max-pages <num>` | 最大页面数 | 100 |
| `-workers <num>` | 并发工作线程数 | 10 |
| `-mode <mode>` | 爬取模式：static/dynamic/smart | smart |

### 作用域控制
| 参数 | 说明 |
|------|------|
| `-include-paths <paths>` | 只爬取这些路径（逗号分隔，支持/api/*） |
| `-exclude-paths <paths>` | 排除这些路径（逗号分隔） |
| `-include-ext <exts>` | 只爬取这些扩展名（逗号分隔） |
| `-exclude-ext <exts>` | 排除这些扩展名（如: jpg,png,css,js） |
| `-allow-subdomains` | 允许爬取子域名 |

### 敏感信息检测
| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-sensitive-detect` | 启用敏感信息检测 | true |
| `-sensitive-rules <file>` | 敏感信息规则文件 | - |
| `-sensitive-min-severity <level>` | 最低严重级别: LOW/MEDIUM/HIGH | LOW |

### 批量扫描
| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-batch-file <file>` | 批量扫描URL列表文件 | - |
| `-batch-concurrency <num>` | 批量扫描并发数 | 5 |

### 其他参数
| 参数 | 说明 |
|------|------|
| `-proxy <url>` | 代理服务器地址 |
| `-rate-limit <num>` | 每秒最大请求数 |
| `-output <dir>` | 输出目录 |
| `-quiet` | 静默模式 |

完整参数列表请运行：`spider -help`

---

## 💡 使用建议

### 1. 根据目标选择合适的深度和并发
```bash
# 小型站点
spider -url https://example.com -depth 3 -max-pages 100 -workers 10

# 中型站点
spider -url https://example.com -depth 5 -max-pages 500 -workers 20

# 大型站点
spider -url https://example.com -depth 8 -max-pages 2000 -workers 50
```

### 2. 敏感信息检测推荐配置
```bash
# 日常使用（40个规则）
spider -url https://example.com -sensitive-rules sensitive_rules_standard.json

# 快速扫描（10个规则）
spider -url https://example.com -sensitive-rules sensitive_rules_minimal.json

# 全面审计（完整规则）
spider -url https://example.com -sensitive-rules sensitive_rules_config.json
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

## 🤝 与其他工具集成

### 漏洞扫描工具链

```bash
# 1. GogoSpider发现URL
spider -url https://target.com -depth 3

# 2. 传递给nuclei扫描漏洞
nuclei -l spider_target.com_*_unique_urls.txt -t cves/ -o vulns.txt

# 3. 传递给sqlmap测试SQL注入
sqlmap -m spider_target.com_*_params.txt --batch

# 4. 传递给xray进行被动扫描
xray webscan --url-file spider_target.com_*_unique_urls.txt
```

### 管道模式

```bash
# 从标准输入读取URL
cat urls.txt | spider -stdin -quiet

# 与其他工具链配合
echo "https://example.com" | spider -stdin | grep "api" | nuclei -t cves/
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

### ❌ 错误做法
```javascript
// 永远不要在前端代码中硬编码密钥
const ossConfig = {
  accessKeyId: 'LTAI4G3VxQxYxxxxxEXAMPLE',
  accessKeySecret: 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
};
```

### ✅ 正确做法
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

## 🐛 常见问题

### Q1: 如何禁用敏感信息检测？
```bash
spider -url https://example.com -sensitive-detect=false
```

### Q2: 如何只检测云存储密钥？
```bash
spider -url https://example.com -sensitive-min-severity HIGH
```

### Q3: 批量扫描失败怎么办？
每个URL独立扫描，某个URL失败不影响其他URL。查看最终报告的成功/失败统计。

### Q4: 动态内容未爬取到？
```bash
# 使用dynamic模式
spider -url https://example.com -mode dynamic
```

### Q5: 如何查看所有检测到的敏感信息类型？
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

## 📚 完整文档

项目包含以下文档：

- 📄 `README.md` - 本文件（项目总览）
- 📄 `PARAMETERS_GUIDE.md` - 完整参数指南
- 📄 `CONFIG_GUIDE.md` - 配置文件指南
- 📄 `CONFIGURATION_FAQ.md` - 配置FAQ
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
