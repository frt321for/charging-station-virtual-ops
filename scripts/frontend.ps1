[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("status", "start", "stop", "restart")]
    [string]$Action = "status",

    [int]$Port = 5173
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$FrontendDir = Join-Path $ProjectRoot "src\frontend"
$RunDir = Join-Path $ProjectRoot ".run"
$PidFile = Join-Path $RunDir "frontend.pid"
$OutLog = Join-Path $RunDir "frontend.out.log"
$ErrLog = Join-Path $RunDir "frontend.err.log"

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

function Stop-ProcessTree {
    param([int]$ProcessId)

    $children = Get-CimInstance Win32_Process -Filter "ParentProcessId = $ProcessId" -ErrorAction SilentlyContinue
    foreach ($child in $children) {
        Stop-ProcessTree -ProcessId $child.ProcessId
    }

    $process = Get-Process -Id $ProcessId -ErrorAction SilentlyContinue
    if ($process) {
        Stop-Process -Id $ProcessId -Force
    }
}

function Stop-FrontendPortOwners {
    param([int]$TargetPort)

    $connections = Get-NetTCPConnection -LocalPort $TargetPort -State Listen -ErrorAction SilentlyContinue
    foreach ($connection in $connections) {
        $processInfo = Get-CimInstance Win32_Process -Filter "ProcessId = $($connection.OwningProcess)" -ErrorAction SilentlyContinue
        if ($processInfo -and $processInfo.CommandLine -like "*$FrontendDir*" -and $processInfo.CommandLine -like "*vite*") {
            Stop-ProcessTree -ProcessId $connection.OwningProcess
            Write-Host "Stopped frontend port owner. PID: $($connection.OwningProcess)"
        }
    }
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
        Write-Host "Frontend: running (PID: $($process.Id))"
    } else {
        Write-Host "Frontend: stopped"
    }

    $status = if (Test-LocalPortOpen -TargetPort $Port) { "open" } else { "closed" }
    Write-Host "Frontend port localhost:$Port $status"
}

function Start-Frontend {
    if (-not (Test-Path $FrontendDir)) {
        throw "Frontend directory not found: $FrontendDir"
    }

    $existing = Get-ManagedProcess
    if ($existing) {
        Write-Host "Frontend already running. PID: $($existing.Id)"
        Show-Status
        return
    }

    New-Item -ItemType Directory -Force -Path $RunDir | Out-Null

    $process = Start-Process `
        -FilePath "pnpm.cmd" `
        -ArgumentList @("dev", "--host", "127.0.0.1", "--port", "$Port") `
        -WorkingDirectory $FrontendDir `
        -PassThru `
        -WindowStyle Hidden `
        -RedirectStandardOutput $OutLog `
        -RedirectStandardError $ErrLog

    $process.Id | Set-Content -Path $PidFile -Encoding ascii
    Write-Host "Frontend started. PID: $($process.Id)"
}

function Stop-Frontend {
    $process = Get-ManagedProcess
    if ($process) {
        Stop-ProcessTree -ProcessId $process.Id
        Write-Host "Stopped frontend. PID: $($process.Id)"
    } else {
        Write-Host "Frontend is not running."
    }

    Stop-FrontendPortOwners -TargetPort $Port
    Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
}

switch ($Action) {
    "status" { Show-Status }
    "start" { Start-Frontend }
    "stop" { Stop-Frontend }
    "restart" {
        Stop-Frontend
        Start-Sleep -Milliseconds 500
        Start-Frontend
    }
}
