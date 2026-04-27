param(
    [string]$url,
    [string]$filename,
    [string]$defaultDir
)

# 定义发送系统原生通知的函数
function Show-Notification {
    param([string]$Title,[string]$Message)
    
    try {
        # 动态加载 Windows 通知相关的原生 API
        [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
        [Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

        $xmlString = @"
    <toast>
        <visual>
            <binding template="ToastText02">
                <text id="1">$Title</text>
                <text id="2">$Message</text>
            </binding>
        </visual>
    </toast>
"@
        $xml = New-Object Windows.Data.Xml.Dom.XmlDocument
        $xml.LoadXml($xmlString)
        $toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
    
        # 借用 PowerShell 的系统标识发送通知
        $appId = "{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe"
        [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($appId).Show($toast)
    } catch {
        # Ignore toast failures (e.g. unsupported host, missing WinRT types)
    }
}

# 1. 发送开始通知
Show-Notification -Title "m3u8 任务开始" -Message "正在后台下载: $filename"

# 2. 执行下载，并将输出转为字符串捕获
$output = m3u8d-cli download --M3u8Url "$url" --FileName "$filename" --SaveDir "$defaultDir" | Out-String

# 3. 判断是否成功并发送结束通知
if ($output -match "下载成功") {
    Show-Notification -Title "m3u8 下载成功" -Message "文件已成功保存: $filename"
    
    # 4. 执行你后续的脚本 (确保 some_script.bat 也在同一目录或提供绝对路径)
    & ".\some_script.bat"
} else {
    Show-Notification -Title "m3u8 下载异常" -Message "未检测到成功标志，请检查。"
}