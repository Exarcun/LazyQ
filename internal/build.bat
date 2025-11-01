@echo off
echo Building LazyQ with icon...

REM Build for Windows with icon embedded
REM Note: Run this from the project root directory
go build -o LazyQ.exe

echo.
echo Build complete! LazyQ.exe created.
echo.
pause
