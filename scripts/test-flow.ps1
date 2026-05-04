[CmdletBinding()]
param(
    [string]$BaseUrl = "http://127.0.0.1:8080/api/v1",
    [string]$Username = "ops.admin",
    [string]$Password = "Password2026!",
    [switch]$SkipSimulator
)

$ErrorActionPreference = "Stop"

$script:StepCount = 0
$script:Headers = @{}

function Write-Step {
    param([string]$Name)
    $script:StepCount += 1
    Write-Host ("[{0:00}] {1}" -f $script:StepCount, $Name)
}

function Assert-True {
    param(
        [object]$Condition,
        [string]$Message
    )
    if (-not [bool]$Condition) {
        throw $Message
    }
}

function Invoke-FlowApi {
    param(
        [ValidateSet("GET", "POST", "PATCH")]
        [string]$Method,
        [string]$Path,
        [object]$Body = $null,
        [hashtable]$Headers = $script:Headers,
        [switch]$AllowError
    )

    $uri = if ($Path.StartsWith("http")) { $Path } else { "$BaseUrl$Path" }
    $params = @{
        Method      = $Method
        Uri         = $uri
        Headers     = $Headers
        ContentType = "application/json"
    }
    if ($null -ne $Body) {
        $params.Body = ($Body | ConvertTo-Json -Depth 12 -Compress)
    }

    try {
        return Invoke-RestMethod @params
    } catch {
        if ($AllowError) {
            return $_.Exception.Response
        }
        throw
    }
}

function First-Item {
    param([object[]]$Items)
    if ($null -eq $Items -or $Items.Count -eq 0) {
        return $null
    }
    return $Items[0]
}

Write-Step "health check"
$health = Invoke-FlowApi -Method GET -Path "/health" -Headers @{}
Assert-True ($health.code -eq 0) "health check failed"

Write-Step "login as operations admin"
$login = Invoke-FlowApi -Method POST -Path "/auth/login" -Headers @{} -Body @{
    username = $Username
    password = $Password
}
Assert-True ($login.data.token) "login token missing"
$script:Headers = @{ Authorization = "Bearer $($login.data.token)" }

Write-Step "current user, users, roles"
$me = Invoke-FlowApi -Method GET -Path "/auth/me"
$users = Invoke-FlowApi -Method GET -Path "/users"
$roles = Invoke-FlowApi -Method GET -Path "/roles"
Assert-True ($me.data.username -eq $Username) "current user mismatch"
Assert-True ($users.data.list.Count -gt 0) "users list is empty"
Assert-True ($roles.data.list.Count -gt 0) "roles list is empty"

Write-Step "permission negative check"
$maintenanceLogin = Invoke-FlowApi -Method POST -Path "/auth/login" -Headers @{} -Body @{
    username = "maintenance.tech"
    password = $Password
}
$maintenanceHeaders = @{ Authorization = "Bearer $($maintenanceLogin.data.token)" }
$forbidden = Invoke-FlowApi -Method GET -Path "/config/sites/HQ-CAMPUS" -Headers $maintenanceHeaders -AllowError
Assert-True ($forbidden.StatusCode.value__ -eq 403) "maintenance user should not manage config"

Write-Step "site topology and sessions"
$sites = Invoke-FlowApi -Method GET -Path "/sites"
$site = First-Item $sites.data.list
Assert-True ($null -ne $site) "site list is empty"
$topology = Invoke-FlowApi -Method GET -Path "/sites/$($site.code)/topology"
Assert-True ($topology.data.site.code -eq $site.code) "topology site mismatch"

if (-not $SkipSimulator) {
    Write-Step "simulator once full lifecycle"
    $sim = Invoke-FlowApi -Method POST -Path "/simulator/once" -Body @{
        chargerCount = 1
        connectorCount = 1
        onlineRate = 1
        faultRate = 1
        loadCurve = "commute"
    }
    Assert-True ($sim.data.generated -ge 1) "simulator did not generate sessions"
}

Write-Step "refresh session detail"
$sessions = Invoke-FlowApi -Method GET -Path "/sessions"
$session = First-Item @($sessions.data.list | Where-Object { $_.status -eq "pending_billing" })
if ($null -eq $session) {
    $session = First-Item $sessions.data.list
}
Assert-True ($null -ne $session) "session list is empty"
$detail = Invoke-FlowApi -Method GET -Path "/sessions/$($session.sessionNo)"
Assert-True ($detail.data.sessionNo -eq $session.sessionNo) "session detail mismatch"

Write-Step "load control snapshot and manual record"
$load = Invoke-FlowApi -Method GET -Path "/sites/$($site.code)/load-control"
Assert-True ($load.data.siteCode -eq $site.code) "load control site mismatch"
$targetPower = 3.5
$loadRecord = Invoke-FlowApi -Method POST -Path "/load-control/records" -Body @{
    siteId = $site.code
    sessionId = $session.sessionNo
    actionType = "limit_power"
    reason = "test-flow regression"
    beforeLoadKw = [double]$load.data.currentLoadKw
    afterLoadKw = [double]$load.data.currentLoadKw
    targetPowerKw = $targetPower
    status = "applied"
    operatorName = $me.data.displayName
}
Assert-True ($loadRecord.data.recordNo) "load control record missing"

Write-Step "billing draft and reconciliation"
$draft = Invoke-FlowApi -Method POST -Path "/sessions/$($session.sessionNo)/billing-draft" -Body @{
    generatedBy = $me.data.displayName
}
Assert-True ($draft.data.billNo) "billing draft missing"
$drafts = Invoke-FlowApi -Method GET -Path "/billing-drafts?siteId=$($site.code)"
Assert-True ($drafts.data.list.Count -gt 0) "billing drafts empty"
$exceptions = Invoke-FlowApi -Method GET -Path "/reconciliation-exceptions?siteId=$($site.code)"
if ($exceptions.data.list.Count -gt 0) {
    $exception = First-Item $exceptions.data.list
    $review = Invoke-FlowApi -Method POST -Path "/reconciliation-exceptions/$($exception.id)/review" -Body @{
        status = "reviewing"
        reviewerName = $me.data.displayName
        note = "test-flow review"
    }
    Assert-True ($review.data.status -eq "reviewing") "reconciliation review failed"
}
$export = Invoke-FlowApi -Method GET -Path "/reconciliation-exceptions/export?siteId=$($site.code)&generatedBy=$([uri]::EscapeDataString($me.data.displayName))"
Assert-True ($export.data.exportNo) "reconciliation export missing"

Write-Step "maintenance fault and work order"
$maintenance = Invoke-FlowApi -Method GET -Path "/maintenance?siteId=$($site.code)"
if ($maintenance.data.faults.Count -gt 0) {
    $fault = First-Item $maintenance.data.faults
    $workOrder = Invoke-FlowApi -Method POST -Path "/faults/$($fault.id)/work-order" -Body @{
        assigneeName = "maintenance-shift-a"
        impactScope = "charger"
        title = "test-flow fault handling"
        description = "regression work order"
        actorName = $me.data.displayName
    }
    Assert-True ($workOrder.data.workOrderNo) "work order missing"
    if ($workOrder.data.status -eq "open") {
        $transition = Invoke-FlowApi -Method POST -Path "/work-orders/$($workOrder.data.id)/transition" -Body @{
            targetStatus = "assigned"
            assigneeName = "maintenance-shift-a"
            actorName = $me.data.displayName
            note = "test-flow assign"
            payload = @{}
        }
        Assert-True ($transition.data.status -eq "assigned") "work order transition failed"
    }
    $events = Invoke-FlowApi -Method GET -Path "/work-orders/$($workOrder.data.id)/events"
    Assert-True ($events.data.list.Count -gt 0) "work order events empty"
}

Write-Step "ai assistant endpoints"
$aiSession = Invoke-FlowApi -Method POST -Path "/ai/session-explanations" -Body @{ sessionId = $session.sessionNo }
Assert-True ($aiSession.data.content) "ai session explanation empty"
if ($maintenance.data.workOrders.Count -gt 0) {
    $wo = First-Item $maintenance.data.workOrders
    $aiWorkOrder = Invoke-FlowApi -Method POST -Path "/ai/work-order-summaries" -Body @{ workOrderId = $wo.id }
    Assert-True ($aiWorkOrder.data.content) "ai work order summary empty"
}
$aiRisk = Invoke-FlowApi -Method POST -Path "/ai/congestion-risk" -Body @{ siteId = $site.code; horizonHours = 4 }
$aiReport = Invoke-FlowApi -Method POST -Path "/ai/daily-reports" -Body @{ siteId = $site.code; businessDate = (Get-Date -Format "yyyy-MM-dd") }
$aiQA = Invoke-FlowApi -Method POST -Path "/ai/station-qa" -Body @{ siteId = $site.code; question = "当前站点有哪些运营风险" }
Assert-True ($aiRisk.data.content -and $aiReport.data.content -and $aiQA.data.content) "ai endpoint content missing"
Write-Host "AI provider: $($aiQA.data.provider)"

Write-Step "configuration read and update"
$config = Invoke-FlowApi -Method GET -Path "/config/sites/$($site.code)"
Assert-True ($config.data.site.code -eq $site.code) "config site mismatch"
$sitePatch = Invoke-FlowApi -Method PATCH -Path "/config/sites/$($site.code)" -Body @{
    name = $config.data.site.name
    campus = $config.data.site.campus
}
Assert-True ($sitePatch.data.code -eq $site.code) "site config patch failed"
$policyCode = "FLOW-" + (Get-Date -Format "yyyyMMddHHmmss")
$newPolicy = Invoke-FlowApi -Method POST -Path "/config/pricing-policies" -Body @{
    siteId = $site.code
    code = $policyCode
    name = "test-flow pricing"
    chargerType = "all"
    status = "draft"
    periods = @(
        @{
            label = "all-day"
            startMinute = 0
            endMinute = 1440
            energyPricePerKwh = 0.72
            serviceFeePerKwh = 0.18
            occupancyFeePerMinute = 0
        }
    )
}
Assert-True ($newPolicy.data.code -eq $policyCode) "pricing policy create failed"

Write-Step "audit filters"
$auditAll = Invoke-FlowApi -Method GET -Path "/audit-logs?page=1&pageSize=10"
$auditObject = Invoke-FlowApi -Method GET -Path "/audit-logs?entityType=AUTH&page=1&pageSize=10"
$auditAction = Invoke-FlowApi -Method GET -Path "/audit-logs?action=login&page=1&pageSize=10"
$auditActor = Invoke-FlowApi -Method GET -Path "/audit-logs?actorName=$([uri]::EscapeDataString($me.data.displayName))&page=1&pageSize=10"
Assert-True ($auditAll.data.pagination.total -gt 0) "audit logs empty"
Assert-True ($auditObject.data.pagination.total -gt 0) "audit object filter failed"
Assert-True ($auditAction.data.pagination.total -gt 0) "audit action filter failed"
Assert-True ($auditActor.data.pagination.total -gt 0) "audit actor filter failed"

Write-Step "done"
Write-Host "test-flow passed"
