[CmdletBinding()]
param(
    [string]$ProjectRoot = (Get-Location).Path,
    [string]$KeyPath = "C:\Users\zf\.ssh\id_ed25519",
    [string]$Router = "root@iStoreOS",
    [string]$HostListen = "0.0.0.0",
    [int]$Port = 8899,
    [string]$SaveDir = "/root/downloads/resd-mini"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Require-Command {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Missing required command: $Name"
    }
}

function Assert-PathExists {
    param([Parameter(Mandatory = $true)][string]$Path)
    try {
        if (-not (Test-Path -LiteralPath $Path)) {
            throw "Required path not found: $Path"
        }
    } catch [System.UnauthorizedAccessException] {
        # UNC keys can be ACL-protected; let ssh/scp validate at runtime.
        if ($Path.StartsWith("\\")) {
            Write-Warning "Path access denied during precheck, continue and defer validation to ssh/scp: $Path"
            return
        }
        throw
    }
}

function Run-External {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $true)][string[]]$ArgumentList
    )

    & $FilePath @ArgumentList
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed ($LASTEXITCODE): $FilePath $($ArgumentList -join ' ')"
    }
}

function Assert-RouterFormat {
    param([Parameter(Mandatory = $true)][string]$Value)

    if ($Value -notmatch '^[a-z_][a-z0-9_-]*@[^\s]+$') {
        throw "Invalid router format: $Value. Expected format: user@host"
    }

    $hostPart = $Value.Split('@', 2)[1]
    if ($hostPart -match '^\d+\.\d+\.\d+\.\d+$') {
        [void][System.Net.IPAddress]::Parse($hostPart)
    }
}

function Assert-SafeRemoteValue {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Value
    )

    if ($Value -match "['`\r\n]") {
        throw "Unsafe value for $Name"
    }
}

Require-Command -Name go
Require-Command -Name ssh
Require-Command -Name scp

Assert-RouterFormat -Value $Router
Assert-PathExists -Path $KeyPath
Assert-SafeRemoteValue -Name "HostListen" -Value $HostListen
Assert-SafeRemoteValue -Name "SaveDir" -Value $SaveDir
if ($Port -lt 1 -or $Port -gt 65535) {
    throw "Port out of range: $Port"
}

$ProjectRoot = (Resolve-Path -LiteralPath $ProjectRoot).Path
Set-Location -LiteralPath $ProjectRoot
$routerHost = $Router.Split('@', 2)[1]

$initSrc = Join-Path $ProjectRoot "build\istoreos\resd-mini.init"
$configSrc = Join-Path $ProjectRoot "build\istoreos\resd-mini.config.example"
Assert-PathExists -Path $initSrc
Assert-PathExists -Path $configSrc

$distDir = Join-Path $ProjectRoot "dist"
$binLocal = Join-Path $distDir "resd-mini"
New-Item -ItemType Directory -Force -Path $distDir | Out-Null

Write-Host "==> Detecting router architecture"
$arch = (& ssh -o BatchMode=yes -i $KeyPath $Router "uname -m").Trim()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($arch)) {
    throw "Failed to detect router architecture"
}
Write-Host "Router architecture: $arch"

$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
Remove-Item Env:GOARM -ErrorAction SilentlyContinue

switch ($arch) {
    "aarch64" { $env:GOARCH = "arm64" }
    "armv7l"  { $env:GOARCH = "arm"; $env:GOARM = "7" }
    "armv6l"  { $env:GOARCH = "arm"; $env:GOARM = "6" }
    "x86_64"  { $env:GOARCH = "amd64" }
    default    { throw "Unsupported architecture: $arch" }
}

Write-Host "==> Building binary (GOOS=$env:GOOS GOARCH=$env:GOARCH GOARM=$($env:GOARM))"
Run-External -FilePath "go" -ArgumentList @("build", "-trimpath", "-ldflags", "-s -w", "-o", $binLocal, ".")
Assert-PathExists -Path $binLocal

Write-Host "==> Uploading staged files"
Run-External -FilePath "scp" -ArgumentList @("-o", "BatchMode=yes", "-i", $KeyPath, $binLocal, "${Router}:/tmp/resd-mini.bin.new")
Run-External -FilePath "scp" -ArgumentList @("-o", "BatchMode=yes", "-i", $KeyPath, $initSrc, "${Router}:/tmp/resd-mini.init.new")
Run-External -FilePath "scp" -ArgumentList @("-o", "BatchMode=yes", "-i", $KeyPath, $configSrc, "${Router}:/tmp/resd-mini.config.new")

$localHash = (Get-FileHash -LiteralPath $binLocal -Algorithm SHA256).Hash.ToLowerInvariant()
$remoteHash = (& ssh -o BatchMode=yes -i $KeyPath $Router "sha256sum /tmp/resd-mini.bin.new | cut -d ' ' -f1").Trim().ToLowerInvariant()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($remoteHash)) {
    throw "Failed to calculate remote checksum"
}
if ($localHash -ne $remoteHash) {
    throw "Checksum mismatch for uploaded binary. Local=$localHash Remote=$remoteHash"
}

$remoteScript = @"
set -eu
umask 022

mkdir -p /usr/bin /etc/init.d /etc/config

if [ -f /usr/bin/resd-mini ]; then cp -f /usr/bin/resd-mini /usr/bin/resd-mini.bak; fi
if [ -f /etc/init.d/resd-mini ]; then cp -f /etc/init.d/resd-mini /etc/init.d/resd-mini.bak; fi
if [ -f /etc/config/resd-mini ]; then cp -f /etc/config/resd-mini /etc/config/resd-mini.bak; fi

cp -f /tmp/resd-mini.bin.new /usr/bin/resd-mini
cp -f /tmp/resd-mini.init.new /etc/init.d/resd-mini
cp -f /tmp/resd-mini.config.new /etc/config/resd-mini
sed -i 's/\r$//' /etc/init.d/resd-mini /etc/config/resd-mini
chmod 0755 /usr/bin/resd-mini
chmod 0755 /etc/init.d/resd-mini
chmod 0644 /etc/config/resd-mini

mkdir -p '$SaveDir'
chmod +x /etc/init.d/resd-mini

uci set resd-mini.main.host='$HostListen'
uci set resd-mini.main.port='$Port'
uci set resd-mini.main.save_dir='$SaveDir'
uci commit resd-mini

/etc/init.d/resd-mini enable
/etc/init.d/resd-mini stop || true
/etc/init.d/resd-mini start

i=0
until wget -qO- 'http://127.0.0.1:$Port/api/v1/health' >/dev/null 2>&1; do
    i=`$((i + 1))
    if [ "`$i" -ge 20 ]; then
        echo "health check timeout after start" >&2
        exit 1
    fi
    sleep 1
done
"@

Write-Host "==> Installing and restarting service"
Run-External -FilePath "ssh" -ArgumentList @("-o", "BatchMode=yes", "-i", $KeyPath, $Router, $remoteScript)

Write-Host "==> Local health check"
$health = Invoke-WebRequest -UseBasicParsing -Uri "http://$routerHost`:$Port/api/v1/health" -TimeoutSec 8
if ($health.StatusCode -ne 200) {
    throw "Gateway health check failed: HTTP $($health.StatusCode)"
}

Write-Host "Deployment finished successfully"
Write-Host "Health payload: $($health.Content)"
Write-Host ("Tail logs: ssh -i `"{0}`" {1} 'logread -f | grep resd-mini'" -f $KeyPath, $Router)
