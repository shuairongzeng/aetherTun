param(
    [string]$OutputName = "Aether-portable"
)

$ErrorActionPreference = "Stop"

$projectRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$buildBin = Join-Path $projectRoot "build\bin"
$portableAssets = Join-Path $projectRoot "packaging\portable"
$stagingDir = Join-Path $buildBin $OutputName
$zipPath = Join-Path $buildBin "$OutputName.zip"

$requiredFiles = @(
    @{ Source = (Join-Path $buildBin "Aether.exe"); Target = "Aether.exe" },
    @{ Source = (Join-Path $buildBin "aether-core.exe"); Target = "aether-core.exe" },
    @{ Source = (Join-Path $buildBin "wintun.dll"); Target = "wintun.dll" },
    @{ Source = (Join-Path $portableAssets "config.example.json"); Target = "config.example.json" },
    @{ Source = (Join-Path $portableAssets "README.txt"); Target = "README.txt" }
)

foreach ($file in $requiredFiles) {
    if (-not (Test-Path $file.Source)) {
        throw "Required file not found: $($file.Source)"
    }
}

if (Test-Path $stagingDir) {
    Remove-Item $stagingDir -Recurse -Force
}

New-Item -ItemType Directory -Path $stagingDir | Out-Null

foreach ($file in $requiredFiles) {
    Copy-Item $file.Source (Join-Path $stagingDir $file.Target) -Force
}

if (Test-Path $zipPath) {
    Remove-Item $zipPath -Force
}

Compress-Archive -Path $stagingDir -DestinationPath $zipPath -CompressionLevel Optimal

Write-Host "Portable staging directory: $stagingDir"
Write-Host "Portable zip package: $zipPath"
