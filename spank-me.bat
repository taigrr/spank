@echo off
setlocal
title SPANK - Windows Launcher - MADE BY SOUMIC :)
echo ==========================================
echo    SPANK - Windows Interactive Launcher For Widows Users made by - (Soumic) Github:- /Soumic28.
echo ==========================================
echo.
echo Please choose your Audio Pack:
echo.
echo  1) Pain mode (Classic "Ow!")
echo  2) Sexy mode (Escalating Female Voice)
echo  3) Halo mode (Death sounds)
echo.
set /p choice="Enter choice (1, 2, or 3): "

set FLAGS=--mouse
if "%choice%"=="1" set MODE=Pain
if "%choice%"=="2" set FLAGS=%FLAGS% --sexy & set MODE=Sexy
if "%choice%"=="3" set FLAGS=%FLAGS% --halo & set MODE=Halo

if "%MODE%"=="" (
    color 0C
    echo Error: Invalid choice "%choice%".
    echo Please run the script again and pick 1, 2, or 3.
    pause
    exit /b
)

cls
echo ==========================================
echo    SPANK is now RUNNING
echo ==========================================
echo.
echo Selected Mode: %MODE%
echo Trigger: Touchpad Tap or Mouse Click
echo.
echo [KEEP THIS WINDOW OPEN TO STAY ACTIVE]
echo.

if exist "spank.exe" (
    .\spank.exe %FLAGS%
) else (
    color 0C
    echo [ERROR] spank.exe not found in this folder.
    echo Please make sure spank.exe and spank-me.bat are in the same directory.
    pause
)
