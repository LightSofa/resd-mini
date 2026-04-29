$ErrorActionPreference = "Stop"

function Assert-True {
    param(
        [bool]$Condition,
        [string]$Message
    )
    if (-not $Condition) {
        throw $Message
    }
}

function Assert-ScriptParses {
    param([string]$Path)

    $tokens = $null
    $errors = $null
    [System.Management.Automation.Language.Parser]::ParseFile($Path, [ref]$tokens, [ref]$errors) | Out-Null

    Assert-True ($errors.Count -eq 0) ("{0} has parser errors: {1}" -f $Path, (($errors | ForEach-Object { $_.Message }) -join "; "))
}

function Assert-Utf8Bom {
    param([string]$Path)

    $bytes = [System.IO.File]::ReadAllBytes($Path)
    Assert-True ($bytes.Length -ge 3) ("{0} is too short to contain a UTF-8 BOM" -f $Path)
    Assert-True (($bytes[0] -eq 0xEF) -and ($bytes[1] -eq 0xBB) -and ($bytes[2] -eq 0xBF)) ("{0} must be saved as UTF-8 with BOM for Windows PowerShell 5.1" -f $Path)
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$wrapper = Join-Path $PSScriptRoot "m3u8d-cli-wrapper.ps1"
$uploader = (Get-ChildItem -LiteralPath $PSScriptRoot -Filter "*uploader.quark.ps1" | Select-Object -First 1).FullName

Assert-ScriptParses -Path $wrapper
Assert-ScriptParses -Path $uploader
Assert-Utf8Bom -Path $uploader

$wrapperText = Get-Content -LiteralPath $wrapper -Raw
$uploaderText = Get-Content -LiteralPath $uploader -Raw

Assert-True ($uploaderText -match '\[string\]\$SOURCE_FILE') "uploader must accept SOURCE_FILE"
Assert-True ($uploaderText -match '\[string\]\$DEST') "uploader must accept optional DEST"
Assert-True ($uploaderText -match '\[string\]\$LOG_FILE') "uploader must accept optional LOG_FILE"
Assert-True ($uploaderText -match 'alist:/quark/') "uploader must default to the quark destination"
Assert-True ($uploaderText -match 'rclone\s+move') "uploader must move via rclone"
Assert-True ($uploaderText.Contains('$script:WriteLogFile = -not [string]::IsNullOrWhiteSpace($script:ResolvedLogFile)')) "uploader must only write a log file when LOG_FILE is provided"
Assert-True ($uploaderText -match 'if\s+\(\$script:WriteLogFile\)\s+\{[\s\S]*Cleanup-OldUploaderLogs\s+-LogDir\s+\$uploaderLogDir\s+-KeepDays\s+1') "uploader logs must be retained for 1 day when LOG_FILE is provided"
Assert-True ($uploaderText -match 'if\s+\(\$script:WriteLogFile\)\s+\{[\s\S]*rclone\s+@rcloneArgs\s+2>&1\s+\|\s+Out-String') "uploader must buffer rclone output when LOG_FILE is provided"
Assert-True ($uploaderText -match '\}\s+else\s+\{[\s\S]*rclone\s+@rcloneArgs\s+2>&1\s+\|\s+ForEach-Object') "uploader must stream rclone output when LOG_FILE is not provided"

Assert-True ($wrapperText -notmatch 'Join-Path\s+\$PSScriptRoot\s+"\.\\uploader\.quark\.ps1"') "wrapper must not hardcode uploader.quark.ps1"
Assert-True ($wrapperText -match '-nextScript\s+''\$nextScriptEsc''') "detached monitor command must forward nextScript"
Assert-True ($wrapperText -match 'IsNullOrWhiteSpace\(\$nextScript\)') "wrapper must treat nextScript as optional"
Assert-True ($wrapperText -notmatch '\[string\[\]\]\$Args') "external process helper must not use PowerShell automatic variable name Args"
Assert-True ($wrapperText -match 'Invoke-ExternalUtf8\s+-FileName\s+"powershell\.exe"') "wrapper must invoke nextScript in a child PowerShell process"
Assert-True ($wrapperText -match 'Invoke-ExternalUtf8\s+-FileName\s+"powershell\.exe"\s+-ArgumentList\s+@\(') "wrapper must pass nextScript arguments through ArgumentList"
Assert-True ($wrapperText -match '"-File",\s*\$nextScript,\s*"-SOURCE_FILE",\s*\(\$targetFile\s*\+\s*"\.mp4"\),\s*"-LOG_FILE",\s*\$logFile') "wrapper must pass downloaded file and log to nextScript"
Assert-True ($wrapperText -match 'Cleanup-OldLogs\s+-LogDir\s+\$logDir\s+-KeepDays\s+1') "wrapper logs must be retained for 1 day"

$ps5 = Get-Command powershell.exe -ErrorAction SilentlyContinue
if ($ps5) {
    $oldErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $ps5Output = & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $uploader -SOURCE_FILE "__resd_missing_source__.mp4" 2>&1 | Out-String
        $ps5ExitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $oldErrorActionPreference
    }
    Assert-True ($ps5ExitCode -eq 3) ("uploader must parse and reach missing source check under Windows PowerShell 5.1; exit={0}; output={1}" -f $ps5ExitCode, $ps5Output)
}

Write-Host "script contract tests passed"
