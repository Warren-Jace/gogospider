@echo off
chcp 65001 >nul
echo ╔═══════════════════════════════════════════════════════════════╗
echo ║      GogoSpider - x.lydaas.com 扫描脚本                       ║
echo ╚═══════════════════════════════════════════════════════════════╝
echo.

echo [检查] 验证环境...
echo.

REM 检查可执行文件
if not exist spider.exe (
    echo ❌ 错误: spider.exe 不存在
    echo    请先编译: go build -o spider.exe cmd/spider/main.go
    pause
    exit /b 1
)
echo ✓ spider.exe 存在

REM 检查配置文件
if not exist config_lydaas.json (
    echo ❌ 错误: config_lydaas.json 不存在
    echo    该配置文件应该已经创建，请检查
    pause
    exit /b 1
)
echo ✓ config_lydaas.json 存在

REM 检查规则文件
if not exist sensitive_rules_config.json (
    echo ⚠️  警告: sensitive_rules_config.json 不存在
    echo    敏感信息检测可能无法工作
    echo.
    set /p continue="是否继续？(Y/N): "
    if /i not "%continue%"=="Y" exit /b 0
) else (
    echo ✓ sensitive_rules_config.json 存在
)

echo.
echo ════════════════════════════════════════════════════════════════
echo   准备开始扫描
echo ════════════════════════════════════════════════════════════════
echo.
echo 目标网站: https://x.lydaas.com
echo 配置文件: config_lydaas.json
echo 最大深度: 5层
echo 子域名: 允许
echo 敏感信息检测: 启用
echo.
echo ════════════════════════════════════════════════════════════════
echo.

REM 提示用户
set /p start="按 Y 开始扫描，其他键取消: "
if /i not "%start%"=="Y" (
    echo.
    echo 已取消扫描
    pause
    exit /b 0
)

echo.
echo ════════════════════════════════════════════════════════════════
echo   开始扫描...
echo ════════════════════════════════════════════════════════════════
echo.

REM 记录开始时间
echo 开始时间: %date% %time%
echo.

REM 运行爬虫
spider.exe -config config_lydaas.json

REM 记录结束时间
echo.
echo ════════════════════════════════════════════════════════════════
echo   扫描完成
echo ════════════════════════════════════════════════════════════════
echo.
echo 结束时间: %date% %time%
echo.

REM 列出生成的文件
echo 生成的文件：
echo.
dir /B spider_x.lydaas.com_*.txt 2>nul
dir /B spider_x.lydaas.com_*.json 2>nul

echo.
echo ════════════════════════════════════════════════════════════════
echo.

REM 检查是否生成了敏感信息文件
dir /B *_sensitive.txt 2>nul >nul
if %ERRORLEVEL% EQU 0 (
    echo ✅ 发现敏感信息文件！
    echo.
    for /f "delims=" %%f in ('dir /B /O-D *_sensitive.txt 2^>nul') do (
        echo 文件名: %%f
        echo ----------------------------------------
        echo 报告摘要（前20行）:
        powershell -Command "Get-Content '%%f' -TotalCount 20"
        goto :done_sensitive
    )
    :done_sensitive
) else (
    echo ℹ️  未发现敏感信息文件
    echo    可能原因：
    echo    1. 目标网站没有敏感信息泄露（正常情况）
    echo    2. 规则文件未正确加载
    echo    3. 爬取的页面数量较少
)

echo.
echo ════════════════════════════════════════════════════════════════
echo   使用提示
echo ════════════════════════════════════════════════════════════════
echo.
echo 如需深度扫描，可以增加参数：
echo   spider.exe -config config_lydaas.json -depth 8 -max-pages 1000
echo.
echo 如需使用标准规则（性能更好）：
echo   spider.exe -config config_lydaas.json -sensitive-rules sensitive_rules_standard.json
echo.
echo 如需禁用敏感信息检测：
echo   spider.exe -config config_lydaas.json -sensitive-detect=false
echo.
echo 详细说明请查看: 配置文件使用说明.md
echo.

pause

