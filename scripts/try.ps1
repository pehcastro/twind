& {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    $demo = if ($args.Count) { $args[0] } else { 'docs' }
    $rest = @($args | Select-Object -Skip 1)
    if ($demo -notin 'docs', 'landing', 'portfolio', 'playground', 'gallery') {
        throw "twind: no demo named '$demo': docs, landing, portfolio, playground or gallery"
    }
    $machine = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    $arch = @{ AMD64 = 'amd64'; ARM64 = 'arm64' }[$machine]
    if (-not $arch) {
        throw "twind: no Windows build for $machine"
    }
    $asset = "twind_windows_$arch.zip"
    $base = 'https://github.com/pehcastro/twind/releases/latest/download'
    if ($env:TWIND_VERSION) {
        $base = "https://github.com/pehcastro/twind/releases/download/$env:TWIND_VERSION"
    }
    if ($env:TWIND_BASE) {
        $base = $env:TWIND_BASE
    }

    $root = Join-Path $env:LOCALAPPDATA 'twind\try'
    New-Item -ItemType Directory -Force $root | Out-Null
    $work = Join-Path $root ('.tmp-' + [guid]::NewGuid())
    New-Item -ItemType Directory $work | Out-Null
    try {
        Invoke-WebRequest -UseBasicParsing "$base/checksums.txt" -OutFile "$work\checksums.txt"
        $want = $null
        foreach ($line in Get-Content "$work\checksums.txt") {
            $sum, $name = -split $line
            if ($name -eq $asset) { $want = $sum }
        }
        if ($want -notmatch '^[0-9a-f]{64}$') {
            throw "twind: $base/checksums.txt has no sha256 for $asset"
        }
        $dir = Join-Path $root $want
        if (Test-Path "$dir\twind.exe") {
            Write-Host "twind: $asset $want from the cache in $dir"
        } else {
            Write-Host "twind: fetching $base/$asset"
            Invoke-WebRequest -UseBasicParsing "$base/$asset" -OutFile "$work\$asset"
            $got = (Get-FileHash -Algorithm SHA256 "$work\$asset").Hash.ToLower()
            if ($got -ne $want) {
                throw "twind: $asset has sha256 $got, checksums.txt says ${want}: refusing to run it"
            }
            Write-Host "twind: sha256 $got matches checksums.txt"
            Expand-Archive "$work\$asset" "$work\twind"
            Move-Item "$work\twind" $dir
        }
    } finally {
        Remove-Item -Recurse -Force $work
    }

    if ($demo -eq 'docs') {
        & "$dir\twind.exe" docs @rest
    } else {
        & "$dir\twind.exe" try $demo @rest
    }
} @args
