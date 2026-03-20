#!/usr/bin/env bash
set -euo pipefail

REPO_SLUG="${GOCHAT_DEPLOYER_REPO:-FlameInTheDark/gochat-deployment}"
DEPLOYER_VERSION="${GOCHAT_DEPLOYER_VERSION:-latest}"
USE_RELEASE="${GOCHAT_DEPLOYER_USE_RELEASE:-0}"

SCRIPT_SOURCE="${BASH_SOURCE[0]:-}"
if [[ -n "$SCRIPT_SOURCE" && "$SCRIPT_SOURCE" != "-" ]]; then
  SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_SOURCE")" && pwd)"
else
  SCRIPT_DIR="$PWD"
fi

info() {
  printf '[gochat] %s\n' "$*"
}

die() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

truthy() {
  case "${1,,}" in
    1|true|yes|on) return 0 ;;
    *) return 1 ;;
  esac
}

is_repo_root() {
  local root="$1"
  [[ -f "$root/go.mod" ]] &&
    [[ -f "$root/main.go" ]] &&
    [[ -f "$root/bundle.go" ]] &&
    [[ -d "$root/deployer" ]]
}

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux\n' ;;
    Darwin) printf 'darwin\n' ;;
    MINGW*|MSYS*|CYGWIN*) printf 'windows\n' ;;
    *)
      die "unsupported operating system: $(uname -s)"
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    aarch64|arm64) printf 'arm64\n' ;;
    *)
      die "unsupported architecture: $(uname -m)"
      ;;
  esac
}

binary_name_for_os() {
  local os_name="$1"
  if [[ "$os_name" == "windows" ]]; then
    printf 'gochat-deployer.exe\n'
  else
    printf 'gochat-deployer\n'
  fi
}

download_file() {
  local url="$1"
  local output="$2"

  if command_exists curl; then
    curl -fsSL "$url" -o "$output"
    return
  fi
  if command_exists wget; then
    wget -qO "$output" "$url"
    return
  fi

  die "curl or wget is required to download release assets"
}

build_local_binary() {
  local repo_root="$1"
  local os_name
  local output
  local go_cache
  local go_mod_cache

  os_name="$(detect_os)"
  output="$repo_root/.generated/bin/$(binary_name_for_os "$os_name")"
  go_cache="$repo_root/.generated/go-build"
  go_mod_cache="$repo_root/.generated/gomodcache"

  mkdir -p "$(dirname "$output")"
  mkdir -p "$go_cache" "$go_mod_cache"
  info "Building local deployer binary from $repo_root"
  (
    cd "$repo_root"
    GOCACHE="$go_cache" GOMODCACHE="$go_mod_cache" go build -o "$output" .
  )
  printf '%s\n' "$output"
}

download_release_binary() {
  local os_name arch_name archive_ext binary_name asset_name url
  local cache_root cache_dir binary_path temp_dir archive_path extracted_path

  os_name="$(detect_os)"
  arch_name="$(detect_arch)"
  binary_name="$(binary_name_for_os "$os_name")"

  if [[ "$os_name" == "windows" ]]; then
    archive_ext="zip"
  else
    archive_ext="tar.gz"
  fi

  asset_name="gochat-deployer_${os_name}_${arch_name}.${archive_ext}"
  if [[ "$DEPLOYER_VERSION" == "latest" ]]; then
    url="https://github.com/${REPO_SLUG}/releases/latest/download/${asset_name}"
  else
    url="https://github.com/${REPO_SLUG}/releases/download/${DEPLOYER_VERSION}/${asset_name}"
  fi

  cache_root="${XDG_CACHE_HOME:-$HOME/.cache}/gochat-deployer"
  cache_dir="$cache_root/${DEPLOYER_VERSION}/${os_name}-${arch_name}"
  binary_path="$cache_dir/$binary_name"

  if [[ -f "$binary_path" ]]; then
    chmod +x "$binary_path" 2>/dev/null || true
    printf '%s\n' "$binary_path"
    return
  fi

  mkdir -p "$cache_dir"
  temp_dir="$(mktemp -d)"
  archive_path="$temp_dir/$asset_name"

  info "Downloading deployer release asset $asset_name"
  download_file "$url" "$archive_path"

  if [[ "$archive_ext" == "zip" ]]; then
    command_exists unzip || die "unzip is required to extract $asset_name"
    unzip -qo "$archive_path" -d "$temp_dir"
  else
    command_exists tar || die "tar is required to extract $asset_name"
    tar -xzf "$archive_path" -C "$temp_dir"
  fi

  extracted_path="$temp_dir/$binary_name"
  [[ -f "$extracted_path" ]] || die "release archive did not contain $binary_name"

  mv "$extracted_path" "$binary_path"
  chmod +x "$binary_path" 2>/dev/null || true
  rm -rf "$temp_dir"

  printf '%s\n' "$binary_path"
}

main() {
  local repo_root_candidate binary_path

  repo_root_candidate="$(cd "$SCRIPT_DIR/.." && pwd)"
  if is_repo_root "$repo_root_candidate" && ! truthy "$USE_RELEASE" && command_exists go; then
    binary_path="$(build_local_binary "$repo_root_candidate")"
  else
    binary_path="$(download_release_binary)"
  fi

  exec "$binary_path" "$@"
}

main "$@"
