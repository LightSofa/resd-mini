@echo off
:: 一行代码隐藏调用 PowerShell 脚本
powershell -ExecutionPolicy Bypass -File "E:\project\resd-mini\doc\m3u8d-cli-wrapper.ps1" -url "%url%" -filename "%filename%" -saveDir "%defaultDir%" -tempDir "X:\appTemp\Resd Mini"