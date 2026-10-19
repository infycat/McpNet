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

# Build to a temporary name first; only replace the official exe after success.
$tmp = Join-Path $root 'McpNet.build.exe'
if (Test-Path $tmp) { Remove-Item $tmp -Force }
go build -a -buildvcs=false -tags "desktop,production,wv2runtime.download" `
    -ldflags "-H windowsgui -s -w" -o $tmp .
if (-not (Test-Path $tmp)) {
    throw 'Build did not produce McpNet.build.exe'
}
Move-Item $tmp "$root\McpNet.exe" -Force

Write-Host "Built: $root\McpNet.exe"
