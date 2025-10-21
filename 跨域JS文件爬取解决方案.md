# 跨域JS文件URL提取解决方案

## 📋 问题分析

### 典型场景
```
目标网站: http://example.com
页面引用: <script src="https://cdn.example.net/static/app.js"></script>

app.js 内容包含:
- fetch('/api/users')
- window.location = '/admin/panel'
- axios.get('/products/list')
```

**问题核心**：
- JS文件托管在 `cdn.example.net`（跨域）
- 当前爬虫只爬取 `example.com` 域名
- 错过了JS中的 `/api/users`, `/admin/panel` 等目标域名的路径

---

## 🎯 解决思路（3个方案）

### 方案1：智能跨域资源白名单 ⭐⭐⭐⭐⭐
**核心思想**：允许下载跨域的静态资源（JS/CSS），但仍限制在目标域名爬取

#### 实现步骤：
```
第1步：识别资源类型
  ├─ JS文件: .js, application/javascript
  ├─ CSS文件: .css, text/css
  └─ 其他静态资源: 字体、图片等

第2步：跨域资源处理策略
  ├─ 如果是目标域名 → 正常爬取
  ├─ 如果是跨域 + 静态资源 → 下载并分析内容
  └─ 如果是跨域 + 非静态资源 → 记录但不爬取

第3步：从跨域JS中提取目标域名URL
  ├─ 下载JS文件内容
  ├─ 正则匹配: /api/*, /admin/*, 相对路径等
  ├─ 拼接成完整URL: http://目标域名/api/*
  └─ 加入爬取队列
```

**优点**：
- ✅ 不会漏掉JS中的重要路径
- ✅ 仍然遵循域名限制原则
- ✅ 代码改动小，逻辑清晰

**缺点**：
- ⚠ 需要解析JS内容（正则匹配）

---

### 方案2：CDN域名智能识别 ⭐⭐⭐⭐
**核心思想**：自动识别和信任常见的CDN域名

#### 实现步骤：
```
第1步：建立CDN域名特征库
  常见CDN: 
  - *.cloudflare.com
  - *.amazonaws.com
  - *.jsdelivr.net
  - *.unpkg.com
  - cdn.*, static.*, assets.*

第2步：同源域名识别
  - cdn.example.com (与example.com同主域)
  - static.example.com
  - assets.example.com

第3步：智能信任策略
  if (是CDN域名 OR 同主域名) AND (是静态资源):
    → 下载并分析
  else:
    → 跳过
```

**优点**：
- ✅ 自动识别常见场景
- ✅ 用户无需手动配置
- ✅ 智能化程度高

**缺点**：
- ⚠ 需要维护CDN特征库
- ⚠ 可能误判

---

### 方案3：配置化白名单 ⭐⭐⭐
**核心思想**：让用户手动配置允许的外部域名

#### 实现步骤：
```
第1步：配置文件设置
{
  "target_domain": "example.com",
  "allowed_external_domains": [
    "cdn.example.com",
    "static.example.com",
    "cdnjs.cloudflare.com"
  ],
  "analyze_external_js": true
}

第2步：爬取时检查
  if (domain == target_domain):
    → 完整爬取
  elif (domain in allowed_external_domains):
    → 仅下载分析，不爬取其链接
  else:
    → 跳过

第3步：从允许的外部JS中提取URL
  → 提取相对路径和目标域名的路径
  → 拼接后加入爬取队列
```

**优点**：
- ✅ 灵活可控
- ✅ 用户知道爬取了什么
- ✅ 安全性高

**缺点**：
- ⚠ 需要用户手动配置
- ⚠ 使用门槛稍高

---

## 🔥 推荐方案：方案1 + 方案2 混合

结合两者优点，实现最佳效果：

```go
// 伪代码
func shouldAnalyzeExternalResource(url string, targetDomain string) bool {
    domain := extractDomain(url)
    
    // 1. 是目标域名 - 完整爬取
    if domain == targetDomain {
        return true
    }
    
    // 2. 不是静态资源 - 跳过
    if !isStaticResource(url) {
        return false
    }
    
    // 3. 是同主域名 - 分析
    if isSameBaseDomain(domain, targetDomain) {
        return true  // cdn.example.com vs example.com
    }
    
    // 4. 是已知CDN - 分析
    if isKnownCDN(domain) {
        return true  // jsdelivr.net, cloudflare.com等
    }
    
    // 5. 其他情况 - 可配置
    if domain in config.AllowedDomains {
        return true
    }
    
    return false
}
```

---

## 🛠️ 具体实现细节

### 1. JS内容分析器
```go
type JSAnalyzer struct {
    targetDomain string
    patterns     []*regexp.Regexp
}

// 从JS中提取目标域名的URL
func (ja *JSAnalyzer) ExtractURLs(jsContent string) []string {
    urls := []string{}
    
    // 匹配模式
    patterns := []string{
        `['"]/([\w\-\/]+)['"]`,           // '/api/users'
        `fetch\(['"]([^'"]+)['"]`,        // fetch('/api')
        `axios\.(get|post)\(['"]([^'"]+)` // axios.get('/api')
        `window\.location\s*=\s*['"]([^'"]+)` // window.location = '/admin'
        `href\s*:\s*['"]([^'"]+)['"]`,    // href: '/page'
    }
    
    // 提取URL
    for _, pattern := range patterns {
        matches := regexp.FindAllStringSubmatch(jsContent, pattern)
        for _, match := range matches {
            path := match[len(match)-1]
            if strings.HasPrefix(path, "/") {
                // 拼接完整URL
                fullURL := "http://" + ja.targetDomain + path
                urls = append(urls, fullURL)
            }
        }
    }
    
    return deduplicateURLs(urls)
}
```

### 2. 同源域名判断
```go
func isSameBaseDomain(domain1, domain2 string) bool {
    // example.com vs cdn.example.com
    parts1 := strings.Split(domain1, ".")
    parts2 := strings.Split(domain2, ".")
    
    if len(parts1) >= 2 && len(parts2) >= 2 {
        // 比较主域名
        base1 := parts1[len(parts1)-2] + "." + parts1[len(parts1)-1]
        base2 := parts2[len(parts2)-2] + "." + parts2[len(parts2)-1]
        return base1 == base2
    }
    
    return false
}
```

### 3. CDN识别
```go
var knownCDNPatterns = []string{
    "cdn.", "static.", "assets.", "img.", "image.",
    "jsdelivr.net", "unpkg.com", "cloudflare.com",
    "amazonaws.com", "azureedge.net", "aliyuncs.com",
}

func isKnownCDN(domain string) bool {
    for _, pattern := range knownCDNPatterns {
        if strings.Contains(domain, pattern) {
            return true
        }
    }
    return false
}
```

---

## 📊 实现效果对比

### 优化前：
```
爬取目标: http://example.com
发现JS: https://cdn.example.com/app.js (跨域，跳过)
结果: 
  ✓ 发现 15个页面链接
  ✗ 错过 app.js 中的 8个API端点
  ✗ 错过 app.js 中的 3个管理页面
```

### 优化后：
```
爬取目标: http://example.com
发现JS: https://cdn.example.com/app.js
  → 识别为同源CDN
  → 下载并分析内容
  → 提取出 11个目标域名URL
  
结果:
  ✓ 发现 15个页面链接
  ✓ 发现 8个API端点 (从JS提取)
  ✓ 发现 3个管理页面 (从JS提取)
  提升覆盖率: +42%
```

---

## 🎯 实施优先级

### Phase 1 (核心功能)
1. **实现JS内容下载** - 允许下载跨域静态资源
2. **实现URL提取器** - 从JS中提取相对路径
3. **实现同源判断** - 识别同主域名

### Phase 2 (智能优化)
4. **实现CDN识别** - 自动识别常见CDN
5. **优化正则匹配** - 提高URL提取准确率
6. **添加配置选项** - 允许用户自定义

### Phase 3 (增强功能)
7. **JS代码美化** - 处理压缩混淆的JS
8. **动态执行JS** - 处理更复杂的动态生成URL
9. **结果统计** - 显示从跨域JS发现的URL数量

---

## ⚠️ 注意事项

1. **性能考虑**
   - 下载JS文件会增加请求数
   - 建议对JS文件大小做限制（如最大5MB）
   - 可以缓存已分析的JS文件

2. **安全性**
   - 仅分析内容，不执行JS代码
   - 避免下载可疑域名的资源
   - 设置超时时间

3. **准确性**
   - 正则匹配可能有误报
   - 建议URL去重验证
   - 记录提取来源便于调试

---

## 🚀 快速开始

我建议从 **方案1（智能跨域资源白名单）** 开始实现，步骤如下：

1. 修改URL过滤逻辑，允许跨域静态资源
2. 创建JS内容分析器
3. 从JS中提取相对路径URL
4. 将提取的URL拼接后加入爬取队列
5. 测试效果

你觉得这个思路如何？我可以马上开始实现代码。

