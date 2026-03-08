@echo off
REM run.bat - Build and demo hotreload on Windows (no make required)

echo [hotreload] Building...
if not exist bin mkdir bin
go build -o bin\hotreload.exe .
if %errorlevel% neq 0 (
    echo [hotreload] Build FAILED
    exit /b 1
)
echo [hotreload] Build OK

echo.
echo [hotreload] Starting demo...
echo   Edit testserver\main.go and save the file.
echo   Visit http://localhost:8080 to see the running server.
echo   Press Ctrl+C to stop.
echo.

bin\hotreload.exe ^
    --root .\testserver ^
    --build "go build -C .\testserver -o ..\bin\testserver.exe ." ^
    --exec ".\bin\testserver.exe" ^
    --log-level debug
