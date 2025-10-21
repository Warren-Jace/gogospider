@echo off
echo 开始运行爬虫测试...
echo 目标: http://testphp.vulnweb.com/
echo 配置文件: test_config.json
echo.

go run cmd/spider/main.go -config test_config.json

echo.
echo 测试完成
pause