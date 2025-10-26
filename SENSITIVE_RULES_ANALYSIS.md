# 敏感信息规则重复问题分析与解决方案

## 📊 当前问题

### 问题1: 规则来源重复

**代码内置规则** (`core/sensitive_info_detector.go`)
- 位置: `initializePatterns()` 方法
- 数量: 约35个规则
- 特点: 硬编码在代码中，修改需要重新编译

**外部规则文件** (`sensitive_rules_config.json`)
- 位置: JSON配置文件
- 数量: 约40个规则
- 特点: 可以动态加载，无需重新编译

### 问题2: 存在重复的规则

| 规则类型 | 代码内置 | 外部文件 | 重复 |
|---------|---------|---------|------|
| AWS Access Key | ✅ | ✅ | ❌ 重复 |
| AWS Secret Key | ✅ | ✅ | ❌ 重复 |
| AWS S3 Bucket | ✅ | ✅ | ❌ 重复 |
| Google API Key | ✅ | ✅ | ❌ 重复 |
| 阿里云AccessKey | ✅ | ✅ | ❌ 重复 |
| 阿里云OSS | ✅ | ✅ | ❌ 重复 |
| 腾讯云SecretId | ✅ | ✅ | ❌ 重复 |
| 腾讯云COS | ✅ | ✅ | ❌ 重复 |
| JWT Token | ✅ | ✅ | ❌ 重复 |
| GitHub Token | ✅ | ✅ | ❌ 重复 |
| Slack Token | ✅ | ✅ | ❌ 重复 |
| 数据库连接字符串 | ✅ | ✅ | ❌ 重复 |
| ... | ... | ... | ... |

**重复率**: 约80%的规则存在重复

---

## 🎯 解决方案

### 方案1: 只保留外部规则文件（推荐⭐）

**优点**:
- ✅ 无需重新编译即可更新规则
- ✅ 用户可以自定义规则
- ✅ 规则统一管理，避免混乱
- ✅ 支持规则版本控制

**缺点**:
- ⚠️ 如果用户不提供规则文件，需要有默认规则

**实现**:
1. 移除代码内置规则
2. 程序启动时检查是否有规则文件
3. 如果没有，使用内置的最小规则集（5-10个最重要的规则）
4. 如果有，完全使用外部规则文件

---

### 方案2: 合并规则模式（当前实现）

**当前逻辑**:
```go
// core/sensitive_info_detector.go
func NewSensitiveInfoDetector() *SensitiveInfoDetector {
    sid := &SensitiveInfoDetector{...}
    sid.initializePatterns()  // 加载内置规则
    return sid
}

// 可以后续加载外部规则
func (sid *SensitiveInfoDetector) MergeRulesFromFile(filename string) error {
    // 外部规则会覆盖同名内置规则
}
```

**问题**:
- 内置规则和外部规则都会加载
- 同名规则会被覆盖，但不同名的规则会累加
- 用户不清楚最终使用了哪些规则

---

### 方案3: 配置文件合并（解决两个配置文件问题）⭐

**当前情况**:
- `example_config_optimized.json` - 主配置文件（250行）
- `sensitive_rules_config.json` - 敏感规则配置（380行）

**问题**: 需要维护两个配置文件，比较繁琐

**解决方案**: 将敏感规则合并到主配置文件中

**合并后的结构**:
```json
{
  "target_url": "https://example.com",
  "depth_settings": { ... },
  "scope_settings": { ... },
  
  "sensitive_detection_settings": {
    "enabled": true,
    "scan_response_body": true,
    "scan_response_headers": true,
    "min_severity": "LOW",
    
    "rules": {
      "AWS S3 Access Key": {
        "pattern": "(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}",
        "severity": "HIGH",
        "mask": true,
        "description": "AWS Access Key ID"
      },
      "阿里云OSS AccessKey": {
        "pattern": "(?i)(aliyun|oss)[_-]?access[_-]?key[_-]?(id|ID)['\"]?\\s*[:=]\\s*['\"]?(LTAI[A-Za-z0-9]{12,20})",
        "severity": "HIGH",
        "mask": true,
        "description": "阿里云OSS AccessKey ID"
      }
      // ... 更多规则
    }
  }
}
```

**优点**:
- ✅ 只需要维护一个配置文件
- ✅ 配置更集中，更容易管理
- ✅ 用户可以快速启用/禁用敏感检测
- ✅ 支持规则的增删改

**缺点**:
- ⚠️ 配置文件会变得更大（约600行）
- ⚠️ 需要修改代码加载逻辑

---

## 💡 最佳实践建议

### 推荐方案: 方案1 + 方案3 组合

**第一步**: 清理代码内置规则
```go
// core/sensitive_info_detector.go
func (sid *SensitiveInfoDetector) initializePatterns() {
    // 只保留5个最关键的规则作为后备
    // 1. AWS Access Key
    // 2. 私钥文件
    // 3. JWT Token
    // 4. 数据库连接字符串
    // 5. 中国身份证号
}
```

**第二步**: 保留独立的敏感规则文件
- 不合并到主配置文件（配置文件会太大）
- 但提供一个简化的规则文件选项

**第三步**: 提供三级规则文件
```
sensitive_rules_minimal.json    (10个规则，快速扫描)
sensitive_rules_standard.json   (40个规则，标准扫描) ← 默认
sensitive_rules_full.json       (100个规则，全面扫描)
```

**第四步**: 在主配置文件中引用规则文件
```json
{
  "sensitive_detection_settings": {
    "enabled": true,
    "rules_file": "sensitive_rules_standard.json",
    "rules_preset": "standard"  // minimal | standard | full | custom
  }
}
```

---

## 🔧 实施步骤

### 步骤1: 清理内置规则（减少重复）

**修改文件**: `core/sensitive_info_detector.go`

```go
func (sid *SensitiveInfoDetector) initializePatterns() {
    // 🔧 只保留5个最核心的规则作为后备
    // 用户没有提供规则文件时使用这些规则
    
    sid.addPattern("Private Key",
        regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE KEY-----`),
        "HIGH", true)
    
    sid.addPattern("AWS Access Key",
        regexp.MustCompile(`(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}`),
        "HIGH", true)
    
    sid.addPattern("Database Connection",
        regexp.MustCompile(`(?i)(mysql|postgres|mongodb)://[^:]+:[^@]+@[^/]+`),
        "HIGH", true)
    
    sid.addPattern("JWT Token",
        regexp.MustCompile(`eyJ[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*`),
        "MEDIUM", true)
    
    sid.addPattern("Chinese ID Card",
        regexp.MustCompile(`[1-9]\d{5}(18|19|20)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]`),
        "HIGH", true)
    
    // 其他规则从外部文件加载
}
```

---

### 步骤2: 创建规则预设文件

**文件1**: `sensitive_rules_minimal.json` (最小规则集)
```json
{
  "description": "最小规则集 - 只检测最高危的10个规则",
  "version": "3.0",
  "rules": {
    "AWS S3 Access Key": {...},
    "阿里云OSS AccessKey": {...},
    "腾讯云COS SecretId": {...},
    "私钥文件": {...},
    "数据库连接字符串": {...},
    "微信AppSecret": {...},
    "支付宝私钥": {...},
    "管理员密码": {...},
    "中国身份证": {...},
    "内网IP": {...}
  }
}
```

**文件2**: `sensitive_rules_standard.json` (标准规则集)
- 即当前的 `sensitive_rules_config.json`

**文件3**: `sensitive_rules_full.json` (完整规则集)
- 扩展到100+规则

---

### 步骤3: 修改主配置文件引用方式

**修改文件**: `example_config_optimized.json`

```json
{
  "sensitive_detection_settings": {
    "enabled": true,
    "scan_response_body": true,
    "scan_response_headers": true,
    "min_severity": "LOW",
    
    // 🆕 新增：规则预设选项
    "rules_preset": "standard",
    
    // 原有：自定义规则文件路径
    "rules_file": "./sensitive_rules_config.json",
    
    // 说明：
    // - rules_preset: minimal | standard | full
    // - 如果指定了 rules_file，则忽略 rules_preset
    // - 如果都不指定，使用代码内置的最小规则集
    
    "output_file": "",
    "realtime_output": true
  }
}
```

---

### 步骤4: 更新加载逻辑

**修改文件**: `cmd/spider/main.go`

```go
// 初始化敏感信息检测器
sensitiveDetector := core.NewSensitiveInfoDetector()

// 加载规则的优先级：
// 1. 命令行指定的规则文件
// 2. 配置文件中的 rules_file
// 3. 配置文件中的 rules_preset
// 4. 使用内置最小规则集

if sensitiveRulesFile != "" {
    // 命令行指定的规则文件（最高优先级）
    err := sensitiveDetector.LoadRulesFromFile(sensitiveRulesFile)
    if err != nil {
        log.Fatalf("加载敏感规则失败: %v", err)
    }
} else if cfg.SensitiveDetectionSettings.RulesFile != "" {
    // 配置文件中指定的规则文件
    err := sensitiveDetector.LoadRulesFromFile(cfg.SensitiveDetectionSettings.RulesFile)
    if err != nil {
        log.Warnf("加载敏感规则失败，使用内置规则: %v", err)
    }
} else if preset := cfg.SensitiveDetectionSettings.RulesPreset; preset != "" {
    // 使用规则预设
    presetFile := fmt.Sprintf("sensitive_rules_%s.json", preset)
    err := sensitiveDetector.LoadRulesFromFile(presetFile)
    if err != nil {
        log.Warnf("加载预设规则失败，使用内置规则: %v", err)
    }
}
// 如果都没有，使用initializePatterns()中的内置规则

log.Infof("敏感信息检测器已初始化，共 %d 条规则", len(sensitiveDetector.GetPatterns()))
```

---

## 📊 对比总结

| 方案 | 内置规则 | 外部规则 | 配置文件数量 | 灵活性 | 推荐度 |
|------|---------|---------|------------|--------|--------|
| 当前实现 | 35个 | 40个 | 2个 | ⭐⭐ | ❌ |
| 方案1 | 5个（后备） | 全部 | 2个 | ⭐⭐⭐⭐ | ✅ |
| 方案2 | 35个 | 合并 | 2个 | ⭐⭐⭐ | ⚠️ |
| 方案3 | 35个 | 40个 | 1个（合并） | ⭐⭐⭐ | ⚠️ |
| **推荐方案** | **5个** | **预设** | **1+3** | **⭐⭐⭐⭐⭐** | **✅✅** |

**推荐方案说明**:
- 1个主配置文件: `example_config_optimized.json`
- 3个预设规则文件: `minimal.json` / `standard.json` / `full.json`
- 5个内置后备规则（代码中）
- 用户可以创建自己的自定义规则文件

---

## 🎯 用户使用体验

### 使用场景1: 快速扫描（新手）
```bash
./main.exe -url https://example.com
# 自动使用标准规则集（40个规则）
```

### 使用场景2: 最小规则集（性能优先）
```bash
./main.exe -url https://example.com -config config.json
```
```json
// config.json
{
  "sensitive_detection_settings": {
    "rules_preset": "minimal"  // 只检测10个最重要的规则
  }
}
```

### 使用场景3: 全面扫描
```bash
./main.exe -url https://example.com -config config.json
```
```json
{
  "sensitive_detection_settings": {
    "rules_preset": "full"  // 检测100+规则
  }
}
```

### 使用场景4: 自定义规则
```bash
./main.exe -url https://example.com -sensitive-rules my_company_rules.json
```

---

## 📝 总结

### 问题
1. ❌ 代码内置规则和外部规则80%重复
2. ❌ 需要维护两个配置文件（主配置+敏感规则）
3. ❌ 用户不清楚最终使用了哪些规则

### 解决方案
1. ✅ 内置规则减少到5个最核心规则（作为后备）
2. ✅ 保持独立的规则文件（不合并到主配置，避免文件过大）
3. ✅ 提供3个预设规则文件（minimal/standard/full）
4. ✅ 在主配置文件中添加 `rules_preset` 选项
5. ✅ 清晰的加载优先级：命令行 > 配置文件rules_file > 配置文件rules_preset > 内置规则

### 优点
- ✅ 减少重复，代码更清晰
- ✅ 提高灵活性，用户可以选择规则集大小
- ✅ 配置简单，一个主配置文件 + 可选的规则文件
- ✅ 向后兼容，现有的规则文件依然可用

---

## 🚀 实施建议

**第一步**: 创建规则预设文件（立即可做）
- `sensitive_rules_minimal.json`
- `sensitive_rules_standard.json` (重命名现有文件)
- `sensitive_rules_full.json`

**第二步**: 更新文档（立即可做）
- 在 README.md 中说明规则预设选项
- 添加使用示例

**第三步**: 修改代码（可选，下个版本）
- 清理内置规则，只保留5个核心规则
- 添加规则预设加载逻辑
- 更新配置文件结构

**第四步**: 发布新版本
- v3.1: 规则优化版
- 更新 CHANGELOG.md

