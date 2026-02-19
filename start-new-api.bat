@echo off
setlocal

:: 设置项目根路径
set "PROJECT_ROOT=G:\AIChat\new_api\new-api"

:: 切换到项目目录 (/d 参数用于跨盘符切换)
cd /d "%PROJECT_ROOT%"

echo ==========================================
echo [INFO] 正在启动 New API (动态权重版本)...
echo [INFO] 工作目录: %PROJECT_ROOT%
echo ==========================================

:: 检查 .env 文件是否存在
if not exist ".env" (
    echo [ERROR] 配置文件 .env 不存在，请先创建。
    pause
    exit /b 1
)

:: 检查编译好的可执行文件是否存在
if not exist "new-api.exe" (
    echo [WARN] 未找到 new-api.exe，正在编译...
    
    :: 检查 Go 环境
    where go >nul 2>nul
    if %ERRORLEVEL% neq 0 (
        echo [ERROR] 未找到 Go 环境，请确保 Go 已安装并添加到系统变量 PATH 中。
        pause
        exit /b 1
    )
    
    echo [INFO] 执行 go build -o new-api.exe ...
    go build -o new-api.exe
    
    if %ERRORLEVEL% neq 0 (
        echo [ERROR] 编译失败，错误代码: %ERRORLEVEL%
        pause
        exit /b 1
    )
    
    echo [SUCCESS] 编译完成！
    echo.
)

:: 显示版本信息
echo [INFO] 启动信息:
echo   - 可执行文件: new-api.exe
echo   - 包含功能: 动态权重管理
echo   - 管理页面: http://localhost:3000/console/dynamic-weight
echo ==========================================
echo.

:: 运行编译好的可执行文件
echo [INFO] 启动服务...
new-api.exe

:: 如果程序异常退出，保留窗口查看错误
if %ERRORLEVEL% neq 0 (
    echo.
    echo [ERROR] 服务异常退出，错误代码: %ERRORLEVEL%
    pause
)

endlocal
