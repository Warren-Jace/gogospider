# Spider-golang 使用示例

## 编译项目

```bash
# 克隆项目后，进入项目目录
cd Spider-golang

# 初始化模块并下载依赖
go mod tidy

# 构建项目
go build -o spider.exe cmd/spider/main.go
```

## 基本使用

```bash
# 基本爬取
./spider.exe -url=https://example.com

# 深度爬取
./spider.exe -url=https://example.com -deep=true

# 设置爬取深度
./spider.exe -url=https://example.com -depth=5

# 使用BFS算法
./spider.exe -url=https://example.com -algorithm=BFS
```

## 使用Makefile构建

```bash
# 构建当前平台版本
make build

# 构建Windows版本
make build-windows

# 构建Linux版本
make build-linux

# 构建macOS版本
make build-mac

# 构建所有平台版本
make build-all

# 运行程序
make run

# 清理构建文件
make clean

# 查看帮助
make help
```

## 功能说明

### 多层次爬取策略
- **静态爬虫**: 基于HTML解析提取链接与参数
- **动态爬虫**: 集成Chromium支持JS渲染
- **JS分析**: 提取API端点、参数及隐藏链接
- **API推测**: 基于模式识别推测潜在接口

### 爬取控制与配置
- 递归爬取，支持深度与广度配置
- 调度算法：DFS/BFS切换

### 反爬机制处理
- User-Agent伪造
- 请求速率控制

### 重复率去除
- 基于内容相似度去重
- URL模式识别

## 输出示例

```
开始爬取: https://example.com
深度设置: 3
调度算法: DFS
使用静态爬虫...
静态爬虫完成，发现 15 个链接, 8 个资源, 2 个表单, 3 个API
处理发现的参数...
为URL https://example.com 生成 2 个参数变体
  变体: https://example.com?param_a=value1
  变体: https://example.com?param_a=value1&param_b=value2
开始递归爬取...
递归爬取: https://example.com/about
递归爬取: https://example.com/contact

=== 爬取完成 ===
总共发现:
  链接: 42
  资源: 25
  表单: 5
  API: 8

发现的链接 (前10个):
  1. https://example.com
  2. https://example.com/about
  3. https://example.com/contact
  4. https://example.com/products
  ...

发现的API (前10个):
  1. /api/users
  2. /api/products
  3. /v1/auth/login
  ...
```