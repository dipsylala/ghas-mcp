# Build script for ghas-mcp
# Produces binaries for Windows, Linux, and macOS (amd64 + arm64)

param(
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"

$module  = "github.com/dipsylala/ghas-mcp"
$ldflags = "-s -w -X main.version=$Version"
$outDir  = "dist"

New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64";  Ext = ".exe" },
    @{ GOOS = "windows"; GOARCH = "arm64";  Ext = ".exe" },
    @{ GOOS = "linux";   GOARCH = "amd64";  Ext = ""     },
    @{ GOOS = "linux";   GOARCH = "arm64";  Ext = ""     },
    @{ GOOS = "darwin";  GOARCH = "amd64";  Ext = ""     },
    @{ GOOS = "darwin";  GOARCH = "arm64";  Ext = ""     }
)

foreach ($t in $targets) {
    $name = "ghas-mcp-$($t.GOOS)-$($t.GOARCH)$($t.Ext)"
    $out  = Join-Path $outDir $name
    Write-Host "Building $name ..."
    $env:GOOS   = $t.GOOS
    $env:GOARCH = $t.GOARCH
    go build -ldflags $ldflags -o $out $module
}

$env:GOOS   = ""
$env:GOARCH = ""
Write-Host "`nDone. Binaries in $outDir/"
