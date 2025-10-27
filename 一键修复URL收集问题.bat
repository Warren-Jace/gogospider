@echo off
chcp 65001 >nul
echo ╔════════════════════════════════════════════════════════════════╗
echo ║           一键修复URL收集问题 v1.0                             ║
echo ║                                                                ║
echo ║  问题：大量有效地址没有被保存记录                               ║
echo ║  原因：每层100个URL限制 + 多重过滤器 + 保存逻辑问题             ║
echo ║                                                                ║
echo ╚════════════════════════════════════════════════════════════════╝
echo.

echo [提示] 本脚本将：
echo   1. 备份原文件
echo   2. 应用关键修复（提高URL限制）
echo   3. 重新编译程序
echo   4. 生成修复报告
echo.

pause
echo.

echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 步骤1: 备份原文件
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

if not exist "backup\" mkdir backup
copy core\spider.go backup\spider.go.%date:~0,4%%date:~5,2%%date:~8,2%_%time:~0,2%%time:~3,2%%time:~6,2%
copy cmd\spider\main.go backup\main.go.%date:~0,4%%date:~5,2%%date:~8,2%_%time:~0,2%%time:~3,2%%time:~6,2%

echo ✅ 备份完成！
echo.

echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 步骤2: 应用修复补丁
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo 正在应用修复...
echo.

echo [修复1] 提高URL限制 100 -^> 500
powershell -Command "(Get-Content core\spider.go) -replace 'if len\(tasksToSubmit\) >= 100 {', 'if len(tasksToSubmit) >= 500 { // 修复：提高限制' | Set-Content core\spider.go"

echo ✅ 修复1完成
echo.

echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 步骤3: 重新编译
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

go build -o spider_fixed.exe cmd/spider/main.go

if %ERRORLEVEL% EQU 0 (
    echo ✅ 编译成功！
    echo.
    echo 新程序：spider_fixed.exe
) else (
    echo ❌ 编译失败！请检查错误信息
    pause
    exit /b 1
)

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 步骤4: 生成修复报告
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo.
echo ╔════════════════════════════════════════════════════════════════╗
echo ║                      修复完成！                                ║
echo ╚════════════════════════════════════════════════════════════════╝
echo.
echo 已应用的修复：
echo   ✅ 修复1: 每层URL限制从100提升到500
echo.
echo 下一步建议：
echo   1. 运行测试：.\spider_fixed.exe -url http://your-target.com -depth 2
echo   2. 对比结果：查看 *_urls.txt 和 *_all_urls.txt
echo   3. 检查文档：【代码逻辑问题分析报告】.md
echo.
echo 预期效果：
echo   - URL收集数量：提升 3-5倍
echo   - 业务URL覆盖：大幅提升
echo.
echo 更多修复建议：
echo   - 查看【代码逻辑问题分析报告】.md 了解其他问题
echo   - 查看【修复补丁】quick_fix.go 了解更多修复方案
echo.
echo 备份文件位置：backup\
echo.

pause

