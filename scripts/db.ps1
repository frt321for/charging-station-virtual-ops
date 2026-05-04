[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("status", "up", "down")]
    [string]$Action = "status"
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $ProjectRoot "src\backend"
$EnvFile = Join-Path $ProjectRoot ".env.local"

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

function Import-LocalEnv {
    if (-not (Test-Path $EnvFile)) {
        return
    }

    Get-Content $EnvFile | ForEach-Object {
        if ($_ -match "^\s*#" -or $_ -notmatch "=") {
            return
        }

        $parts = $_.Split("=", 2)
        [Environment]::SetEnvironmentVariable($parts[0].Trim(), $parts[1].Trim(), "Process")
    }
}

if (-not (Test-LocalPortOpen -TargetPort 15432)) {
    throw "PostgreSQL tunnel is not open. Run scripts/tunnels.ps1 start first."
}

Import-LocalEnv

Push-Location $BackendDir
try {
    go run .\cmd\migrate -action $Action
} finally {
    Pop-Location
}
