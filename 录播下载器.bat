@echo off
chcp 65001 >nul
@setlocal

@if /I "%~1"=="--worker" goto worker

@cd /d "F:\Videos\财学堂录播"

rem 用法:
rem 1) 录播下载器.bat "url" "filename"
rem 2) 录播下载器.bat --url "url" --filename "filename"
rem 3) 录播下载器.bat -u "url" -f "filename"
@set "url="
@set "filename="
@set "pos1="
@set "pos2="

:parse_args
@if "%~1"=="" goto after_parse

@if /I "%~1"=="--url" (
    @set "url=%~2"
    @shift
    @shift
    goto parse_args
)

@if /I "%~1"=="-u" (
    @set "url=%~2"
    @shift
    @shift
    goto parse_args
)

@if /I "%~1"=="--filename" (
    @set "filename=%~2"
    @shift
    @shift
    goto parse_args
)

@if /I "%~1"=="-f" (
    @set "filename=%~2"
    @shift
    @shift
    goto parse_args
)

@if "%pos1%"=="" (
    @set "pos1=%~1"
) else if "%pos2%"=="" (
    @set "pos2=%~1"
)
@shift
@goto parse_args

:after_parse
@if "%url%"=="" @set "url=%pos1%"
@if "%filename%"=="" @set "filename=%pos2%"

@if "%url%"=="" set /p "url=请输入 URL: "
@if "%filename%"=="" set /p "filename=请输入 视频名称: "

@if "%url%"=="" (
    echo 未输入 URL，脚本结束。
    pause
    exit /b 1
)

@if "%filename%"=="" (
    echo 未输入 视频名称，脚本结束。
    pause
    exit /b 1
)

start "录播下载器-下载中" cmd /k ""%~f0" --worker "%url%" "%filename%""
echo 已在新窗口启动下载任务。
exit /b 0

:worker
@set "url=%~2"
@set "filename=%~3"
@cd /d "F:\Videos\财学堂录播"

echo 开始下载：%filename%
m3u8d-cli download --M3u8Url "%url%" --FileName "%filename%"
set "ec=%errorlevel%"
echo.
if not "%ec%"=="0" (
    echo 下载失败，退出码: %ec%
    echo 说明: 若出现 403，通常是 m3u8 链接已过期或鉴权参数无效，请重新获取最新链接。
) else (
    echo 下载完成。
)
pause
exit /b %ec%
