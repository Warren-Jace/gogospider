# Cookie使用指南 - 突破登录墙

## 📌 什么是登录墙？

登录墙是指网站要求用户登录才能访问内容的机制。对于爬虫来说，这是一个常见问题：

```
┌─────────────────────────────────────────┐
│  没有Cookie的爬虫                        │
│                                         │
│  🕷️ 访问 https://x.lydaas.com          │
│   ↓                                    │
│  🔒 被重定向到登录页面                   │
│   ↓                                    │
│  ❌ 无法访问业务内容                     │
│  ❌ 只能爬取登录页面                     │
│  ❌ 爬取500个页面，全是登录页面变体      │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  使用Cookie的爬虫                        │
│                                         │
│  🕷️ 访问 https://x.lydaas.com          │
│  🍪 携带登录Cookie                      │
│   ↓                                    │
│  ✅ 直接访问业务内容                     │
│  ✅ 爬取真实页面                        │
│  ✅ 发现敏感信息                        │
└─────────────────────────────────────────┘
```

---

## 🍪 如何获取Cookie

### 方法1：Chrome浏览器（推荐）

1. 打开 Chrome，访问目标网站并登录
2. 按 `F12` 打开开发者工具
3. 切换到 **Application** 标签
4. 左侧选择 **Cookies** → 点击网站域名
5. 右侧会显示所有Cookie
6. 找到重要的Cookie（如 `session_id`, `auth_token`, `user_token` 等）
7. 复制名称和值

或者使用Network方法：
1. 按 `F12` → **Network** 标签
2. 刷新页面
3. 点击任意请求
4. 查看 **Request Headers** → **Cookie**
5. 复制完整Cookie字符串

### 方法2：Edge浏览器

类似Chrome，按 `F12` → **Application** → **Cookies**

### 方法3：Firefox浏览器

1. 按 `F12` → **Storage** 标签
2. 展开 **Cookies**
3. 选择网站域名
4. 右键Cookie → **Copy** → **Copy Value**

### 方法4：使用浏览器插件

- **EditThisCookie** (Chrome)
- **Cookie Editor** (Firefox/Chrome)

---

## 📝 创建Cookie文件

GogoSpider支持3种Cookie文件格式：

### 格式1：简单格式（最简单）⭐

创建文件 `cookies.txt`：
```
session_id=abc123xyz456
auth_token=def789uvw012
user_id=12345
```

或者一行（用分号分隔）：
```
session_id=abc123xyz456; auth_token=def789uvw012; user_id=12345
```

### 格式2：JSON格式（推荐）⭐⭐

创建文件 `cookies.json`：
```json
{
  "session_id": "abc123xyz456",
  "auth_token": "def789uvw012",
  "user_id": "12345",
  "remember_me": "true"
}
```

### 格式3：Netscape格式（兼容性最好）⭐⭐⭐

创建文件 `cookies_netscape.txt`：
```
# Netscape HTTP Cookie File
.lydaas.com	TRUE	/	FALSE	0	session_id	abc123xyz456
.lydaas.com	TRUE	/	FALSE	0	auth_token	def789uvw012
x.lydaas.com	TRUE	/	FALSE	0	user_id	12345
```

格式说明：
```
域名 \t 子域名标志 \t 路径 \t Secure标志 \t 过期时间 \t Cookie名 \t Cookie值
```

---

## 🚀 使用Cookie爬取

### 方法1：使用Cookie文件（推荐）

```bash
# 使用简单格式Cookie文件
spider.exe -url https://x.lydaas.com -cookie-file cookies.txt

# 使用JSON格式Cookie文件  
spider.exe -url https://x.lydaas.com -cookie-file cookies.json

# 使用Netscape格式Cookie文件
spider.exe -url https://x.lydaas.com -cookie-file cookies_netscape.txt
```

### 方法2：使用Cookie字符串（快速测试）

```bash
# 直接在命令行传入Cookie
spider.exe -url https://x.lydaas.com -cookie "session_id=abc123; auth_token=def456"
```

### 方法3：组合使用

```bash
# 使用Cookie + 其他参数
spider.exe -url https://x.lydaas.com \
  -cookie-file cookies.txt \
  -depth 5 \
  -max-pages 1000 \
  -allow-subdomains \
  -sensitive-rules sensitive_rules_config.json
```

---

## 📊 预期输出

### 成功加载Cookie后的输出：

```
╔═══════════════════════════════════════════════════════════════╗
║            Spider Ultimate - 智能Web爬虫系统                 ║
╚═══════════════════════════════════════════════════════════════╝

⏳ 正在加载Cookie文件: cookies.txt
[Cookie] 从简单格式加载了 3 个Cookie
[Cookie] 已加载 3 个Cookie:
  - session_id = abc123xyz4...uvw012
  - auth_token = def789uvw0...xyz456
  - user_id = 12345

✅ 已加载敏感信息规则文件: ./sensitive_rules_config.json

[*] 开始爬取: https://x.lydaas.com
...
```

### 爬取过程中：

```
【第 1 层爬取】最大深度: 5
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[静态爬虫] 开始爬取: https://x.lydaas.com
[静态爬虫] 🍪 使用Cookie认证
[静态爬虫] 发现 25 个<a>标签
[静态爬虫] 有效链接: 20个
[技术栈] 检测到: React 18.2.0, Nginx
[敏感信息] ⚠️  发现 3 处高危敏感信息！

✅ 成功进入业务页面！
```

### 爬取结束时的登录墙检测报告：

#### 无登录墙（正常情况）：
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔍 登录墙检测报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
总爬取页面: 500 个
登录页面: 5 个 (1.0%)
正常页面: 495 个
唯一登录URL: 1 个

✅ 未发现登录墙，所有页面均可正常访问
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

#### 有登录墙（Cookie无效）：
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔍 登录墙检测报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
总爬取页面: 500 个
登录页面: 485 个 (97.0%)
正常页面: 15 个
唯一登录URL: 2 个

⚠️  警告：登录页面占比过高（>50%），建议使用Cookie认证
   详细说明：如何解决登录问题.md
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 🎯 实际使用示例

### 示例1：x.lydaas.com（您的网站）

#### 步骤1：获取Cookie

1. 在浏览器中登录 https://x.lydaas.com
2. 按 F12 → Application → Cookies → x.lydaas.com
3. 找到并复制重要Cookie（如 `session_id`, `token` 等）

#### 步骤2：创建Cookie文件

创建 `cookies_lydaas.json`：
```json
{
  "session_id": "你的session值",
  "auth_token": "你的token值"
}
```

#### 步骤3：使用Cookie爬取

```bash
# 方法1：使用Cookie文件
spider.exe -url https://x.lydaas.com -cookie-file cookies_lydaas.json -depth 5

# 方法2：使用Cookie字符串
spider.exe -url https://x.lydaas.com -cookie "session_id=xxx; auth_token=yyy" -depth 5
```

### 示例2：测试Cookie功能

使用本地测试（无需真实Cookie）：

```bash
test_local_sensitive.bat
```

### 示例3：批量扫描 + Cookie

如果多个站点需要相同的Cookie：

```bash
spider.exe -batch-file targets.txt -cookie-file cookies.txt -batch-concurrency 5
```

---

## ⚠️ 常见问题

### Q1: Cookie过期了怎么办？

**A:** Cookie有有效期，如果爬取中途Cookie过期：

**症状：**
- 突然开始出现大量登录页面
- 登录墙检测报告显示比例上升

**解决：**
- 重新从浏览器获取Cookie
- 使用 `记住我` 选项获取长期Cookie

### Q2: Cookie对所有子域名都有效吗？

**A:** 取决于Cookie的Domain属性：
- `.lydaas.com` → 对所有子域名有效 ✅
- `x.lydaas.com` → 只对该子域名有效

**建议：**在Netscape格式中使用 `.lydaas.com`（带点前缀）

### Q3: 需要哪些Cookie？

**A:** 通常需要：
- `session_id` / `PHPSESSID` / `JSESSIONID` - 会话ID
- `auth_token` / `access_token` - 认证Token
- `user_id` / `uid` - 用户ID
- `remember_me` - 记住登录状态

**提示：**复制浏览器请求中的所有Cookie更保险

### Q4: Cookie文件的安全性？

**⚠️ 重要提醒：**
- Cookie文件包含敏感认证信息
- 不要提交到Git仓库
- 使用后及时删除
- 不要分享给其他人

**建议：**在 `.gitignore` 中添加：
```
cookies*.txt
cookies*.json
```

### Q5: 如何判断Cookie是否有效？

**A:** 查看爬取开始后：

**Cookie有效：**
```
[静态爬虫] 发现 25 个<a>标签        ← 发现很多链接
[静态爬虫] 有效链接: 20个
```

**Cookie无效（仍然是登录页面）：**
```
[静态爬虫] 发现 0 个<a>标签         ← 没有发现链接
[静态爬虫] 有效链接: 0个
```

并且登录墙检测报告会显示高占比。

---

## 🔒 安全建议

1. **使用测试账号**
   - 不要使用管理员账号的Cookie
   - 创建专门的测试账号

2. **定期轮换Cookie**
   - Cookie可能包含敏感权限
   - 扫描后及时退出登录/更换Cookie

3. **限制扫描范围**
   - 使用Scope设置限制爬取范围
   - 避免爬取到敏感的管理页面

4. **本地保存**
   - 不要通过网络传输Cookie文件
   - 存储在本地安全位置

---

## 📁 Cookie文件示例

### 示例1：简单格式（cookies_simple.txt）

```
session_id=abc123xyz456789
auth_token=Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
user_id=10001
lang=zh-CN
```

### 示例2：JSON格式（cookies.json）

```json
{
  "session_id": "abc123xyz456789",
  "auth_token": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
  "user_id": "10001",
  "lang": "zh-CN",
  "remember_me": "true"
}
```

### 示例3：Netscape格式（cookies_netscape.txt）

```
# Netscape HTTP Cookie File
# This is a generated file! Do not edit.
.lydaas.com	TRUE	/	FALSE	1767139200	session_id	abc123xyz456789
.lydaas.com	TRUE	/	TRUE	1767139200	auth_token	Bearer_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
x.lydaas.com	TRUE	/	FALSE	0	user_id	10001
.lydaas.com	TRUE	/	FALSE	0	lang	zh-CN
```

---

## 🎬 完整使用流程

### 流程1：首次爬取发现登录墙

```bash
# 第一次尝试（没有Cookie）
spider.exe -url https://x.lydaas.com -depth 5

# 结果：
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🔍 登录墙检测报告
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 总爬取页面: 500 个
# 登录页面: 485 个 (97.0%)
# 正常页面: 15 个
# 
# ⚠️  警告：登录页面占比过高（>50%），建议使用Cookie认证
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 流程2：获取Cookie并重新爬取

```bash
# 1. 在浏览器中登录
# 2. 复制Cookie到 cookies.json
# 3. 使用Cookie重新爬取

spider.exe -url https://x.lydaas.com -cookie-file cookies.json -depth 5

# 结果：
# ⏳ 正在加载Cookie文件: cookies.json
# [Cookie] 从JSON加载了 3 个Cookie
# [Cookie] 已加载 3 个Cookie:
#   - session_id = abc123xyz4...
#   - auth_token = def789uvw0...
#   - user_id = 12345
# 
# ✅ 已加载敏感信息规则文件: ./sensitive_rules_config.json
# 
# [*] 开始爬取: https://x.lydaas.com
# 
# [静态爬虫] 发现 45 个<a>标签           ← 成功进入！
# [敏感信息] ⚠️  发现 5 处高危敏感信息！  ← 发现敏感信息！
# 
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🔍 登录墙检测报告
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 总爬取页面: 500 个
# 登录页面: 3 个 (0.6%)                ← 只有少量登录页面
# 正常页面: 497 个                     ← 大部分是正常页面
# 
# ✅ 未发现登录墙，所有页面均可正常访问   ← 成功！
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 🛡️ 登录墙自动检测

GogoSpider v3.2 新增了自动登录墙检测功能！

### 检测机制

程序会自动检测：

1. **URL模式检测**
   - `/login`, `/signin`, `/auth/` 等路径
   - 自动识别登录页面

2. **内容信号检测**
   - 检查页面是否包含"登录"、"sign in"等关键词
   - 检查是否有"username"、"password"等表单字段

3. **统计分析**
   - 每100个页面检查一次登录墙占比
   - 如果 >50% 是登录页面，立即发出警告

### 自动过滤

**智能跳过登录页面变体：**
```
检测到：
  https://auth.lydaas.com/login?redirect_uri=/page1
  https://auth.lydaas.com/login?redirect_uri=/page2
  https://auth.lydaas.com/login?redirect_uri=/page3
  ...

自动跳过：
  [登录墙过滤] 本层跳过 245 个重复的登录页面变体
```

---

## 📚 命令参考

### Cookie相关参数

```bash
-cookie-file string
        Cookie文件路径（支持3种格式）
        格式：简单格式、JSON格式、Netscape格式

-cookie string
        Cookie字符串（格式：name1=value1; name2=value2）
        适合快速测试，不用创建文件
```

### 完整示例

```bash
# 示例1：使用Cookie文件 + 深度扫描
spider.exe \
  -url https://x.lydaas.com \
  -cookie-file cookies.json \
  -depth 8 \
  -max-pages 2000 \
  -allow-subdomains \
  -sensitive-rules sensitive_rules_config.json

# 示例2：使用Cookie字符串 + 批量扫描
spider.exe \
  -batch-file targets.txt \
  -cookie "session_id=xxx; token=yyy" \
  -batch-concurrency 10

# 示例3：使用Cookie + 配置文件
spider.exe \
  -config config_lydaas.json \
  -cookie-file cookies.txt
```

---

## ✅ 成功标志

Cookie成功使用的标志：

1. ✅ 启动时显示 `[Cookie] 已加载 X 个Cookie`
2. ✅ 爬取过程中发现很多有效链接（不是0个）
3. ✅ 登录墙检测报告显示登录页面占比<10%
4. ✅ 生成了敏感信息文件
5. ✅ 发现了真实的业务内容

---

## 📖 相关文档

- **如何解决登录问题.md** - 登录墙详细说明
- **三个问题的完整解答.md** - 问题分析
- **快速总结.txt** - 快速参考

---

## 🎉 快速开始（针对x.lydaas.com）

```bash
# 步骤1：在浏览器登录 x.lydaas.com
# 步骤2：复制Cookie到 cookies.json
# 步骤3：运行爬虫

spider.exe -url https://x.lydaas.com -cookie-file cookies.json -depth 5
```

就这么简单！🚀

