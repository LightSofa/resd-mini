[CmdletBinding()]
param(
    [string]$ProjectRoot = $PSScriptRoot,
    [string]$OutputDir = "dist",
    [switch]$InstallDeps,
    [switch]$BuildArm64,
    [switch]$BuildNsis
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Require-Command {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Missing required command: $Name"
    }
}

function Run-External {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $true)][string[]]$ArgumentList,
        [string]$WorkDir
    )

    if ([string]::IsNullOrWhiteSpace($WorkDir)) {
        & $FilePath @ArgumentList
    } else {
        Push-Location -LiteralPath $WorkDir
        try {
            & $FilePath @ArgumentList
        } finally {
            Pop-Location
        }
    }

    if ($LASTEXITCODE -ne 0) {
        throw "Command failed ($LASTEXITCODE): $FilePath $($ArgumentList -join ' ')"
    }
}

function Get-VersionFromAppGo {
    param([Parameter(Mandatory = $true)][string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        return "dev"
    }

    $content = Get-Content -LiteralPath $Path -Raw
    $m = [regex]::Match($content, '\d+\.\d+\.\d+')
    if ($m.Success) {
        return $m.Value
    }
    return "dev"
}

Require-Command -Name go
Require-Command -Name npm

$ProjectRoot = (Resolve-Path -LiteralPath $ProjectRoot).Path
$webDir = Join-Path $ProjectRoot "web"
$appGo = Join-Path $ProjectRoot "core\app.go"
$outputPath = Join-Path $ProjectRoot $OutputDir
$buildDir = Join-Path $ProjectRoot "build"
$buildBinDir = Join-Path $buildDir "bin"
$nsiX64 = Join-Path $buildDir "resd-mini-x64.nsi"
$nsiArm = Join-Path $buildDir "resd-mini-arm.nsi"

if (-not (Test-Path -LiteralPath $webDir)) {
    throw "Web directory not found: $webDir"
}
if ($BuildNsis) {
    if (-not (Get-Command makensis -ErrorAction SilentlyContinue)) {
        throw "Missing required command for NSIS packaging: makensis"
    }
    if (-not (Test-Path -LiteralPath $nsiX64)) {
        throw "NSIS script not found: $nsiX64"
    }
    if ($BuildArm64 -and -not (Test-Path -LiteralPath $nsiArm)) {
        throw "NSIS script not found: $nsiArm"
    }
}

Write-Host "==> Project root: $ProjectRoot"
Write-Host "==> Step 1/4: front-end dependencies"

$nodeModules = Join-Path $webDir "node_modules"
if ($InstallDeps -or -not (Test-Path -LiteralPath $nodeModules)) {
    Run-External -FilePath npm -ArgumentList @("install") -WorkDir $webDir
} else {
    Write-Host "node_modules exists, skip npm install (use -InstallDeps to force)"
}

Write-Host "==> Step 2/4: build web/dist"
Run-External -FilePath npm -ArgumentList @("run", "build-only") -WorkDir $webDir

Write-Host "==> Step 3/4: go build sanity check (all packages)"
Run-External -FilePath go -ArgumentList @("build", "./...") -WorkDir $ProjectRoot

Write-Host "==> Step 4/4: build Windows binaries"
New-Item -ItemType Directory -Force -Path $outputPath | Out-Null
if ($BuildNsis) {
    New-Item -ItemType Directory -Force -Path $buildBinDir | Out-Null
}

$version = Get-VersionFromAppGo -Path $appGo
$oldGoos = $env:GOOS
$oldGoarch = $env:GOARCH
$oldCgo = $env:CGO_ENABLED

try {
    $env:CGO_ENABLED = "0"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"

    $amd64Out = Join-Path $outputPath "resd-mini-x64.exe.exe"
    Run-External -FilePath go -ArgumentList @("build", "-trimpath", "-ldflags", "-H=windowsgui", "-o", $amd64Out, ".") -WorkDir $ProjectRoot
    Write-Host "Built: $amd64Out"
    if ($BuildNsis) {
        Copy-Item -Force -LiteralPath $amd64Out -Destination (Join-Path $buildBinDir "resd-mini-x64.exe")
    }

    if ($BuildArm64) {
        $env:GOARCH = "arm64"
        $arm64Out = Join-Path $outputPath "resd-mini-arm64.exe"
        Run-External -FilePath go -ArgumentList @("build", "-trimpath", "-ldflags", "-H=windowsgui", "-o", $arm64Out, ".") -WorkDir $ProjectRoot
        Write-Host "Built: $arm64Out"
        if ($BuildNsis) {
            Copy-Item -Force -LiteralPath $arm64Out -Destination (Join-Path $buildBinDir "resd-mini-arm.exe")
        }
    }

    if ($BuildNsis) {
        Run-External -FilePath makensis -ArgumentList @($nsiX64) -WorkDir $buildDir
        $x64Nsis = Join-Path $buildBinDir "resd-mini-x64-nsis.exe"
        if (-not (Test-Path -LiteralPath $x64Nsis)) {
            throw "NSIS package not generated: $x64Nsis"
        }
        $x64NsisOut = Join-Path $outputPath "resd-mini_${version}_windows_amd64_nsis.exe"
        Move-Item -Force -LiteralPath $x64Nsis -Destination $x64NsisOut
        Write-Host "Built: $x64NsisOut"

        if ($BuildArm64) {
            Run-External -FilePath makensis -ArgumentList @($nsiArm) -WorkDir $buildDir
            $armNsis = Join-Path $buildBinDir "resd-mini-arm-nsis.exe"
            if (-not (Test-Path -LiteralPath $armNsis)) {
                throw "NSIS package not generated: $armNsis"
            }
            $armNsisOut = Join-Path $outputPath "resd-mini_${version}_windows_arm64_nsis.exe"
            Move-Item -Force -LiteralPath $armNsis -Destination $armNsisOut
            Write-Host "Built: $armNsisOut"
        }
    }
} finally {
    $env:GOOS = $oldGoos
    $env:GOARCH = $oldGoarch
    $env:CGO_ENABLED = $oldCgo
}

Write-Host "==> Done"
