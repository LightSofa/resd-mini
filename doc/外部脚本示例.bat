wt new-tab m3u8d-cli download --M3u8Url "%url%" --FileName "%filename%" --SaveDir "%defaultDir%" | findstr /C:"下载成功" > nul  
if %errorlevel% == 0 (  
    call your_script.bat  
)