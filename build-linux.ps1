# Build script for Linux (PowerShell)
# Usage: .\build-linux.ps1

Write-Host "Building for Linux (arm64)..." -ForegroundColor Cyan

# Start timer
$startTime = Get-Date

# Download dependencies first
Write-Host "Downloading dependencies..." -ForegroundColor Yellow
go mod download
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to download dependencies!" -ForegroundColor Red
    exit 1
}

# Tidy modules
Write-Host "Tidying modules..." -ForegroundColor Yellow
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to tidy modules!" -ForegroundColor Red
    exit 1
}

# Set environment variables for Linux ARM64
$env:GOOS = "linux"
$env:GOARCH = "arm64"
$env:CGO_ENABLED = "0"

# Build the application
# Target: ./cmd/server directory where main.go resides
Write-Host "Building application..." -ForegroundColor Yellow
go build -ldflags="-s -w" -o botforge ./cmd/server

# Stop timer and calculate duration
$endTime = Get-Date
$duration = $endTime - $startTime

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful!" -ForegroundColor Green
    Write-Host "Output: botforge" -ForegroundColor Yellow
    
    # Show elapsed time
    Write-Host "Time Elapsed: $($duration.Minutes)m $($duration.Seconds)s $($duration.Milliseconds)ms" -ForegroundColor Cyan
    
    # Show file info
    if (Test-Path "botforge") {
        $fileInfo = Get-Item "botforge"
        $sizeMB = [math]::Round($fileInfo.Length / 1MB, 2)
        Write-Host "File size: $sizeMB MB" -ForegroundColor Yellow
    }
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

# Reset environment variables
Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue