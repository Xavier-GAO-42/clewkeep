[CmdletBinding()]
param(
    [Parameter()]
    [ValidatePattern('^[0-9A-Za-z][0-9A-Za-z._-]*$')]
    [string]$Version = '0.1.0-rc.1'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = [System.IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$distDirectory = [System.IO.Path]::GetFullPath((Join-Path $repoRoot 'dist'))
$expectedDistDirectory = [System.IO.Path]::GetFullPath((Join-Path $repoRoot 'dist'))
if ($distDirectory -ne $expectedDistDirectory -or $distDirectory -eq $repoRoot) {
    throw "Refusing to clean unexpected dist path: $distDirectory"
}

$goCommand = Get-Command go -ErrorAction SilentlyContinue
if ($null -eq $goCommand) {
    $goCandidate = Join-Path ${env:ProgramFiles} 'Go\bin\go.exe'
    if (-not (Test-Path -LiteralPath $goCandidate -PathType Leaf)) {
        throw 'Go was not found on PATH or under Program Files.'
    }
    $goPath = $goCandidate
} else {
    $goPath = $goCommand.Source
}

$fixedArchiveTime = [DateTimeOffset]::new(2000, 1, 1, 0, 0, 0, [TimeSpan]::Zero)
$fixedTarUnixTime = 946684800

function Copy-BuildSources {
    param(
        [Parameter(Mandatory)] [string]$SourceRoot,
        [Parameter(Mandatory)] [string]$DestinationRoot
    )

    $sourceFiles = Get-ChildItem -LiteralPath $SourceRoot -Recurse -File | Where-Object {
        $_.Name -eq 'go.mod' -or $_.Name -eq 'go.sum' -or $_.Extension -eq '.go'
    }
    foreach ($sourceFile in $sourceFiles) {
        $relativePath = $sourceFile.FullName.Substring($SourceRoot.Length).TrimStart('\', '/')
        $destinationPath = Join-Path $DestinationRoot $relativePath
        $destinationParent = Split-Path -Parent $destinationPath
        [System.IO.Directory]::CreateDirectory($destinationParent) | Out-Null
        Copy-Item -LiteralPath $sourceFile.FullName -Destination $destinationPath
    }
}

function Set-BuildVersion {
    param(
        [Parameter(Mandatory)] [string]$MainPath,
        [Parameter(Mandatory)] [string]$BuildVersion
    )

    $source = [System.IO.File]::ReadAllText($MainPath)
    $pattern = 'const version = "[^"]+"'
    if ([regex]::Matches($source, $pattern).Count -ne 1) {
        throw 'Expected exactly one version constant in cmd/ctx/main.go.'
    }
    $updated = [regex]::Replace($source, $pattern, "const version = `"$BuildVersion`"")
    [System.IO.File]::WriteAllText($MainPath, $updated, [System.Text.UTF8Encoding]::new($false))
}

function New-DeterministicZip {
    param(
        [Parameter(Mandatory)] [string]$ArchivePath,
        [Parameter(Mandatory)] [object[]]$Entries
    )

    Add-Type -AssemblyName System.IO.Compression
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archiveStream = [System.IO.File]::Open($ArchivePath, [System.IO.FileMode]::CreateNew)
    try {
        $archive = [System.IO.Compression.ZipArchive]::new(
            $archiveStream,
            [System.IO.Compression.ZipArchiveMode]::Create,
            $true
        )
        try {
            foreach ($entrySpec in $Entries) {
                $entry = $archive.CreateEntry(
                    $entrySpec.Name,
                    [System.IO.Compression.CompressionLevel]::Optimal
                )
                $entry.LastWriteTime = $fixedArchiveTime
                $entryStream = $entry.Open()
                $inputStream = [System.IO.File]::OpenRead($entrySpec.Path)
                try {
                    $inputStream.CopyTo($entryStream)
                } finally {
                    $inputStream.Dispose()
                    $entryStream.Dispose()
                }
            }
        } finally {
            $archive.Dispose()
        }
    } finally {
        $archiveStream.Dispose()
    }
}

function Set-TarText {
    param(
        [Parameter(Mandatory)] [byte[]]$Header,
        [Parameter(Mandatory)] [int]$Offset,
        [Parameter(Mandatory)] [int]$Length,
        [Parameter(Mandatory)] [string]$Value
    )

    $bytes = [System.Text.Encoding]::ASCII.GetBytes($Value)
    if ($bytes.Length -gt $Length) {
        throw "Tar field is too long: $Value"
    }
    [System.Array]::Copy($bytes, 0, $Header, $Offset, $bytes.Length)
}

function Get-TarOctal {
    param(
        [Parameter(Mandatory)] [long]$Value,
        [Parameter(Mandatory)] [int]$Length
    )

    $octal = [Convert]::ToString($Value, 8)
    if ($octal.Length -gt ($Length - 1)) {
        throw "Value does not fit in tar field: $Value"
    }
    return $octal.PadLeft($Length - 1, '0') + [char]0
}

function Write-TarEntry {
    param(
        [Parameter(Mandatory)] [System.IO.Stream]$TarStream,
        [Parameter(Mandatory)] [string]$SourcePath,
        [Parameter(Mandatory)] [string]$EntryName,
        [Parameter(Mandatory)] [int]$Mode
    )

    $length = (Get-Item -LiteralPath $SourcePath).Length
    $header = [byte[]]::new(512)
    Set-TarText $header 0 100 $EntryName
    Set-TarText $header 100 8 (Get-TarOctal $Mode 8)
    Set-TarText $header 108 8 (Get-TarOctal 0 8)
    Set-TarText $header 116 8 (Get-TarOctal 0 8)
    Set-TarText $header 124 12 (Get-TarOctal $length 12)
    Set-TarText $header 136 12 (Get-TarOctal $fixedTarUnixTime 12)
    for ($index = 148; $index -lt 156; $index++) {
        $header[$index] = 32
    }
    $header[156] = [byte][char]'0'
    Set-TarText $header 257 6 ("ustar" + [char]0)
    Set-TarText $header 263 2 '00'

    $checksum = 0
    foreach ($value in $header) {
        $checksum += $value
    }
    $checksumText = [Convert]::ToString($checksum, 8).PadLeft(6, '0') + [char]0 + ' '
    Set-TarText $header 148 8 $checksumText

    $TarStream.Write($header, 0, $header.Length)
    $inputStream = [System.IO.File]::OpenRead($SourcePath)
    try {
        $inputStream.CopyTo($TarStream)
    } finally {
        $inputStream.Dispose()
    }

    $remainder = $length % 512
    if ($remainder -ne 0) {
        $padding = [byte[]]::new([int](512 - $remainder))
        $TarStream.Write($padding, 0, $padding.Length)
    }
}

function New-DeterministicTarGz {
    param(
        [Parameter(Mandatory)] [string]$ArchivePath,
        [Parameter(Mandatory)] [object[]]$Entries
    )

    $tarPath = "$ArchivePath.tar"
    $tarStream = [System.IO.File]::Open($tarPath, [System.IO.FileMode]::CreateNew)
    try {
        foreach ($entrySpec in $Entries) {
            Write-TarEntry $tarStream $entrySpec.Path $entrySpec.Name $entrySpec.Mode
        }
        $endBlocks = [byte[]]::new(1024)
        $tarStream.Write($endBlocks, 0, $endBlocks.Length)
    } finally {
        $tarStream.Dispose()
    }

    try {
        $archiveStream = [System.IO.File]::Open($ArchivePath, [System.IO.FileMode]::CreateNew)
        try {
            $gzipStream = [System.IO.Compression.GZipStream]::new(
                $archiveStream,
                [System.IO.Compression.CompressionLevel]::Optimal,
                $true
            )
            try {
                $tarInput = [System.IO.File]::OpenRead($tarPath)
                try {
                    $tarInput.CopyTo($gzipStream)
                } finally {
                    $tarInput.Dispose()
                }
            } finally {
                $gzipStream.Dispose()
            }
        } finally {
            $archiveStream.Dispose()
        }
    } finally {
        Remove-Item -LiteralPath $tarPath -Force -ErrorAction SilentlyContinue
    }
}

$targets = @(
    [pscustomobject]@{ OS = 'windows'; Arch = 'amd64'; Extension = '.zip' },
    [pscustomobject]@{ OS = 'windows'; Arch = 'arm64'; Extension = '.zip' },
    [pscustomobject]@{ OS = 'darwin'; Arch = 'amd64'; Extension = '.tar.gz' },
    [pscustomobject]@{ OS = 'darwin'; Arch = 'arm64'; Extension = '.tar.gz' },
    [pscustomobject]@{ OS = 'linux'; Arch = 'amd64'; Extension = '.tar.gz' },
    [pscustomobject]@{ OS = 'linux'; Arch = 'arm64'; Extension = '.tar.gz' }
)

$temporaryRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("clewkeep-dist-" + [Guid]::NewGuid().ToString('N'))
$sourceRoot = Join-Path $temporaryRoot 'source'
$stageRoot = Join-Path $temporaryRoot 'stage'

try {
    if (Test-Path -LiteralPath $distDirectory) {
        Remove-Item -LiteralPath $distDirectory -Recurse -Force
    }
    [System.IO.Directory]::CreateDirectory($distDirectory) | Out-Null
    [System.IO.Directory]::CreateDirectory($sourceRoot) | Out-Null
    [System.IO.Directory]::CreateDirectory($stageRoot) | Out-Null

    Copy-BuildSources $repoRoot $sourceRoot
    Set-BuildVersion (Join-Path $sourceRoot 'cmd\ctx\main.go') $Version

    foreach ($target in $targets) {
        $binaryName = if ($target.OS -eq 'windows') { 'ctx.exe' } else { 'ctx' }
        $targetStage = Join-Path $stageRoot ("{0}-{1}" -f $target.OS, $target.Arch)
        [System.IO.Directory]::CreateDirectory($targetStage) | Out-Null
        $binaryPath = Join-Path $targetStage $binaryName

        $previousGoos = $env:GOOS
        $previousGoarch = $env:GOARCH
        $previousCgo = $env:CGO_ENABLED
        try {
            $env:GOOS = $target.OS
            $env:GOARCH = $target.Arch
            $env:CGO_ENABLED = '0'
            Push-Location $sourceRoot
            try {
                & $goPath build -trimpath -buildvcs=false '-ldflags=-s -w -buildid=' -o $binaryPath ./cmd/ctx
                if ($LASTEXITCODE -ne 0) {
                    throw "Go build failed for $($target.OS)/$($target.Arch)."
                }
            } finally {
                Pop-Location
            }
        } finally {
            $env:GOOS = $previousGoos
            $env:GOARCH = $previousGoarch
            $env:CGO_ENABLED = $previousCgo
        }

        $entries = @(
            [pscustomobject]@{ Name = $binaryName; Path = $binaryPath; Mode = 493 },
            [pscustomobject]@{ Name = 'README.md'; Path = (Join-Path $repoRoot 'README.md'); Mode = 420 },
            [pscustomobject]@{ Name = 'LICENSE'; Path = (Join-Path $repoRoot 'LICENSE'); Mode = 420 }
        )
        $archiveName = "clewkeep-$Version-$($target.OS)-$($target.Arch)$($target.Extension)"
        $archivePath = Join-Path $distDirectory $archiveName
        if ($target.Extension -eq '.zip') {
            New-DeterministicZip $archivePath $entries
        } else {
            New-DeterministicTarGz $archivePath $entries
        }
        Write-Host "built $archiveName"
    }

    $archives = Get-ChildItem -LiteralPath $distDirectory -File | Sort-Object Name
    $checksumLines = foreach ($archive in $archives) {
        $hash = (Get-FileHash -LiteralPath $archive.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        "$hash  $($archive.Name)"
    }
    # WriteAllLines joins with Environment.NewLine (CRLF on Windows), which breaks
    # `sha256sum -c` on macOS/Linux. Force LF so the checksum file is portable.
    $checksumContent = ($checksumLines -join "`n") + "`n"
    $checksumPath = Join-Path $distDirectory 'SHA256SUMS'
    [System.IO.File]::WriteAllText($checksumPath, $checksumContent, [System.Text.UTF8Encoding]::new($false))
    Write-Host "wrote SHA256SUMS"
} finally {
    if (Test-Path -LiteralPath $temporaryRoot) {
        Remove-Item -LiteralPath $temporaryRoot -Recurse -Force
    }
}
