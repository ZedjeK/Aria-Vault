# Aria Vaultwarden, étape 1 : compile le moteur en console (aria-moteur.exe).
# Vaultwarden et son interface web sont pris dans un dossier déjà compilé,
# par exemple ton installation actuelle ou l'artefact de vaultwarden_windows_build.
#
# Usage (PowerShell normal, pas besoin d'admin) :
#   powershell -ExecutionPolicy Bypass -File .\build.ps1
#   powershell -ExecutionPolicy Bypass -File .\build.ps1 -Vaultwarden D:\vaultwarden-1.37.3

param([string] $Vaultwarden = 'C:\Vaultwarden')

$ErrorActionPreference = 'Stop'

$Root   = $PSScriptRoot
$Src    = Join-Path $Root 'src'
$Assets = Join-Path $Src 'internal\vaultwarden\assets'

function Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }

function Invoke-Native {
    param([string]$Exe, [string[]]$Arguments)
    & $Exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Exe a échoué (code $LASTEXITCODE)" }
}

Step 'Vérification des outils'
if (-not (Get-Command 'go' -ErrorAction SilentlyContinue)) { throw 'go introuvable. Installe Go 1.26 ou plus : https://go.dev/dl/' }
if (-not (Get-Command 'tar.exe' -ErrorAction SilentlyContinue)) { throw 'tar.exe introuvable (fourni avec Windows 10 et 11).' }
Write-Host ("  " + (go version))

Step "Vaultwarden à embarquer depuis $Vaultwarden"
$exe = Join-Path $Vaultwarden 'vaultwarden.exe'
$web = Join-Path $Vaultwarden 'web-vault'
if (-not (Test-Path $exe)) { throw "$exe introuvable. Indique le bon dossier avec -Vaultwarden." }
if (-not (Test-Path (Join-Path $web 'index.html'))) { throw "$web\index.html introuvable : l'interface web doit être à côté de vaultwarden.exe." }
Write-Host ("  vaultwarden.exe SHA-256 : " + (Get-FileHash $exe -Algorithm SHA256).Hash.ToLower())
$buildInfo = Join-Path $Vaultwarden 'BUILDINFO.txt'
if (Test-Path $buildInfo) {
    Get-Content $buildInfo | Where-Object { $_ -match '^(Version|Commit)' } | ForEach-Object { Write-Host "  $_" }
}

New-Item -ItemType Directory -Force -Path $Assets | Out-Null

# vaultwarden.exe -> vaultwarden.exe.gz
$gzPath = Join-Path $Assets 'vaultwarden.exe.gz'
$in  = [IO.File]::OpenRead($exe)
$out = [IO.File]::Create($gzPath)
try {
    $gz = New-Object IO.Compression.GZipStream($out, [IO.Compression.CompressionLevel]::Optimal)
    try { $in.CopyTo($gz) } finally { $gz.Dispose() }
} finally {
    $in.Dispose()
    $out.Dispose()
}
Write-Host "  vaultwarden.exe.gz prêt"

# web-vault\ -> web-vault.tar.gz (dossier racine « web-vault »)
Invoke-Native 'tar.exe' @('-czf', (Join-Path $Assets 'web-vault.tar.gz'), '-C', $Vaultwarden, 'web-vault')
Write-Host "  web-vault.tar.gz prêt"

Step 'Compilation du moteur'
Push-Location $Src
try {
    Invoke-Native 'go' @('build', '-trimpath', '-ldflags', '-s -w', '-o', (Join-Path $Root 'aria-moteur.exe'), './cmd/aria-moteur')
} finally {
    Pop-Location
}

Step 'Terminé'
Write-Host "  Moteur prêt : $(Join-Path $Root 'aria-moteur.exe')" -ForegroundColor Green
Write-Host '  Première fois : .\aria-moteur.exe configurer'
Write-Host '  Puis          : .\aria-moteur.exe'
