[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("status", "start", "stop", "restart")]
    [string]$Action = "status",

    [int]$Port = 8080
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $ProjectRoot "src\backend"
$RunDir = Join-Path $ProjectRoot ".run"
$PidFile = Join-Path $RunDir "backend.pid"
$OutLog = Join-Path $RunDir "backend.out.log"
$ErrLog = Join-Path $RunDir "backend.err.log"
$EnvFile = Join-Path $ProjectRoot ".env.local"
$BinaryPath = Join-Path $RunDir "backend-api.exe"

function Get-ManagedProcess {
    if (-not (Test-Path $PidFile)) {
        return $null
    }

    $pidText = Get-Content $PidFile -ErrorAction SilentlyContinue
    if (-not $pidText) {
        return $null
    }

    return Get-Process -Id ([int]$pidText) -ErrorAction SilentlyContinue
}

function Test-LocalPortOpen {
    param([int]$TargetPort)

    $client = [System.Net.Sockets.TcpClient]::new()
    try {
        $task = $client.ConnectAsync("127.0.0.1", $TargetPort)
        return $task.Wait(500) -and $client.Connected
    } finally {
        $client.Dispose()
    }
}

function Show-Status {
    $process = Get-ManagedProcess
    if ($process) {
        Write-Host "Backend: running (PID: $($process.Id))"
    } else {
        Write-Host "Backend: stopped"
    }

    $status = if (Test-LocalPortOpen -TargetPort $Port) { "open" } else { "closed" }
    Write-Host "Backend port localhost:$Port $status"
}

function Start-Backend {
    if (-not (Test-Path $BackendDir)) {
        throw "Backend directory not found: $BackendDir"
    }

    $existing = Get-ManagedProcess
    if ($existing) {
        Write-Host "Backend already running. PID: $($existing.Id)"
        Show-Status
        return
    }

    New-Item -ItemType Directory -Force -Path $RunDir | Out-Null

    $pgOpen = Test-LocalPortOpen -TargetPort 15432
    if (-not $pgOpen) {
        throw "PostgreSQL tunnel is not open. Run scripts/tunnels.ps1 start first."
    }

    if (Test-Path $EnvFile) {
        Get-Content $EnvFile | ForEach-Object {
            if ($_ -match "^\s*#" -or $_ -notmatch "=") {
                return
            }

            $parts = $_.Split("=", 2)
            [Environment]::SetEnvironmentVariable($parts[0].Trim(), $parts[1].Trim(), "Process")
        }
    }

    [Environment]::SetEnvironmentVariable("APP_PORT", "$Port", "Process")

    Push-Location $BackendDir
    try {
        go build -o $BinaryPath ./cmd/api
    } finally {
        Pop-Location
    }

    $process = Start-Process `
        -FilePath $BinaryPath `
        -ArgumentList @("--host", "127.0.0.1", "--port", "$Port") `
        -WorkingDirectory $BackendDir `
        -PassThru `
        -WindowStyle Hidden `
        -RedirectStandardOutput $OutLog `
        -RedirectStandardError $ErrLog

    $process.Id | Set-Content -Path $PidFile -Encoding ascii
    Write-Host "Backend started. PID: $($process.Id)"
}

function Stop-Backend {
    $process = Get-ManagedProcess
    if ($process) {
        Stop-Process -Id $process.Id -Force
        Write-Host "Stopped backend. PID: $($process.Id)"
    } else {
        Write-Host "Backend is not running."
    }

    Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
}

switch ($Action) {
    "status" { Show-Status }
    "start" { Start-Backend }
    "stop" { Stop-Backend }
    "restart" {
        Stop-Backend
        Start-Sleep -Milliseconds 500
        Start-Backend
    }
}
