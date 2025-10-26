@echo off
chcp 65001 >nul
echo ========================================
echo   敏感信息检测功能测试脚本
echo ========================================
echo.

echo [测试1] 检查规则文件是否存在
echo ----------------------------------------
if exist sensitive_rules_config.json (
    echo ✓ 默认规则文件存在: sensitive_rules_config.json
) else (
    echo ✗ 默认规则文件不存在: sensitive_rules_config.json
    echo   请确保文件存在或使用其他规则文件
)

if exist sensitive_rules_standard.json (
    echo ✓ 标准规则文件存在: sensitive_rules_standard.json
) else (
    echo ✗ 标准规则文件不存在: sensitive_rules_standard.json
)

if exist sensitive_rules_minimal.json (
    echo ✓ 精简规则文件存在: sensitive_rules_minimal.json
) else (
    echo ✗ 精简规则文件不存在: sensitive_rules_minimal.json
)
echo.

echo [测试2] 检查可执行文件
echo ----------------------------------------
if exist spider.exe (
    echo ✓ 爬虫程序存在: spider.exe
) else (
    echo ✗ 爬虫程序不存在，请先编译:
    echo   go build -o spider.exe cmd/spider/main.go
    pause
    exit /b 1
)
echo.

echo [提示] 修复内容说明
echo ----------------------------------------
echo 修复前: 即使启用敏感信息检测，也不会加载规则文件
echo 修复后: 自动加载配置中指定的默认规则文件
echo.

echo [使用建议] 
echo ----------------------------------------
echo 1. 快速扫描（推荐新手）:
echo    spider.exe -url https://example.com
echo.
echo 2. 使用标准规则（推荐）:
echo    spider.exe -url https://example.com -sensitive-rules sensitive_rules_standard.json
echo.
echo 3. 全面扫描:
echo    spider.exe -url https://example.com -sensitive-rules sensitive_rules_config.json
echo.
echo 4. 禁用敏感信息检测:
echo    spider.exe -url https://example.com -sensitive-detect=false
echo.

echo [预期输出] 修复成功的标志
echo ----------------------------------------
echo 启动时应该看到以下提示之一:
echo ✅ 已加载敏感信息规则文件: ./sensitive_rules_config.json
echo 或
echo ✅ 已加载敏感信息规则文件: sensitive_rules_standard.json
echo.
echo 如果看到警告提示，说明规则文件未找到:
echo ⚠️  警告: 加载敏感规则失败: ...
echo 💡 提示: 请使用 -sensitive-rules 参数指定规则文件
echo.

echo [生成的敏感信息文件]
echo ----------------------------------------
echo 成功扫描后会生成以下文件（如果发现敏感信息）:
echo   spider_[domain]_[timestamp]_sensitive.txt   - 文本格式报告
echo   spider_[domain]_[timestamp]_sensitive.json  - JSON格式报告
echo.

echo ========================================
echo   测试完成！详细说明请查看:
echo   敏感信息检测修复说明.md
echo ========================================
echo.

pause

