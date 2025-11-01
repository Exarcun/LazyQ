@echo off
echo Building LazyQ with icon...

REM Create output directory if it doesn't exist
if not exist "output" mkdir output

REM Build for Windows with icon embedded
go build -o output\LazyQ.exe

echo.
echo Build complete! Executable is in the output folder.
echo.
pause
