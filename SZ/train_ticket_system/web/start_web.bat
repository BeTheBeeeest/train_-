@echo off
echo ========================================
echo   Train Ticket System - Web Frontend
echo ========================================
echo.
echo Starting web page...
echo.
echo Tips:
echo 1. Make sure the server is running
echo 2. Using simulated data for demo
echo 3. Page will open in default browser
echo.
echo ========================================
echo.

cd /d "%~dp0"
start "" "%~dp0index.html"

echo Web page opened in browser!
echo.
pause
