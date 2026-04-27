param(
    [string]$url,
    [string]$filename,
    [string]$defaultDir,
    [string]$TsTempDir
)

function Get-FirstNonEmptyLine {
    param([string]$Text)
    if ([string]::IsNullOrWhiteSpace($Text)) { return "" }
    foreach ($line in ($Text -split "\r?\n")) {
        $t = $line.Trim()
        if ($t) { return $t }
    }
    return ""
}

function Quote-Arg {
    param([string]$Arg)
    if ($null -eq $Arg) { return '""' }
    if ($Arg -notmatch '[\s"]') { return $Arg }

    # Windows CreateProcess quoting rules (simplified)
    $sb = New-Object System.Text.StringBuilder
    [void]$sb.Append('"')
    $bs = 0
    foreach ($ch in $Arg.ToCharArray()) {
        if ($ch -eq '\') {
            $bs++
            continue
        }
        if ($ch -eq '"') {
            if ($bs -gt 0) { [void]$sb.Append('\', $bs * 2) }
            $bs = 0
            [void]$sb.Append('\\"')
            continue
        }
        if ($bs -gt 0) { [void]$sb.Append('\', $bs) }
        $bs = 0
        [void]$sb.Append($ch)
    }
    if ($bs -gt 0) { [void]$sb.Append('\', $bs * 2) }
    [void]$sb.Append('"')
    return $sb.ToString()
}

function Invoke-ExternalUtf8 {
    param(
        [string]$FileName,
        [string[]]$Args
    )

    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $FileName
    $psi.UseShellExecute = $false
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true

    # Force UTF-8 decoding if the properties exist (Windows PowerShell/.NET Framework compatibility).
    try {
        $utf8 = New-Object System.Text.UTF8Encoding($false)
        $psi.StandardOutputEncoding = $utf8
        $psi.StandardErrorEncoding = $utf8
    } catch {}

    $argLine = ""
    if ($Args -and $Args.Count -gt 0) {
        $argLine = ($Args | ForEach-Object { Quote-Arg -Arg $_ }) -join " "
    }
    $psi.Arguments = $argLine

    $p = New-Object System.Diagnostics.Process
    $p.StartInfo = $psi

    [void]$p.Start()
    $stdout = $p.StandardOutput.ReadToEnd()
    $stderr = $p.StandardError.ReadToEnd()
    $p.WaitForExit()

    return @{
        ExitCode = $p.ExitCode
        Stdout   = $stdout
        Stderr   = $stderr
    }
}

function Escape-Xml {
    param([string]$Text)
    if ($null -eq $Text) { return "" }
    return [System.Security.SecurityElement]::Escape($Text)
}

function To-FileUri {
    param([string]$Path)
    if ([string]::IsNullOrWhiteSpace($Path)) { return "" }
    try {
        $full = [System.IO.Path]::GetFullPath($Path)
        return ([System.Uri]::new($full)).AbsoluteUri
    } catch {
        return ""
    }
}

function New-LogFile {
    param([string]$BaseDir)
    $stamp = (Get-Date).ToString("yyyyMMdd-HHmmss")

    $base = ($BaseDir | ForEach-Object { $_.Trim() })
    if (-not [string]::IsNullOrWhiteSpace($base)) {
        $base = $base -replace '/', '\'
        if ($base.StartsWith("\\\\")) {
            $base = "\\\\" + (($base.Substring(2)) -replace '\\\\+', '\')
        } else {
            $base = $base -replace '\\\\+', '\'
        }
    }

    $candidates = @()
    if (-not [string]::IsNullOrWhiteSpace($base)) {
        $candidates += (Join-Path $base "resd-mini-logs")
    }
    $candidates += (Join-Path $env:TEMP "resd-mini-logs")

    foreach ($dir in $candidates) {
        try {
            New-Item -ItemType Directory -Force -Path $dir -ErrorAction Stop | Out-Null
            $probe = Join-Path $dir ".resd-mini-write-test"
            Set-Content -LiteralPath $probe -Value "ok" -Encoding Ascii -ErrorAction Stop
            Remove-Item -LiteralPath $probe -Force -ErrorAction Stop
            return (Join-Path $dir ("resd-mini-action-{0}.log" -f $stamp))
        } catch {
            # try next candidate
        }
    }

    return (Join-Path $env:TEMP ("resd-mini-action-{0}.log" -f $stamp))
}

function Write-Log {
    param([string]$Path,[string]$Text)
    try {
        $ts = (Get-Date).ToString("yyyy-MM-dd HH:mm:ss")
        $line = ("[{0}] {1}" -f $ts, $Text)
        $enc = New-Object System.Text.UTF8Encoding($true)
        [System.IO.File]::AppendAllText($Path, $line + [Environment]::NewLine, $enc)
    } catch {}
}

function Cleanup-OldLogs {
    param([string]$LogDir,[int]$KeepDays)
    if ([string]::IsNullOrWhiteSpace($LogDir)) { return }
    if ($KeepDays -le 0) { return }
    try {
        $threshold = (Get-Date).AddDays(-1 * $KeepDays)
        Get-ChildItem -LiteralPath $LogDir -Filter "resd-mini-action-*.log" -ErrorAction SilentlyContinue |
        Where-Object { $_.LastWriteTime -lt $threshold } |
        ForEach-Object { Remove-Item -LiteralPath $_.FullName -Force -ErrorAction SilentlyContinue }

        Get-ChildItem -LiteralPath $LogDir -Filter "resd-mini-open-*.lnk" -ErrorAction SilentlyContinue |
        Where-Object { $_.LastWriteTime -lt $threshold } |
        ForEach-Object { Remove-Item -LiteralPath $_.FullName -Force -ErrorAction SilentlyContinue }
    } catch {}
}

function New-ExplorerSelectShortcut {
    param([string]$TargetFile,[string]$OutDir)
    if ([string]::IsNullOrWhiteSpace($TargetFile)) { return "" }
    if ([string]::IsNullOrWhiteSpace($OutDir)) { return "" }
    try {
        $full = [System.IO.Path]::GetFullPath($TargetFile)
        $stamp = (Get-Date).ToString("yyyyMMdd-HHmmss")
        $lnkPath = Join-Path $OutDir ("resd-mini-open-{0}.lnk" -f $stamp)

        $shell = New-Object -ComObject WScript.Shell
        $sc = $shell.CreateShortcut($lnkPath)
        $sc.TargetPath = (Join-Path $env:WINDIR "explorer.exe")
        $sc.Arguments = "/select,`"$full`""
        $sc.WorkingDirectory = (Split-Path -Parent $full)
        $sc.IconLocation = (Join-Path $env:WINDIR "explorer.exe") + ",0"
        $sc.Save()

        if (Test-Path -LiteralPath $lnkPath) { return $lnkPath }
        return ""
    } catch {
        return ""
    }
}

function New-OpenLogShortcut {
    param([string]$LogFile,[string]$OutDir)
    if ([string]::IsNullOrWhiteSpace($LogFile)) { return "" }
    if ([string]::IsNullOrWhiteSpace($OutDir)) { return "" }
    try {
        $full = [System.IO.Path]::GetFullPath($LogFile)
        $stamp = (Get-Date).ToString("yyyyMMdd-HHmmss")
        $lnkPath = Join-Path $OutDir ("resd-mini-open-{0}.lnk" -f $stamp)

        $shell = New-Object -ComObject WScript.Shell
        $sc = $shell.CreateShortcut($lnkPath)
        $sc.TargetPath = (Join-Path $env:WINDIR "notepad.exe")
        $sc.Arguments = "`"$full`""
        $sc.WorkingDirectory = (Split-Path -Parent $full)
        $sc.IconLocation = (Join-Path $env:WINDIR "notepad.exe") + ",0"
        $sc.Save()

        if (Test-Path -LiteralPath $lnkPath) { return $lnkPath }
        return ""
    } catch {
        return ""
    }
}

function Show-Notification {
    param([string]$Title,[string]$Message,[string]$LaunchUri)

    try {
        $notifyDir = $script:logDir
        if ([string]::IsNullOrWhiteSpace($notifyDir)) {
            $notifyDir = Join-Path $env:TEMP "resd-mini-logs"
        }
        New-Item -ItemType Directory -Force -Path $notifyDir -ErrorAction SilentlyContinue | Out-Null

        $helper = Join-Path $notifyDir "resd-mini-notify-helper.ps1"
        $helperContent = @'
param(
    [string]$Title,
    [string]$Message,
    [string]$Target
)

$ErrorActionPreference = "Stop"
try {
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$script:notify = New-Object System.Windows.Forms.NotifyIcon
$script:notify.Icon = [System.Drawing.SystemIcons]::Information
$script:notify.Visible = $true
$script:notify.Text = if ($Title.Length -gt 63) { $Title.Substring(0, 63) } else { $Title }
$script:notify.BalloonTipTitle = $Title
$script:notify.BalloonTipText = $Message
$script:notify.BalloonTipIcon = [System.Windows.Forms.ToolTipIcon]::Info

$script:targetPath = $Target
try {
    if (-not [string]::IsNullOrWhiteSpace($Target)) {
        $uri = [System.Uri]::new($Target)
        if ($uri.IsFile) {
            $script:targetPath = $uri.LocalPath
        }
    }
} catch {}

$openTarget = {
    try {
        if (-not [string]::IsNullOrWhiteSpace($script:targetPath)) {
            Start-Process -FilePath $script:targetPath
        }
    } catch {}
    try { $script:notify.Visible = $false; $script:notify.Dispose() } catch {}
    [System.Windows.Forms.Application]::Exit()
}

$script:notify.add_BalloonTipClicked($openTarget)
$script:notify.add_Click($openTarget)

$timer = New-Object System.Windows.Forms.Timer
if ([string]::IsNullOrWhiteSpace($script:targetPath)) {
    $timer.Interval = 10000
} else {
    $timer.Interval = 300000
}
$timer.add_Tick({
    try { $script:notify.Visible = $false; $script:notify.Dispose() } catch {}
    $timer.Stop()
    [System.Windows.Forms.Application]::Exit()
})
$timer.Start()

$script:notify.ShowBalloonTip(300000)
[System.Windows.Forms.Application]::Run()
} catch {
    try {
        $errPath = Join-Path $PSScriptRoot "resd-mini-notify-error.log"
        $msg = "{0}`r`n{1}" -f (Get-Date).ToString("yyyy-MM-dd HH:mm:ss"), ($_ | Out-String)
        [System.IO.File]::AppendAllText($errPath, $msg + [Environment]::NewLine, [System.Text.Encoding]::UTF8)
    } catch {}
}
'@
        $enc = New-Object System.Text.UTF8Encoding($true)
        [System.IO.File]::WriteAllText($helper, $helperContent, $enc)

        $helperEsc = $helper -replace "'", "''"
        $titleEsc = $Title -replace "'", "''"
        $messageEsc = $Message -replace "'", "''"
        $targetEsc = $LaunchUri -replace "'", "''"
        $notifyCommand = "& '$helperEsc' -Title '$titleEsc' -Message '$messageEsc' -Target '$targetEsc'"
        $encoded = [Convert]::ToBase64String([System.Text.Encoding]::Unicode.GetBytes($notifyCommand))
        $notifyArgLine = "-STA -NoProfile -ExecutionPolicy Bypass -EncodedCommand $encoded"
        Start-Process -WindowStyle Hidden -FilePath "powershell.exe" -ArgumentList $notifyArgLine | Out-Null
    } catch {
        if (-not [string]::IsNullOrWhiteSpace($script:logFile)) {
            Write-Log -Path $script:logFile -Text ("notifyError={0}" -f ($_ | Out-String))
        }
        # ignore notification failures
    }
}

$defaultDir = ($defaultDir | ForEach-Object { $_.Trim() })
$defaultDir = $defaultDir -replace '/', '\'
if ($defaultDir.StartsWith("\\\\")) {
    $defaultDir = "\\\\" + (($defaultDir.Substring(2)) -replace '\\\\+', '\')
} else {
    $defaultDir = $defaultDir -replace '\\\\+', '\'
}

$TsTempDir = ($TsTempDir | ForEach-Object { $_.Trim() })
if (-not [string]::IsNullOrWhiteSpace($TsTempDir)) {
    $TsTempDir = $TsTempDir -replace '/', '\'
    if ($TsTempDir.StartsWith("\\\\")) {
        $TsTempDir = "\\\\" + (($TsTempDir.Substring(2)) -replace '\\\\+', '\')
    } else {
        $TsTempDir = $TsTempDir -replace '\\\\+', '\'
    }
}

$logBaseDir = $defaultDir
if (-not [string]::IsNullOrWhiteSpace($TsTempDir)) {
    $logBaseDir = $TsTempDir
}

$logFile = New-LogFile -BaseDir $logBaseDir
$logDir = Split-Path -Parent $logFile
Cleanup-OldLogs -LogDir $logDir -KeepDays 3

Write-Log -Path $logFile -Text ("url={0}" -f $url)
Write-Log -Path $logFile -Text ("filename={0}" -f $filename)
Write-Log -Path $logFile -Text ("defaultDir={0}" -f $defaultDir)
Write-Log -Path $logFile -Text ("TsTempDir={0}" -f $TsTempDir)
Write-Log -Path $logFile -Text ("logBaseDir={0}" -f $logBaseDir)

Show-Notification -Title "m3u8 任务开始" -Message ("正在后台下载: {0}" -f $filename) -LaunchUri ""

$cmd = Get-Command m3u8d-cli -ErrorAction SilentlyContinue
if (-not $cmd) {
    $msg = "m3u8d-cli not found in PATH"
    Write-Log -Path $logFile -Text $msg
    Show-Notification -Title "m3u8 下载异常" -Message ("未找到 m3u8d-cli。日志: {0}" -f $logFile)
    [Console]::Error.WriteLine($msg)
    [Console]::Error.WriteLine(("log: {0}" -f $logFile))
    exit 127
}

# Prefer invoking via PowerShell (&) so aliases/functions/shims work correctly.
try {
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    $OutputEncoding = $utf8NoBom
    [Console]::OutputEncoding = $utf8NoBom
    [Console]::InputEncoding = $utf8NoBom
} catch {}

$m3u8Args = @(
    "download",
    "--M3u8Url", $url,
    "--FileName", $filename,
    "--SaveDir", $defaultDir
)
if (-not [string]::IsNullOrWhiteSpace($TsTempDir)) {
    $m3u8Args += @("--TsTempDir", $TsTempDir)
}

$output = (& m3u8d-cli @m3u8Args 2>&1 | Out-String)
$exitCode = $LASTEXITCODE

Write-Log -Path $logFile -Text ("exitCode={0}" -f $exitCode)
Write-Log -Path $logFile -Text "----- m3u8d-cli output begin -----"
Write-Log -Path $logFile -Text ($output.TrimEnd())
Write-Log -Path $logFile -Text "----- m3u8d-cli output end -----"

# m3u8d-cli: "下载成功" is the only reliable success indicator (exit code can be 0 on failure).
if ($output -match "下载成功") {
    $targetFile = Join-Path $defaultDir $filename
    $launch = ""
    if (Test-Path -LiteralPath $targetFile) {
        $lnk = New-ExplorerSelectShortcut -TargetFile $targetFile -OutDir $logDir
        if (-not [string]::IsNullOrWhiteSpace($lnk)) {
            $launch = To-FileUri -Path $lnk
        }
    }
    if ([string]::IsNullOrWhiteSpace($launch)) {
        $launch = To-FileUri -Path $defaultDir
        Show-Notification -Title "m3u8 下载成功" -Message ("文件已成功保存: {0} (点击打开目录)" -f $filename) -LaunchUri $launch
    } else {
        Show-Notification -Title "m3u8 下载成功" -Message ("文件已成功保存: {0} (点击打开并选中文件)" -f $filename) -LaunchUri $launch
    }

    $next = Join-Path $PSScriptRoot "some_script.bat"
    if (Test-Path -LiteralPath $next) {
        & $next
    }
    exit 0
}

$first = Get-FirstNonEmptyLine -Text $output
if ([string]::IsNullOrWhiteSpace($first)) { $first = "未检测到成功标志，请检查。" }

$lnkLog = New-OpenLogShortcut -LogFile $logFile -OutDir $logDir
$openLog = ""
if (-not [string]::IsNullOrWhiteSpace($lnkLog)) {
    $openLog = To-FileUri -Path $lnkLog
} else {
    $openLog = To-FileUri -Path $logFile
}
Show-Notification -Title "m3u8 下载异常" -Message ("{0} (点击打开日志)" -f $first) -LaunchUri $openLog

# resd-mini uses stderr's first line as the failure summary; keep it clean.
[Console]::Error.WriteLine(("m3u8d-cli failed (exit={0}). log: {1}" -f $exitCode, $logFile))
if (-not [string]::IsNullOrWhiteSpace($output)) {
    [Console]::Error.WriteLine($output.TrimEnd())
}

if ($exitCode -ne 0) { exit $exitCode }
exit 1
