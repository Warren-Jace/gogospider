@echo off
chcp 65001 >nul
echo ═══════════════════════════════════════════════════════
echo   GogoSpider v3.4 - 快速启动脚本
echo   默认算法: HYBRID（广度优先 + 优先级策略 + 自适应学习）
echo ═══════════════════════════════════════════════════════
echo.

REM 设置目标URL（请修改为你的目标）
set TARGET_URL=http://x.lydaas.com

echo 【配置信息】
echo   目标URL: %TARGET_URL%
echo   配置文件: config.json
echo   调度策略: HYBRID混合策略（BFS+优先级+自适应）
echo   最大深度: 5层
echo   自适应学习: 已启用
echo.

echo 【开始爬取】
echo   正在启动爬虫...
echo.

REM 运行爬虫
spider.exe -url %TARGET_URL% -config config.json

echo.
echo ═══════════════════════════════════════════════════════
echo   爬取完成！
echo ═══════════════════════════════════════════════════════
pause

