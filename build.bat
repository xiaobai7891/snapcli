@echo off
echo Building SnapCLI for Windows...

if not exist build mkdir build

echo Downloading dependencies...
go mod download
go mod tidy

echo Building...
go build -ldflags="-s -w -H windowsgui" -o build\snapcli.exe .\cmd\snapcli

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Build successful!
    echo Output: build\snapcli.exe
    echo.
    echo To run: build\snapcli.exe
) else (
    echo.
    echo Build failed!
)

pause
