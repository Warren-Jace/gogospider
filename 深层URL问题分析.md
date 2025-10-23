# 深层URL发现不足问题分析

## 🔍 问题诊断

### 当前爬取逻辑

```
第1层: 爬取根URL http://testphp.vulnweb.com/
  └─ 发现20个链接

第2层: crawlRecursively() 爬取这20个链接
  └─ 发现更多链接

第3层: ❌ 没有继续爬取！
```

**问题**: `crawlRecursively()` 只执行一次，不是真正的递归！

### 代码分析

```go
// core/spider.go 第238-240行
if s.config.DepthSettings.MaxDepth > 1 {
    s.crawlRecursively()  // ❌ 只调用一次！
}
```

**期望的逻辑**:
```
第1层 → 第2层 → 第3层 → 第4层 → 第5层
  ↓      ↓      ↓      ↓      ↓
 20个   14个    8个    6个    4个 链接
```

**实际的逻辑**:
```
第1层 → 第2层 → 结束
  ↓      ↓
 20个   14个 链接
```

## ⚠️ 为什么深层URL发现不了

### crawlergo发现但Spider未发现的URL

| URL | 需要的路径 | 当前Spider深度 |
|-----|-----------|---------------|
| `comment.php?aid=1` | / → artists.php → artist=1 → 评论 | 第4层 ❌ |
| `product.php?pic=1` | / → categories → cat=1 → 产品 | 第4层 ❌ |
| `showimage.php?file=...` | / → listproducts → 产品 → 图片 | 第5层 ❌ |
| `BuyProduct-1/` | / → Shop → Details → 购买 | 第5层 ❌ |
| `secured/newuser.php` | / → signup → 提交表单 | 第4层 ❌ |

**原因**: 当前只爬取到第2层，所以这些第4-5层的URL都发现不了！

## 🔧 解决方案

### 方案1: 实现真正的递归爬取（推荐）

修改递归逻辑，使其真正递归执行多层。

### 方案2: 增加深度到10层

简单粗暴，但会大幅增加爬取时间。

### 方案3: 智能深度扩展

检测到有新链接就继续爬取，直到没有新链接。

