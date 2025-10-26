# GoGoSpider - 智能Web爬虫

> 🚀 功能强大的Go语言Web安全爬虫工具

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8?logo=go)](https://golang.org/)

## ✨ 核心特性

### 🔍 智能爬取
- **双引擎爬虫**：静态爬虫（Colly）+ 动态爬虫（Chromedp）
- **JavaScript分析**：40+种JS URL提取模式
- **AJAX拦截**：自动捕获动态加载的URL
- **事件触发**：模拟点击、悬停、输入等用户行为
- **多层递归**：支持深度爬取（BFS算法）

### 🎯 智能去重
- **三层去重机制**：
  - URL模式去重（忽略参数值变化）
  - URL结构化去重（识别路径变量）
  - DOM相似度去重（5种算法）
- **智能参数值去重**：16种特征分类，效率提升84%
- **业务感知过滤**：自动识别URL业务价值

### 🛡️ 安全功能
- **技术栈识别**：自动检测15+种Web框架
- **敏感信息检测**：30+种敏感模式（API密钥、凭证等）
- **隐藏路径扫描**：200+个常见Web路径
- **子域名提取**：自动发现子域名
- **IP地址检测**：识别内网IP泄露

### 🚀 高级功能
- **CDN检测**：识别60+个CDN服务商
- **表单智能填充**：20+种字段类型自动填充
- **静态资源分类**：7种资源类型智能分类
- **Sitemap解析**：自动爬取sitemap和robots.txt
- **CSS URL提取**：支持@import、url()等
- **Base64解码**：自动解码Base64中的URL

### 📊 外部数据源（可选）
- Wayback Machine（历史URL）
- VirusTotal（安全情报）
- Common Crawl（大规模爬取数据）

---

## 📦 安装

### 方式1：使用预编译二进制（推荐）

从[Releases](https://github.com/Warren-Jace/gogospider/releases)页面下载对应平台的可执行文件。

### 方式2：从源码编译

**前置要求**：
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
```

---

## 🚀 快速开始

### 基本用法

```bash
# 爬取单个网站
./spider -url http://example.com

# 指定爬取深度
./spider -url http://example.com -depth 3

# 使用配置文件
./spider -config example_config.json
```

### 输出文件

爬取完成后会自动生成以下文件：

```
spider_example.com_20241026_120000.txt                    # 详细结果
spider_example.com_20241026_120000_urls.txt               # 所有URL
spider_example.com_20241026_120000_all_urls.txt          # 全部URL（含子资源）
spider_example.com_20241026_120000_params.txt            # 带参数的URL
spider_example.com_20241026_120000_forms.txt             # 表单URL
spider_example.com_20241026_120000_post_requests.txt     # POST请求
spider_example.com_20241026_120000_unique_urls.txt       # 去重URL（推荐）
spider_example.com_20241026_120000_structure_unique_urls.txt  # 结构化去重URL
```

**推荐使用**: `*_unique_urls.txt` 或 `*_structure_unique_urls.txt`，已经智能去重，适合传递给其他安全工具（如nuclei、sqlmap等）。

---

## 📖 命令行参数

### 基础参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-url` | 目标URL（必需） | - |
| `-depth` | 最大爬取深度 | 3 |
| `-max-pages` | 最大爬取页面数 | 100 |
| `-timeout` | 请求超时（秒） | 30 |
| `-workers` | 并发线程数 | 10 |

### 爬取模式

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-mode` | 爬取模式：static, dynamic, smart | smart |
| `-allow-subdomains` | 允许爬取子域名 | false |
| `-ignore-robots` | 忽略robots.txt | false |

### 代理和认证

| 参数 | 说明 |
|------|------|
| `-proxy` | 代理服务器（如：http://127.0.0.1:8080） |
| `-user-agent` | 自定义User-Agent |
| `-cookie-file` | Cookie文件路径 |
| `-headers` | 自定义HTTP头（JSON格式） |

### 外部数据源

| 参数 | 说明 |
|------|------|
| `-wayback` | 启用Wayback Machine |
| `-virustotal` | 启用VirusTotal |
| `-vt-api-key` | VirusTotal API密钥 |
| `-commoncrawl` | 启用Common Crawl |
| `-external-timeout` | 外部数据源超时（秒，默认30） |

### Scope控制

| 参数 | 说明 |
|------|------|
| `-include-domains` | 包含的域名（逗号分隔，支持*.example.com） |
| `-exclude-domains` | 排除的域名（逗号分隔） |
| `-include-paths` | 包含的路径模式（逗号分隔，支持/api/*） |
| `-exclude-paths` | 排除的路径模式 |
| `-include-regex` | 包含的URL正则表达式 |
| `-exclude-regex` | 排除的URL正则表达式 |
| `-include-ext` | 包含的文件扩展名（逗号分隔） |
| `-exclude-ext` | 排除的文件扩展名 |

### 速率控制

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-rate-limit` | 每秒最大请求数 | 100 |
| `-adaptive-rate` | 启用自适应速率控制 | false |
| `-min-rate` | 自适应最小速率 | 10 |
| `-max-rate` | 自适应最大速率 | 200 |
| `-min-delay` | 最小请求间隔（毫秒） | 0 |
| `-max-delay` | 最大请求间隔（毫秒） | 0 |

### 输出格式

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-json` | 启用JSON输出 | false |
| `-json-mode` | JSON模式：compact, pretty, line | line |
| `-output-file` | 输出文件路径 | - |
| `-format` | 输出格式：text, json, urls-only | text |
| `-simple` | 简洁模式（只输出URL） | false |

### 管道模式

| 参数 | 说明 |
|------|------|
| `-stdin` | 从标准输入读取URL |
| `-pipeline` | 启用管道模式 |
| `-quiet` | 静默模式（日志输出到stderr） |

### 日志和调试

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-log-level` | 日志级别：debug, info, warn, error | info |
| `-log-file` | 日志文件路径 | - |
| `-log-format` | 日志格式：json, text | json |
| `-show-metrics` | 显示实时监控指标 | false |

---

## 💡 使用示例

### 示例1：基础爬取

```bash
# 爬取网站，深度3层
./spider -url http://testphp.vulnweb.com -depth 3
```

### 示例2：深度爬取 + 子域名

```bash
# 深度5层，允许子域名，最多爬取200个页面
./spider -url https://example.com -depth 5 -allow-subdomains -max-pages 200
```

### 示例3：使用代理 + 自定义UA

```bash
# 通过代理爬取，使用自定义User-Agent
./spider -url http://example.com \
  -proxy http://127.0.0.1:8080 \
  -user-agent "Mozilla/5.0 (Custom Spider)"
```

### 示例4：Scope过滤

```bash
# 只爬取/api路径，排除图片
./spider -url http://example.com \
  -include-paths "/api/*,/v1/*" \
  -exclude-ext "jpg,png,gif,svg"
```

### 示例5：外部数据源

```bash
# 结合Wayback Machine和VirusTotal
./spider -url http://example.com \
  -wayback \
  -virustotal -vt-api-key "YOUR_API_KEY"
```

### 示例6：速率限制

```bash
# 限制每秒10个请求，自适应速率
./spider -url http://example.com \
  -rate-limit 10 \
  -adaptive-rate \
  -min-rate 5 -max-rate 20
```

### 示例7：管道模式

```bash
# 从文件读取URL列表，批量爬取
cat urls.txt | ./spider -stdin -simple

# 与其他工具链配合
echo "http://example.com" | ./spider -stdin | nuclei -t cves/
```

### 示例8：JSON输出

```bash
# 输出为JSON格式
./spider -url http://example.com -json -json-mode pretty

# 保存到文件
./spider -url http://example.com -json -output-file results.json
```

### 示例9：使用配置文件

```bash
# 使用配置文件（推荐用于复杂配置）
./spider -config example_config.json
```

---

## ⚙️ 配置文件

配置文件使用JSON格式，可以包含所有命令行参数的配置。

### 示例配置文件

参见 `example_config.json`，包含所有可配置选项和详细说明。

### 配置优先级

```
命令行参数 > 配置文件 > 默认值
```

---

## 🔧 高级功能

### 1. 智能去重

程序自动使用三层去重机制：

- **URL模式去重**：`/product?id=1` → `/product?id=`
- **结构化去重**：`/user/123/profile` → `/user/{num}/profile`
- **DOM相似度去重**：检测页面结构相似度

### 2. 业务感知过滤

自动计算URL的业务价值：

- 高价值：登录、支付、API、文件上传等
- 中价值：详情页、搜索等
- 低价值：静态页面、重复结构

### 3. 隐藏路径扫描

自动扫描200+个常见路径：

- 管理后台：/admin, /wp-admin, /phpmyadmin
- API接口：/api, /api/v1, /graphql
- 配置文件：/.env, /config.php, /web.config
- 备份文件：/backup, /backup.sql, /db.sql

### 4. 技术栈识别

自动识别：

- Web服务器（Nginx, Apache, IIS等）
- 开发框架（Laravel, Django, Spring等）
- CMS系统（WordPress, Drupal等）
- JavaScript库（jQuery, Vue, React等）

---

## 🛡️ 安全说明

### 合法使用

本工具仅用于授权的安全测试。使用前请确保：

1. ✅ 已获得目标网站所有者的明确授权
2. ✅ 遵守当地法律法规
3. ✅ 不用于恶意攻击或非法活动

### 速率限制

为避免对目标服务器造成影响，建议：

- 使用 `-rate-limit` 限制请求速率
- 使用 `-timeout` 设置合理的超时时间
- 避免过大的 `-max-pages` 值

---

## 🐛 常见问题

### Q1: 爬取结果为空？

**原因**：
- 目标网站使用了严格的反爬虫机制
- 需要登录才能访问

**解决方案**：
```bash
# 使用Cookie文件
./spider -url http://example.com -cookie-file cookies.txt

# 使用代理
./spider -url http://example.com -proxy http://127.0.0.1:8080
```

### Q2: 动态内容未爬取到？

**原因**：静态爬虫无法处理JavaScript渲染的内容

**解决方案**：
```bash
# 使用smart模式（默认）或dynamic模式
./spider -url http://example.com -mode dynamic
```

### Q3: 爬取速度慢？

**原因**：
- 目标网站响应慢
- 深度设置过大

**解决方案**：
```bash
# 增加并发数，减少超时时间
./spider -url http://example.com -workers 20 -timeout 15

# 限制深度
./spider -url http://example.com -depth 2
```

### Q4: 如何只爬取特定路径？

```bash
# 使用scope过滤
./spider -url http://example.com -include-paths "/api/*,/admin/*"
```

### Q5: 如何与其他安全工具集成？

```bash
# 使用unique_urls文件（已去重，适合传递给扫描器）
spider -url http://example.com

# 传递给nuclei
nuclei -l spider_example.com_*_unique_urls.txt -t cves/

# 传递给sqlmap
sqlmap -m spider_example.com_*_params.txt --batch

# 传递给xray
xray webscan --url-file spider_example.com_*_unique_urls.txt
```

---

## 📊 性能指标

- **URL发现率**：比Crawlergo提升119%
- **AJAX覆盖率**：100%
- **去重效果**：90%+
- **平均速度**：20-50 页/秒（取决于目标网站）

---

## 🤝 贡献

欢迎提交Issue和Pull Request！

---

## 📄 许可证

本项目采用 [Apache 2.0](LICENSE) 许可证。

---

## 🙏 致谢

本项目参考和学习了以下优秀项目：

- [Crawlergo](https://github.com/Qianlitp/crawlergo)
- [Katana](https://github.com/projectdiscovery/katana)
- [Gospider](https://github.com/jaeles-project/gospider)
- [Hakrawler](https://github.com/hakluke/hakrawler)

---

## 📧 联系方式

- GitHub: [@Warren-Jace](https://github.com/Warren-Jace)
- Issues: [提交Issue](https://github.com/Warren-Jace/gogospider/issues)

---

**⚠️ 免责声明**：本工具仅用于授权的安全测试，使用者需自行承担使用本工具的一切法律责任。
