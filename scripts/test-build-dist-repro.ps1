[CmdletBinding()]
param(
    [Parameter()]
    [string]$Version = '0.1.0-repro-test'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = [System.IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$temporaryBase = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
$trimSeparators = [char[]]@([System.IO.Path]::DirectorySeparatorChar, [System.IO.Path]::AltDirectorySeparatorChar)
$temporaryPrefix = $temporaryBase.TrimEnd($trimSeparators) + [System.IO.Path]::DirectorySeparatorChar
$temporaryRoot = [System.IO.Path]::GetFullPath((Join-Path $temporaryBase ("clewkeep-repro-" + [Guid]::NewGuid().ToString('N'))))
if (-not $temporaryRoot.StartsWith($temporaryPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing unexpected temporary path: $temporaryRoot"
}
$repoStatusBefore = @(& git -C $repoRoot -c core.quotePath=true status --porcelain=v1 --untracked-files=all)
if ($LASTEXITCODE -ne 0) {
    throw 'git status failed'
}
$documentHashesBefore = @{}
foreach ($document in @('README.md', 'LICENSE')) {
    $documentHashesBefore[$document] = (Get-FileHash -LiteralPath (Join-Path $repoRoot $document) -Algorithm SHA256).Hash
}

function Set-DocumentLineEnding {
    param(
        [Parameter(Mandatory)] [string]$Path,
        [Parameter(Mandatory)] [string]$NewLine
    )

    $content = [System.IO.File]::ReadAllText($Path)
    $content = $content.Replace("`r`n", "`n").Replace("`r", "`n").Replace("`n", $NewLine)
    [System.IO.File]::WriteAllText($Path, $content, [System.Text.UTF8Encoding]::new($false))
}

function Copy-BuildInputs {
    param([Parameter(Mandatory)] [string]$Destination)

    [System.IO.Directory]::CreateDirectory($Destination) | Out-Null
    $relativePaths = & git -C $repoRoot -c core.quotePath=false ls-files -- '*.go' go.mod go.sum README.md LICENSE scripts/build-dist.ps1
    if ($LASTEXITCODE -ne 0) {
        throw 'git ls-files failed'
    }
    foreach ($relativePath in $relativePaths) {
        $source = Join-Path $repoRoot $relativePath
        if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
            continue
        }
        $target = Join-Path $Destination $relativePath
        [System.IO.Directory]::CreateDirectory((Split-Path -Parent $target)) | Out-Null
        Copy-Item -LiteralPath $source -Destination $target
    }
}

function Get-ArtifactHashes {
    param([Parameter(Mandatory)] [string]$DistPath)

    $result = @{}
    foreach ($file in Get-ChildItem -LiteralPath $DistPath -File) {
        $result[$file.Name] = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    }
    return $result
}

function Assert-NormalizedDocument {
    param([Parameter(Mandatory)] [string]$Path)

    $bytes = [System.IO.File]::ReadAllBytes($Path)
    if ($bytes.Length -ge 3 -and $bytes[0] -eq 0xef -and $bytes[1] -eq 0xbb -and $bytes[2] -eq 0xbf) {
        throw "UTF-8 BOM found in normalized document: $Path"
    }
    $strictUtf8 = [System.Text.UTF8Encoding]::new($false, $true)
    $text = $strictUtf8.GetString($bytes)
    if ($text.Contains("`r")) {
        throw "CR found in normalized document: $Path"
    }
}

function Assert-Checksums {
    param([Parameter(Mandatory)] [string]$DistPath)

    $checksumPath = Join-Path $DistPath 'SHA256SUMS'
    Assert-NormalizedDocument $checksumPath
    $checksumText = [System.IO.File]::ReadAllText($checksumPath)
    if (-not $checksumText.EndsWith("`n")) {
        throw 'SHA256SUMS must end with LF'
    }
    $lines = @($checksumText.TrimEnd([char]10).Split([char]10))
    if ($lines.Count -ne 6) {
        throw "Expected six checksum lines, found $($lines.Count)"
    }
    $seen = @{}
    foreach ($line in $lines) {
        if ($line -notmatch '^([0-9a-f]{64})  ([0-9A-Za-z._-]+)$') {
            throw "Malformed checksum line: $line"
        }
        $expected = $Matches[1]
        $name = $Matches[2]
        if ($seen.ContainsKey($name) -or $name -eq 'SHA256SUMS') {
            throw "Duplicate or invalid checksum entry: $name"
        }
        $seen[$name] = $true
        $archivePath = Join-Path $DistPath $name
        if (-not (Test-Path -LiteralPath $archivePath -PathType Leaf)) {
            throw "Checksum refers to missing archive: $name"
        }
        $actual = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $expected) {
            throw "Checksum mismatch: $name"
        }
    }
}

function Assert-PackageContents {
    param(
        [Parameter(Mandatory)] [string]$DistPath,
        [Parameter(Mandatory)] [string]$ExpectedReadme,
        [Parameter(Mandatory)] [string]$ExpectedLicense,
        [Parameter(Mandatory)] [string]$InspectRoot,
        [Parameter(Mandatory)] [string]$ExpectedVersion
    )

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archives = Get-ChildItem -LiteralPath $DistPath -File | Where-Object { $_.Name -ne 'SHA256SUMS' }
    if ($archives.Count -ne 6) {
        throw "Expected six archives, found $($archives.Count)"
    }
    foreach ($archive in $archives) {
        $target = Join-Path $InspectRoot $archive.BaseName
        [System.IO.Directory]::CreateDirectory($target) | Out-Null
        if ($archive.Extension -eq '.zip') {
            [System.IO.Compression.ZipFile]::ExtractToDirectory($archive.FullName, $target)
            $binaryName = 'ctx.exe'
        } else {
            & tar -xzf $archive.FullName -C $target
            if ($LASTEXITCODE -ne 0) {
                throw "tar failed for $($archive.Name)"
            }
            $binaryName = 'ctx'
        }
        $actualEntries = @(Get-ChildItem -LiteralPath $target -Recurse -File | ForEach-Object {
            $_.FullName.Substring($target.Length).TrimStart('\', '/')
        } | Sort-Object)
        $unexpectedDirectories = @(Get-ChildItem -LiteralPath $target -Recurse -Directory)
        if ($unexpectedDirectories.Count -ne 0) {
            throw "Unexpected directories in $($archive.Name)"
        }
        $expectedEntries = @('LICENSE', 'README.md', $binaryName) | Sort-Object
        if (Compare-Object $actualEntries $expectedEntries) {
            throw "Unexpected entries in $($archive.Name): $($actualEntries -join ', ')"
        }
        foreach ($document in @('README.md', 'LICENSE')) {
            $expected = if ($document -eq 'README.md') { $ExpectedReadme } else { $ExpectedLicense }
            $actual = (Get-FileHash -LiteralPath (Join-Path $target $document) -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($actual -ne $expected) {
                throw "$document is not normalized in $($archive.Name)"
            }
            Assert-NormalizedDocument (Join-Path $target $document)
        }
        if ($archive.Name -like '*-windows-amd64.zip') {
            $versionOutput = (& (Join-Path $target $binaryName) version) -join "`n"
            if ($LASTEXITCODE -ne 0 -or $versionOutput -cne "ctx $ExpectedVersion") {
                throw "Embedded version mismatch in $($archive.Name): $versionOutput"
            }
        }
    }
}

try {
    $lfRoot = Join-Path $temporaryRoot 'lf'
    $crlfRoot = Join-Path $temporaryRoot 'crlf'
    Copy-BuildInputs $lfRoot
    Copy-BuildInputs $crlfRoot
    Set-DocumentLineEnding (Join-Path $lfRoot 'README.md') "`n"
    Set-DocumentLineEnding (Join-Path $lfRoot 'LICENSE') "`n"
    Set-DocumentLineEnding (Join-Path $crlfRoot 'README.md') "`r`n"
    Set-DocumentLineEnding (Join-Path $crlfRoot 'LICENSE') "`r`n"

    & (Join-Path $lfRoot 'scripts\build-dist.ps1') -Version $Version
    & (Join-Path $crlfRoot 'scripts\build-dist.ps1') -Version $Version

    $lfHashes = Get-ArtifactHashes (Join-Path $lfRoot 'dist')
    $crlfHashes = Get-ArtifactHashes (Join-Path $crlfRoot 'dist')
    if ($lfHashes.Count -ne 7 -or $crlfHashes.Count -ne 7) {
        throw "Expected seven distribution files in each build"
    }
    foreach ($name in $lfHashes.Keys) {
        if (-not $crlfHashes.ContainsKey($name) -or $lfHashes[$name] -ne $crlfHashes[$name]) {
            throw "LF/CRLF build mismatch: $name"
        }
    }

    Assert-Checksums (Join-Path $lfRoot 'dist')
    Assert-Checksums (Join-Path $crlfRoot 'dist')

    $expectedReadme = (Get-FileHash -LiteralPath (Join-Path $lfRoot 'README.md') -Algorithm SHA256).Hash.ToLowerInvariant()
    $expectedLicense = (Get-FileHash -LiteralPath (Join-Path $lfRoot 'LICENSE') -Algorithm SHA256).Hash.ToLowerInvariant()
    Assert-PackageContents (Join-Path $lfRoot 'dist') $expectedReadme $expectedLicense (Join-Path $temporaryRoot 'inspect') $Version
    $repoStatusAfter = @(& git -C $repoRoot -c core.quotePath=true status --porcelain=v1 --untracked-files=all)
    if ($LASTEXITCODE -ne 0 -or ($repoStatusBefore -join "`n") -cne ($repoStatusAfter -join "`n")) {
        throw 'Reproducibility test changed the repository worktree'
    }
    foreach ($document in @('README.md', 'LICENSE')) {
        $after = (Get-FileHash -LiteralPath (Join-Path $repoRoot $document) -Algorithm SHA256).Hash
        if ($after -ne $documentHashesBefore[$document]) {
            throw "Reproducibility test modified $document"
        }
    }
    Write-Host 'PASS: LF and CRLF document worktrees produced identical normalized archives and checksums.'
} finally {
    if (Test-Path -LiteralPath $temporaryRoot) {
        $resolved = [System.IO.Path]::GetFullPath($temporaryRoot)
        if (-not $resolved.StartsWith($temporaryPrefix, [StringComparison]::OrdinalIgnoreCase) -or $resolved -eq $temporaryBase) {
            throw "Refusing unexpected cleanup path: $resolved"
        }
        $temporaryItem = Get-Item -LiteralPath $resolved -Force
        if (-not $temporaryItem.PSIsContainer -or ($temporaryItem.Attributes -band [System.IO.FileAttributes]::ReparsePoint)) {
            throw "Refusing to recursively clean a non-directory or reparse point: $resolved"
        }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}
