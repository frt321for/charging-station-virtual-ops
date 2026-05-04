[CmdletBinding()]
param(
    [string]$BaseUrl = "http://127.0.0.1:8080/api/v1",
    [string]$Username = "ai.assistant",
    [string]$Password = "Password2026!",
    [string]$SiteId = "HQ-CAMPUS",
    [string]$ExpectProvider = "",
    [string]$ModelScopeApiKey = "",
    [string]$ModelScopeBaseUrl = "https://api-inference.modelscope.cn/v1",
    [string]$ModelScopeModel = "deepseek-ai/DeepSeek-V3.2",
    [switch]$RestartBackend
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot

function Invoke-AIRequest {
    param(
        [ValidateSet("GET", "POST")]
        [string]$Method,
        [string]$Path,
        [object]$Body = $null,
        [hashtable]$Headers = @{}
    )

    $params = @{
        Method      = $Method
        Uri         = "$BaseUrl$Path"
        Headers     = $Headers
        ContentType = "application/json"
    }
    if ($null -ne $Body) {
        $params.Body = ($Body | ConvertTo-Json -Depth 8 -Compress)
    }
    return Invoke-RestMethod @params
}

function Wait-BackendReady {
    $deadline = (Get-Date).AddSeconds(25)
    while ((Get-Date) -lt $deadline) {
        try {
            $health = Invoke-AIRequest -Method GET -Path "/health" -Headers @{}
            if ($health.code -eq 0) {
                return
            }
        } catch {
            Start-Sleep -Milliseconds 500
        }
        Start-Sleep -Milliseconds 500
    }
    throw "backend did not become ready"
}

if ($ModelScopeApiKey.Trim() -ne "") {
    $env:MODELSCOPE_API_KEY = $ModelScopeApiKey.Trim()
    $env:MODELSCOPE_BASE_URL = $ModelScopeBaseUrl
    $env:MODELSCOPE_MODEL = $ModelScopeModel
    if ($ExpectProvider -eq "") {
        $ExpectProvider = "modelscope"
    }
}

if ($RestartBackend) {
    & (Join-Path $ProjectRoot "scripts\backend.ps1") restart | Out-Host
    Wait-BackendReady
}

$login = Invoke-AIRequest -Method POST -Path "/auth/login" -Body @{
    username = $Username
    password = $Password
}

if (-not $login.data.token) {
    throw "login failed: token missing"
}

$headers = @{ Authorization = "Bearer $($login.data.token)" }
$result = Invoke-AIRequest -Method POST -Path "/ai/station-qa" -Headers $headers -Body @{
    siteId = $SiteId
    question = "请给出当前站点的一个运营风险和一个处理建议"
}

$provider = [string]$result.data.provider
$model = [string]$result.data.model
$content = [string]$result.data.content

if ($ExpectProvider -ne "" -and $provider -ne $ExpectProvider) {
    throw "unexpected AI provider: expected $ExpectProvider, got $provider"
}

Write-Host "AI provider: $provider"
Write-Host "AI model: $model"
Write-Host "AI content length: $($content.Length)"
Write-Host "AI verification passed"
