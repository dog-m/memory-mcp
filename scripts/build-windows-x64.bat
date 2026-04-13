@echo off
cd ..

SETLOCAL
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
set BUILD_DIR=build
set OUTPUT_NAME=%BUILD_DIR%\memory-mcp.exe

if not exist %BUILD_DIR% mkdir %BUILD_DIR%

echo Building memory-mcp for Windows x64...
go build -ldflags "-s -w" -o %OUTPUT_NAME% ./src

if %ERRORLEVEL% EQU 0 (
    echo Build successful: %OUTPUT_NAME%
    exit /b 0
) else (
    echo Build failed!
    pause
    exit /b 1
)
