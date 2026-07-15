[CmdletBinding()]
param(
    [Parameter()]
    [string]$Version = '0.1.0-rc.2'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = [System.IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$temporaryBase = [System.IO.Path]::GetFullPath((Join-Path ([System.IO.Path]::GetTempPath()) '.'))
$temporaryRoot = [System.IO.Path]::GetFullPath((Join-Path $temporaryBase ("clewkeep-clean-install-" + [Guid]::NewGuid().ToString('N'))))
$distDirectory = [System.IO.Path]::GetFullPath((Join-Path $repoRoot 'dist'))
$utf8 = [System.Text.UTF8Encoding]::new($false)

function Remove-SafeDirectory {
    param(
        [Parameter(Mandatory)] [string]$Path,
        [Parameter(Mandatory)] [string]$ExpectedParent
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return
    }
    $resolved = [System.IO.Path]::GetFullPath($Path)
    $parent = [System.IO.Path]::GetFullPath((Split-Path -Parent $resolved))
    $expected = [System.IO.Path]::GetFullPath($ExpectedParent)
    if (-not $parent.Equals($expected, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing unexpected cleanup path: $resolved"
    }
    $item = Get-Item -LiteralPath $resolved -Force
    if (-not $item.PSIsContainer -or ($item.Attributes -band [System.IO.FileAttributes]::ReparsePoint)) {
        throw "Refusing to recursively clean a non-directory or reparse point: $resolved"
    }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}

$temporaryParent = [System.IO.Path]::GetFullPath((Split-Path -Parent $temporaryRoot))
if (-not $temporaryParent.Equals($temporaryBase, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing unexpected temporary path: $temporaryRoot"
}
$distParent = [System.IO.Path]::GetFullPath((Split-Path -Parent $distDirectory))
if (-not $distParent.Equals($repoRoot, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing unexpected dist path: $distDirectory"
}

$savedEnvironment = @{}
foreach ($name in @('USERPROFILE', 'HOME', 'CTX_HOME')) {
    $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
$repoStatusBefore = @(& git -C $repoRoot -c core.quotePath=true status --porcelain=v1 --untracked-files=all)
if ($LASTEXITCODE -ne 0) {
    throw 'git status failed'
}

try {
    [System.IO.Directory]::CreateDirectory($temporaryRoot) | Out-Null
    & (Join-Path $PSScriptRoot 'build-dist.ps1') -Version $Version

    $archiveName = "clewkeep-$Version-windows-amd64.zip"
    $archivePath = Join-Path $distDirectory $archiveName
    $checksumLine = Get-Content -LiteralPath (Join-Path $distDirectory 'SHA256SUMS') |
        Where-Object { $_ -match ([regex]::Escape($archiveName) + '$') } |
        Select-Object -First 1
    if ($checksumLine -notmatch '^([0-9a-f]{64})  ([0-9A-Za-z._-]+)$') {
        throw 'Windows archive checksum entry is missing or malformed'
    }
    $expectedHash = $Matches[1]
    $actualHash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne $expectedHash) {
        throw 'Windows archive checksum mismatch'
    }

    $installRoot = Join-Path $temporaryRoot 'install'
    [System.IO.Directory]::CreateDirectory($installRoot) | Out-Null
    Expand-Archive -LiteralPath $archivePath -DestinationPath $installRoot
    $ctx = Join-Path $installRoot 'ctx.exe'
    $versionOutput = (& $ctx version) -join "`n"
    if ($LASTEXITCODE -ne 0 -or $versionOutput -cne "ctx $Version") {
        throw "Installed version mismatch: $versionOutput"
    }

    $syntheticHome = Join-Path $temporaryRoot 'home'
    $ctxHome = Join-Path $temporaryRoot 'ctx-home'
    $codexRoot = Join-Path $syntheticHome '.codex\sessions\2026\01\01'
    $claudeRoot = Join-Path $syntheticHome '.claude\projects\clean-project'
    [System.IO.Directory]::CreateDirectory($codexRoot) | Out-Null
    [System.IO.Directory]::CreateDirectory($claudeRoot) | Out-Null
    $env:USERPROFILE = $syntheticHome
    $env:HOME = $syntheticHome
    $env:CTX_HOME = $ctxHome

    $codexPath = Join-Path $codexRoot 'clean-codex-001.jsonl'
    $codexLines = @(
        '{"type":"session_meta","payload":{"id":"clean-codex-001","cwd":"C:\\synthetic\\codex-project","timestamp":"2026-01-01T00:00:00Z","source":"exec","originator":"clean-test","cli_version":"test"}}',
        '{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"rare-clean-install-needle"}]}}'
    )
    [System.IO.File]::WriteAllText($codexPath, ($codexLines -join "`n") + "`n", $utf8)

    $claudePath = Join-Path $claudeRoot 'clean-claude-001.jsonl'
    $claudeLine = '{"sessionId":"clean-claude-001","cwd":"C:\\synthetic\\claude-project","timestamp":"2026-01-01T00:00:00Z","type":"user","message":{"role":"user","content":"rare-claude-clean-needle"}}'
    [System.IO.File]::WriteAllText($claudePath, $claudeLine + "`n", $utf8)
    $nativeHashes = @{
        $codexPath = (Get-FileHash -LiteralPath $codexPath -Algorithm SHA256).Hash
        $claudePath = (Get-FileHash -LiteralPath $claudePath -Algorithm SHA256).Hash
    }

    $catalog = ((& $ctx scan --json) -join "`n") | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0 -or $catalog.format -cne 'CtxCatalog' -or $catalog.schema_version -cne '0.2') {
        throw 'Clean scan did not produce a schema 0.2 catalog'
    }
    $records = @($catalog.threads)
    $warningsProperty = $catalog.PSObject.Properties['warnings']
    $warningCount = if ($null -eq $warningsProperty) { 0 } else { @($warningsProperty.Value).Count }
    if ($records.Count -ne 2 -or $warningCount -ne 0) {
        throw "Clean scan expected 2 records and 0 warnings; got $($records.Count) records"
    }
    $providers = @($records.provider | Sort-Object -Unique)
    if (Compare-Object $providers @('claude-code', 'codex')) {
        throw "Unexpected providers: $($providers -join ', ')"
    }

    $hits = @(((& $ctx search 'rare-clean-install-needle' --provider codex --project codex-project --limit 5 --json) -join "`n") | ConvertFrom-Json)
    if ($LASTEXITCODE -ne 0 -or $hits.Count -ne 1) {
        throw "Filtered search expected one result; got $($hits.Count)"
    }
    $shown = ((& $ctx show ([string]$hits[0].thread_id) --json) -join "`n") | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0 -or [string]$shown.thread.id -cne [string]$hits[0].thread_id) {
        throw 'Search result was not directly show-addressable'
    }

    $fullCatalog = ((& $ctx scan --full --json) -join "`n") | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0 -or @($fullCatalog.threads).Count -ne 2) {
        throw 'Full scan changed the clean record count'
    }
    $status = ((& $ctx status --json) -join "`n") | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0 -or $status.threads -ne 2 -or $status.projects -ne 2 -or $status.warnings -ne 0) {
        throw 'Status did not report 2 records, 2 projects, and 0 warnings'
    }
    $doctorJson = (& $ctx doctor --json) -join "`n"
    $doctor = ConvertFrom-Json -InputObject $doctorJson
    $failedDoctorChecks = @()
    foreach ($check in $doctor) {
        if ($null -eq $check.PSObject.Properties['status'] -or [string]$check.status -cne 'ok') {
            $failedDoctorChecks += $check
        }
    }
    if ($LASTEXITCODE -ne 0 -or $failedDoctorChecks.Count -ne 0) {
        throw "Doctor reported a failed clean-install check: $($failedDoctorChecks | ConvertTo-Json -Compress)"
    }

    foreach ($nativePath in $nativeHashes.Keys) {
        if ((Get-FileHash -LiteralPath $nativePath -Algorithm SHA256).Hash -ne $nativeHashes[$nativePath]) {
            throw 'A synthetic native transcript was modified'
        }
    }

    Remove-SafeDirectory $installRoot $temporaryRoot
    if (Test-Path -LiteralPath $ctx) {
        throw 'Uninstall left ctx.exe behind'
    }
    Write-Host 'PASS: clean Windows install, scan, filtered search, show, doctor, native immutability, and uninstall.'
} finally {
    foreach ($name in $savedEnvironment.Keys) {
        if ($null -eq $savedEnvironment[$name]) {
            Remove-Item ("Env:" + $name) -ErrorAction SilentlyContinue
        } else {
            [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process')
        }
    }
    Remove-SafeDirectory $temporaryRoot $temporaryBase
    Remove-SafeDirectory $distDirectory $repoRoot
    $repoStatusAfter = @(& git -C $repoRoot -c core.quotePath=true status --porcelain=v1 --untracked-files=all)
    if ($LASTEXITCODE -ne 0 -or ($repoStatusBefore -join "`n") -cne ($repoStatusAfter -join "`n")) {
        throw 'Clean-install test changed the repository worktree'
    }
}
