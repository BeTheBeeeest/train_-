@echo off
chcp 65001 >nul
echo.
echo ============================================
echo   火车票候补系统 - 设计图表查看器
echo ============================================
echo.
echo 正在打开设计图表页面...
echo.
echo 功能说明：
echo - 类模型图：展示候补功能的数据关系
echo - 活动图：展示候补功能的业务流程
echo - 状态转换图：展示订单状态变化
echo.
echo 提示：
echo 1. 页面将在默认浏览器中打开
echo 2. 图表使用Mermaid.js动态生成
echo 3. 支持点击导航快速跳转到各个图表
echo.

start "" "diagram.html"

echo 设计图表页面已打开！
echo.
echo 按任意键关闭此窗口...
pause >nul
