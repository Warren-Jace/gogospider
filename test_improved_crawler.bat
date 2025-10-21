@echo off
echo 正在使用优化配置运行爬虫...
go run cmd/spider/main.go -config improved_config.json

echo.
echo 爬虫运行完成，请查看生成的报告文件。
pause