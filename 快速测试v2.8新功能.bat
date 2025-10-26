@echo off
chcp 65001 >nul
echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║         gogospider v2.8 新功能快速测试                       ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.

echo 提示：本脚本将测试v2.8的所有新功能
echo      请确保已创建测试HTML文件
echo.
pause

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo  测试1/4: Base64 URL解码功能
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
echo 测试文件: http://localhost/test_base64.html
echo 预期结果: 提取Base64编码的URL
echo.

spider_v2.8.exe -url http://localhost/test_base64.html -depth 1 -output test1_base64.json

if %ERRORLEVEL% EQU 0 (
    echo ✅ 测试1完成！查看 test1_base64.json
) else (
    echo ❌ 测试1失败，请检查：
    echo    1. 是否创建了 test_base64.html
    echo    2. 是否启动了本地Web服务器
)

echo.
pause

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo  测试2/4: CSS URL提取功能
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
echo 测试文件: http://localhost/test_css.html
echo 预期结果: 提取CSS中的url()、@import、@font-face
echo.

spider_v2.8.exe -url http://localhost/test_css.html -depth 1 -output test2_css.json

if %ERRORLEVEL% EQU 0 (
    echo ✅ 测试2完成！查看 test2_css.json
) else (
    echo ❌ 测试2失败
)

echo.
pause

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo  测试3/4: srcset响应式图片支持
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
echo 测试文件: http://localhost/test_srcset.html
echo 预期结果: 提取srcset、picture标签中的所有图片URL
echo.

spider_v2.8.exe -url http://localhost/test_srcset.html -depth 1 -output test3_srcset.json

if %ERRORLEVEL% EQU 0 (
    echo ✅ 测试3完成！查看 test3_srcset.json
) else (
    echo ❌ 测试3失败
)

echo.
pause

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo  测试4/4: 资源智能分类（核心功能）
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
echo 测试文件: http://localhost/test_classification.html
echo 预期结果: 静态资源只收集不请求，页面/JS/CSS正常爬取
echo.

spider_v2.8.exe -url http://localhost/test_classification.html -depth 2 -output test4_classification.json

if %ERRORLEVEL% EQU 0 (
    echo ✅ 测试4完成！查看 test4_classification.json
) else (
    echo ❌ 测试4失败
)

echo.
echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║                  测试完成！                                  ║
echo ╠══════════════════════════════════════════════════════════════╣
echo ║                                                              ║
echo ║  生成的结果文件：                                            ║
echo ║    test1_base64.json         - Base64解码测试                ║
echo ║    test2_css.json            - CSS提取测试                   ║
echo ║    test3_srcset.json         - srcset测试                    ║
echo ║    test4_classification.json - 资源分类测试                  ║
echo ║                                                              ║
echo ║  查看要点：                                                  ║
echo ║    1. base64_decoded_urls 字段 - Base64解码的URL            ║
echo ║    2. css_extracted_urls 字段  - CSS提取的URL               ║
echo ║    3. srcset_images 字段       - srcset图片URL               ║
echo ║    4. resource_classification  - 资源分类统计                ║
echo ║                                                              ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.

pause

echo.
echo 提示：如果测试失败，请检查：
echo   1. 是否创建了测试HTML文件
echo   2. 是否启动了本地Web服务器（如python -m http.server）
echo   3. 测试文件是否在正确的位置
echo.
echo 详细使用说明请查看：
echo   - 🎉v2.8编译成功-快速测试.md
echo   - v2.8使用指南.md
echo.

pause

