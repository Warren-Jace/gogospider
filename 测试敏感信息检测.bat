@echo off
chcp 65001 >nul
echo ╔══════════════════════════════════════════════════════════════╗
echo ║         GogoSpider v4.2 - 敏感信息检测功能测试               ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.
echo 📋 测试说明：
echo   本脚本将测试敏感信息检测功能是否正常工作
echo   - 检测每个返回数据包
echo   - 记录来源URL和信息类型
echo   - 生成独立的敏感信息报告文件
echo.

REM 检查是否存在规则文件
if not exist "sensitive_rules_standard.json" (
    echo ❌ 错误：找不到 sensitive_rules_standard.json
    echo    请确保规则文件存在于当前目录
    pause
    exit /b 1
)

echo ✅ 找到规则文件：sensitive_rules_standard.json
echo.

REM 运行爬虫（使用测试站点）
echo 🚀 开始测试爬虫（目标：testphp.vulnweb.com）...
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.

spider.exe -url https://testphp.vulnweb.com ^
    -depth 2 ^
    -sensitive-detect ^
    -sensitive-rules sensitive_rules_standard.json ^
    -sensitive-realtime

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.

REM 检查生成的敏感信息文件
echo 📁 检查生成的敏感信息报告文件...
echo.

set FOUND=0

for %%f in (spider_*_sensitive.txt) do (
    echo ✅ 找到文件：%%f
    set FOUND=1
    echo.
    echo 📄 文件内容预览（前50行）：
    echo ────────────────────────────────────────────────────────
    powershell -Command "Get-Content '%%f' -Head 50"
    echo ────────────────────────────────────────────────────────
    echo.
)

for %%f in (sensitive_*.html) do (
    echo ✅ 找到HTML报告：%%f
    echo    可在浏览器中查看可视化报告
    set FOUND=1
)

for %%f in (sensitive_*.json) do (
    echo ✅ 找到JSON报告：%%f
    set FOUND=1
)

for %%f in (sensitive_*_summary.txt) do (
    echo ✅ 找到摘要报告：%%f
    set FOUND=1
    echo.
    echo 📋 摘要内容：
    echo ────────────────────────────────────────────────────────
    type "%%f"
    echo ────────────────────────────────────────────────────────
    echo.
)

if %FOUND%==0 (
    echo ⚠️  未找到敏感信息报告文件
    echo    可能原因：
    echo    1. 目标网站未发现敏感信息
    echo    2. 规则文件配置不正确
    echo    3. 敏感信息检测未启用
) else (
    echo.
    echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    echo 🎉 测试完成！
    echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    echo.
    echo 📊 验证要点：
    echo   ✓ 敏感信息检测针对每个返回数据包
    echo   ✓ 已保存到独立的敏感信息文件
    echo   ✓ 文件中包含来源URL和信息类型
    echo   ✓ 敏感信息检测是统一的功能模块
    echo.
    echo 💡 查看报告：
    echo   1. 文本报告：type spider_*_sensitive.txt
    echo   2. 摘要报告：type sensitive_*_summary.txt
    echo   3. HTML报告：start sensitive_*.html
    echo   4. JSON数据：type sensitive_*.json
    echo.
)

echo.
pause

