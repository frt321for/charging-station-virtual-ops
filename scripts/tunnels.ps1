[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("status", "start", "stop", "restart")]
    [string]$Action = "status",

    [string]$SshHost = "aliyun2738",
    [int]$PostgresLocalPort = 15432,
    [int]$ValkeyLocalPort = 16379,
    [int]$NatsLocalPort = 14222,
    [int]$NatsMonitorLocalPort = 18222
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$RunDir = Join-Path $ProjectRoot ".run"
$PidFile = Join-Path $RunDir "ssh-tunnels.pid"
$OutLog = Join-Path $RunDir "ssh-tunnels.out.log"
$ErrLog = Join-Path $RunDir "ssh-tunnels.err.log"

function Get-TunnelProcess {
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
    param([int]$Port)

    $client = [System.Net.Sockets.TcpClient]::new()
    try {
        $task = $client.ConnectAsync("127.0.0.1", $Port)
        return $task.Wait(500) -and $client.Connected
    } finally {
        $client.Dispose()
    }
}

function Test-LocalPortFree {
    param([int]$Port)

    $used = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue
    if ($used) {
        throw "Local port $Port is already in use."
    }
}

function Wait-LocalPort {
    param(
        [int]$Port,
        [int]$TimeoutSeconds = 10
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        if (Test-LocalPortOpen -Port $Port) {
            return
        }
        Start-Sleep -Milliseconds 300
    }

    throw "Timed out waiting for local port $Port."
}

function Show-TunnelStatus {
    $process = Get-TunnelProcess
    if ($process) {
        Write-Host "SSH tunnels: running (PID: $($process.Id))"
    } else {
        Write-Host "SSH tunnels: stopped"
    }

    $ports = @(
        @("PostgreSQL", $PostgresLocalPort),
        @("Valkey", $ValkeyLocalPort),
        @("NATS", $NatsLocalPort),
        @("NATS Monitor", $NatsMonitorLocalPort)
    )

    foreach ($entry in $ports) {
        $open = Test-LocalPortOpen -Port $entry[1]
        $status = if ($open) { "open" } else { "closed" }
        Write-Host ("{0,-14} localhost:{1,-6} {2}" -f $entry[0], $entry[1], $status)
    }
}

function Start-Tunnels {
    New-Item -ItemType Directory -Force -Path $RunDir | Out-Null

    $existingProcess = Get-TunnelProcess
    if ($existingProcess) {
        Write-Host "SSH tunnels already running. PID: $($existingProcess.Id)"
        Show-TunnelStatus
        return
    }

    if (Test-Path $PidFile) {
        Remove-Item -LiteralPath $PidFile -Force
    }

    Test-LocalPortFree -Port $PostgresLocalPort
    Test-LocalPortFree -Port $ValkeyLocalPort
    Test-LocalPortFree -Port $NatsLocalPort
    Test-LocalPortFree -Port $NatsMonitorLocalPort

    $sshArgs = @(
        "-N",
        "-o", "ExitOnForwardFailure=yes",
        "-L", "$PostgresLocalPort`:127.0.0.1:5432",
        "-L", "$ValkeyLocalPort`:127.0.0.1:6379",
        "-L", "$NatsLocalPort`:127.0.0.1:4222",
        "-L", "$NatsMonitorLocalPort`:127.0.0.1:8222",
        $SshHost
    )

    $process = Start-Process `
        -FilePath "ssh" `
        -ArgumentList $sshArgs `
        -PassThru `
        -WindowStyle Hidden `
        -RedirectStandardOutput $OutLog `
        -RedirectStandardError $ErrLog

    $process.Id | Set-Content -Path $PidFile -Encoding ascii

    Start-Sleep -Seconds 1
    if ($process.HasExited) {
        Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
        $errorText = Get-Content $ErrLog -Raw -ErrorAction SilentlyContinue
        throw "SSH tunnel process exited early. $errorText"
    }

    Wait-LocalPort -Port $PostgresLocalPort
    Wait-LocalPort -Port $ValkeyLocalPort
    Wait-LocalPort -Port $NatsLocalPort
    Wait-LocalPort -Port $NatsMonitorLocalPort

    Write-Host "SSH tunnels started. PID: $($process.Id)"
    Show-TunnelStatus
}

function Stop-Tunnels {
    $process = Get-TunnelProcess
    if ($process) {
        Stop-Process -Id $process.Id -Force
        Write-Host "Stopped SSH tunnels. PID: $($process.Id)"
    } else {
        Write-Host "SSH tunnels are not running."
    }

    Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
}

switch ($Action) {
    "status" {
        Show-TunnelStatus
    }
    "start" {
        Start-Tunnels
    }
    "stop" {
        Stop-Tunnels
    }
    "restart" {
        Stop-Tunnels
        Start-Sleep -Milliseconds 500
        Start-Tunnels
    }
}
