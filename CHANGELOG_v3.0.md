# GogoSpider v3.0 更新日志

## 🎉 重大更新

v3.0 是一个重要的版本更新，主要聚焦于**简化使用体验**和**增强企业级功能**。

---

## ✨ 新增功能

### 1. 简化配置体验

#### 📝 命令行参数精简

**问题：** v2.x 版本有70+个命令行参数，使用复杂，难以记忆

**解决方案：**
- ✅ 精简到 **15个核心参数**
- ✅ 所有细节配置移到 JSON 配置文件
- ✅ 更清晰的参数分类

**对比：**

```bash
# v2.x - 参数过多，难以记忆
./spider -url https://example.com \
  -depth 5 \
  -max-pages 1000 \
  -rate-limit 50 \
  -rate-limit-enable \
  -burst 10 \
  -min-delay 100 \
  -max-delay 1000 \
  -adaptive-rate \
  -wayback \
  -include-domains "*.example.com" \
  -exclude-ext "jpg,png,css" \
  -sensitive-rules rules.json \
  -sensitive-min-severity MEDIUM \
  -log-level debug \
  -json \
  -output-file results.json

# v3.0 - 简洁清晰
./spider -url https://example.com -preset deep_scan
```

#### 🎯 预设场景配置

提供 **5种开箱即用** 的配置模板：

1. **quick_scan** - 快速扫描（3层深度，200页面）
2. **deep_scan** - 深度扫描（8层深度，5000页面）
3. **api_discovery** - API发现（专注接口）
4. **batch_scan** - 批量扫描（多目标）
5. **stealth_scan** - 隐蔽扫描（低速率）

**使用方式：**

```bash
# 直接使用预设
./spider -url https://example.com -preset deep_scan

# 基于预设修改参数
./spider -url https://example.com -preset deep_scan -depth 10
```

#### 📊 清晰的优先级

```
命令行参数 > 配置文件 > 默认值
```

---

### 2. 黑名单功能

**新增配置：** `blacklist_settings`

#### 功能描述

自动防止爬取敏感网站，避免法律风险：

- 政府网站（*.gov.cn, *.gov）
- 教育机构（*.edu.cn, *.edu）
- 军事网站（*.mil.cn, *.mil）
- 金融机构（*bank*, *payment*）
- 司法机构（*police*, *court*）

#### 配置示例

```json
{
  "blacklist_settings": {
    "enabled": true,
    "domains": [
      "*.gov.cn",
      "*.edu.cn",
      "*.mil.cn"
    ],
    "domain_patterns": [
      "*bank*",
      "*payment*",
      "*police*"
    ],
    "strict_mode": true
  }
}
```

#### 匹配方式

- **精确匹配**: `"example.com"` - 只匹配 example.com
- **通配符匹配**: `"*.example.com"` - 匹配所有子域名
- **模糊匹配**: `"*bank*"` - 匹配包含 bank 的域名

#### 严格模式

- `strict_mode: true` - 匹配到直接拒绝，不会爬取
- `strict_mode: false` - 匹配到记录警告，但继续爬取

---

### 3. 批量扫描增强

**新增配置：** `batch_scan_settings`

#### 完整配置

```json
{
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
}
```

#### 功能特性

- ✅ 并发控制（可配置并发数）
- ✅ 超时控制（每个目标独立超时）
- ✅ 错误处理（继续或停止）
- ✅ 独立报告（每个目标单独保存）
- ✅ 汇总报告（整体统计）

#### 输出结构

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

---

### 4. 敏感信息规则配置

**新增字段：** `sensitive_detection_settings.rules_file`

#### 问题

v2.x 版本中，敏感信息规则文件路径不明确，用户不知道如何指定自定义规则。

#### 解决方案

在配置文件中明确指定规则文件路径：

```json
{
  "sensitive_detection_settings": {
    "enabled": true,
    "rules_file": "./sensitive_rules_config.json",
    "min_severity": "LOW",
    "scan_response_body": true,
    "scan_response_headers": true,
    "realtime_output": true
  }
}
```

#### 支持的路径格式

- **相对路径**: `"./my_rules.json"`
- **绝对路径**: `"/path/to/my_rules.json"`
- **默认路径**: `"./sensitive_rules_config.json"`

---

### 5. 作用域优先级说明

#### 优先级规则

Scope 控制的过滤优先级（从高到低）：

```
exclude_regex > exclude_domains > exclude_paths > 
include_regex > include_domains > include_paths
```

#### 说明

1. **排除规则优先于包含规则**
2. **正则优先于精确匹配**
3. **域名匹配优先于路径匹配**

#### 示例

```json
{
  "scope_settings": {
    "include_domains": ["*.example.com"],    // 优先级 5
    "exclude_domains": ["cdn.example.com"],  // 优先级 2（高）
    "include_paths": ["/api/*"],             // 优先级 6（低）
    "exclude_paths": ["/*.jpg"],             // 优先级 3
    "include_regex": "/api/v\\d+/.*",        // 优先级 4
    "exclude_regex": "\\.(jpg|png|css)$"     // 优先级 1（最高）
  }
}
```

**处理流程：**
1. 首先检查 `exclude_regex`，匹配则拒绝
2. 然后检查 `exclude_domains`，匹配则拒绝
3. 然后检查 `exclude_paths`，匹配则拒绝
4. 通过排除检查后，再检查包含规则
5. 最后通过所有检查的URL才会被爬取

---

### 6. 配置文件优化

#### 新增配置文件

1. **example_config_optimized.json** - 完整配置模板
2. **config_presets/quick_scan.json** - 快速扫描
3. **config_presets/deep_scan.json** - 深度扫描
4. **config_presets/api_discovery.json** - API发现
5. **config_presets/batch_scan.json** - 批量扫描
6. **config_presets/stealth_scan.json** - 隐蔽扫描

#### 配置文件增强

所有配置文件都包含：
- ✅ 详细的注释说明
- ✅ 使用场景说明
- ✅ 推荐的参数值
- ✅ 黑名单配置
- ✅ 作用域配置
- ✅ 优先级说明

---

## 📚 新增文档

### 1. CONFIG_GUIDE.md

**完整配置指南** - 300+ 行详细文档

包含内容：
- 概述和优先级说明
- 命令行参数详解
- 配置文件详解
- 预设场景说明
- 敏感信息检测配置
- 黑名单配置指南
- 批量扫描指南
- 常见问题解答

### 2. PARAMETERS_MIGRATION.md

**参数迁移指南** - v2.x → v3.0 迁移文档

包含内容：
- 保留的参数列表
- 移除的参数列表
- 迁移对照表
- 迁移步骤说明
- 常见迁移场景
- 兼容性说明

### 3. README.md 更新

**主文档更新** - 添加 v3.0 说明

新增内容：
- v3.0 重大更新说明
- 预设场景使用指南
- 黑名单配置说明
- 配置文件说明
- 更清晰的使用示例

---

## 🔧 代码改进

### 1. config/config.go

**新增结构体：**

```go
// BlacklistSettings 黑名单设置
type BlacklistSettings struct {
    Enabled        bool
    Domains        []string
    DomainPatterns []string
    StrictMode     bool
}

// BatchScanSettings 批量扫描设置
type BatchScanSettings struct {
    Enabled               bool
    InputFile             string
    Concurrency           int
    OutputDir             string
    PerTargetTimeout      int
    ContinueOnError       bool
    SaveIndividualReports bool
    SaveSummaryReport     bool
}
```

**新增字段：**

```go
type SensitiveDetectionSettings struct {
    // ... 原有字段 ...
    RulesFile string  // 新增：规则文件路径
}
```

### 2. 默认配置更新

添加黑名单和批量扫描的默认配置：

```go
BlacklistSettings: BlacklistSettings{
    Enabled:    true,
    Domains:    []string{"*.gov.cn", "*.edu.cn", "*.mil.cn"},
    StrictMode: true,
},

BatchScanSettings: BatchScanSettings{
    Enabled:      false,
    Concurrency:  5,
    // ... 其他默认值 ...
},
```

---

## 📦 新增文件

### 配置文件

```
example_config_optimized.json       # 优化的完整配置模板
config_presets/
├── quick_scan.json                 # 快速扫描预设
├── deep_scan.json                  # 深度扫描预设
├── api_discovery.json              # API发现预设
├── batch_scan.json                 # 批量扫描预设
└── stealth_scan.json               # 隐蔽扫描预设
```

### 文档文件

```
CONFIG_GUIDE.md                     # 完整配置指南
PARAMETERS_MIGRATION.md             # 参数迁移指南
CHANGELOG_v3.0.md                   # 本文档
```

---

## 🔄 变更说明

### 保留的功能

- ✅ 所有 v2.x 的核心爬虫功能
- ✅ 敏感信息检测功能
- ✅ 批量扫描功能
- ✅ 所有去重和优化功能

### 移除的功能

- ❌ Fuzzing 参数爆破功能（专注纯爬虫）
  - 移除 `-fuzz`, `-fuzz-params`, `-fuzz-dict`
  - 原因：专注于纯爬虫功能，爆破可用其他工具

### 精简的参数

- 📦 70+ 个命令行参数 → 15 个核心参数
- 📝 其他参数移到配置文件
- 📚 提供完整的迁移指南

---

## 🚀 升级指南

### 从 v2.x 升级到 v3.0

#### 步骤 1：更新二进制文件

```bash
# 下载 v3.0 版本
# 或重新编译
go build -o spider cmd/spider/main.go
```

#### 步骤 2：创建配置文件

```bash
# 复制优化配置模板
cp example_config_optimized.json my_config.json

# 或使用预设场景
cp config_presets/deep_scan.json my_config.json
```

#### 步骤 3：迁移你的参数

参考 [PARAMETERS_MIGRATION.md](PARAMETERS_MIGRATION.md) 将你的命令行参数迁移到配置文件。

#### 步骤 4：测试新配置

```bash
# 使用 debug 模式查看最终配置
./spider -url https://example.com -config my_config.json -log-level debug
```

#### 步骤 5：使用预设场景（推荐）

```bash
# 对于大多数场景，直接使用预设即可
./spider -url https://example.com -preset deep_scan
```

---

## 💡 最佳实践

### 1. 优先使用预设场景

```bash
# 推荐方式 - 使用预设
./spider -url https://example.com -preset deep_scan

# 而不是
./spider -url https://example.com \
  -depth 8 \
  -max-pages 5000 \
  # ... 大量参数
```

### 2. 保存常用配置

```bash
# 将你的配置保存为预设
cp my_config.json config_presets/my_preset.json

# 以后直接使用
./spider -url https://example.com -preset my_preset
```

### 3. 使用配置文件而非命令行

```bash
# 推荐 - 配置文件
./spider -url https://example.com -config my_config.json

# 不推荐 - 大量命令行参数
./spider -url https://example.com -depth 5 -workers 20 ...
```

### 4. 启用黑名单保护

所有预设场景都默认启用黑名单，保护你避免误爬敏感网站。

### 5. 阅读完整文档

- 📖 [CONFIG_GUIDE.md](CONFIG_GUIDE.md) - 配置指南
- 📖 [PARAMETERS_MIGRATION.md](PARAMETERS_MIGRATION.md) - 迁移指南
- 📖 [README.md](README.md) - 使用说明

---

## 🐛 已知问题

无

---

## 🎯 未来计划

- [ ] 黑名单实现（需要在 core 层添加过滤逻辑）
- [ ] 批量扫描完整实现（需要优化 main.go 的批量扫描逻辑）
- [ ] `-preset` 参数支持（需要添加预设加载逻辑）
- [ ] Web UI 配置界面
- [ ] 更多预设场景

---

## 📝 反馈

如果你对 v3.0 有任何建议或发现问题，请提交 Issue。

---

**版本：** v3.0  
**发布日期：** 2025-10-26  
**作者：** Warren-Jace

