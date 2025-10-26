@echo off
REM Spider Ultimate 使用示例
REM 展示各种常用命令

echo ========================================
echo  Spider Ultimate 使用示例
echo ========================================
echo.
echo 选择要执行的示例:
echo.
echo 1. 基础爬取 (深度3层，静态模式)
echo 2. 完整爬取 (深度5层，智能模式)
echo 3. 参数爆破 (启用GET/POST参数fuzz)
echo 4. 使用代理 (通过Burp Suite)
echo 5. 批量爬取 (从文件读取URL列表)
echo 6. 使用配置文件
echo 7. Pipeline模式 (简洁输出)
echo 8. JSON格式输出
echo 9. 查看程序帮助
echo 0. 退出
echo.

set /p choice=请输入选项 (0-9): 

if "%choice%"=="1" goto example1
if "%choice%"=="2" goto example2
if "%choice%"=="3" goto example3
if "%choice%"=="4" goto example4
if "%choice%"=="5" goto example5
if "%choice%"=="6" goto example6
if "%choice%"=="7" goto example7
if "%choice%"=="8" goto example8
if "%choice%"=="9" goto example9
if "%choice%"=="0" exit /b 0

echo 无效选项，请重新运行
pause
exit /b 1

:example1
echo.
echo ========================================
echo 示例1: 基础爬取
echo ========================================
echo 命令: spider.exe -url http://testphp.vulnweb.com -mode static -depth 3
echo.
pause
spider.exe -url http://testphp.vulnweb.com -mode static -depth 3
goto end

:example2
echo.
echo ========================================
echo 示例2: 完整爬取（智能模式）
echo ========================================
echo 命令: spider.exe -url http://testphp.vulnweb.com -mode smart -depth 5
echo.
echo 说明: 启用静态+动态双引擎，深度5层
echo.
pause
spider.exe -url http://testphp.vulnweb.com -mode smart -depth 5
goto end

:example3
echo.
echo ========================================
echo 示例3: 参数爆破
echo ========================================
echo 命令: spider.exe -url http://testphp.vulnweb.com -fuzz -depth 4
echo.
echo 说明: 启用GET/POST参数枚举，自动发现隐藏参数
echo.
pause
spider.exe -url http://testphp.vulnweb.com -fuzz -depth 4
goto end

:example4
echo.
echo ========================================
echo 示例4: 使用代理（Burp Suite）
echo ========================================
echo 命令: spider.exe -url http://testphp.vulnweb.com -proxy http://127.0.0.1:8080
echo.
echo 说明: 所有请求通过Burp Suite代理，方便进一步测试
echo 确保Burp Suite已启动并监听8080端口
echo.
pause
spider.exe -url http://testphp.vulnweb.com -proxy http://127.0.0.1:8080
goto end

:example5
echo.
echo ========================================
echo 示例5: 批量爬取
echo ========================================
echo.
echo 首先创建URL列表文件...
echo http://testphp.vulnweb.com > batch_urls.txt
echo http://testhtml5.vulnweb.com >> batch_urls.txt
echo.
echo 已创建 batch_urls.txt
echo.
echo 命令: for /F %%i in (batch_urls.txt) do spider.exe -url %%i -mode static
echo.
pause
for /F %%i in (batch_urls.txt) do (
    echo.
    echo [批量爬取] 正在爬取: %%i
    spider.exe -url %%i -mode static -depth 2
    echo.
)
goto end

:example6
echo.
echo ========================================
echo 示例6: 使用配置文件
echo ========================================
echo 命令: spider.exe -config example_config.json
echo.
echo 说明: 使用预定义的配置文件，所有参数从JSON读取
if not exist example_config.json (
    echo.
    echo [错误] example_config.json 不存在
    echo 请先创建配置文件或运行 test_spider.ps1
    pause
    goto end
)
echo.
pause
spider.exe -config example_config.json
goto end

:example7
echo.
echo ========================================
echo 示例7: Pipeline模式（简洁输出）
echo ========================================
echo 命令: spider.exe -url http://testphp.vulnweb.com -simple -format urls-only
echo.
echo 说明: 只输出URL，适合与其他工具配合使用
echo 例如: spider.exe -url ... -simple -format urls-only ^| findstr /i "admin"
echo.
pause
spider.exe -url http://testphp.vulnweb.com -simple -format urls-only -depth 2
goto end

:example8
echo.
echo ========================================
echo 示例8: JSON格式输出
echo ========================================
echo 命令: spider.exe -url http://testphp.vulnweb.com -format json -depth 2
echo.
echo 说明: 以JSON格式输出结果，方便程序化处理
echo.
pause
spider.exe -url http://testphp.vulnweb.com -format json -depth 2
goto end

:example9
echo.
echo ========================================
echo 程序帮助信息
echo ========================================
spider.exe -h
goto end

:end
echo.
echo ========================================
echo 示例执行完成
echo ========================================
echo.
echo 查看输出文件:
if exist spider_*.txt (
    dir /b spider_*.txt
)
if exist responses (
    echo responses\ 目录 - 包含原始响应数据
)
echo.
pause

