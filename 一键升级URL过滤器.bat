@echo off
chcp 65001 >nul
echo.
echo ╔════════════════════════════════════════════════════════════╗
echo ║      一键升级URL过滤器到 v2.0 (黑名单机制)                ║
echo ║      预期效果: URL收集数提升 5-10倍                       ║
echo ╚════════════════════════════════════════════════════════════╝
echo.

:: 检查是否已备份
if exist core\url_validator.go.backup (
    echo [警告] 发现备份文件，可能已经升级过
    echo.
    choice /C YN /M "是否继续升级（会覆盖当前配置）"
    if errorlevel 2 (
        echo.
        echo 已取消升级
        pause
        exit /b 0
    )
)

echo [步骤1/5] 备份旧版URL验证器...
copy core\url_validator.go core\url_validator.go.backup >nul
if %ERRORLEVEL% EQU 0 (
    echo ✓ 备份成功: core\url_validator.go.backup
) else (
    echo ✗ 备份失败
    pause
    exit /b 1
)
echo.

echo [步骤2/5] 备份爬虫主文件...
copy core\spider.go core\spider.go.backup >nul
if %ERRORLEVEL% EQU 0 (
    echo ✓ 备份成功: core\spider.go.backup
) else (
    echo ✗ 备份失败
    pause
    exit /b 1
)
echo.

echo [步骤3/5] 检查新版验证器...
if exist core\url_validator_v2.go (
    echo ✓ 找到新版验证器: core\url_validator_v2.go
) else (
    echo ✗ 未找到新版验证器文件
    echo.
    echo 请确保以下文件存在:
    echo   - core\url_validator_v2.go
    pause
    exit /b 1
)
echo.

echo [步骤4/5] 修改爬虫主文件...
echo.
echo 需要手动修改 core\spider.go 文件:
echo.
echo 找到第157行左右的代码:
echo   urlValidator:      NewURLValidator(),
echo.
echo 替换为:
echo   urlValidator:      NewSmartURLValidatorCompat(),
echo.
echo 按任意键打开文件进行修改...
pause >nul
notepad core\spider.go
echo.

echo [步骤5/5] 编译测试...
echo.
echo 正在编译新版爬虫...
go build -o spider_v3.6_new.exe cmd/spider/main.go
if %ERRORLEVEL% EQU 0 (
    echo ✓ 编译成功: spider_v3.6_new.exe
    echo.
    echo ╔════════════════════════════════════════════════════════════╗
    echo ║                    升级成功！                              ║
    echo ╚════════════════════════════════════════════════════════════╝
    echo.
    echo 接下来的步骤:
    echo.
    echo 1. 运行对比测试:
    echo    .\test_validator_comparison.bat
    echo.
    echo 2. 测试新版爬虫:
    echo    spider_v3.6_new.exe -url http://x.lydaas.com -depth 2 -config config.json
    echo.
    echo 3. 对比结果:
    echo    - 旧版: ~11-59个URL
    echo    - 新版: ~200-300个URL (提升5-10倍)
    echo.
    echo 4. 如果效果满意，替换旧版:
    echo    copy spider_v3.6_new.exe spider_v3.5.exe
    echo.
    echo 5. 如需回滚:
    echo    copy core\url_validator.go.backup core\url_validator.go
    echo    copy core\spider.go.backup core\spider.go
    echo    go build -o spider_v3.5.exe cmd/spider/main.go
    echo.
) else (
    echo ✗ 编译失败
    echo.
    echo 可能的原因:
    echo 1. 没有正确修改 core\spider.go
    echo 2. 代码存在语法错误
    echo.
    echo 如需回滚:
    echo    copy core\spider.go.backup core\spider.go
    echo.
    pause
    exit /b 1
)

pause

