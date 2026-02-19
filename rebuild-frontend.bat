@echo off
setlocal

echo ==========================================
echo [INFO] 重新编译前端
echo ==========================================

cd web

echo [INFO] 正在编译前端...
call npm run build

if %ERRORLEVEL% neq 0 (
    echo [ERROR] 前端编译失败
    pause
    exit /b 1
)

echo.
echo ==========================================
echo [SUCCESS] 前端编译完成！
echo ==========================================
echo.
echo 请按 Ctrl+Shift+R 刷新浏览器以清除缓存
echo.
pause

endlocal
