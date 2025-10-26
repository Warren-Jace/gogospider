@echo off
chcp 65001 >nul
echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║         gogospider v2.8 完整功能测试                         ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.

echo 本脚本将测试v2.8的所有新功能：
echo   1. 200个常见路径扫描
echo   2. URL去重保存功能
echo   3. 优先级队列算法
echo   4. 资源智能分类
echo.
echo 提示：可以用真实网站测试，也可以用本地测试页面
echo.

set /p TEST_URL="请输入测试URL（例如：https://testphp.vulnweb.com）: "

if "%TEST_URL%"=="" (
    echo 未输入URL，使用默认测试地址
    set TEST_URL=https://testphp.vulnweb.com
)

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo  测试目标: %TEST_URL%
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
pause

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo  测试1/2: BFS模式（默认，推荐）
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
echo 特点：逐层扫描，稳定可靠，精确深度控制
echo 功能：自动扫描200路径 + URL去重保存
echo.

spider_v2.8_final.exe -url %TEST_URL% -depth 3

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ 测试1完成！
    echo.
    echo 请检查生成的文件：
    echo   1. *_unique_urls.txt    - 去重URL（给其他工具用）
    echo   2. *_all_urls.txt       - 所有URL（完整记录）
    echo   3. *_params.txt         - 带参数URL
    echo   4. 查看控制台输出中的[路径发现]提示
    echo.
) else (
    echo ❌ 测试1失败
)

pause

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo  测试2/2: 优先级队列模式（实验性）
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
echo 特点：智能排序，高价值URL优先
echo 功能：优先发现 /admin, /api, /login等重要路径
echo.

spider_v2.8_final.exe -config config_v2.8_priority_mode.json

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ 测试2完成！
    echo.
    echo 对比两种模式的输出：
    echo   BFS模式：按层级爬取（第1层、第2层...）
    echo   优先级模式：按优先级爬取（高分URL先爬）
    echo.
) else (
    echo.
    echo ⚠️  测试2失败（可能需要修改config中的target_url）
    echo.
)

echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║                  测试完成！                                  ║
echo ╠══════════════════════════════════════════════════════════════╣
echo ║                                                              ║
echo ║  生成的关键文件：                                            ║
echo ║                                                              ║
echo ║  *_unique_urls.txt  🎯 去重URL（最重要）                    ║
echo ║    • 参数值已清空                                            ║
echo ║    • 去重率90%+                                              ║
echo ║    • 适合给sqlmap/nuclei/xray使用                           ║
echo ║                                                              ║
echo ║  *_all_urls.txt     📋 所有URL                              ║
echo ║    • 完整URL列表                                             ║
echo ║    • 包含所有静态资源                                        ║
echo ║    • 用于资产盘点                                            ║
echo ║                                                              ║
echo ║  控制台输出关注：                                            ║
echo ║    [路径发现] 发现 X/200 个常见业务路径                     ║
echo ║    [URL去重] 减少 X个 (X%)                                  ║
echo ║    [资源分类] 跳过 X个静态资源                              ║
echo ║                                                              ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.

echo 使用去重URL进行漏洞扫描：
echo.
echo   # nuclei
echo   nuclei -l *_unique_urls.txt -t cves/
echo.
echo   # sqlmap
echo   cat *_unique_urls.txt ^| xargs -I {} sqlmap -u {}
echo.
echo   # xray
echo   cat *_unique_urls.txt ^| xray webscan
echo.

pause

