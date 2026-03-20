[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ForwardArgs
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$RepoSlug = if ($env:GOCHAT_DEPLOYER_REPO) { $env:GOCHAT_DEPLOYER_REPO } else { 'FlameInTheDark/gochat-deployment' }
$DeployerVersion = if ($env:GOCHAT_DEPLOYER_VERSION) { $env:GOCHAT_DEPLOYER_VERSION } else { 'latest' }
$UseRelease = if ($env:GOCHAT_DEPLOYER_USE_RELEASE) { $env:GOCHAT_DEPLOYER_USE_RELEASE } else { '0' }

function Write-Info {
    param([string]$Message)
    Write-Host "[gochat] $Message"
}

function Test-Truthy {
    param([string]$Value)

    if ([string]::IsNullOrWhiteSpace($Value)) {
        return $false
    }

    switch ($Value.Trim().ToLowerInvariant()) {
        '1' { return $true }
        'true' { return $true }
        'yes' { return $true }
        'on' { return $true }
        default { return $false }
    }
}

function Test-Command {
    param([string]$Name)
    return [bool](Get-Command $Name -ErrorAction SilentlyContinue)
}

function Test-RepoRoot {
    param([string]$Path)

    return (
        (Test-Path -LiteralPath (Join-Path $Path 'go.mod')) -and
        (Test-Path -LiteralPath (Join-Path $Path 'main.go')) -and
        (Test-Path -LiteralPath (Join-Path $Path 'bundle.go')) -and
        (Test-Path -LiteralPath (Join-Path $Path 'deployer'))
    )
}

function Get-OsName {
    if ([System.Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([System.Runtime.InteropServices.OSPlatform]::Windows)) {
        return 'windows'
    }
    if ([System.Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([System.Runtime.InteropServices.OSPlatform]::Linux)) {
        return 'linux'
    }
    if ([System.Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([System.Runtime.InteropServices.OSPlatform]::OSX)) {
        return 'darwin'
    }

    throw 'Unsupported operating system'
}

function Get-ArchName {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
    switch ($arch) {
        'x64' { return 'amd64' }
        'arm64' { return 'arm64' }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Get-BinaryName {
    param([string]$OsName)

    if ($OsName -eq 'windows') {
        return 'gochat-deployer.exe'
    }

    return 'gochat-deployer'
}

function Invoke-External {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$Arguments = @(),
        [string]$WorkingDirectory
    )

    if ($WorkingDirectory) {
        Push-Location $WorkingDirectory
    }

    try {
        & $FilePath @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "$FilePath exited with code $LASTEXITCODE"
        }
    }
    finally {
        if ($WorkingDirectory) {
            Pop-Location
        }
    }
}

function Build-LocalBinary {
    param([string]$RepoRoot)

    $osName = Get-OsName
    $outputPath = Join-Path $RepoRoot '.generated\bin'
    $binaryPath = Join-Path $outputPath (Get-BinaryName $osName)
    $goCache = Join-Path $RepoRoot '.generated\go-build'
    $goModCache = Join-Path $RepoRoot '.generated\gomodcache'
    $previousGoCache = $env:GOCACHE
    $previousGoModCache = $env:GOMODCACHE

    New-Item -ItemType Directory -Force -Path $outputPath | Out-Null
    New-Item -ItemType Directory -Force -Path $goCache | Out-Null
    New-Item -ItemType Directory -Force -Path $goModCache | Out-Null
    Write-Info "Building local deployer binary from $RepoRoot"
    try {
        $env:GOCACHE = $goCache
        $env:GOMODCACHE = $goModCache
        Invoke-External -FilePath 'go' -Arguments @('build', '-o', $binaryPath, '.') -WorkingDirectory $RepoRoot
    }
    finally {
        if ($null -ne $previousGoCache) {
            $env:GOCACHE = $previousGoCache
        }
        else {
            Remove-Item Env:GOCACHE -ErrorAction SilentlyContinue
        }

        if ($null -ne $previousGoModCache) {
            $env:GOMODCACHE = $previousGoModCache
        }
        else {
            Remove-Item Env:GOMODCACHE -ErrorAction SilentlyContinue
        }
    }
    return $binaryPath
}

function Get-CacheRoot {
    $localAppData = [Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)
    if ($localAppData) {
        return (Join-Path $localAppData 'gochat-deployer')
    }

    return (Join-Path $HOME '.cache\gochat-deployer')
}

function Get-ReleaseBinary {
    $osName = Get-OsName
    $archName = Get-ArchName
    $binaryName = Get-BinaryName $osName
    $archiveExt = if ($osName -eq 'windows') { 'zip' } else { 'tar.gz' }
    $assetName = "gochat-deployer_${osName}_${archName}.${archiveExt}"

    if ($DeployerVersion -eq 'latest') {
        $url = "https://github.com/$RepoSlug/releases/latest/download/$assetName"
    }
    else {
        $url = "https://github.com/$RepoSlug/releases/download/$DeployerVersion/$assetName"
    }

    $cacheDir = Join-Path (Get-CacheRoot) (Join-Path $DeployerVersion "$osName-$archName")
    $binaryPath = Join-Path $cacheDir $binaryName
    if (Test-Path -LiteralPath $binaryPath) {
        return $binaryPath
    }

    New-Item -ItemType Directory -Force -Path $cacheDir | Out-Null
    $tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString('N'))
    $archivePath = Join-Path $tempRoot $assetName
    New-Item -ItemType Directory -Force -Path $tempRoot | Out-Null

    try {
        Write-Info "Downloading deployer release asset $assetName"
        $requestArgs = @{
            Uri     = $url
            OutFile = $archivePath
        }
        $webRequest = Get-Command Invoke-WebRequest
        if ($webRequest.Parameters.ContainsKey('UseBasicParsing')) {
            $requestArgs.UseBasicParsing = $true
        }
        Invoke-WebRequest @requestArgs

        if ($archiveExt -eq 'zip') {
            Expand-Archive -Path $archivePath -DestinationPath $tempRoot -Force
        }
        else {
            if (-not (Test-Command 'tar')) {
                throw "tar is required to extract $assetName"
            }
            Invoke-External -FilePath 'tar' -Arguments @('-xzf', $archivePath, '-C', $tempRoot)
        }

        $extractedPath = Join-Path $tempRoot $binaryName
        if (-not (Test-Path -LiteralPath $extractedPath)) {
            throw "Release archive did not contain $binaryName"
        }

        Move-Item -Force -Path $extractedPath -Destination $binaryPath
        return $binaryPath
    }
    finally {
        if (Test-Path -LiteralPath $tempRoot) {
            Remove-Item -Recurse -Force -LiteralPath $tempRoot
        }
    }
}

$repoRootCandidate = if ($PSScriptRoot) {
    [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
}
else {
    (Get-Location).Path
}

$binary = if ((Test-RepoRoot $repoRootCandidate) -and -not (Test-Truthy $UseRelease) -and (Test-Command 'go')) {
    Build-LocalBinary -RepoRoot $repoRootCandidate
}
else {
    Get-ReleaseBinary
}

& $binary @ForwardArgs
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
