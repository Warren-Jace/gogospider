@echo off
chcp 65001 >nul
echo ========================================
echo   敏感信息检测功能实际测试
echo ========================================
echo.

echo [步骤1] 检查测试文件
if exist test_sensitive.html (
    echo ✓ 测试HTML文件已创建: test_sensitive.html
) else (
    echo ✗ 测试文件不存在
    echo   请先运行上面的创建命令
    pause
    exit /b 1
)
echo.

echo [步骤2] 启动爬虫扫描测试文件
echo.
echo 命令：spider.exe -url file:///%CD%/test_sensitive.html -depth 1
echo.

spider.exe -url "file:///%CD%/test_sensitive.html" -depth 1

echo.
echo ========================================
echo [步骤3] 检查结果
echo ========================================
echo.

echo 查找生成的敏感信息文件...
echo.

dir /B *_sensitive.txt 2>nul
if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ 找到敏感信息文件！修复成功！
    echo.
    echo 最新的敏感信息报告：
    for /f "delims=" %%f in ('dir /B /O-D *_sensitive.txt 2^>nul') do (
        echo ----------------------------------------
        type "%%f"
        goto :done
    )
    :done
) else (
    echo ❌ 未找到敏感信息文件
    echo.
    echo 请检查上面的输出，确认：
    echo 1. 是否显示 "✅ 已加载敏感信息规则文件"
    echo 2. 是否显示 "[敏感信息] 发现 X 处敏感信息"
)

echo.
echo ========================================
pause

