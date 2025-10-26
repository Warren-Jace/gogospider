# 命令行参数迁移指南

## v2.x → v3.0 参数变更

为了简化使用，v3.0 将大部分参数移到了配置文件中。以下是参数迁移对照表。

---

## 保留的命令行参数（15个核心参数）

### ✅ 核心参数（必需）

| 参数 | 说明 | 旧版 | 新版 |
|------|------|------|------|
| `-url` | 目标URL | ✅ | ✅ |

### ✅ 基础参数

| 参数 | 说明 | 旧版 | 新版 |
|------|------|------|------|
| `-config` | 配置文件路径 | ✅ | ✅ |
| `-depth` | 最大爬取深度 | ✅ | ✅ |
| `-max-pages` | 最大页面数 | ✅ | ✅ |
| `-workers` | 并发工作线程数 | ✅ | ✅ |

### ✅ 模式参数

| 参数 | 说明 | 旧版 | 新版 |
|------|------|------|------|
| `-mode` | 爬取模式 | ✅ | ✅ |
| `-preset` | 预设场景配置 | ❌ | ✅ **新增** |

### ✅ 输出参数

| 参数 | 说明 | 旧版 | 新版 |
|------|------|------|------|
| `-output` | 输出目录 | ✅ | ✅ |
| `-json` | JSON输出 | ✅ | ✅ |
| `-quiet` | 静默模式 | ✅ | ✅ |

### ✅ 高级参数

| 参数 | 说明 | 旧版 | 新版 |
|------|------|------|------|
| `-proxy` | 代理服务器 | ✅ | ✅ |
| `-allow-subdomains` | 允许子域名 | ✅ | ✅ |
| `-batch-file` | 批量扫描文件 | ✅ | ✅ |

### ✅ 工具参数

| 参数 | 说明 | 旧版 | 新版 |
|------|------|------|------|
| `-version` | 显示版本 | ✅ | ✅ |
| `-help` | 显示帮助 | ✅ | ✅ |

---

## 移到配置文件的参数（约55个）

### 🔧 反检测设置

| 旧版参数 | 配置文件位置 |
|---------|------------|
| `-timeout` | `anti_detection_settings.timeout` |
| `-user-agent` | `anti_detection_settings.user_agents` |
| `-cookie-file` | 改用 `-headers` 参数或配置文件 |
| `-headers` | `anti_detection_settings` 中自定义 |
| `-ignore-robots` | 已移除，默认忽略 |

**迁移示例：**

```json
"anti_detection_settings": {
  "request_delay": 500000000,
  "random_delay": true,
  "timeout": 30,
  "retry_times": 3,
  "user_agents": [
    "Mozilla/5.0 ..."
  ]
}
```

### 🔧 Chrome/动态爬虫设置

| 旧版参数 | 配置文件位置 |
|---------|------------|
| `-chrome-path` | `strategy_settings.chrome_path` |

**迁移示例：**

```json
"strategy_settings": {
  "enable_dynamic_crawler": true,
  "chrome_path": "/path/to/chrome"
}
```

### 🔧 Fuzzing 参数（已移除）

| 旧版参数 | v3.0 状态 |
|---------|----------|
| `-fuzz` | ❌ 已移除（专注纯爬虫） |
| `-fuzz-params` | ❌ 已移除 |
| `-fuzz-dict` | ❌ 已移除 |

### 🔧 日志参数

| 旧版参数 | 配置文件位置 |
|---------|------------|
| `-log-level` | `log_settings.level` |
| `-log-file` | `log_settings.output_file` |
| `-log-format` | `log_settings.format` |
| `-show-metrics` | `log_settings.show_metrics` |

**迁移示例：**

```json
"log_settings": {
  "level": "INFO",
  "output_file": "",
  "format": "json",
  "show_metrics": true
}
```

### 🔧 JSON 输出参数

| 旧版参数 | 配置文件位置 |
|---------|------------|
| `-json-mode` | `output_settings.json_mode` |
| `-output-file` | `output_settings.output_file` |
| `-include-all` | `output_settings.include_all` |

**迁移示例：**

```json
"output_settings": {
  "format": "json",
  "output_file": "results.jsonl",
  "json_mode": "line",
  "include_all": true
}
```

### 🔧 速率控制参数

| 旧版参数 | 配置文件位置 |
|---------|------------|
| `-rate-limit-enable` | `rate_limit_settings.enabled` |
| `-rate-limit` | `rate_limit_settings.requests_per_second` |
| `-burst` | `rate_limit_settings.burst_size` |
| `-min-delay` | `rate_limit_settings.min_delay` |
| `-max-delay` | `rate_limit_settings.max_delay` |
| `-adaptive-rate` | `rate_limit_settings.adaptive` |
| `-min-rate` | `rate_limit_settings.adaptive_min_rate` |
| `-max-rate` | `rate_limit_settings.adaptive_max_rate` |

**迁移示例：**

```json
"rate_limit_settings": {
  "enabled": true,
  "requests_per_second": 50,
  "burst_size": 10,
  "adaptive": true,
  "adaptive_min_rate": 10,
  "adaptive_max_rate": 100
}
```

### 🔧 外部数据源参数

| 旧版参数 | 配置文件位置 |
|---------|------------|
| `-wayback` | `external_source_settings.enable_wayback_machine` |
| `-virustotal` | `external_source_settings.enable_virustotal` |
| `-vt-api-key` | `external_source_settings.virustotal_api_key` |
| `-commoncrawl` | `external_source_settings.enable_common_crawl` |
| `-external-timeout` | `external_source_settings.timeout` |

**迁移示例：**

```json
"external_source_settings": {
  "enabled": true,
  "enable_wayback_machine": true,
  "enable_virustotal": false,
  "virustotal_api_key": "",
  "enable_common_crawl": false,
  "timeout": 30
}
```

### 🔧 Scope 控制参数

| 旧版参数 | 配置文件位置 |
|---------|------------|
| `-include-domains` | `scope_settings.include_domains` |
| `-exclude-domains` | `scope_settings.exclude_domains` |
| `-include-paths` | `scope_settings.include_paths` |
| `-exclude-paths` | `scope_settings.exclude_paths` |
| `-include-regex` | `scope_settings.include_regex` |
| `-exclude-regex` | `scope_settings.exclude_regex` |
| `-include-ext` | `scope_settings.include_extensions` |
| `-exclude-ext` | `scope_settings.exclude_extensions` |

**迁移示例：**

```json
"scope_settings": {
  "enabled": true,
  "include_domains": ["*.example.com"],
  "exclude_domains": ["cdn.example.com"],
  "include_paths": ["/api/*"],
  "exclude_paths": ["/*.jpg"],
  "exclude_extensions": ["jpg", "png", "css"]
}
```

### 🔧 管道模式参数

| 旧版参数 | 配置文件位置 |
|---------|------------|
| `-pipeline` | `pipeline_settings.enabled` |
| `-stdin` | `pipeline_settings.enable_stdin` |
| `-simple` | 使用 `-quiet` 代替 |
| `-format` | `output_settings.format` |

**迁移示例：**

```json
"pipeline_settings": {
  "enabled": true,
  "enable_stdin": true,
  "enable_stdout": true,
  "quiet": true
}
```

### 🔧 敏感信息检测参数

| 旧版参数 | 配置文件位置 |
|---------|------------|
| `-sensitive-detect` | `sensitive_detection_settings.enabled` |
| `-sensitive-scan-body` | `sensitive_detection_settings.scan_response_body` |
| `-sensitive-scan-headers` | `sensitive_detection_settings.scan_response_headers` |
| `-sensitive-min-severity` | `sensitive_detection_settings.min_severity` |
| `-sensitive-output` | `sensitive_detection_settings.output_file` |
| `-sensitive-realtime` | `sensitive_detection_settings.realtime_output` |
| `-sensitive-rules` | `sensitive_detection_settings.rules_file` |

**迁移示例：**

```json
"sensitive_detection_settings": {
  "enabled": true,
  "scan_response_body": true,
  "scan_response_headers": true,
  "min_severity": "LOW",
  "rules_file": "./sensitive_rules_config.json",
  "realtime_output": true
}
```

### 🔧 批量扫描参数

| 旧版参数 | 配置文件位置 |
|---------|------------|
| `-batch-concurrency` | `batch_scan_settings.concurrency` |

**迁移示例：**

```json
"batch_scan_settings": {
  "enabled": true,
  "input_file": "targets.txt",
  "concurrency": 5,
  "output_dir": "./batch_results"
}
```

---

## 迁移步骤

### 步骤 1：创建配置文件

从模板开始：

```bash
# 复制优化配置模板
cp example_config_optimized.json my_config.json

# 或使用预设场景
cp config_presets/deep_scan.json my_config.json
```

### 步骤 2：迁移参数

将你原来的命令行参数写入配置文件：

**旧版命令：**
```bash
./spider \
  -url https://example.com \
  -depth 5 \
  -max-pages 1000 \
  -rate-limit 50 \
  -wayback \
  -include-domains "*.example.com" \
  -exclude-ext "jpg,png,css" \
  -log-level debug \
  -json \
  -output-file results.json
```

**新版命令：**
```bash
./spider -url https://example.com -config my_config.json
```

**my_config.json：**
```json
{
  "depth_settings": {
    "max_depth": 5,
    "max_pages": 1000
  },
  "rate_limit_settings": {
    "enabled": true,
    "requests_per_second": 50
  },
  "external_source_settings": {
    "enabled": true,
    "enable_wayback_machine": true
  },
  "scope_settings": {
    "enabled": true,
    "include_domains": ["*.example.com"],
    "exclude_extensions": ["jpg", "png", "css"]
  },
  "log_settings": {
    "level": "DEBUG"
  },
  "output_settings": {
    "format": "json",
    "output_file": "results.json"
  }
}
```

### 步骤 3：测试配置

```bash
# 使用 debug 模式查看最终配置
./spider -url https://example.com -config my_config.json -log-level debug
```

### 步骤 4：保存常用配置

将你的配置保存为预设：

```bash
# 保存到预设目录
cp my_config.json config_presets/my_preset.json

# 以后直接使用
./spider -url https://example.com -preset my_preset
```

---

## 常见迁移场景

### 场景 1：快速扫描

**旧版：**
```bash
./spider -url https://example.com -depth 3 -max-pages 200 -workers 5
```

**新版：**
```bash
./spider -url https://example.com -preset quick_scan
```

或：
```bash
./spider -url https://example.com -depth 3 -max-pages 200 -workers 5
```

### 场景 2：深度扫描 + 外部数据源

**旧版：**
```bash
./spider \
  -url https://example.com \
  -depth 8 \
  -wayback \
  -virustotal \
  -vt-api-key "YOUR_KEY" \
  -rate-limit 30
```

**新版：**
```bash
# 修改 config_presets/deep_scan.json 添加 VT 配置
./spider -url https://example.com -preset deep_scan
```

### 场景 3：API 发现

**旧版：**
```bash
./spider \
  -url https://example.com \
  -include-paths "/api/*,/v1/*" \
  -exclude-ext "jpg,png,css,html" \
  -json \
  -output-file api_results.json
```

**新版：**
```bash
./spider -url https://example.com -preset api_discovery
```

### 场景 4：批量扫描

**旧版：**
```bash
./spider \
  -batch-file targets.txt \
  -batch-concurrency 5 \
  -depth 4
```

**新版：**
```bash
./spider -batch-file targets.txt -preset batch_scan
```

---

## 兼容性说明

### 保留的兼容性

v3.0 仍然支持所有旧版命令行参数，但：
1. **建议** 使用配置文件
2. **推荐** 使用预设场景
3. **废弃** 的参数会显示警告

### 移除的功能

以下功能在 v3.0 中已移除：

1. **Fuzzing 功能**（`-fuzz`, `-fuzz-params`, `-fuzz-dict`）
   - 原因：专注于纯爬虫功能
   - 替代：使用其他专业 fuzzing 工具

2. **简单模式**（`-simple`）
   - 原因：与 `-quiet` 功能重复
   - 替代：使用 `-quiet` 参数

---

## 反馈和建议

如果你觉得某个参数应该保留在命令行，或者对迁移有任何问题，请提交 Issue。

---

**文档版本：** v3.0  
**最后更新：** 2025-10-26

