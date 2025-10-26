@echo off
chcp 65001 >nul
echo ================================================================================
echo                   Spider Ultimate - URL文件使用示例
echo ================================================================================
echo.
echo 本脚本演示如何使用Spider输出的URL文件
echo.

REM 检查是否已编译
if not exist "spider_fixed.exe" (
    echo [错误] 未找到 spider_fixed.exe
    echo 请先运行 build.bat 编译程序
    pause
    exit /b 1
)

echo [步骤1] 爬取测试网站...
echo.
spider_fixed.exe -url https://xss-quiz.int21h.jp/ -depth 2
echo.

echo ================================================================================
echo.
echo [步骤2] 查看生成的URL文件...
echo.

REM 查找最新生成的文件
for /f "delims=" %%i in ('dir /b /od spider_*_all_urls.txt 2^>nul') do set LATEST_ALL=%%i
for /f "delims=" %%i in ('dir /b /od spider_*_params.txt 2^>nul') do set LATEST_PARAMS=%%i
for /f "delims=" %%i in ('dir /b /od spider_*_apis.txt 2^>nul') do set LATEST_APIS=%%i

if defined LATEST_ALL (
    echo ✓ 找到完整URL文件: %LATEST_ALL%
    for /f %%a in ('find /c /v "" ^< %LATEST_ALL%') do set ALL_COUNT=%%a
    echo   - 包含 %ALL_COUNT% 个URL
    echo.
    
    echo 【前10个URL预览】:
    powershell -Command "Get-Content '%LATEST_ALL%' | Select-Object -First 10"
    echo   ...
    echo.
) else (
    echo ✗ 未找到URL文件
)

if defined LATEST_PARAMS (
    echo ✓ 找到参数URL文件: %LATEST_PARAMS%
    for /f %%a in ('find /c /v "" ^< %LATEST_PARAMS%') do set PARAMS_COUNT=%%a
    echo   - 包含 %PARAMS_COUNT% 个带参数的URL
    echo.
)

if defined LATEST_APIS (
    echo ✓ 找到API文件: %LATEST_APIS%
    for /f %%a in ('find /c /v "" ^< %LATEST_APIS%') do set APIS_COUNT=%%a
    echo   - 包含 %APIS_COUNT% 个API接口
    echo.
)

echo ================================================================================
echo.
echo [步骤3] 使用URL文件的示例...
echo.

if defined LATEST_ALL (
    echo 【示例1】统计URL类型:
    powershell -Command "$urls = Get-Content '%LATEST_ALL%'; $admin = ($urls | Where-Object { $_ -match 'admin' }).Count; $api = ($urls | Where-Object { $_ -match 'api' }).Count; $param = ($urls | Where-Object { $_ -match '\?' }).Count; Write-Host \"  - 管理后台相关: $admin 个\"; Write-Host \"  - API接口: $api 个\"; Write-Host \"  - 带参数: $param 个\""
    echo.
    
    echo 【示例2】提取所有带参数的URL:
    powershell -Command "Get-Content '%LATEST_ALL%' | Where-Object { $_ -match '\?' } | Select-Object -First 5"
    echo   ...
    echo.
    
    echo 【示例3】查找高价值URL:
    powershell -Command "Get-Content '%LATEST_ALL%' | Where-Object { $_ -match '(admin|login|upload|config|api|auth)' } | Select-Object -First 5"
    echo   ...
    echo.
    
    echo 【示例4】与其他工具集成:
    echo.
    echo   # 使用httpx批量探测状态
    echo   type %LATEST_ALL% ^| httpx -status-code -title
    echo.
    echo   # 使用nuclei批量扫描漏洞
    echo   nuclei -l %LATEST_ALL% -t vulnerabilities/
    echo.
    echo   # 使用ffuf进行参数Fuzz（如果有参数文件）
    if defined LATEST_PARAMS (
        echo   ffuf -w wordlist.txt -u FUZZ -ic ^< %LATEST_PARAMS%
    )
    echo.
    echo   # 使用sqlmap批量SQL注入测试（如果有参数文件）
    if defined LATEST_PARAMS (
        echo   sqlmap -m %LATEST_PARAMS% --batch --level=5
    )
    echo.
)

echo ================================================================================
echo.
echo [完成] URL文件使用演示完成！
echo.
echo 生成的文件说明：
echo   - *_all_urls.txt   : 所有URL（推荐）
echo   - *_params.txt     : 带参数的URL（参数Fuzz）
echo   - *_apis.txt       : API接口（API测试）
echo   - *_forms.txt      : 表单URL（表单注入）
echo   - *_urls.txt       : 兼容旧版
echo.
echo 详细说明请查看：URL输出文件说明.md
echo.
echo ================================================================================
pause

