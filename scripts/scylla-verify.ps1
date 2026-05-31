param(
    [string]$SourceNamespace = "gochat",
    [string]$SourcePod = "gochat-scylla-0",
    [string]$SourceContainer = "",
    [string]$TargetNamespace = "gochat-scylla",
    [string]$TargetPod = "",
    [string]$TargetContainer = "scylla",
    [string]$TargetHost = "gochat-scylla-client.gochat-scylla.svc.cluster.local",
    [string]$Keyspace = "gochat",
    [string]$OutDir = ".generated/scylla-verify"
)

$ErrorActionPreference = "Stop"

function Invoke-Cql {
    param(
        [string]$Namespace,
        [string]$Pod,
        [string]$Container,
        [string]$Host,
        [string]$Query
    )

    $args = @("-n", $Namespace, "exec", $Pod)
    if ($Container -ne "") {
        $args += @("-c", $Container)
    }
    $args += @("--", "cqlsh")
    if ($Host -ne "") {
        $args += @($Host)
    }
    $args += @("-e", $Query)

    $output = & kubectl @args
    if ($LASTEXITCODE -ne 0) {
        throw "cqlsh failed in $Namespace/$Pod"
    }
    return $output
}

function Resolve-TargetPod {
    if ($TargetPod -ne "") {
        return $TargetPod
    }

    $json = & kubectl -n $TargetNamespace get pods -l app.kubernetes.io/name=scylla -o json | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0 -or @($json.items).Count -eq 0) {
        throw "No ScyllaDB target pods found in namespace $TargetNamespace"
    }
    return @($json.items | Where-Object { $_.status.phase -eq "Running" } | Select-Object -First 1).metadata.name
}

function Get-Tables {
    param(
        [string]$Namespace,
        [string]$Pod,
        [string]$Container,
        [string]$Host
    )

    $raw = Invoke-Cql -Namespace $Namespace -Pod $Pod -Container $Container -Host $Host -Query "SELECT table_name FROM system_schema.tables WHERE keyspace_name='$Keyspace';"
    return @($raw | ForEach-Object { $_.Trim() } | Where-Object { $_ -and $_ -notmatch "^table_name|^-|^\(|rows\)" } | Sort-Object)
}

function Get-Counts {
    param(
        [string]$Namespace,
        [string]$Pod,
        [string]$Container,
        [string]$Host,
        [string[]]$Tables
    )

    $counts = @{}
    foreach ($table in $Tables) {
        $raw = Invoke-Cql -Namespace $Namespace -Pod $Pod -Container $Container -Host $Host -Query "SELECT count(*) FROM $Keyspace.$table;"
        $match = ($raw | Select-String -Pattern "^\s*\d+\s*$" | Select-Object -First 1)
        if ($null -eq $match) {
            throw "Could not parse count for $Namespace/$Pod $Keyspace.$table"
        }
        $counts[$table] = [int64]$match.Matches.Value.Trim()
    }
    return $counts
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$resolvedTargetPod = Resolve-TargetPod
$sourceTables = Get-Tables -Namespace $SourceNamespace -Pod $SourcePod -Container $SourceContainer -Host ""
$targetTables = Get-Tables -Namespace $TargetNamespace -Pod $resolvedTargetPod -Container $TargetContainer -Host $TargetHost
$tables = @($sourceTables + $targetTables | Sort-Object -Unique)

$sourceCounts = Get-Counts -Namespace $SourceNamespace -Pod $SourcePod -Container $SourceContainer -Host "" -Tables $tables
$targetCounts = Get-Counts -Namespace $TargetNamespace -Pod $resolvedTargetPod -Container $TargetContainer -Host $TargetHost -Tables $tables

$rows = foreach ($table in $tables) {
    [pscustomobject]@{
        table = $table
        source = $sourceCounts[$table]
        target = $targetCounts[$table]
        match = ($sourceCounts[$table] -eq $targetCounts[$table])
    }
}

$report = [pscustomobject]@{
    generated_at = (Get-Date).ToString("o")
    source = "$SourceNamespace/$SourcePod"
    target = "$TargetNamespace/$resolvedTargetPod via $TargetHost"
    keyspace = $Keyspace
    tables = $rows
}

$reportPath = Join-Path $OutDir "scylla-counts.json"
$report | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $reportPath -Encoding UTF8
$rows | Format-Table -AutoSize

$mismatches = @($rows | Where-Object { -not $_.match })
if ($mismatches.Count -gt 0) {
    throw "ScyllaDB verification failed: $($mismatches.Count) table count mismatch(es). Report: $reportPath"
}

Write-Host "ScyllaDB verification passed. Report: $reportPath"
