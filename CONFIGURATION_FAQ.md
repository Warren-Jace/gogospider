# 配置文件常见问题解答

## 📝 问题1: exclude_extensions 的作用是什么？

### 简单回答
`exclude_extensions` 用于**过滤URL**，不爬取指定扩展名的文件。

### 详细说明

#### 作用位置
在 `scope_settings` 配置块中：
```json
{
  "scope_settings": {
    "exclude_extensions": [
      "jpg", "jpeg", "png", "gif",  // 图片
      "css", "js",                    // 样式和脚本
      "woff", "woff2", "ttf",        // 字体
      "mp4", "mp3", "avi",           // 视频音频
      "pdf", "doc", "zip"            // 文档压缩包
    ]
  }
}
```

#### 实际效果

**示例1**: 不配置 `exclude_extensions`
```
爬取的URL包括:
✅ https://example.com/api/users
✅ https://example.com/login.php
✅ https://example.com/images/logo.png       ← 图片也会爬
✅ https://example.com/static/style.css      ← CSS也会爬
✅ https://example.com/js/app.js             ← JS也会爬
✅ https://example.com/docs/manual.pdf       ← PDF也会爬

结果: 爬取1000个URL，其中700个是静态资源（浪费时间）
```

**示例2**: 配置 `exclude_extensions`
```json
{
  "exclude_extensions": ["png", "css", "js", "pdf"]
}
```
```
爬取的URL包括:
✅ https://example.com/api/users
✅ https://example.com/login.php
❌ https://example.com/images/logo.png       ← 被过滤
❌ https://example.com/static/style.css      ← 被过滤
❌ https://example.com/js/app.js             ← 被过滤
❌ https://example.com/docs/manual.pdf       ← 被过滤

结果: 只爬取300个URL，都是动态页面（高效）
```

---

### 为什么需要排除静态资源？

#### 原因1: 提高效率
- 图片、字体、视频等静态资源**不包含业务逻辑**
- 爬取这些文件**浪费时间和带宽**
- 过滤后可以**专注于动态页面和API**

#### 原因2: 减少无效请求
```
不过滤静态资源:
- 爬取1000个URL，其中700个是图片/字体/CSS/JS
- 实际有价值的只有300个
- 效率: 30%

过滤静态资源:
- 只爬取300个有价值的URL
- 效率: 100%
```

#### 原因3: 避免误报
- 敏感信息检测不需要扫描图片、字体等
- 减少误报，提高准确性

---

### 推荐配置

#### 配置1: 基础过滤（推荐）
```json
{
  "exclude_extensions": [
    "jpg", "jpeg", "png", "gif", "svg", "ico",
    "css", "js", "woff", "woff2", "ttf", "eot"
  ]
}
```
**说明**: 排除图片、样式、脚本、字体

#### 配置2: 完整过滤（推荐⭐）
```json
{
  "exclude_extensions": [
    "jpg", "jpeg", "png", "gif", "svg", "ico",
    "css", "js", "woff", "woff2", "ttf", "eot",
    "mp4", "mp3", "avi", "mov",
    "pdf", "doc", "docx", "xls", "xlsx",
    "zip", "rar", "tar", "gz"
  ]
}
```
**说明**: 排除所有静态资源和文档

#### 配置3: 不过滤（不推荐）
```json
{
  "exclude_extensions": []
}
```
**说明**: 只在特殊场景使用（如需要分析JS文件中的敏感信息）

---

### 特殊场景

#### 场景1: 需要分析JS文件
```json
{
  "exclude_extensions": [
    "jpg", "jpeg", "png", "gif", "svg", "ico",
    "woff", "woff2", "ttf", "eot",
    "mp4", "mp3", "avi", "mov"
  ]
  // 注意: 不排除 js 和 css
}
```
**说明**: JS文件可能包含API端点、敏感信息

#### 场景2: 只爬取特定扩展名
使用 `include_extensions` 替代：
```json
{
  "include_extensions": ["php", "jsp", "aspx", "do", "action"],
  "exclude_extensions": []
}
```
**说明**: 只爬取动态页面

---

### 与其他配置的关系

#### 优先级
```
exclude_regex > exclude_extensions > include_extensions > include_regex
```

#### 组合使用
```json
{
  "scope_settings": {
    // 只包含API路径
    "include_paths": ["/api/*", "/v1/*"],
    
    // 排除静态资源扩展名
    "exclude_extensions": ["jpg", "png", "css", "js"],
    
    // 进一步用正则排除
    "exclude_regex": ".*\\.(jpg|png|gif)$"
  }
}
```

---

## 📝 问题2: 敏感信息规则如何配置？

### 方式1: 使用规则预设（推荐⭐）

#### 最小规则集（性能优先）
```json
{
  "sensitive_detection_settings": {
    "enabled": true,
    "rules_file": "sensitive_rules_minimal.json"
  }
}
```
**说明**: 只检测10个最高危规则（云存储密钥、私钥、数据库密码等）

#### 标准规则集（推荐）
```json
{
  "sensitive_detection_settings": {
    "enabled": true,
    "rules_file": "sensitive_rules_standard.json"
  }
}
```
**说明**: 40+个规则，覆盖常见场景

#### 完整规则集（全面扫描）
```json
{
  "sensitive_detection_settings": {
    "enabled": true,
    "rules_file": "sensitive_rules_config.json"
  }
}
```
**说明**: 所有规则，最全面

---

### 方式2: 自定义规则文件

创建自己的规则文件：
```json
{
  "description": "我的公司敏感规则",
  "version": "1.0",
  "rules": {
    "公司内部API密钥": {
      "pattern": "MYCOMPANY_[A-Z0-9]{32}",
      "severity": "HIGH",
      "mask": true,
      "description": "公司内部API密钥"
    }
  }
}
```

使用：
```json
{
  "sensitive_detection_settings": {
    "rules_file": "./my_company_rules.json"
  }
}
```

---

### 方式3: 命令行覆盖

```bash
# 使用最小规则集
./main.exe -url https://example.com -sensitive-rules sensitive_rules_minimal.json

# 使用自定义规则
./main.exe -url https://example.com -sensitive-rules my_rules.json

# 禁用敏感检测
./main.exe -url https://example.com -sensitive-detect=false
```

---

## 📝 问题3: 配置文件太多，如何简化？

### 当前配置文件
```
example_config_optimized.json     (主配置文件 250行)
sensitive_rules_config.json       (敏感规则 380行)
sensitive_rules_minimal.json      (最小规则 50行)
sensitive_rules_standard.json     (标准规则 150行)
```

### 简化方案

#### 方案1: 只用一个配置文件（推荐新手）
```json
{
  "target_url": "https://example.com",
  "depth_settings": { "max_depth": 5 },
  "sensitive_detection_settings": {
    "enabled": true,
    "rules_file": "sensitive_rules_standard.json"
  }
}
```

使用：
```bash
./main.exe -config simple_config.json
```

#### 方案2: 使用命令行 + 默认配置（推荐熟练用户）
```bash
./main.exe -url https://example.com -depth 5 -workers 20
```
**说明**: 不需要配置文件，所有配置都在命令行

#### 方案3: 使用预设场景（推荐⭐）
```bash
# 使用深度扫描预设
./main.exe -url https://example.com -config config_presets/deep_scan.json

# 使用API发现预设
./main.exe -url https://example.com -config config_presets/api_discovery.json
```

---

## 📝 问题4: 代码内置规则和外部规则有什么区别？

### 对比

| 特性 | 代码内置规则 | 外部规则文件 |
|------|------------|------------|
| 位置 | `core/sensitive_info_detector.go` | `sensitive_rules_*.json` |
| 数量 | 35个（v3.0将减少到5个） | 10-100+个 |
| 修改 | 需要重新编译代码 | 直接编辑JSON文件 |
| 用途 | 作为后备规则 | 主要规则来源 |
| 优先级 | 低 | 高（会覆盖内置规则） |

### 推荐使用

**日常使用**: 使用外部规则文件
```bash
./main.exe -url https://example.com -sensitive-rules sensitive_rules_standard.json
```

**极简使用**: 不指定规则文件（使用内置规则）
```bash
./main.exe -url https://example.com
# 会使用代码内置的5个核心规则
```

**自定义使用**: 创建自己的规则文件
```bash
./main.exe -url https://example.com -sensitive-rules my_rules.json
```

---

## 📝 问题5: 如何选择合适的规则集？

### 决策树

```
需要检测敏感信息吗?
├─ 否 → -sensitive-detect=false
└─ 是
    ├─ 快速扫描（时间紧）
    │   └─ sensitive_rules_minimal.json (10个规则)
    │
    ├─ 日常扫描（推荐）
    │   └─ sensitive_rules_standard.json (40个规则)
    │
    ├─ 全面扫描（安全审计）
    │   └─ sensitive_rules_config.json (100+个规则)
    │
    └─ 特定场景（自定义）
        └─ 创建自己的规则文件
```

### 性能对比

| 规则集 | 规则数量 | 性能影响 | 覆盖率 | 推荐场景 |
|--------|---------|---------|--------|---------|
| minimal | 10 | < 2% | 60% | 快速扫描 |
| standard | 40 | < 5% | 90% | 日常使用⭐ |
| full | 100+ | < 10% | 100% | 全面审计 |
| custom | 自定义 | 视规则数 | 视需求 | 特定场景 |

---

## 💡 最佳实践建议

### 建议1: 始终排除静态资源
```json
{
  "exclude_extensions": [
    "jpg", "png", "css", "js", "woff", "ttf",
    "mp4", "pdf", "zip"
  ]
}
```

### 建议2: 使用标准规则集
```json
{
  "sensitive_detection_settings": {
    "rules_file": "sensitive_rules_standard.json"
  }
}
```

### 建议3: 只检测高危敏感信息
```json
{
  "sensitive_detection_settings": {
    "min_severity": "HIGH"
  }
}
```

### 建议4: 使用配置文件而不是命令行
```bash
# 不推荐（参数太多）
./main.exe -url https://example.com -depth 5 -workers 20 -exclude-ext "jpg,png" ...

# 推荐（使用配置文件）
./main.exe -config my_config.json
```

---

## 🚀 快速上手

### 场景1: 我是新手，想快速开始
```bash
./main.exe -url https://example.com
```
**说明**: 使用默认配置，自动启用标准规则集

### 场景2: 我想要最大性能
```bash
./main.exe -url https://example.com \
  -exclude-ext "jpg,png,css,js,woff,ttf,mp4,pdf,zip" \
  -sensitive-detect=false \
  -workers 50
```

### 场景3: 我想要最全面的扫描
```bash
./main.exe -url https://example.com \
  -config config_presets/deep_scan.json \
  -sensitive-rules sensitive_rules_config.json
```

---

## 📚 相关文档

- `PARAMETERS_GUIDE.md` - 参数使用指南
- `SENSITIVE_RULES_ANALYSIS.md` - 敏感规则分析
- `README.md` - 项目总览
- `CONFIG_GUIDE.md` - 配置指南

---

**总结**:
1. `exclude_extensions` 用于过滤静态资源，提高效率
2. 敏感规则推荐使用外部文件，灵活可配置
3. 配置简化：使用预设场景或命令行参数
4. 日常使用推荐 `sensitive_rules_standard.json`

