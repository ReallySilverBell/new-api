@echo off
chcp 65001 >nul
setlocal

:: 1. 设置路径并进入目录
set "PROJECT_ROOT=G:\AIChat\new_api\new-api"
cd /d "%PROJECT_ROOT%"

:: 2. 核心检查：如果没有 exe 则尝试编译
if not exist "new-api.exe" (
    echo [编译] 未找到程序，正在尝试 go build...
    go build -o new-api.exe || (echo [错误] 编译失败，请检查 Go 环境 & pause & exit /b 1)
)

:: 3. 运行服务
echo [启动] New API 正在运行...
echo 管理页面: http://localhost:3000/console/dynamic-weight
new-api.exe

:: 4. 异常退出保护
if %ERRORLEVEL% neq 0 (
    echo [错误] 程序异常退出 (代码: %ERRORLEVEL%)
    pause
)

endlocal