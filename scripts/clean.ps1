$ErrorActionPreference = 'Continue'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# Remove build/test artifacts, never the official McpNet.exe.
Get-ChildItem $root -File -Filter '*.log' | Remove-Item -Force
Get-ChildItem $root -File | Where-Object { $_.Name -like 'McpNet_test*.exe' -or $_.Name -like 'McpNet_icontest*.exe' -or $_.Name -like 'McpNet_*check.exe' } | Remove-Item -Force
Get-ChildItem "$root\build" -Recurse -File -Filter '*.exe' | Remove-Item -Force
if (Test-Path "$root\frontend\dist.rebuilt") { Remove-Item "$root\frontend\dist.rebuilt" -Recurse -Force }
if (Test-Path "$root\frontend\node_modules") { Remove-Item "$root\frontend\node_modules" -Recurse -Force }

Write-Host 'Removed build/test artifacts. Kept McpNet.exe.'
