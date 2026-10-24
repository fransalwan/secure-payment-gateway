<#
.SYNOPSIS
    Secure Payment Gateway - Public Tunnel Launcher for Remote Interviews & Demos
.DESCRIPTION
    Exposes localhost:8080 to a secure, public HTTPS URL using Cloudflare Tunnel or ngrok.
    Zero-config, free, and does not require complex firewall forwarding.
.EXAMPLE
    .\scripts\tunnel.ps1
    .\scripts\tunnel.ps1 -Provider ngrok
#>
param (
    [ValidateSet("auto", "cloudflare", "ngrok")]
    [string]$Provider = "auto",
    [int]$Port = 8080
)

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "  SECURE PAYMENT GATEWAY - PUBLIC TUNNEL LAUNCHER         " -ForegroundColor Yellow
Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "Target Local Service : http://localhost:$Port" -ForegroundColor Gray

# Verify local gateway is active
try {
    $res = Invoke-RestMethod -Uri "http://localhost:$Port/health" -Method Get -TimeoutSec 2 -ErrorAction Stop
    Write-Host "[OK] Gateway service verified active (Status: $($res.data.status))" -ForegroundColor Green
} catch {
    Write-Host "[WARN] Gateway is not running on http://localhost:$Port yet!" -ForegroundColor Red
    Write-Host "       Starting it automatically in the background or run 'make run' first." -ForegroundColor Yellow
}

$hasCloudflared = [bool](Get-Command cloudflared -ErrorAction SilentlyContinue)
$hasNgrok = [bool](Get-Command ngrok -ErrorAction SilentlyContinue)

if ($Provider -eq "auto") {
    if ($hasCloudflared) {
        $Provider = "cloudflare"
    } elseif ($hasNgrok) {
        $Provider = "ngrok"
    } else {
        Write-Host "[ERROR] Neither 'cloudflared' nor 'ngrok' was detected on your system!" -ForegroundColor Red
        Write-Host "Install Cloudflare Tunnel: winget install Cloudflare.cloudflared" -ForegroundColor White
        Write-Host "Or install ngrok:          winget install ngrok" -ForegroundColor White
        exit 1
    }
}

if ($Provider -eq "cloudflare") {
    if (-not $hasCloudflared) {
        Write-Host "[ERROR] cloudflared.exe not found. Install via 'winget install Cloudflare.cloudflared'" -ForegroundColor Red
        exit 1
    }
    Write-Host "`n[STARTING] Launching Cloudflare Tunnel on port $Port..." -ForegroundColor Cyan
    Write-Host "No account or login required. Cloudflare will generate a public HTTPS URL below:`n" -ForegroundColor DarkGray
    cloudflared tunnel --url "http://localhost:$Port"
} elseif ($Provider -eq "ngrok") {
    if (-not $hasNgrok) {
        Write-Host "[ERROR] ngrok.exe not found. Install via 'winget install ngrok'" -ForegroundColor Red
        exit 1
    }
    Write-Host "`n[STARTING] Launching ngrok on port $Port..." -ForegroundColor Cyan
    ngrok http $Port
}
