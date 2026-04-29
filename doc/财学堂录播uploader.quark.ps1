param(
    [string]$SOURCE_FILE,
    [string]$DEST = "openlist:/quark/财学堂录播",
    [string]$LOG_FILE
)

function Write-UploaderLog {
    param([string]$Text)

    $ts = (Get-Date).ToString("yyyy-MM-dd HH:mm:ss")
    $line = ("[{0}] {1}" -f $ts, $Text)
    Write-Output $line

    if (-not $script:WriteLogFile) { return }
    try {
        $enc = New-Object System.Text.UTF8Encoding($true)
        [System.IO.File]::AppendAllText($script:ResolvedLogFile, $line + [Environment]::NewLine, $enc)
    } catch {}
}

function Cleanup-OldUploaderLogs {
    param([string]$LogDir,[int]$KeepDays)
    if ([string]::IsNullOrWhiteSpace($LogDir)) { return }
    if ($KeepDays -le 0) { return }
    try {
        $threshold = (Get-Date).AddDays(-1 * $KeepDays)
        Get-ChildItem -LiteralPath $LogDir -Filter "resd-mini-uploader-*.log" -ErrorAction SilentlyContinue |
        Where-Object { $_.LastWriteTime -lt $threshold } |
        ForEach-Object { Remove-Item -LiteralPath $_.FullName -Force -ErrorAction SilentlyContinue }
    } catch {}
}

try {
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    $OutputEncoding = $utf8NoBom
    [Console]::OutputEncoding = $utf8NoBom
    [Console]::InputEncoding = $utf8NoBom
} catch {}

if ([string]::IsNullOrWhiteSpace($SOURCE_FILE)) {
    [Console]::Error.WriteLine("SOURCE_FILE is required")
    exit 2
}

if ([string]::IsNullOrWhiteSpace($DEST)) {
    $DEST = "openlist:/quark/财学堂录播"
}

$script:ResolvedLogFile = $LOG_FILE
$script:WriteLogFile = -not [string]::IsNullOrWhiteSpace($script:ResolvedLogFile)

if ($script:WriteLogFile) {
    $uploaderLogDir = Split-Path -Parent $script:ResolvedLogFile
    Cleanup-OldUploaderLogs -LogDir $uploaderLogDir -KeepDays 1
}

Write-UploaderLog ("source={0}" -f $SOURCE_FILE)
Write-UploaderLog ("dest={0}" -f $DEST)

if (-not (Test-Path -LiteralPath $SOURCE_FILE)) {
    $msg = "source file not found: {0}" -f $SOURCE_FILE
    Write-UploaderLog $msg
    [Console]::Error.WriteLine($msg)
    exit 3
}

$cmd = Get-Command rclone -ErrorAction SilentlyContinue
if (-not $cmd) {
    $msg = "rclone not found in PATH"
    Write-UploaderLog $msg
    [Console]::Error.WriteLine($msg)
    exit 127
}

# 执行增量同步
$rcloneArgs = @(
    "move",
    $SOURCE_FILE,
    $DEST,
    "--size-only",
    "--timeout", "30m",
    "--contimeout", "30s",
    "--retries", "1",
    "--low-level-retries", "1",
    "-v"
)

Write-UploaderLog "----- rclone output begin -----"
if ($script:WriteLogFile) {
    $output = (& rclone @rcloneArgs 2>&1 | Out-String)
    $exitCode = $LASTEXITCODE
    if (-not [string]::IsNullOrWhiteSpace($output)) {
        Write-UploaderLog ($output.TrimEnd())
    }
} else {
    & rclone @rcloneArgs 2>&1 | ForEach-Object {
        $line = $_.ToString()
        if (-not [string]::IsNullOrWhiteSpace($line)) {
            Write-UploaderLog $line
        }
    }
    $exitCode = $LASTEXITCODE
}
Write-UploaderLog "----- rclone output end -----"
Write-UploaderLog ("exitCode={0}" -f $exitCode)

if ($exitCode -ne 0) {
    if ($script:WriteLogFile) {
        [Console]::Error.WriteLine(("rclone move failed (exit={0}). log: {1}" -f $exitCode, $script:ResolvedLogFile))
    } else {
        [Console]::Error.WriteLine(("rclone move failed (exit={0})" -f $exitCode))
    }
    exit $exitCode
}

exit 0
