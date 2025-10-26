# GogoSpider 参数使用指南

## 📖 参数分类说明

本文档将70+个命令行参数按使用场景和功能进行分类，帮助您快速找到需要的参数。

---

## 🎯 快速场景选择

### 场景1: 快速扫描（初学者推荐）
```bash
./main.exe -url https://example.com
```
**说明**: 使用默认配置即可，简单快速

---

### 场景2: 深度全面扫描
```bash
./main.exe -url https://example.com \
  -depth 5 \
  -max-pages 1000 \
  -workers 20 \
  -mode dynamic
```
**适用**: 安全测试、漏洞挖掘、API发现

---

### 场景3: API接口发现
```bash
./main.exe -url https://example.com \
  -include-paths "/api/*,/v1/*,/v2/*" \
  -exclude-ext "jpg,png,css,js,ico" \
  -depth 5
```
**适用**: 后端接口分析、API文档生成

---

### 场景4: 隐蔽低速扫描
```bash
./main.exe -url https://example.com \
  -rate-limit 5 \
  -min-delay 500 \
  -max-delay 2000 \
  -adaptive-rate
```
**适用**: 敏感目标、避免触发WAF/IDS

---

### 场景5: 批量站点扫描
```bash
./main.exe -batch-file targets.txt \
  -batch-concurrency 10 \
  -output ./batch_results
```
**适用**: 多站点资产盘点、批量安全检查

---

### 场景6: 敏感信息专项扫描
```bash
./main.exe -url https://example.com \
  -sensitive-rules sensitive_rules_config.json \
  -sensitive-min-severity HIGH \
  -sensitive-output sensitive.json
```
**适用**: 密钥泄露检查、合规审计

---

## 📂 参数分类详解

### 一、核心参数（必需）

| 参数 | 说明 | 默认值 | 示例 |
|------|------|--------|------|
| `-url` | 🔴 目标URL（必需） | - | `-url https://example.com` |
| `-config` | 配置文件路径 | - | `-config config.json` |

---

### 二、基础爬取参数

#### 2.1 深度和范围控制

| 参数 | 说明 | 默认值 | 推荐值 | 场景 |
|------|------|--------|--------|------|
| `-depth` | 最大爬取深度 | 3 | 快速:2, 深度:5-8 | 控制递归层数 |
| `-max-pages` | 最大页面数 | 100 | 快速:50, 深度:1000+ | 防止无限爬取 |
| `-workers` | 并发线程数 | 10 | 快速:5, 深度:20-50 | 提高爬取速度 |

**使用建议**:
- 小型站点: `-depth 3 -max-pages 100 -workers 10`
- 中型站点: `-depth 5 -max-pages 500 -workers 20`
- 大型站点: `-depth 8 -max-pages 2000 -workers 50`

---

#### 2.2 爬取模式

| 参数 | 说明 | 默认值 | 场景 |
|------|------|--------|------|
| `-mode` | 爬取模式 | smart | 见下表 |

**模式说明**:
- `static` - 静态爬虫（快速，只解析HTML）
- `dynamic` - 动态爬虫（慢但全面，使用Chrome）
- `smart` - 智能模式（自动选择，推荐）

**选择指南**:
```bash
# 静态网站（传统网站）
-mode static

# 单页应用SPA（React/Vue/Angular）
-mode dynamic

# 不确定网站类型
-mode smart
```

---

### 三、作用域控制参数（Scope）

#### 3.1 域名控制

| 参数 | 说明 | 示例 |
|------|------|------|
| `-include-domains` | 只爬取这些域名 | `-include-domains "*.example.com,api.test.com"` |
| `-exclude-domains` | 排除这些域名 | `-exclude-domains "cdn.example.com"` |
| `-allow-subdomains` | 允许爬取子域名 | `-allow-subdomains` |

**组合使用**:
```bash
# 只爬取主域名和API域名
-include-domains "example.com,api.example.com"

# 爬取所有子域名但排除CDN
-allow-subdomains -exclude-domains "cdn.example.com,static.example.com"
```

---

#### 3.2 路径控制

| 参数 | 说明 | 示例 |
|------|------|------|
| `-include-paths` | 只爬取这些路径 | `-include-paths "/api/*,/admin/*"` |
| `-exclude-paths` | 排除这些路径 | `-exclude-paths "/logout,/signout"` |
| `-include-regex` | URL包含正则 | `-include-regex ".*\\.php"` |
| `-exclude-regex` | URL排除正则 | `-exclude-regex ".*\\.(jpg\|png)"` |

**使用场景**:
```bash
# 只爬取API路径
-include-paths "/api/*,/v1/*,/v2/*"

# 排除登出和下载路径
-exclude-paths "/logout,/signout,/download/*"

# 只爬取PHP文件
-include-regex ".*\\.php"
```

---

#### 3.3 文件扩展名控制

| 参数 | 说明 | 示例 |
|------|------|------|
| `-include-ext` | 只爬取这些扩展名 | `-include-ext "php,jsp,aspx"` |
| `-exclude-ext` | 排除这些扩展名 | `-exclude-ext "jpg,png,css,js"` |

**常用组合**:
```bash
# 只爬取动态页面
-include-ext "php,jsp,aspx,do,action"

# 排除所有静态资源（推荐）
-exclude-ext "jpg,jpeg,png,gif,svg,ico,css,js,woff,woff2,ttf,mp4,mp3,pdf,zip"
```

**❓ exclude-ext 作用解释**:
- **作用**: 过滤URL，不爬取指定扩展名的文件
- **目的**: 排除图片、字体、视频等静态资源，提高效率
- **效果**: 节省时间和带宽，专注于动态内容
- **建议**: 始终排除静态资源，除非有特殊需求

---

### 四、网络和代理参数

| 参数 | 说明 | 默认值 | 示例 |
|------|------|--------|------|
| `-timeout` | 请求超时（秒） | 30 | `-timeout 60` |
| `-proxy` | 代理服务器 | - | `-proxy http://127.0.0.1:8080` |
| `-user-agent` | 自定义User-Agent | - | `-user-agent "MyBot/1.0"` |
| `-headers` | 自定义HTTP头（JSON） | - | `-headers '{"Token":"xxx"}'` |
| `-cookie-file` | Cookie文件路径 | - | `-cookie-file cookies.txt` |

**使用场景**:
```bash
# 通过代理扫描
-proxy http://127.0.0.1:8080

# 认证扫描（需要登录）
-cookie-file session_cookies.txt -headers '{"Authorization":"Bearer xxx"}'
```

---

### 五、速率控制参数（防止封禁）

| 参数 | 说明 | 默认值 | 场景 |
|------|------|--------|------|
| `-rate-limit` | 每秒最大请求数 | 100 | 根据目标调整 |
| `-rate-limit-enable` | 启用速率限制 | false | 避免压力过大 |
| `-burst` | 允许突发请求数 | 10 | 初始加速 |
| `-min-delay` | 最小延迟（毫秒） | 0 | 隐蔽扫描用 |
| `-max-delay` | 最大延迟（毫秒） | 0 | 随机延迟范围 |
| `-adaptive-rate` | 自适应速率控制 | false | 智能调整速度 |
| `-min-rate` | 自适应最小速率 | 10 | 自适应下限 |
| `-max-rate` | 自适应最大速率 | 200 | 自适应上限 |

**场景配置**:
```bash
# 快速扫描（内网/测试环境）
-rate-limit 100 -workers 50

# 普通扫描（一般网站）
-rate-limit 20 -adaptive-rate -min-rate 10 -max-rate 50

# 隐蔽扫描（敏感目标）
-rate-limit 5 -min-delay 500 -max-delay 2000

# 极速扫描（无限制）
# 不设置任何速率参数
```

---

### 六、敏感信息检测参数

#### 6.1 基础参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-sensitive-detect` | 启用敏感信息检测 | true |
| `-sensitive-scan-body` | 扫描响应体 | true |
| `-sensitive-scan-headers` | 扫描响应头 | true |
| `-sensitive-min-severity` | 最低严重级别 | LOW |
| `-sensitive-output` | 敏感信息输出文件 | 自动生成 |
| `-sensitive-realtime` | 实时输出敏感信息 | true |

#### 6.2 自定义规则

| 参数 | 说明 | 示例 |
|------|------|------|
| `-sensitive-rules` | 外部规则文件 | `-sensitive-rules custom_rules.json` |

**使用场景**:
```bash
# 只检测高危敏感信息（云存储密钥、数据库密码）
-sensitive-min-severity HIGH

# 禁用敏感信息检测（性能优先）
-sensitive-detect=false

# 使用自定义规则
-sensitive-rules ./my_company_rules.json
```

---

### 七、外部数据源参数

| 参数 | 说明 | 默认值 | 用途 |
|------|------|--------|------|
| `-wayback` | 从Wayback Machine获取历史URL | false | 发现已下线的页面 |
| `-virustotal` | 从VirusTotal获取URL | false | 发现被报告的URL |
| `-vt-api-key` | VirusTotal API密钥 | - | VT认证 |
| `-commoncrawl` | 从CommonCrawl获取URL | false | 从网络爬虫数据获取 |
| `-external-timeout` | 外部源超时（秒） | 30 | 防止卡死 |

**使用建议**:
```bash
# 全面扫描（包含历史URL）
-wayback -virustotal -vt-api-key "your-key" -commoncrawl

# 注意：外部数据源会大幅增加爬取URL数量和时间
```

---

### 八、输出和日志参数

#### 8.1 输出格式

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-output` | 输出目录 | ./ |
| `-output-file` | 输出文件路径 | 自动生成 |
| `-format` | 输出格式 | text |
| `-json` | 启用JSON输出 | false |
| `-json-mode` | JSON模式 | line |
| `-include-all` | 包含所有字段 | false |

**格式说明**:
- `text` - 文本格式（易读）
- `json` - JSON格式（程序处理）
- `urls-only` - 只输出URL（管道模式）

```bash
# 文本输出（默认）
-format text

# JSON行分隔输出（推荐）
-json -json-mode line

# 传递给其他工具
-format urls-only -simple
```

---

#### 8.2 日志参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-log-level` | 日志级别 | info |
| `-log-file` | 日志文件路径 | 控制台 |
| `-log-format` | 日志格式 | json |
| `-show-metrics` | 显示实时指标 | false |
| `-quiet` | 静默模式 | false |
| `-simple` | 简洁模式 | false |

**日志级别**:
- `debug` - 调试信息（最详细）
- `info` - 一般信息（推荐）
- `warn` - 警告信息
- `error` - 错误信息（最简洁）

```bash
# 调试模式（排查问题）
-log-level debug -log-file debug.log

# 静默模式（只要结果）
-quiet -simple

# 监控模式（查看实时性能）
-show-metrics
```

---

### 九、批量扫描参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-batch-file` | URL列表文件 | - |
| `-batch-concurrency` | 批量并发数 | 5 |

**使用方法**:
```bash
# 创建目标文件
cat > targets.txt << EOF
https://www.example.com
https://api.example.com
https://admin.example.com
EOF

# 批量扫描
./main.exe -batch-file targets.txt -batch-concurrency 10
```

---

### 十、管道和集成参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-stdin` | 从标准输入读取URL | false |
| `-pipeline` | 启用管道模式 | false |
| `-simple` | 简洁模式 | false |
| `-quiet` | 静默模式 | false |

**管道集成**:
```bash
# 从标准输入读取
cat urls.txt | ./main.exe -stdin -quiet

# 传递给nuclei
./main.exe -url https://example.com -simple | nuclei -silent

# 与httpx结合
./main.exe -url https://example.com -format urls-only | httpx -silent
```

---

### 十一、高级参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-chrome-path` | Chrome浏览器路径 | 自动查找 |
| `-ignore-robots` | 忽略robots.txt | false |
| `-fuzz` | 启用参数模糊测试 | false |
| `-fuzz-params` | 要fuzz的参数 | - |
| `-fuzz-dict` | Fuzz字典文件 | - |

---

## 🎯 完整使用示例

### 示例1: 企业内网扫描
```bash
./main.exe -url https://internal.company.com \
  -depth 6 \
  -max-pages 2000 \
  -workers 30 \
  -rate-limit 50 \
  -include-paths "/api/*,/admin/*" \
  -exclude-ext "jpg,png,css,js" \
  -sensitive-detect=true \
  -sensitive-min-severity MEDIUM \
  -output ./results \
  -log-level info
```

---

### 示例2: 外部SaaS平台扫描
```bash
./main.exe -url https://saas-platform.com \
  -depth 5 \
  -max-pages 1000 \
  -workers 20 \
  -rate-limit 10 \
  -adaptive-rate \
  -min-delay 200 \
  -max-delay 800 \
  -mode dynamic \
  -sensitive-rules sensitive_rules_config.json \
  -sensitive-min-severity HIGH
```

---

### 示例3: API接口扫描
```bash
./main.exe -url https://api.service.com \
  -include-paths "/api/*,/v1/*,/v2/*,/v3/*" \
  -exclude-ext "jpg,png,css,js,ico,svg,woff,ttf" \
  -include-ext "json" \
  -depth 4 \
  -max-pages 500 \
  -workers 15 \
  -format json \
  -json-mode line
```

---

### 示例4: 批量资产扫描
```bash
./main.exe -batch-file company_assets.txt \
  -batch-concurrency 10 \
  -depth 4 \
  -max-pages 300 \
  -workers 10 \
  -rate-limit 20 \
  -sensitive-detect=true \
  -sensitive-output batch_sensitive.json \
  -output ./batch_results
```

---

## 📊 参数优先级

```
命令行参数 > 配置文件 > 默认值
```

**示例**:
```bash
# 配置文件中 depth=5，命令行指定 depth=3
# 最终使用: depth=3
./main.exe -config config.json -url https://example.com -depth 3
```

---

## 💡 最佳实践建议

### 1. 新手入门
```bash
# 第一次使用，先用默认配置
./main.exe -url https://example.com
```

### 2. 日常使用
```bash
# 创建配置文件，保存常用设置
./main.exe -config my_config.json -url https://target.com
```

### 3. 性能优化
```bash
# 排除静态资源 + 增加并发 + 深度爬取
./main.exe -url https://target.com \
  -exclude-ext "jpg,png,css,js,woff,ttf,mp4,mp3,pdf,zip" \
  -depth 6 \
  -workers 30 \
  -max-pages 2000
```

### 4. 隐蔽扫描
```bash
# 低速 + 随机延迟 + 自适应速率
./main.exe -url https://sensitive-target.com \
  -rate-limit 5 \
  -min-delay 500 \
  -max-delay 2000 \
  -adaptive-rate \
  -user-agent "Mozilla/5.0 ..."
```

---

## 🔧 故障排查

### 问题1: 爬取不到动态内容
**解决**: 使用动态模式
```bash
-mode dynamic -chrome-path "C:\Program Files\Google\Chrome\Application\chrome.exe"
```

### 问题2: 速度太慢
**解决**: 增加并发，取消速率限制
```bash
-workers 50 -rate-limit 100
```

### 问题3: 被目标网站封禁
**解决**: 降低速率，增加延迟
```bash
-rate-limit 5 -min-delay 1000 -max-delay 3000
```

### 问题4: 内存占用过高
**解决**: 限制最大页面数
```bash
-max-pages 500
```

---

## 📚 相关文档

- `README.md` - 项目总览
- `CONFIG_GUIDE.md` - 配置文件指南
- `example_config_optimized.json` - 配置文件示例
- `sensitive_rules_config.json` - 敏感信息规则

---

## 🙏 说明

本指南将70+个参数按场景和功能分类，帮助您快速找到所需参数。

**建议使用顺序**:
1. 先看"快速场景选择"，找到最接近的场景
2. 再看对应的"参数分类详解"，了解参数含义
3. 查看"完整使用示例"，复制修改使用

如有疑问，欢迎提Issue！

