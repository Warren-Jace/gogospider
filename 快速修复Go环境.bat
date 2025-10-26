@echo off
chcp 65001 >nul
echo ================================
echo Go环境快速修复脚本
echo ================================
echo.

echo [检查] 正在检查Go环境...
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] Go未安装或PATH配置错误
    echo.
    echo 请从以下地址下载Go:
    echo https://golang.org/dl/go1.23.6.windows-amd64.msi
    echo.
    pause
    exit /b 1
)

echo [信息] Go版本:
go version
echo.

echo [检查] 验证标准库...
go env GOROOT >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] GOROOT未设置
    pause
    exit /b 1
)

for /f "delims=" %%i in ('go env GOROOT') do set GOROOT=%%i
echo [信息] GOROOT: %GOROOT%
echo.

if not exist "%GOROOT%\src\runtime\runtime.go" (
    echo [错误] 标准库文件缺失！
    echo 路径: %GOROOT%\src\
    echo.
    echo 解决方案:
    echo 1. 重新下载并安装Go
    echo 2. 或者从官网下载zip完整解压
    echo.
    echo 下载地址:
    echo https://golang.org/dl/go1.23.6.windows-amd64.msi
    echo https://golang.org/dl/go1.23.6.windows-amd64.zip
    echo.
    pause
    exit /b 1
)

echo [成功] 标准库文件存在
echo.

echo [测试] 尝试编译测试程序...
echo package main > test_compile.go
echo import "fmt" >> test_compile.go
echo func main() { fmt.Println("OK") } >> test_compile.go

go build -o test_compile.exe test_compile.go 2>compile_error.log
if %errorlevel% neq 0 (
    echo [错误] 编译失败！
    echo.
    type compile_error.log
    echo.
    echo 您的Go安装可能不完整，请重新安装。
    del test_compile.go >nul 2>&1
    del compile_error.log >nul 2>&1
    pause
    exit /b 1
)

echo [成功] 编译测试通过
del test_compile.go >nul 2>&1
del test_compile.exe >nul 2>&1
del compile_error.log >nul 2>&1
echo.

echo [编译] 开始编译spider程序...
echo.

go build -o spider_fixed.exe cmd\spider\main.go
if %errorlevel% neq 0 (
    echo [错误] spider编译失败
    echo.
    echo 请查看上面的错误信息
    pause
    exit /b 1
)

echo.
echo ================================
echo [成功] 编译完成！
echo ================================
echo.
echo 生成文件: spider_fixed.exe
echo.
echo 测试命令:
echo   spider_fixed.exe -url https://xss-quiz.int21h.jp/ -depth 2
echo.
echo 或使用配置文件:
echo   spider_fixed.exe -config config_smart_dedup.json
echo.
pause

