[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("status", "start", "stop", "restart", "logs")]
    [string]$Action = "status",

    [string]$SshHost = "aliyun2738",
    [string]$RemoteDir = "/opt/charging-ops",
    [string]$Service = ""
)

$ErrorActionPreference = "Stop"

function Invoke-Cloud {
    param([string]$Command)
    ssh $SshHost "cd $RemoteDir && $Command"
}

switch ($Action) {
    "status" {
        Invoke-Cloud "docker compose ps && echo '--- health' && docker inspect --format '{{.Name}} {{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{end}}' charging_ops_db charging_ops_valkey charging_ops_nats 2>/dev/null || true"
    }
    "start" {
        Invoke-Cloud "docker compose up -d"
        Invoke-Cloud "docker compose ps"
    }
    "stop" {
        Invoke-Cloud "docker compose stop"
    }
    "restart" {
        Invoke-Cloud "docker compose restart"
        Invoke-Cloud "docker compose ps"
    }
    "logs" {
        if ($Service) {
            Invoke-Cloud "docker compose logs --tail=120 $Service"
        } else {
            Invoke-Cloud "docker compose logs --tail=120"
        }
    }
}
