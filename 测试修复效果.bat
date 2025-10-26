@echo off
chcp 65001 >nul
echo ════════════════════════════════════════════════════════
echo   GogoSpider 修复版本测试脚本
echo ════════════════════════════════════════════════════════
echo.

echo [1/5] 清理旧的测试文件...
if exist log-test.log del /q log-test.log
if exist spider_testphp.vulnweb.com_*_all_urls.txt del /q spider_testphp.vulnweb.com_*_all_urls.txt
if exist spider_testphp.vulnweb.com_*_excluded.txt del /q spider_testphp.vulnweb.com_*_excluded.txt
echo       ✓ 清理完成
echo.

echo [2/5] 运行修复版本程序（输出到文件）...
echo       这可能需要1-2分钟，请稍候...
.\spider_fixed.exe -url http://testphp.vulnweb.com/ -config .\config.json > log-test.log 2>&1
echo       ✓ 爬取完成
echo.

echo [3/5] 检查中文编码...
findstr /C:"爬虫" log-test.log >nul
if %errorlevel%==0 (
    echo       ✓ 中文显示正常
) else (
    echo       ✗ 中文可能有问题
)
echo.

echo [4/5] 统计结果文件...
for %%f in (spider_testphp.vulnweb.com_*_all_urls.txt) do (
    for /f %%a in ('type "%%f" ^| find /c /v ""') do (
        echo       all_urls.txt: %%a 个URL
    )
)

for %%f in (spider_testphp.vulnweb.com_*_excluded.txt) do (
    if exist "%%f" (
        for /f %%a in ('type "%%f" ^| find /c /v ""') do (
            echo       excluded.txt: %%a 行
        )
    )
)
echo.

echo [5/5] 验证关键内容...
echo       检查是否包含外部链接...
findstr /C:"acunetix.com" spider_testphp.vulnweb.com_*_all_urls.txt >nul
if %errorlevel%==0 (
    echo       ✓ all_urls.txt包含外部链接
) else (
    echo       ⚠ all_urls.txt不包含外部链接（检查excluded.txt）
)

echo       检查是否包含图片资源...
findstr /C:".jpg" spider_testphp.vulnweb.com_*_all_urls.txt >nul
if %errorlevel%==0 (
    echo       ✓ all_urls.txt包含图片资源
) else (
    echo       ⚠ all_urls.txt不包含图片资源（检查excluded.txt）
)
echo.

echo ════════════════════════════════════════════════════════
echo   测试完成！
echo ════════════════════════════════════════════════════════
echo.
echo 请检查以下文件：
echo   - log-test.log : 完整日志（检查中文是否正常）
echo   - spider_testphp.vulnweb.com_*_all_urls.txt : 所有发现的URL
echo   - spider_testphp.vulnweb.com_*_excluded.txt : 排除的URL分类
echo.
pause

