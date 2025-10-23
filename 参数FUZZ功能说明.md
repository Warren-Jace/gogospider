# Spider-golang 参数FUZZ功能详解

## ✅ 功能确认

**当前程序具备完整的参数FUZZ功能！** 主要实现在 `core/param_handler.go` 文件中。

---

## 🎯 核心FUZZ功能

### 1. **常规参数变体生成** (`GenerateParamVariations`)
**代码位置**: param_handler.go: 181-329行

#### 功能说明：
为带参数的URL自动生成多种变体，用于发现隐藏功能和测试不同参数组合。

#### 生成的变体类型：

| 变体类型 | 示例 | 说明 |
|---------|------|------|
| **原始URL** | `artists.php?artist=1` | 保留原始URL |
| **添加常见参数** | `artists.php?artist=1&id=1` | 添加id/page/category/product/user/token |
| **参数值变化** | `artists.php?artist=admin` | 使用admin/test/debug/123等值 |
| **HPP参数污染** | `artists.php?artist=1&artist=duplicate_value` | 重复参数测试HTTP参数污染 |
| **移除参数** | `artists.php?artist=` | 测试参数缺失情况 |
| **特定站点变体** | `cart.php?price=199&addcart=1` | 针对特定页面的专用参数 |

#### 示例代码：
```go
// 为URL生成变体
variations := paramHandler.GenerateParamVariations("http://testphp.vulnweb.com/artists.php?artist=1")

// 输出示例：
// artists.php?artist=1
// artists.php?artist=1&id=1
// artists.php?artist=1&id=admin
// artists.php?artist=1&page=1
// artists.php?artist=1&artist=duplicate_value (HPP)
```

---

### 2. **安全测试参数变体** (`GenerateSecurityTestVariations`)
**代码位置**: param_handler.go: 540-624行

#### 功能说明：
专门用于安全漏洞扫描，生成包含攻击payload的URL变体。

#### 支持的漏洞类型：

| 漏洞类型 | Payload数量 | Payload示例 |
|---------|------------|------------|
| **SQL注入** | 5个 | `'`, `"`, `1' OR '1'='1`, `1" OR "1"="1`, `'; DROP TABLE users; --` |
| **XSS跨站脚本** | 3个 | `<script>alert(1)</script>`, `<img src=x onerror=alert(1)>`, `javascript:alert(1)` |
| **文件包含(LFI)** | 4个 | `../../../etc/passwd`, `..\..\windows\system32\drivers\etc\hosts` |
| **命令注入** | 5个 | `; ls`, `| whoami`, `&& dir`, `$(id)`, `` `whoami` `` |
| **隐藏参数发现** | 7个 | `debug=1`, `test=1`, `admin=1`, `dev=1`, `backup=1`, `config=1` |

#### 示例：
```go
// 为URL生成安全测试变体
securityVars := paramHandler.GenerateSecurityTestVariations("http://testphp.vulnweb.com/artists.php?artist=1")

// 输出示例：
// artists.php?artist='                    (SQL注入测试)
// artists.php?artist=' OR '1'='1          (SQL注入测试)
// artists.php?artist=<script>alert(1)</script>  (XSS测试)
// artists.php?artist=../../../etc/passwd  (文件包含测试)
// artists.php?artist=1; ls                (命令注入测试)
// artists.php?artist=1&debug=1            (隐藏参数测试)
```

---

### 3. **参数模糊测试列表生成** (`GenerateParameterFuzzList`)
**代码位置**: param_handler.go: 627-692行

#### 功能说明：
在不知道具体参数的情况下，暴力枚举常见参数名，发现隐藏的功能入口。

#### 参数字典（80+个常见参数）：

```
【通用参数】
id, page, limit, offset, sort, order, search, q, query, filter, 
category, type, status, action, method, format

【用户相关】
user, username, userid, uid, email, password, pass, pwd, token, 
auth, session, key, api_key, access_token

【文件相关】
file, filename, path, dir, folder, upload, download, image, img, 
pic, photo, document, doc, pdf

【数据库相关】
table, column, field, record, row, data, value, insert, update, 
delete, select, where, join

【系统相关】
cmd, command, exec, system, shell, script, function, class, 
method, module, plugin, extension, callback

【调试相关】
debug, test, dev, development, staging, prod, production, admin, 
administrator, root, config, settings, options

【重定向相关】
redirect, return, next, continue, url, link, ref, referer, target, 
destination, forward, back, home, exit

【特殊功能】
preview, view, show, display, print, export, import, backup, 
restore, reset, clear, clean, flush, cache
```

#### 每个参数的测试值：
- 基本值: `1`, `test`, `admin`, `../`, `null`, `true`, `false`
- 空值: `param=`
- 数组: `param[]=1`, `param[0]=1`

#### 示例：
```go
// 生成完整的参数fuzz列表
fuzzList := paramHandler.GenerateParameterFuzzList("http://testphp.vulnweb.com/test.php")

// 输出示例（80个参数 × 9个测试值 = 720个测试URL）：
// test.php?id=1
// test.php?id=test
// test.php?id=admin
// test.php?id=../
// test.php?id=
// test.php?id[]=1
// test.php?page=1
// test.php?debug=1
// ... (共约720个URL)
```

---

### 4. **POST请求参数FUZZ** (`GeneratePOSTVariations`)
**代码位置**: param_handler.go: 715-770行

#### 功能说明：
针对POST表单生成安全测试变体，支持所有常见攻击类型。

#### 支持的测试类型：

| 类型 | 说明 | 示例 |
|------|------|------|
| **SQL注入** | 5种payload | `uname='`, `uname=' OR '1'='1` |
| **XSS** | 3种payload | `uname=<script>alert(1)</script>` |
| **参数污染** | 重复参数 | `uname=admin&uname=duplicate_value` |
| **空值测试** | 空参数值 | `uname=&pass=123` |
| **数组测试** | 数组参数 | `uname[]=admin` |

#### 示例：
```go
// 原始POST请求
postReq := POSTRequest{
    URL: "http://testphp.vulnweb.com/userinfo.php",
    Method: "POST",
    Parameters: map[string]string{
        "uname": "admin",
        "pass": "123456",
    },
}

// 生成FUZZ变体
variations := paramHandler.GeneratePOSTVariations(postReq)

// 输出示例（约50+个变体）：
// POST userinfo.php: uname=admin&pass=123456         (原始)
// POST userinfo.php: uname='&pass=123456             (SQL注入)
// POST userinfo.php: uname=' OR '1'='1&pass=123456   (SQL注入)
// POST userinfo.php: uname=<script>alert(1)</script>&pass=123456  (XSS)
// POST userinfo.php: uname=admin&uname=duplicate&pass=123456  (参数污染)
// POST userinfo.php: uname=&pass=123456              (空值)
// POST userinfo.php: uname[]=admin&pass=123456       (数组)
```

---

### 5. **参数安全分析** (`AnalyzeParameterSecurity`)
**代码位置**: param_handler.go: 496-537行

#### 功能说明：
自动识别参数的安全风险，标记高危参数，优先进行安全测试。

#### 风险分类：

| 风险级别 | 参数类型 | 示例参数 | 检测内容 |
|---------|---------|---------|---------|
| **级别3 (高危)** | 危险参数 | file, path, cmd, exec, system | 可能的RCE/文件包含 |
| **级别3 (高危)** | 文件包含 | file, filename, path, include | 文件包含漏洞 |
| **级别2 (中危)** | 安全参数 | debug, admin, password, token | 敏感功能/信息泄露 |
| **级别2 (中危)** | SQL注入 | id, user, search, query | SQL注入风险 |
| **级别2 (中危)** | XSS | message, comment, content | XSS风险 |
| **级别1 (低危)** | 常规参数 | page, limit, sort | 一般参数 |

#### 示例：
```go
// 分析参数安全性
risk, level := paramHandler.AnalyzeParameterSecurity("file")
// 输出: "FILE_INCLUSION: 可能存在文件包含漏洞", 3

risk, level = paramHandler.AnalyzeParameterSecurity("id")
// 输出: "SQL_INJECTION: 可能存在SQL注入漏洞", 2
```

---

### 6. **多源参数发现** (`DiscoverParametersFromMultipleSources`)
**代码位置**: param_handler.go: 332-366行

#### 功能说明：
从多个来源自动发现隐藏参数，提高FUZZ覆盖率。

#### 参数来源：

| 来源 | 提取方法 | 示例 |
|------|---------|------|
| **HTML表单** | 提取input/select/textarea的name属性 | `<input name="username">` → username |
| **JavaScript代码** | 提取变量名、对象属性、API参数 | `var userId = 123` → userId |
| **HTTP响应头** | 提取Cookie、自定义头参数 | `Set-Cookie: session=xxx` → session |
| **HTML注释** | 提取注释中的参数引用 | `<!-- ?debug=1 -->` → debug |
| **URL查询参数** | 提取URL中的参数 | `?id=1&page=2` → id, page |
| **data-*属性** | 提取HTML5 data属性 | `data-user-id="123"` → user-id |

#### 示例：
```go
// 从多个来源发现参数
params := paramHandler.DiscoverParametersFromMultipleSources(htmlContent, jsContent, headers)

// 可能发现：
// ["username", "password", "userId", "session", "debug", "token", "api_key"]
```

---

## 🔧 程序中的自动FUZZ触发

### 在爬取过程中自动执行

**代码位置**: spider.go: 410-454行

```go
// processParams 处理参数变体生成和安全分析
func (s *Spider) processParams(rawURL string) []string {
    // 1. 提取参数
    params, err := s.paramHandler.ExtractParams(rawURL)
    
    // 2. 安全分析（自动标记高危参数）
    for paramName := range params {
        risk, level := s.paramHandler.AnalyzeParameterSecurity(paramName)
        if level >= 2 { // 中等风险以上
            fmt.Printf("安全发现: SECURITY_PARAM: %s - %s (Risk Level: %d)\n", 
                paramName, risk, level)
        }
    }
    
    // 3. 生成常规参数变体
    variations := s.paramHandler.GenerateParamVariations(rawURL)
    
    // 4. 生成安全测试变体
    securityVariations := s.paramHandler.GenerateSecurityTestVariations(rawURL)
    variations = append(variations, securityVariations...)
    
    // 5. 打印生成的变体
    fmt.Printf("为URL %s 生成 %d 个参数变体（包括安全测试）\n", rawURL, len(variations))
    
    return variations
}
```

### 执行时机：

1. **静态爬虫响应时** (static_crawler.go: 533-548行)
   - 发现带参数的URL时自动生成变体

2. **递归爬取时** (spider.go: 410行)
   - 每个发现的URL都会进行参数分析

---

## 📊 FUZZ效果统计

### 单个URL能生成多少变体？

以 `artists.php?artist=1` 为例：

| 变体类型 | 数量 | 说明 |
|---------|------|------|
| 常规参数变体 | ~50个 | 添加6个常见参数 × 6个测试值 + HPP + 移除参数 |
| SQL注入 | 5个 | 5种SQL payload |
| XSS | 3个 | 3种XSS payload |
| 文件包含 | 4个 | 4种LFI payload |
| 命令注入 | 5个 | 5种命令注入payload |
| 隐藏参数 | 7个 | 7个调试/管理参数 |
| **总计** | **~74个** | **一个URL生成74个测试变体** |

### 对整个网站的FUZZ规模

假设爬取到48个URL（如uu.txt），其中30个带参数：

```
30个带参数URL × 74个变体 = 2,220个测试URL
18个无参数URL × 720个参数fuzz = 12,960个测试URL
总计: 约 15,180 个测试URL
```

---

## 💡 使用示例

### 示例1: 对单个URL进行FUZZ

```go
package main

import (
    "fmt"
    "spider-golang/core"
)

func main() {
    ph := core.NewParamHandler()
    
    // 目标URL
    url := "http://testphp.vulnweb.com/artists.php?artist=1"
    
    // 生成所有变体
    variations := ph.GenerateParamVariations(url)
    securityVars := ph.GenerateSecurityTestVariations(url)
    
    fmt.Printf("常规变体: %d 个\n", len(variations))
    fmt.Printf("安全测试变体: %d 个\n", len(securityVars))
    
    // 输出所有变体
    for _, v := range variations {
        fmt.Println(v)
    }
}
```

### 示例2: 对POST表单进行FUZZ

```go
package main

import (
    "fmt"
    "spider-golang/core"
)

func main() {
    ph := core.NewParamHandler()
    
    // POST请求
    postReq := core.POSTRequest{
        URL:    "http://testphp.vulnweb.com/userinfo.php",
        Method: "POST",
        Parameters: map[string]string{
            "uname": "admin",
            "pass":  "password",
        },
    }
    
    // 生成POST变体
    variations := ph.GeneratePOSTVariations(postReq)
    
    fmt.Printf("生成 %d 个POST测试变体\n", len(variations))
    
    for _, v := range variations {
        fmt.Printf("POST %s: %s\n", v.URL, v.Body)
    }
}
```

### 示例3: 自动参数发现和FUZZ

```go
package main

import (
    "fmt"
    "spider-golang/core"
)

func main() {
    ph := core.NewParamHandler()
    
    // 从HTML/JS/Headers中发现参数
    params := ph.DiscoverParametersFromMultipleSources(htmlContent, jsContent, headers)
    
    fmt.Printf("发现 %d 个参数: %v\n", len(params), params)
    
    // 为每个参数生成测试URL
    baseURL := "http://testphp.vulnweb.com/test.php"
    for _, param := range params {
        testURL := fmt.Sprintf("%s?%s=1", baseURL, param)
        
        // 分析参数安全性
        risk, level := ph.AnalyzeParameterSecurity(param)
        fmt.Printf("参数: %s, 风险: %s (级别%d)\n", param, risk, level)
        
        // 生成FUZZ变体
        if level >= 2 {
            variations := ph.GenerateSecurityTestVariations(testURL)
            fmt.Printf("  生成 %d 个安全测试变体\n", len(variations))
        }
    }
}
```

---

## 🎯 实际应用场景

### 1. 安全测试/渗透测试
- 自动化漏洞扫描
- SQL注入检测
- XSS漏洞检测
- 文件包含漏洞检测
- 命令注入检测

### 2. API测试
- 参数组合测试
- 边界值测试
- 异常输入测试
- 权限绕过测试

### 3. 功能发现
- 隐藏参数发现
- 调试接口发现
- 管理后台发现
- 未授权访问测试

### 4. 自动化测试
- 回归测试
- 兼容性测试
- 压力测试
- 边界测试

---

## 🔥 优势总结

| 特性 | 说明 | 优势 |
|------|------|------|
| **全自动** | 爬取过程中自动FUZZ | 无需手动配置 |
| **多维度** | 6种FUZZ策略 | 覆盖全面 |
| **智能化** | 自动参数发现+风险分析 | 精准高效 |
| **可扩展** | 模块化设计 | 易于定制 |
| **大规模** | 单URL可生成74+变体 | 测试深入 |

---

## ⚠️ 注意事项

1. **合法性**: 仅在授权的目标上使用FUZZ功能
2. **性能**: 大规模FUZZ会产生大量请求，注意速率限制
3. **存储**: 变体结果可能占用较多内存，注意资源管理
4. **误报**: 安全测试变体可能触发WAF，需要配合其他工具验证

---

## 📝 总结

**当前程序具备企业级的参数FUZZ能力！**

✅ **6种FUZZ策略**
✅ **80+个参数字典**
✅ **20+种攻击payload**
✅ **自动安全风险分析**
✅ **支持GET和POST请求**
✅ **智能参数发现**

完全满足安全测试、渗透测试、自动化测试的需求！

---

**生成日期**: 2025-10-23
**版本**: Spider-golang v2.5+


