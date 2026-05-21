# Full-stack smoke: docker compose + Playwright against live API/nginx proxy.
$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

function Wait-Url([string]$Url, [int]$MaxSeconds = 180) {
  $deadline = (Get-Date).AddSeconds($MaxSeconds)
  while ((Get-Date) -lt $deadline) {
    try {
      $r = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 5
      if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 300) {
        Write-Host "OK $Url"
        return
      }
    } catch {
      # retry
    }
    Start-Sleep -Seconds 2
  }
  throw "Timed out waiting for $Url"
}

Write-Host 'Building and starting docker compose...'
docker compose up -d --build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Wait-Url 'http://127.0.0.1:8080/health'
Wait-Url 'http://127.0.0.1/api/health'

Write-Host 'Running Playwright (E2E_COMPOSE=1)...'
$env:E2E_COMPOSE = '1'
pnpm --filter @codeatlas/web exec playwright install chromium
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
pnpm --filter @codeatlas/web test:e2e
exit $LASTEXITCODE
