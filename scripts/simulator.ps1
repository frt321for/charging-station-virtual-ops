[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("status", "once", "start", "stop", "restart")]
    [string]$Action = "status",

    [string]$ApiBase = "http://127.0.0.1:8080",
    [string]$GroupCode = "N-A1",
    [string]$Prefix = "SIM",
    [int]$Chargers = 1,
    [int]$Connectors = 1,
    [double]$OnlineRate = 1,
    [double]$FaultRate = 0,
    [ValidateSet("flat", "commute", "random")]
    [string]$LoadCurve = "flat",
    [int]$Ticks = 5,
    [int]$IntervalSeconds = 5
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $ProjectRoot "src\backend"
$RunDir = Join-Path $ProjectRoot ".run"
$PidFile = Join-Path $RunDir "simulator.pid"
$OutLog = Join-Path $RunDir "simulator.out.log"
$ErrLog = Join-Path $RunDir "simulator.err.log"
$BinaryPath = Join-Path $RunDir "virtual-charger-simulator.exe"

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

function Get-ApiPort {
    $uri = [System.Uri]$ApiBase
    if ($uri.Port -gt 0) {
        return $uri.Port
    }
    if ($uri.Scheme -eq "https") {
        return 443
    }
    return 80
}

function Assert-BackendOpen {
    $port = Get-ApiPort
    if (-not (Test-LocalPortOpen -TargetPort $port)) {
        throw "Backend API is not reachable at $ApiBase. Run scripts/backend.ps1 start first."
    }
}

function Build-Simulator {
    if (-not (Test-Path $BackendDir)) {
        throw "Backend directory not found: $BackendDir"
    }

    New-Item -ItemType Directory -Force -Path $RunDir | Out-Null
    Push-Location $BackendDir
    try {
        go build -o $BinaryPath ./cmd/simulator
    } finally {
        Pop-Location
    }
}

function New-SimulatorArgs {
    param([string]$Mode)

    return @(
        "-mode", $Mode,
        "-api-base", $ApiBase,
        "-group-code", $GroupCode,
        "-prefix", $Prefix,
        "-chargers", "$Chargers",
        "-connectors", "$Connectors",
        "-online-rate", "$OnlineRate",
        "-fault-rate", "$FaultRate",
        "-load-curve", $LoadCurve,
        "-ticks", "$Ticks",
        "-interval", "$($IntervalSeconds)s"
    )
}

function Show-Status {
    $process = Get-ManagedProcess
    if ($process) {
        Write-Host "Simulator: running (PID: $($process.Id))"
    } else {
        Write-Host "Simulator: stopped"
    }

    $port = Get-ApiPort
    $status = if (Test-LocalPortOpen -TargetPort $port) { "open" } else { "closed" }
    Write-Host "Backend API $ApiBase $status"
}

function Invoke-Once {
    Assert-BackendOpen
    Build-Simulator
    $argsList = New-SimulatorArgs -Mode "once"
    & $BinaryPath @argsList
}

function Start-Simulator {
    Assert-BackendOpen
    $existing = Get-ManagedProcess
    if ($existing) {
        Write-Host "Simulator already running. PID: $($existing.Id)"
        Show-Status
        return
    }

    Build-Simulator
    $process = Start-Process `
        -FilePath $BinaryPath `
        -ArgumentList (New-SimulatorArgs -Mode "run") `
        -WorkingDirectory $BackendDir `
        -PassThru `
        -WindowStyle Hidden `
        -RedirectStandardOutput $OutLog `
        -RedirectStandardError $ErrLog

    $process.Id | Set-Content -Path $PidFile -Encoding ascii
    Write-Output "Simulator started. PID: $($process.Id)"
}

function Stop-Simulator {
    $process = Get-ManagedProcess
    if ($process) {
        Stop-Process -Id $process.Id -Force
        Write-Host "Stopped simulator. PID: $($process.Id)"
    } else {
        Write-Host "Simulator is not running."
    }

    Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
}

switch ($Action) {
    "status" { Show-Status }
    "once" { Invoke-Once }
    "start" { Start-Simulator }
    "stop" { Stop-Simulator }
    "restart" {
        Stop-Simulator
        Start-Sleep -Milliseconds 500
        Start-Simulator
    }
}
