[CmdletBinding()]
param()

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\\..")).Path
$CacheRoot = Join-Path $RepoRoot ".cache"
$UvCache = Join-Path $CacheRoot "uv"
$XdgCacheHome = Join-Path $CacheRoot "xdg"
$TorchInductorCache = Join-Path $CacheRoot "torchinductor"
$GoBuildCache = Join-Path $CacheRoot "go-build"

foreach ($path in @($CacheRoot, $UvCache, $XdgCacheHome, $TorchInductorCache, $GoBuildCache)) {
    New-Item -ItemType Directory -Force -Path $path | Out-Null
}

$env:UV_CACHE_DIR = $UvCache
$env:XDG_CACHE_HOME = $XdgCacheHome
$env:TORCHINDUCTOR_CACHE_DIR = $TorchInductorCache
$env:GOCACHE = $GoBuildCache
