$ErrorActionPreference = "Stop"

$repo = "mikolajbadyl/clauductor"
$binary = "clauductor.exe"
$installDir = "$env:LOCALAPPDATA\clauductor"

function Write-Step($msg) { Write-Host "`n  $msg" -ForegroundColor White }
function Write-Info($msg) { Write-Host "  $msg" -ForegroundColor DarkGray }
function Write-Ok($msg)   { Write-Host "  $msg" -ForegroundColor Green }

Write-Host "  🚂 Clauductor" -ForegroundColor White
Write-Host "  Your AI needs a conductor" -ForegroundColor DarkGray

# Detect platform
Write-Step "Detecting platform"

$arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Write-Info "32-bit systems are not supported."
    exit 1
}

Write-Info "Platform: windows/$arch"

# Fetch latest version
Write-Step "Fetching latest release"

$release = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
$tag = $release.tag_name
Write-Info "Version: $tag"

# Download and install
Write-Step "Installing"

$archName = if ($arch -eq "amd64") { "x86_64" } else { $arch }
$archive = "clauductor_Windows_$archName.zip"
$url = "https://github.com/$repo/releases/download/$tag/$archive"
$tmp = Join-Path $env:TEMP "clauductor-install"

if (Test-Path $tmp) { Remove-Item $tmp -Recurse -Force }
New-Item -ItemType Directory -Path $tmp | Out-Null

Write-Info "Downloading..."
Invoke-WebRequest -Uri $url -OutFile "$tmp\$archive"

Write-Info "Extracting..."
Expand-Archive -Path "$tmp\$archive" -DestinationPath $tmp -Force

if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}
Copy-Item "$tmp\$binary" "$installDir\$binary" -Force
Remove-Item $tmp -Recurse -Force

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Info "Added to PATH (restart terminal to use)"
}

Write-Ok "Installed to $installDir\$binary"

# MCP setup
Write-Step "MCP Permission Server"

$binPath = "$installDir\$binary"

if (Get-Command claude -ErrorAction SilentlyContinue) {
    Write-Info "Configuring MCP server via Claude CLI..."
    try { claude mcp remove clauductor-mcp -s user 2>$null } catch {}
    claude mcp add --scope user clauductor-mcp -- $binPath --mcp
    Write-Ok "MCP server configured"
} else {
    Write-Info "Claude CLI not found, configuring manually..."
    $configPath = Join-Path $env:USERPROFILE ".claude.json"

    $mcpEntry = @{
        type    = "stdio"
        command = $binPath
        args    = @("--mcp")
    }

    if (Test-Path $configPath) {
        $config = Get-Content $configPath -Raw | ConvertFrom-Json
        if (-not $config.mcpServers) {
            $config | Add-Member -NotePropertyName "mcpServers" -NotePropertyValue @{}
        }
        $config.mcpServers | Add-Member -NotePropertyName "clauductor-mcp" -NotePropertyValue $mcpEntry -Force
        $config | ConvertTo-Json -Depth 10 | Set-Content $configPath -Encoding UTF8
    } else {
        @{ mcpServers = @{ "clauductor-mcp" = $mcpEntry } } | ConvertTo-Json -Depth 10 | Set-Content $configPath -Encoding UTF8
    }

    Write-Ok "Added MCP server to $configPath"
}

# Summary
Write-Host ""
Write-Host "  All aboard!" -ForegroundColor Green
Write-Host ""
Write-Host "  clauductor" -ForegroundColor White -NoNewline
Write-Host "              Start the server"
Write-Host "  clauductor --help" -ForegroundColor White -NoNewline
Write-Host "       Show all options"
Write-Host ""
