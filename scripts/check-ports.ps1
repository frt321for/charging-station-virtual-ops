[CmdletBinding()]
param(
    [int]$BackendPort = 8080,
    [int]$FrontendPort = 5173,
    [int]$PostgresPort = 15432,
    [int]$ValkeyPort = 16379,
    [int]$NatsPort = 14222,
    [int]$NatsMonitorPort = 18222
)

$ErrorActionPreference = "Stop"

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

function Show-Port {
    param(
        [string]$Name,
        [int]$Port
    )

    $status = if (Test-LocalPortOpen -Port $Port) { "open" } else { "closed" }
    Write-Host ("{0,-18} localhost:{1,-6} {2}" -f $Name, $Port, $status)
}

Show-Port -Name "Backend" -Port $BackendPort
Show-Port -Name "Frontend" -Port $FrontendPort
Show-Port -Name "PostgreSQL tunnel" -Port $PostgresPort
Show-Port -Name "Valkey tunnel" -Port $ValkeyPort
Show-Port -Name "NATS tunnel" -Port $NatsPort
Show-Port -Name "NATS monitor" -Port $NatsMonitorPort
