param(
  [Parameter(Mandatory = $true)]
  [string]$OutputPath,

  [ValidateSet("running", "onboarding")]
  [string]$Preview = "running",

  [int]$Port = 34115,

  [int]$Width = 1440,

  [int]$Height = 1600
)

$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$distPath = Join-Path $projectRoot "frontend/dist"
$resolvedOutput = [System.IO.Path]::GetFullPath((Join-Path $projectRoot $OutputPath))
$outputDir = Split-Path -Parent $resolvedOutput

if (-not (Test-Path $distPath)) {
  throw "前端构建产物不存在：$distPath"
}

New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

$browserCandidates = @(
  "C:\Program Files (x86)\Microsoft\EdgeCore\131.0.2903.86\msedge.exe",
  "C:\Program Files (x86)\Microsoft\EdgeWebView\Application\131.0.2903.86\msedge.exe",
  "C:\Users\Administrator\AppData\Local\Google\Chrome\Application\chrome.exe"
)

$browserPath = $browserCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $browserPath) {
  throw "未找到可用的 Chromium 浏览器可执行文件。"
}

$pythonServer = $null

try {
  $pythonServer = Start-Process -FilePath "python" -ArgumentList @("-m", "http.server", "$Port", "--bind", "127.0.0.1", "--directory", $distPath) -WorkingDirectory $projectRoot -WindowStyle Hidden -PassThru

  $ready = $false
  for ($index = 0; $index -lt 40; $index++) {
    Start-Sleep -Milliseconds 250
    try {
      $client = New-Object System.Net.Sockets.TcpClient
      $client.Connect("127.0.0.1", $Port)
      $client.Close()
      $ready = $true
      break
    } catch {
    }
  }

  if (-not $ready) {
    throw "本地预览服务未能在端口 $Port 上启动。"
  }

  $previewUrl = "http://127.0.0.1:$Port/?preview=$Preview"
  $browserArgs = @(
    "--headless",
    "--disable-gpu",
    "--hide-scrollbars",
    "--window-size=$Width,$Height",
    "--virtual-time-budget=3000",
    "--screenshot=$resolvedOutput",
    $previewUrl
  )
  & $browserPath @browserArgs | Out-Null

  Start-Sleep -Milliseconds 500

  if (-not (Test-Path $resolvedOutput)) {
    throw "截图失败，未生成文件：$resolvedOutput"
  }

  Get-Item $resolvedOutput | Select-Object FullName, Length, LastWriteTime
} finally {
  if ($pythonServer -and -not $pythonServer.HasExited) {
    Stop-Process -Id $pythonServer.Id -Force -ErrorAction SilentlyContinue
  }
}
