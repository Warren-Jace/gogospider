# URL发现路径追踪分析

## 🔍 基于Referer的URL发现路径分析

### 方法论

1. 查看Crawlergo的Referer字段
2. 了解每个URL是从哪个页面发现的
3. 检查Spider是否爬取了相同的来源页面
4. 如果爬取了，为什么没有发现该URL？

---

## 📋 Crawlergo URL的Referer追踪

### 第1层：从根目录发现的URL

**来源**: `http://testphp.vulnweb.com/`（根目录）

| URL | Referer | Spider是否爬取此来源页 | Spider是否发现此URL |
|-----|---------|----------------------|-------------------|
| `/Templates/main_dynamic_template.dwt.php` | `/` | ✅ 是 | ❌ 否 |
| `/index.php` | `/` | ✅ 是 | ✅ 是 |
| `/categories.php` | `/` | ✅ 是 | ✅ 是 |
| `/artists.php` | `/` | ✅ 是 | ✅ 是 |
| `/disclaimer.php` | `/` | ✅ 是 | ✅ 是 |
| `/cart.php` | `/` | ✅ 是 | ✅ 是 |
| `/guestbook.php` | `/` | ✅ 是 | ✅ 是 |
| `/AJAX/index.php` | `/` | ✅ 是 | ✅ 是 |
| `/search.php?test=query` | `/` | ✅ 是 | ✅ 是 |
| `/login.php` | `/` | ✅ 是 | ✅ 是 |
| `/userinfo.php` | `/` | ✅ 是 | ✅ 是 |
| `/privacy.php` | `/` | ✅ 是 | ✅ 是 |
| `/Mod_Rewrite_Shop/` | `/` | ✅ 是 | ✅ 是 |
| `/hpp/` | `/` | ✅ 是 | ✅ 是 |

**分析**: 
- Spider爬取了根页面 ✅
- Spider发现了13/14个链接 ✅
- 🔴 **问题URL**: `/Templates/main_dynamic_template.dwt.php` 

**需要检查**: 为什么Spider在爬取根页面时没有发现这个URL？

---

### 第2层：从categories.php发现的URL

**来源**: `http://testphp.vulnweb.com/categories.php`

| URL | Referer | Spider是否爬取此来源页 | Spider是否发现此URL |
|-----|---------|----------------------|-------------------|
| `/listproducts.php?cat=1` | `/categories.php` | ✅ 是 | ✅ 是 |

**分析**: ✅ 完全覆盖

---

### 第3层：从artists.php发现的URL

**来源**: `http://testphp.vulnweb.com/artists.php`

| URL | Referer | Spider是否爬取此来源页 | Spider是否发现此URL |
|-----|---------|----------------------|-------------------|
| `/comment.php?aid=1` | `/artists.php` | ✅ 是 | ❌ 否 |

**分析**:
- Spider爬取了 `/artists.php` ✅
- Spider发现了29个`<a>`标签 ✅
- Spider收集了3个链接 ✅
- 🔴 **问题**: 为什么没有发现 `comment.php?aid=1`？

**需要检查**: 
1. 这个链接是否在HTML的`<a>`标签中？
2. 还是在JavaScript代码中？
3. 还是需要点击某个元素才出现？

---

### 第4层：从AJAX/index.php发现的URL

**来源**: `http://testphp.vulnweb.com/AJAX/index.php`

| URL | Referer | Spider是否爬取此来源页 | Spider是否发现此URL |
|-----|---------|----------------------|-------------------|
| `/showimage.php?file=` | `/AJAX/index.php` | ✅ 是 | ❌ 否（空参数无效） |
| `/AJAX/showxml.php` | `/AJAX/index.php` | ✅ 是 | ❌ 否 |
| `/AJAX/artists.php` | `/AJAX/index.php` | ✅ 是 | ❌ 否 |
| `/AJAX/categories.php` | `/AJAX/index.php` | ✅ 是 | ❌ 否 |
| `/AJAX/titles.php` | `/AJAX/index.php` | ✅ 是 | ❌ 否 |

**分析**:
- Spider爬取了 `/AJAX/index.php` ✅
- Spider发现了5个`<a>`标签 ✅
- Spider收集了0个链接 ❌
- 🔴 **严重问题**: Spider爬取了这个页面但没发现这4个URL！

**需要检查**:
1. AJAX/index.php的HTML源代码
2. 这些链接是否在`<a>`标签中？
3. 还是通过JavaScript动态生成？

---

### 第5层：从login.php发现的URL

**来源**: `http://testphp.vulnweb.com/login.php`

| URL | Referer | Spider是否爬取此来源页 | Spider是否发现此URL |
|-----|---------|----------------------|-------------------|
| `/signup.php` | `/login.php` | ✅ 是 | ✅ 是 |

**分析**: ✅ 完全覆盖

---

### 第6层：从listproducts.php?cat=1发现的URL

**来源**: `http://testphp.vulnweb.com/listproducts.php?cat=1`

| URL | Referer | Spider是否爬取此来源页 | Spider是否发现此URL |
|-----|---------|----------------------|-------------------|
| `/showimage.php?file=./pictures/1.jpg&size=160` | `/listproducts.php?cat=1` | ✅ 是 | ✅ 是（参数变体） |
| `/comment.php?pid=1` | `/listproducts.php?cat=1` | ✅ 是 | ❌ 否 |
| `/product.php?pic=1` | `/listproducts.php?cat=1` | ✅ 是 | ✅ 是 |
| `/showimage.php?file=./pictures/1.jpg` | `/listproducts.php?cat=1` | ✅ 是 | ✅ 是 |

**分析**:
- Spider爬取了 `/listproducts.php?cat=1` ✅
- Spider发现了47个`<a>`标签 ✅
- Spider收集了12个链接 ✅
- 🔴 **问题**: 为什么没有发现 `comment.php?pid=1`？

**需要检查**: listproducts.php?cat=1 页面的HTML源代码

---

### 第7层：从hpp/?pp=12发现的URL

**来源**: `http://testphp.vulnweb.com/hpp/?pp=12`

| URL | Referer | Spider是否爬取此来源页 | Spider是否发现此URL |
|-----|---------|----------------------|-------------------|
| `/hpp/params.php?p=valid&pp=12` | `/hpp/?pp=12` | ✅ 是 | ✅ 是 |
| `/hpp/params.php?` | `/hpp/?pp=12` | ✅ 是 | ❌ 否（空参数） |
| `/hpp/params.php?aaaa/=%E6%8F%90%E4%BA%A4` | `/hpp/?pp=12` | ✅ 是 | ❌ 否 |

**分析**:
- Spider爬取了 `/hpp/?pp=12` ✅
- Spider发现了4个`<a>`标签 ✅
- Spider收集了1个链接 ✅
- 🔴 **问题**: 为什么只发现1个，没有发现另外2个？

---

## 🔍 关键问题总结

### 需要深入分析的3个页面

| 来源页面 | Spider爬取 | 应该发现 | 实际发现 | 问题 |
|---------|-----------|---------|----------|------|
| `/` | ✅ 25个<a>标签 | 14个 | 13个 | ❌ 缺少Templates URL |
| `/AJAX/index.php` | ✅ 5个<a>标签 | 5个 | 0个 | ❌ 缺少4个AJAX URL |
| `/artists.php` | ✅ 29个<a>标签 | 含comment.php | 3个 | ❌ 缺少comment.php?aid=1 |
| `/listproducts.php?cat=1` | ✅ 47个<a>标签 | 含comment.php | 12个 | ❌ 缺少comment.php?pid=1 |
| `/hpp/?pp=12` | ✅ 4个<a>标签 | 3个 | 1个 | ❌ 缺少2个params.php |

---

## 🎯 下一步诊断计划

### 需要检查的内容

1. **检查根页面HTML** - 查找Templates URL
2. **检查AJAX/index.php的HTML** - 查找4个AJAX URL
3. **检查artists.php的HTML** - 查找comment.php?aid=1
4. **检查listproducts.php?cat=1的HTML** - 查找comment.php?pid=1
5. **检查hpp/?pp=12的HTML** - 查找params.php的其他2个URL

### 检查方法

对于每个页面，需要确认：
- ✓ 这些URL是否在`<a href>`标签中？
- ✓ 还是在`onclick`等事件中？
- ✓ 还是在JavaScript代码中？
- ✓ 还是需要特定的用户交互才会出现？

