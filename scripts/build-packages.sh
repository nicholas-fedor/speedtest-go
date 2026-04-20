#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

VERSION=${VERSION:-$(git -C "${ROOT_DIR}" describe --tags --always --dirty 2>/dev/null || echo "dev")}
COMMIT=$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Strip leading 'v' and replace hyphens with dots for package version fields.
# Both RPM and DEB require the version to start with a digit; when there is no
# git tag the describe output is a bare commit hash, so prefix it with 0.0.0.
PKG_VERSION=$(echo "${VERSION}" | sed 's/^v//' | tr '-' '.')
if [[ ! "${PKG_VERSION}" =~ ^[0-9] ]]; then
    PKG_VERSION="0.0.0.${PKG_VERSION}"
fi

OUTPUT_DIR="${ROOT_DIR}/dist"
mkdir -p "${OUTPUT_DIR}"

BUILD_ARGS=(
    --build-arg "VERSION=${PKG_VERSION}"
    --build-arg "COMMIT=${COMMIT}"
    --build-arg "DATE=${DATE}"
)

build_rpm() {
    echo "==> Building RPM package (CentOS 7) — version ${PKG_VERSION}"
    DOCKER_BUILDKIT=1 docker build \
        "${BUILD_ARGS[@]}" \
        --output "type=local,dest=${OUTPUT_DIR}" \
        -f "${SCRIPT_DIR}/rpm/Dockerfile" \
        "${ROOT_DIR}"
    echo "    RPM written to ${OUTPUT_DIR}"
}

build_deb() {
    echo "==> Building DEB package (Ubuntu 22.04) — version ${PKG_VERSION}"
    DOCKER_BUILDKIT=1 docker build \
        "${BUILD_ARGS[@]}" \
        --output "type=local,dest=${OUTPUT_DIR}" \
        -f "${SCRIPT_DIR}/deb/Dockerfile" \
        "${ROOT_DIR}"
    echo "    DEB written to ${OUTPUT_DIR}"
}

case "${1:-all}" in
    --rpm) build_rpm ;;
    --deb) build_deb ;;
    all)   build_rpm; build_deb ;;
    *)
        echo "Usage: $0 [--rpm | --deb | all]" >&2
        exit 1
        ;;
esac

echo ""
echo "Packages in ${OUTPUT_DIR}:"
ls -lh "${OUTPUT_DIR}"/*.rpm "${OUTPUT_DIR}"/*.deb 2>/dev/null || true
