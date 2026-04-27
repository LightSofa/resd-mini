@echo off
:: 一行代码隐藏调用 PowerShell 脚本
 powershell -WindowStyle Hidden -ExecutionPolicy Bypass -File "download_task.ps1" -url "%url%" -filename "%filename%" -defaultDir "%defaultDir%"