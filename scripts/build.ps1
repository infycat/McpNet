$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# Regenerate the Windows resource object (icon ID 3 + requireAdministrator
# manifest) when windres is available. Wails loads the window/taskbar icon from
# resource ID 3, so app.rc must keep "3 ICON" and must not use rsrc's default ID.
$windres = Get-Command windres -ErrorAction SilentlyContinue
if ($windres) {
    & $windres.Source -i "$root\app.rc" -o "$root\app.syso" -O coff
    if ($LASTEXITCODE -ne 0) { throw "windres failed with exit code $LASTEXITCODE" }
} else {
    Write-Warning 'windres not found; using the checked-in app.syso (icon ID 3 + manifest).'
}

# Keep the runtime UI-injection bundles available to the embedded frontend.
$snSrc = Join-Path $root 'frontend\public\assets'
$snDst = Join-Path $root 'frontend\dist\assets'
if (Test-Path $snSrc) {
    if (-not (Test-Path $snDst)) { New-Item -ItemType Directory -Path $snDst -Force | Out-Null }
    Copy-Item (Join-Path $snSrc 'sn-mcp.1.js') $snDst -Force -ErrorAction SilentlyContinue
    Copy-Item (Join-Path $snSrc 'sn-i18n.396.js') $snDst -Force -ErrorAction SilentlyContinue
}

# Build to a temporary name first; only replace the official exe after success.
$tmp = Join-Path $root 'McpNet.build.exe'
if (Test-Path $tmp) { Remove-Item $tmp -Force }
go build -a -buildvcs=false -tags "desktop,production,wv2runtime.embed" `
    -ldflags "-H windowsgui -s -w -linkmode=internal" -o $tmp .
if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $tmp)) {
    if (Test-Path -LiteralPath $tmp) { Remove-Item -LiteralPath $tmp -Force }
    throw "Build failed with exit code $LASTEXITCODE; the official executable was not changed."
}
Move-Item $tmp "$root\McpNet.exe" -Force

Write-Host "Built: $root\McpNet.exe"
