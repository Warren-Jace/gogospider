# Spider-golang

一个基于Golang开发的跨平台网页爬虫工具，具备多层次爬取策略、智能去重机制和灵活的配置选项。

## 最新改进

### 参数处理优化
- 改进了参数变体生成算法，为不同类型的参数生成更合理的变体：
  - 数字参数：生成+1和-1变体
  - 字符串参数：生成"_variant"后缀变体和空值变体
- 添加了线程安全机制，确保在并发环境下的正确性

### 表单处理优化
- 完善了静态爬虫中的表单提取逻辑
- 在报告中添加了详细的表单信息，包括：
  - 表单动作（Action）
  - 表单方法（Method）
  - 表单字段（Field Name, Type, Value）

## 功能特性

- **多层次爬取策略**：
  - 静态爬虫：解析HTML文档，提取链接、资源和表单信息
  - 动态爬虫：使用Chrome DevTools Protocol处理JavaScript渲染的页面
  - JS分析：分析JavaScript文件中的API端点和敏感信息
  - 参数处理：处理URL参数，生成参数变体进行测试

- **爬取控制与配置**：
  - 递归爬取：支持多层链接的递归爬取
  - 调度算法：支持DFS和BFS两种爬取算法
  - 反爬机制：支持设置请求间隔、随机User-Agent等反爬措施

- **智能去重机制**：
  - URL去重：避免重复爬取相同URL
  - 内容去重：基于文本内容相似度的去重
  - DOM结构去重：基于页面结构相似度的去重

- **跨平台支持**：
  - 支持Windows、Linux和macOS操作系统
  - 提供Makefile简化构建和部署流程

## 项目结构

```
Spider-golang/
├── cmd/
│   └── spider/
│       └── main.go          # 命令行接口
├── core/
│   ├── spider.go            # 主爬虫协调器
│   ├── static_crawler.go    # 静态爬虫实现
│   ├── dynamic_crawler.go   # 动态爬虫实现
│   ├── js_analyzer.go       # JavaScript分析器
│   ├── param_handler.go     # 参数处理器
│   └── duplicate_handler.go # 去重处理器
├── config/
│   └── config.go            # 配置管理
├── examples/
│   └── usage.md             # 使用示例
├── Makefile                 # 构建脚本
├── .gitignore               # Git忽略文件
├── LICENSE                  # 许可证
└── README.md                # 项目说明
```

## 安装与构建

### 使用Makefile构建

```bash
# 构建当前平台的可执行文件
make build

# 构建Windows平台的可执行文件
make build-windows

# 构建Linux平台的可执行文件
make build-linux

# 构建macOS平台的可执行文件
make build-darwin

# 构建所有平台的可执行文件
make build-all
```

### 手动构建

```bash
# 初始化Go模块
go mod tidy

# 构建项目
go build -o spider.exe cmd/spider/main.go
```

## 使用方法

### 基本使用

```bash
# 爬取指定URL
./spider.exe -url=http://example.com/

# 深度爬取
./spider.exe -url=http://example.com/ -depth

# 设置最大爬取深度
./spider.exe -url=http://example.com/ -max-depth=3

# 使用BFS算法
./spider.exe -url=http://example.com/ -algorithm=bfs
```

### 高级选项

```bash
# 启用动态爬虫
./spider.exe -url=http://example.com/ -enable-dynamic

# 启用JS分析
./spider.exe -url=http://example.com/ -enable-js

# 设置请求间隔（毫秒）
./spider.exe -url=http://example.com/ -delay=1000

# 设置User-Agent
./spider.exe -url=http://example.com/ -user-agent="Mozilla/5.0 ..."
```

### 输出报告

程序会生成一个TXT格式的报告文件，文件名格式为：`spider_http_{domain}_{timestamp}.txt`

报告包含以下信息：
- 爬取到的所有URL链接
- 发现的资源文件（CSS、JS、图片等）
- 发现的表单信息（动作、方法、字段）
- 发现的API端点

表单信息在报告中以以下格式显示：
```
Form Action: http://example.com/login.php, Method: post
  Field Name: username, Type: text, Value: 
  Field Name: password, Type: password, Value: 
  Field Name: submit, Type: submit, Value: Login
```

## 技术特点

- **模块化设计**：各功能模块独立实现，便于维护和扩展
- **并发处理**：支持并发爬取，提高效率
- **错误处理**：完善的错误处理机制，确保程序稳定性
- **配置灵活**：支持多种配置选项，满足不同需求
- **跨平台**：支持主流操作系统，便于部署

## 适用场景

- Web安全测试
- 资产发现
- 竞品分析
- 数据采集