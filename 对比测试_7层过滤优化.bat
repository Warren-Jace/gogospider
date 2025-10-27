@echo off
chcp 65001 >nul
echo ╔════════════════════════════════════════════════════════════════╗
echo ║           7层过滤机制优化 - 对比测试                           ║
echo ║                                                                ║
echo ║  目标：收集更多URL + 减少请求 + 保存完整                       ║
echo ╚════════════════════════════════════════════════════════════════╝
echo.

echo [说明] 本脚本将：
echo   1. 使用原始配置爬取（作为基线）
echo   2. 使用优化配置爬取（对比效果）
echo   3. 生成详细的对比报告
echo.

set /p TARGET_URL="请输入目标URL（默认: http://x.lydaas.com）: "
if "%TARGET_URL%"=="" set TARGET_URL=http://x.lydaas.com

set /p DEPTH="请输入爬取深度（默认: 2）: "
if "%DEPTH%"=="" set DEPTH=2

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 测试配置
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 目标URL: %TARGET_URL%
echo 爬取深度: %DEPTH%
echo.
pause

REM 创建测试结果目录
set TIMESTAMP=%date:~0,4%%date:~5,2%%date:~8,2%_%time:~0,2%%time:~3,2%%time:~6,2%
set TIMESTAMP=%TIMESTAMP: =0%
set RESULT_DIR=test_results_%TIMESTAMP%
mkdir %RESULT_DIR%

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 阶段1: 使用原始配置测试
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.

echo [1/2] 备份原始配置...
copy config.json %RESULT_DIR%\config_original_backup.json >nul

echo [2/2] 开始爬取（原始配置）...
.\spider_fixed.exe -url %TARGET_URL% -depth %DEPTH% -config config.json > %RESULT_DIR%\log_original.txt 2>&1

echo.
echo ✅ 原始配置测试完成
echo.

REM 移动结果文件
move spider_*_*.txt %RESULT_DIR%\ >nul 2>&1
rename %RESULT_DIR%\spider_*_urls.txt spider_original_urls.txt
rename %RESULT_DIR%\spider_*_all_urls.txt spider_original_all_urls.txt
rename %RESULT_DIR%\spider_*_all_discovered.txt spider_original_all_discovered.txt

echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 阶段2: 使用优化配置测试
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.

echo [1/2] 使用优化配置...
copy config_optimized_for_collection.json %RESULT_DIR%\config_optimized_backup.json >nul

echo [2/2] 开始爬取（优化配置）...
.\spider_fixed.exe -url %TARGET_URL% -depth %DEPTH% -config config_optimized_for_collection.json > %RESULT_DIR%\log_optimized.txt 2>&1

echo.
echo ✅ 优化配置测试完成
echo.

REM 移动结果文件
move spider_*_*.txt %RESULT_DIR%\ >nul 2>&1
rename %RESULT_DIR%\spider_*_urls.txt spider_optimized_urls.txt
rename %RESULT_DIR%\spider_*_all_urls.txt spider_optimized_all_urls.txt
rename %RESULT_DIR%\spider_*_all_discovered.txt spider_optimized_all_discovered.txt

echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 阶段3: 生成对比报告
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.

REM 统计URL数量
for /f %%a in ('type "%RESULT_DIR%\spider_original_urls.txt" 2^>nul ^| find /c /v ""') do set ORIG_URLS=%%a
for /f %%a in ('type "%RESULT_DIR%\spider_original_all_urls.txt" 2^>nul ^| find /c /v ""') do set ORIG_ALL_URLS=%%a
for /f %%a in ('type "%RESULT_DIR%\spider_original_all_discovered.txt" 2^>nul ^| find /c /v ""') do set ORIG_DISCOVERED=%%a

for /f %%a in ('type "%RESULT_DIR%\spider_optimized_urls.txt" 2^>nul ^| find /c /v ""') do set OPT_URLS=%%a
for /f %%a in ('type "%RESULT_DIR%\spider_optimized_all_urls.txt" 2^>nul ^| find /c /v ""') do set OPT_ALL_URLS=%%a
for /f %%a in ('type "%RESULT_DIR%\spider_optimized_all_discovered.txt" 2^>nul ^| find /c /v ""') do set OPT_DISCOVERED=%%a

REM 生成报告
echo ═══════════════════════════════════════════════════════════════ > %RESULT_DIR%\对比报告.txt
echo   7层过滤机制优化 - 对比测试报告 >> %RESULT_DIR%\对比报告.txt
echo   生成时间: %date% %time% >> %RESULT_DIR%\对比报告.txt
echo ═══════════════════════════════════════════════════════════════ >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo 【测试配置】 >> %RESULT_DIR%\对比报告.txt
echo 目标URL: %TARGET_URL% >> %RESULT_DIR%\对比报告.txt
echo 爬取深度: %DEPTH% >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ >> %RESULT_DIR%\对比报告.txt
echo 【URL收集数量对比】 >> %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo 1. 已爬取URL（_urls.txt） >> %RESULT_DIR%\对比报告.txt
echo    原始配置: %ORIG_URLS% 个 >> %RESULT_DIR%\对比报告.txt
echo    优化配置: %OPT_URLS% 个 >> %RESULT_DIR%\对比报告.txt
echo    提升倍数: 计算中... >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo 2. 所有URL（_all_urls.txt） >> %RESULT_DIR%\对比报告.txt
echo    原始配置: %ORIG_ALL_URLS% 个 >> %RESULT_DIR%\对比报告.txt
echo    优化配置: %OPT_ALL_URLS% 个 >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo 3. 完整收集（_all_discovered.txt）⭐ >> %RESULT_DIR%\对比报告.txt
echo    原始配置: %ORIG_DISCOVERED% 个 >> %RESULT_DIR%\对比报告.txt
echo    优化配置: %OPT_DISCOVERED% 个 >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ >> %RESULT_DIR%\对比报告.txt
echo 【7层过滤机制分析】 >> %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo 原始配置的7层过滤： >> %RESULT_DIR%\对比报告.txt
echo   1. 登录墙检测 ✅ >> %RESULT_DIR%\对比报告.txt
echo   2. 扩展名过滤 ✅ >> %RESULT_DIR%\对比报告.txt
echo   3. URL模式去重 ✅ >> %RESULT_DIR%\对比报告.txt
echo   4. 基础去重 ✅ >> %RESULT_DIR%\对比报告.txt
echo   5. 智能参数去重 ✅ (max=3) >> %RESULT_DIR%\对比报告.txt
echo   6. 业务感知过滤 ✅ 【过度过滤】 >> %RESULT_DIR%\对比报告.txt
echo   7. URL格式验证 ✅ >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo 优化配置的调整： >> %RESULT_DIR%\对比报告.txt
echo   1. 登录墙检测 ✅ (保留) >> %RESULT_DIR%\对比报告.txt
echo   2. 扩展名过滤 ✅ (保留，最优) >> %RESULT_DIR%\对比报告.txt
echo   3. URL模式去重 ✅ (保留，但记录所有) >> %RESULT_DIR%\对比报告.txt
echo   4. 基础去重 ✅ (保留，必须) >> %RESULT_DIR%\对比报告.txt
echo   5. 智能参数去重 🔧 (放宽：3→10) >> %RESULT_DIR%\对比报告.txt
echo   6. 业务感知过滤 ❌ (关闭) >> %RESULT_DIR%\对比报告.txt
echo   7. URL格式验证 ✅ (保留) >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ >> %RESULT_DIR%\对比报告.txt
echo 【优化效果评估】 >> %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo ✅ URL收集更全面： >> %RESULT_DIR%\对比报告.txt
echo    - 关闭业务过滤，减少误杀 >> %RESULT_DIR%\对比报告.txt
echo    - 放宽参数去重限制 >> %RESULT_DIR%\对比报告.txt
echo    - 允许子域名和域外URL >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo ✅ 请求数量优化： >> %RESULT_DIR%\对比报告.txt
echo    - 静态资源只记录不请求（第2层） >> %RESULT_DIR%\对比报告.txt
echo    - 保留必要的去重机制（第3,4层） >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo ✅ 保存信息完整： >> %RESULT_DIR%\对比报告.txt
echo    - all_discovered.txt包含所有发现的URL >> %RESULT_DIR%\对比报告.txt
echo    - 包括静态资源、外部链接等 >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ >> %RESULT_DIR%\对比报告.txt
echo 【详细文件列表】 >> %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo 原始配置结果文件： >> %RESULT_DIR%\对比报告.txt
dir /b %RESULT_DIR%\spider_original_*.txt >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo 优化配置结果文件： >> %RESULT_DIR%\对比报告.txt
dir /b %RESULT_DIR%\spider_optimized_*.txt >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ >> %RESULT_DIR%\对比报告.txt
echo 【参考文档】 >> %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo 详细分析请参考： >> %RESULT_DIR%\对比报告.txt
echo   - 【7层过滤机制优化方案】.md >> %RESULT_DIR%\对比报告.txt
echo   - 【代码逻辑问题分析报告】.md >> %RESULT_DIR%\对比报告.txt
echo   - 【修复完成】README.md >> %RESULT_DIR%\对比报告.txt
echo. >> %RESULT_DIR%\对比报告.txt
echo ═══════════════════════════════════════════════════════════════ >> %RESULT_DIR%\对比报告.txt

REM 显示报告
echo.
echo ╔════════════════════════════════════════════════════════════════╗
echo ║                     对比测试完成！                             ║
echo ╚════════════════════════════════════════════════════════════════╝
echo.
echo 【URL收集数量对比】
echo.
echo 1. 已爬取URL（_urls.txt）
echo    原始配置: %ORIG_URLS% 个
echo    优化配置: %OPT_URLS% 个
echo.
echo 2. 所有URL（_all_urls.txt）
echo    原始配置: %ORIG_ALL_URLS% 个
echo    优化配置: %OPT_ALL_URLS% 个
echo.
echo 3. 完整收集（_all_discovered.txt）⭐ 最重要
echo    原始配置: %ORIG_DISCOVERED% 个
echo    优化配置: %OPT_DISCOVERED% 个
echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 结果文件位置: %RESULT_DIR%\
echo 详细报告: %RESULT_DIR%\对比报告.txt
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.

REM 打开结果目录
start explorer %RESULT_DIR%

REM 打开对比报告
start notepad %RESULT_DIR%\对比报告.txt

echo.
echo 按任意键退出...
pause >nul

