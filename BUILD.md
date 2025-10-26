# 编译说明

## 前置要求

- Go 1.18 或更高版本
- Chrome/Chromium（用于动态爬取功能）

## 编译命令

### Windows

```bash
go build -o spider.exe cmd/spider/main.go
```

### Linux/macOS

```bash
go build -o spider cmd/spider/main.go
```

### 交叉编译

```bash
# Windows 64位
GOOS=windows GOARCH=amd64 go build -o spider_windows_amd64.exe cmd/spider/main.go

# Linux 64位
GOOS=linux GOARCH=amd64 go build -o spider_linux_amd64 cmd/spider/main.go

# macOS 64位（Intel）
GOOS=darwin GOARCH=amd64 go build -o spider_darwin_amd64 cmd/spider/main.go

# macOS ARM64（M1/M2）
GOOS=darwin GOARCH=arm64 go build -o spider_darwin_arm64 cmd/spider/main.go
```

## 运行测试

```bash
# 安装依赖
go mod download

# 运行测试
go test ./...

# 编译并测试
go build -o spider cmd/spider/main.go
./spider -url https://testphp.vulnweb.com -depth 2
```

## 常见问题

### Go环境未配置

如果出现 "cannot find GOROOT" 错误：

1. 下载Go: https://golang.org/dl/
2. 安装并设置环境变量
3. 验证: `go version`

### 依赖下载失败

```bash
# 使用国内代理
go env -w GOPROXY=https://goproxy.cn,direct

# 重新下载依赖
go mod download
```

### Chrome未找到

动态爬虫功能需要Chrome/Chromium：

- Windows: 自动检测Chrome路径
- Linux: `sudo apt install chromium-browser`
- macOS: `brew install chromium`

或使用 `-chrome-path` 参数指定路径。

