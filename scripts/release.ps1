<#
Usage:
  scripts/release.ps1 [--skip-tests] [--no-tag] <X.Y.Z>

Examples:
  .\scripts\release.ps1 --skip-tests 0.2.2
  .\scripts\release.ps1 --no-tag 0.2.2
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0, ValueFromRemainingArguments = $true)]
    [string[]] $ScriptArgs
)

$ErrorActionPreference = "Stop"

$skipTests = $false
$noTag = $false
$newVersion = $null

function Show-Step {
    param([string]$Text)
    Write-Host ""
    Write-Host "-> $Text"
}

function Show-Info {
    param([string]$Text)
    Write-Host "  -> $Text"
}

function Die {
    param([string]$Message)
    throw "ERROR: $Message"
}

function Usage {
    Write-Host "usage: scripts/release.ps1 [--skip-tests] [--no-tag] <X.Y.Z>"
}

function Ensure-Command {
    param([string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        Die "required command not found: $Name"
    }
}

function Invoke-Checked {
    param(
        [string]$Command,
        [string[]]$Arguments,
        [string]$WorkingDirectory = $null
    )

    Ensure-Command $Command

    $prev = $null
    if ($null -ne $WorkingDirectory) {
        $prev = Get-Location
        Set-Location $WorkingDirectory
    }

    try {
        & $Command @Arguments
        if ($LASTEXITCODE -ne 0) {
            Die "'$Command $($Arguments -join ' ')' failed with exit code $LASTEXITCODE"
        }
    } finally {
        if ($null -ne $prev) {
            Set-Location $prev
        }
    }
}

function Replace-FirstMatch {
    param(
        [string]$InputText,
        [string]$Pattern,
        [string]$Replacement
    )

    $match = [regex]::Match($InputText, $Pattern)
    if (-not $match.Success) {
        return $null
    }

    return $InputText.Substring(0, $match.Index) +
        $Replacement +
        $InputText.Substring($match.Index + $match.Length)
}

for ($i = 0; $i -lt $ScriptArgs.Length; $i++) {
    switch ($ScriptArgs[$i]) {
        "--skip-tests" { $skipTests = $true; continue }
        "--no-tag" { $noTag = $true; continue }
        "-h" { Usage; exit 0 }
        "--help" { Usage; exit 0 }
        { $_ -like "-*" } { Die "unknown option: $($_)" }
        default {
            if ($null -ne $newVersion) {
                Die "unexpected argument: $($ScriptArgs[$i])"
            }
            $newVersion = $ScriptArgs[$i]
        }
    }
}

if (-not $newVersion) {
    Usage
    Die "version is required"
}
if ($newVersion -notmatch "^[0-9]+\.[0-9]+\.[0-9]+$") {
    Die "version must be X.Y.Z (got '$newVersion')"
}

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $root

$jsPackagePath = Join-Path $root "js/package.json"
$cliMainPath = Join-Path $root "cli/cmd/guten/main.go"

$packageJsonText = Get-Content -Raw $jsPackagePath
$oldMatch = [regex]::Match($packageJsonText, '"version"\s*:\s*"([^"]+)"')
if (-not $oldMatch.Success) { Die "cannot read version from js/package.json" }
$oldVersion = $oldMatch.Groups[1].Value

if ($oldVersion -eq $newVersion) {
    Die "already at $newVersion"
}

Write-Host ""
Write-Host "Release: $oldVersion -> $newVersion"

Show-Step "Pre-flight"

$currentBranch = (git branch --show-current).Trim()
if ($currentBranch -ne "main") { Die "must be on main" }
if (-not [string]::IsNullOrWhiteSpace((git status --porcelain))) { Die "working tree dirty -- commit/stash first" }
git fetch --quiet origin main
if ((git rev-parse HEAD).Trim() -ne (git rev-parse origin/main).Trim()) {
    Die "behind origin/main -- git pull first"
}
Show-Info "on main, clean, up to date"

foreach ($t in @("go/v$newVersion", "cli/v$newVersion", "js/v$newVersion")) {
    if (git tag --list $t) { Die "tag $t already exists locally (bump to a new version)" }
    if (git ls-remote --tags origin $t) { Die "tag $t already exists on origin (bump to a new version)" }
}
Show-Info "tags go/cli/js v$newVersion are free"

Show-Step "Tests"
if (-not $skipTests) {
    Invoke-Checked -Command "go" -Arguments @("test", "./...") -WorkingDirectory (Join-Path $root "go")
    Show-Info "go OK"

    Invoke-Checked -Command "go" -Arguments @("test", "./...") -WorkingDirectory (Join-Path $root "cli")
    Show-Info "cli OK"

    Invoke-Checked -Command "npm" -Arguments @("--prefix", (Join-Path $root "js"), "run", "test", "--silent")
    Show-Info "js OK"
} else {
    Show-Info "skipped (--skip-tests)"
}

Show-Step "Bump versions (-> $newVersion)"
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
$newPackageJsonText = Replace-FirstMatch -InputText $packageJsonText -Pattern '"version"\s*:\s*"[^"]+"' -Replacement ('"version": "' + $newVersion + '"')
if ($null -eq $newPackageJsonText) {
    Die "cannot update version in js/package.json"
}
[System.IO.File]::WriteAllText($jsPackagePath, $newPackageJsonText, $utf8NoBom)

$cliText = Get-Content -Raw $cliMainPath
$newCliText = Replace-FirstMatch -InputText $cliText -Pattern 'var version = "[^"]+"' -Replacement ('var version = "' + $newVersion + '"')
if ($null -eq $newCliText) {
    Die "cannot update version in cli/cmd/guten/main.go"
}
[System.IO.File]::WriteAllText($cliMainPath, $newCliText, $utf8NoBom)

Show-Info "js/package.json + cli/cmd/guten/main.go"

Show-Step "Commit + push version bump"
git add js/package.json cli/cmd/guten/main.go
git commit -m "chore(release): guten v$newVersion" -m "Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
git push origin main

Show-Step "Tag + push go/v$newVersion (Go module)"
git tag "go/v$newVersion"
git push origin "go/v$newVersion"

Show-Step "Point cli at guten/go@v$newVersion"
$prevGoFlags = $env:GOFLAGS
$prevGoProxy = $env:GOPROXY
$prevGoPrivate = $env:GOPRIVATE
try {
    $env:GOFLAGS = "-mod=mod"
    $env:GOPROXY = "direct"
    $env:GOPRIVATE = "github.com/kitsyai/*"

    Invoke-Checked -Command "go" -Arguments @("get", "github.com/kitsyai/guten/go@v$($newVersion)") -WorkingDirectory (Join-Path $root "cli")
    Invoke-Checked -Command "go" -Arguments @("mod", "tidy") -WorkingDirectory (Join-Path $root "cli")
} finally {
    $env:GOFLAGS = $prevGoFlags
    $env:GOPROXY = $prevGoProxy
    $env:GOPRIVATE = $prevGoPrivate
}

if (-not [string]::IsNullOrWhiteSpace((git status --porcelain cli/go.mod cli/go.sum))) {
    git add cli/go.mod cli/go.sum
    git commit -m "chore(cli): guten/go@v$newVersion" -m "Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
    git push origin main
    Show-Info "cli deps updated"
}

if ($noTag) {
    Write-Host ""
    Write-Host "Bumped + pushed main + go/v$newVersion. Finish with:"
    Write-Host "  git tag cli/v$newVersion js/v$newVersion && git push origin cli/v$newVersion js/v$newVersion"
    exit 0
}

Show-Step "Tag + push cli/v$newVersion and js/v$newVersion (triggers CI)"
git tag "cli/v$newVersion"
git tag "js/v$newVersion"
git push origin "cli/v$newVersion" "js/v$newVersion"

Write-Host ""
Write-Host "Done. CI publishes @kitsy/guten to npm (js/v$newVersion) and cli binaries (cli/v$newVersion)."
