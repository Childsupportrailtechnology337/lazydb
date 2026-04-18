#!/bin/sh
# install.sh — Universal installer for LazyDB
# Usage:
#   curl -sSL https://raw.githubusercontent.com/aymenhmaidiwastaken/lazydb/main/install.sh | sh
#   curl -sSL https://raw.githubusercontent.com/aymenhmaidiwastaken/lazydb/main/install.sh | sh -s -- --version v0.3.0

set -e

REPO="aymenhmaidiwastaken/lazydb"
GITHUB_API="https://api.github.com/repos/${REPO}"
GITHUB_RELEASES="https://github.com/${REPO}/releases"

# --- Helpers ----------------------------------------------------------------

log()   { printf "[lazydb] %s\n" "$*"; }
error() { printf "[lazydb] ERROR: %s\n" "$*" >&2; exit 1; }

need_cmd() {
    if ! command -v "$1" > /dev/null 2>&1; then
        error "need '$1' (command not found)"
    fi
}

# --- Parse flags -------------------------------------------------------------

VERSION=""
while [ $# -gt 0 ]; do
    case "$1" in
        --version|-v)
            VERSION="$2"
            shift 2
            ;;
        --version=*)
            VERSION="${1#*=}"
            shift
            ;;
        *)
            error "unknown option: $1"
            ;;
    esac
done

# --- Detect OS and architecture ---------------------------------------------

detect_os() {
    case "$(uname -s)" in
        Linux*)   echo "linux"   ;;
        Darwin*)  echo "darwin"  ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)        error "unsupported OS: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              error "unsupported architecture: $(uname -m)" ;;
    esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

log "Detected OS: ${OS}, Arch: ${ARCH}"

# --- Resolve version ---------------------------------------------------------

need_cmd curl

if [ -z "${VERSION}" ]; then
    log "Fetching latest release..."
    VERSION="$(curl -sSL "${GITHUB_API}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
    if [ -z "${VERSION}" ]; then
        error "could not determine latest version"
    fi
fi

# Ensure version starts with 'v'
case "${VERSION}" in
    v*) ;;
    *)  VERSION="v${VERSION}" ;;
esac

log "Installing LazyDB ${VERSION}..."

# --- Build download URL ------------------------------------------------------

EXT="tar.gz"
if [ "${OS}" = "windows" ]; then
    EXT="zip"
fi

BINARY="lazydb"
if [ "${OS}" = "windows" ]; then
    BINARY="lazydb.exe"
fi

ARCHIVE="lazydb_${VERSION#v}_${OS}_${ARCH}.${EXT}"
DOWNLOAD_URL="${GITHUB_RELEASES}/download/${VERSION}/${ARCHIVE}"

# --- Download and extract ----------------------------------------------------

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

log "Downloading ${DOWNLOAD_URL}..."
curl -sSL -o "${TMPDIR}/${ARCHIVE}" "${DOWNLOAD_URL}" || error "download failed — check that ${VERSION} exists for ${OS}/${ARCH}"

log "Extracting..."
cd "${TMPDIR}"

if [ "${EXT}" = "zip" ]; then
    need_cmd unzip
    unzip -q "${ARCHIVE}"
else
    tar xzf "${ARCHIVE}"
fi

# --- Install binary ----------------------------------------------------------

INSTALL_DIR="/usr/local/bin"
USE_SUDO=""

if [ ! -d "${INSTALL_DIR}" ] || [ ! -w "${INSTALL_DIR}" ]; then
    if command -v sudo > /dev/null 2>&1; then
        USE_SUDO="sudo"
        log "Need sudo to install to ${INSTALL_DIR}"
    else
        INSTALL_DIR="${HOME}/bin"
        mkdir -p "${INSTALL_DIR}"
        log "No sudo available, installing to ${INSTALL_DIR}"
    fi
fi

${USE_SUDO} install -m 755 "${BINARY}" "${INSTALL_DIR}/${BINARY}"

# --- Verify ------------------------------------------------------------------

if command -v lazydb > /dev/null 2>&1; then
    INSTALLED_VERSION="$(lazydb version 2>/dev/null || true)"
    log "Successfully installed: ${INSTALLED_VERSION}"
else
    log "Installed to ${INSTALL_DIR}/${BINARY}"
    if [ "${INSTALL_DIR}" = "${HOME}/bin" ]; then
        log "Make sure ${INSTALL_DIR} is in your PATH:"
        log "  export PATH=\"\${HOME}/bin:\${PATH}\""
    fi
fi

log ""
log "LazyDB ${VERSION} is ready!"
log "Get started:  lazydb ./mydb.sqlite"
log "More info:    https://github.com/${REPO}"
