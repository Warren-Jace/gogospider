# 🔒 GogoSpider 敏感信息检测系统

## 📋 快速答案

### Q1: 敏感信息是否保存到单独文件？
**✅ 是的！** 保存在独立文件中，文件名：`spider_{domain}_{timestamp}_sensitive.txt`

### Q2: 文件中是否包含来源地址和信息类型？
**✅ 完全包含！** 每条记录都有：
- 敏感信息类型（Type）
- 来源URL地址（SourceURL）
- 精确位置（Location）

### Q3: 敏感信息检测是否针对每个返回数据包？
**✅ 是的！** 每个HTTP响应都会进行检测

### Q4: 敏感信息检测是否是统一功能？
**✅ 完全统一！** 通过统一的检测器、配置和存储管理

---

## 🎯 核心功能

```
┌─────────────────────────────────────────────────────────────┐
│                  GogoSpider 敏感信息检测系统                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  🔍 检测范围：每个HTTP响应                                    │
│     ├─ 响应体（HTML内容）                                    │
│     └─ 响应头（HTTP Headers）                                │
│                                                             │
│  📝 记录信息：                                               │
│     ├─ Type:       敏感信息类型                              │
│     ├─ SourceURL:  来源URL地址                               │
│     ├─ Location:   精确行号                                  │
│     ├─ Severity:   严重程度                                  │
│     └─ Value:      脱敏后的值                                │
│                                                             │
│  💾 保存位置：独立文件                                        │
│     ├─ spider_{domain}_{time}_sensitive.txt                │
│     ├─ sensitive_{domain}_{time}.json                      │
│     ├─ sensitive_{domain}_{time}.csv                       │
│     ├─ sensitive_{domain}_{time}.html                      │
│     └─ sensitive_{domain}_{time}_summary.txt               │
│                                                             │
│  ⚙️  统一管理：                                              │
│     ├─ SensitiveDetector   (检测器)                        │
│     ├─ SensitiveManager    (管理器)                        │
│     └─ SensitiveFindings   (存储)                          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 📊 检测流程

```
HTTP请求 → 获取响应 → addResult()
                         │
                         ▼
                  ┌─────────────┐
                  │ 敏感信息检测 │
                  └──────┬──────┘
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
    扫描响应体      扫描响应头      应用规则
          │              │              │
          └──────────────┴──────────────┘
                         │
                         ▼
                  记录发现信息
                  ├─ Type
                  ├─ SourceURL
                  ├─ Location
                  └─ Severity
                         │
                         ▼
              统一存储 (sensitiveFindings)
                         │
                         ▼
                  爬取结束后导出
                  ├─ TXT  (文本)
                  ├─ JSON (数据)
                  ├─ CSV  (表格)
                  ├─ HTML (可视化)
                  └─ Summary (摘要)
```

---

## 📁 输出文件示例

### 1. 文本报告（.txt）

```
==========================================
   敏感信息泄露检测报告
==========================================

【高危发现】
[1] API密钥泄露
    来源URL: https://example.com/config.js  ← 来源地址
    位置: Line 15                          ← 精确位置
    值: sk_****xyz                         ← 脱敏值
```

### 2. JSON数据（.json）

```json
{
  "findings": [
    {
      "type": "API密钥泄露",
      "source_url": "https://example.com/config.js",
      "location": "Line 15",
      "severity": "HIGH",
      "value": "sk_****xyz"
    }
  ]
}
```

---

## 🚀 使用方法

### 方法1：默认使用（推荐）

```bash
# 敏感信息检测默认开启
./spider -url https://example.com -sensitive-rules sensitive_rules_standard.json
```

### 方法2：自定义配置

```json
// config.json
{
  "sensitive_detection_settings": {
    "enabled": true,
    "scan_response_body": true,
    "scan_response_headers": true,
    "rules_file": "sensitive_rules_standard.json"
  }
}
```

```bash
./spider -config config.json -url https://example.com
```

### 方法3：批量扫描

```bash
./spider -batch-file targets.txt -sensitive-rules sensitive_rules_standard.json
```

---

## 📖 查看结果

### 快速查看

```bash
# 1. 查看摘要（推荐第一步）
type sensitive_*_summary.txt

# 2. 查看详细报告
type spider_*_sensitive.txt

# 3. 浏览器查看（最直观）
start sensitive_*.html
```

### Excel分析

```bash
# 用Excel打开CSV文件
open sensitive_*.csv
```

---

## ⚙️ 配置选项

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `enabled` | 是否启用敏感信息检测 | `true` |
| `scan_response_body` | 是否扫描响应体 | `true` |
| `scan_response_headers` | 是否扫描响应头 | `true` |
| `min_severity` | 最低严重程度 | `LOW` |
| `rules_file` | 规则文件路径 | `sensitive_rules_standard.json` |
| `realtime_output` | 实时输出发现 | `true` |

---

## 🔧 自定义规则

### 规则文件格式

```json
{
  "rules": {
    "API密钥泄露": {
      "pattern": "sk_[a-zA-Z0-9]{32}",
      "severity": "HIGH",
      "mask": true,
      "description": "检测Stripe API密钥"
    },
    "邮箱地址": {
      "pattern": "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
      "severity": "MEDIUM",
      "mask": false,
      "description": "检测邮箱地址"
    }
  }
}
```

### 可用规则文件

- `sensitive_rules_minimal.json` - 最小规则集（仅高危）
- `sensitive_rules_standard.json` - 标准规则集（推荐）
- `sensitive_rules_config.json` - 自定义规则集

---

## 📊 支持的敏感信息类型

### 高危（HIGH）

- API密钥
- 数据库密码
- AWS访问密钥
- 私钥文件
- JWT Token

### 中危（MEDIUM）

- 内部IP地址
- 邮箱地址
- 电话号码
- 身份证号
- 银行卡号

### 低危（LOW）

- 注释中的TODO
- 调试信息
- 内部路径
- 版本号

---

## 🎯 实际案例

### 案例1：发现API密钥泄露

```
【高危发现】
[1] API密钥泄露
    来源URL: https://target.com/js/config.js
    位置: Line 15
    值: AIza****************************xyz
    
建议：立即撤销该API密钥并使用环境变量
```

### 案例2：发现数据库连接信息

```
【高危发现】
[2] 数据库连接字符串
    来源URL: https://target.com/config.php
    位置: Line 23
    值: mysql://admin:p****@localhost/db
    
建议：立即修改数据库密码
```

---

## 📈 统计信息

每次扫描后会生成详细统计：

```
【统计概况】
扫描页面数: 245
发现总数: 18（已去重）
受影响URL: 12

【严重程度分布】
🔴 高危: 3
🟡 中危: 9
🟢 低危: 6

【类型分布】
API密钥泄露: 2
邮箱地址: 8
内部IP: 3
```

---

## ✅ 核心特性

### ✅ 全面检测
- 每个HTTP响应都检测
- 响应体和响应头都扫描
- 支持动态内容

### ✅ 完整记录
- 敏感信息类型
- 来源URL地址
- 精确行号位置
- 严重程度分级

### ✅ 独立保存
- 单独的敏感信息文件
- 与爬取结果分离
- 多种格式可选

### ✅ 统一管理
- 统一的检测器
- 统一的配置
- 统一的导出

### ✅ 智能去重
- 自动去除重复发现
- 按类型+值+URL去重
- 提高报告质量

### ✅ 可视化报告
- 美观的HTML报告
- 颜色区分严重程度
- 支持移动端查看

---

## 📚 相关文档

- [敏感信息检测机制_完整检查报告.md](敏感信息检测机制_完整检查报告.md) - 详细检查报告
- [敏感信息功能_快速说明.md](敏感信息功能_快速说明.md) - 快速使用指南
- [敏感信息统一管理_使用指南.md](敏感信息统一管理_使用指南.md) - 统一管理系统
- [敏感信息收集结果_完整分析报告.md](敏感信息收集结果_完整分析报告.md) - 功能分析

---

## 🧪 快速测试

运行测试脚本：

```bash
# Windows
测试敏感信息检测.bat

# 自动完成：
# 1. 运行爬虫
# 2. 生成报告
# 3. 显示预览
```

---

## 💡 最佳实践

### 1. 选择合适的规则集

```bash
# 快速扫描：使用最小规则集
./spider -url https://target.com -sensitive-rules sensitive_rules_minimal.json

# 全面扫描：使用标准规则集
./spider -url https://target.com -sensitive-rules sensitive_rules_standard.json
```

### 2. 定期更新规则

```bash
# 定期检查和更新规则文件
# 添加新的敏感信息检测模式
```

### 3. 查看HTML报告

```bash
# 使用HTML报告进行可视化分析
start sensitive_*.html
```

### 4. 导出CSV进行数据分析

```bash
# 用Excel打开CSV进行深度分析
open sensitive_*.csv
```

---

## 🔒 安全建议

### 发现敏感信息后的处理步骤：

1. **立即评估影响范围**
   - 查看受影响的URL
   - 确定敏感信息类型
   - 评估严重程度

2. **采取紧急措施**
   - 撤销泄露的API密钥
   - 修改泄露的密码
   - 删除敏感文件

3. **修复源代码**
   - 移除硬编码的敏感信息
   - 使用环境变量
   - 使用密钥管理服务

4. **审查和监控**
   - 定期扫描
   - 监控异常访问
   - 建立安全流程

---

## ❓ 常见问题

### Q: 为什么有些敏感信息被脱敏了？
A: 为了保护安全，敏感信息会自动脱敏显示。完整值仅用于内部处理。

### Q: 如何添加自定义检测规则？
A: 编辑规则文件（JSON格式），添加新的正则表达式模式。

### Q: 误报怎么办？
A: 调整规则文件中的正则表达式，或设置更高的严重程度阈值。

### Q: 可以只检测高危敏感信息吗？
A: 可以，在配置文件中设置 `min_severity: "HIGH"`。

---

## 📞 技术支持

如有问题或建议：
1. 查看完整文档
2. 运行测试脚本
3. 提交Issue

---

**GogoSpider v4.2** - 专业的敏感信息检测系统

让您的Web安全扫描更全面、更专业！

---

生成时间：2025-10-28  
版本：v4.2  
状态：✅ 已验证

