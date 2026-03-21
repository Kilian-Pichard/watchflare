#!/bin/bash
set -e

# Watchflare Agent - Multi-platform Build Script
# Builds agent binaries for Linux and macOS (amd64 + arm64)

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
OUTPUT_DIR="./dist"

echo -e "${GREEN}Building Watchflare Agent v${VERSION}${NC}"
echo "Build time: ${BUILD_TIME}"
echo ""

rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

BUILDS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

CHECKSUM_FILE="${OUTPUT_DIR}/watchflare_checksums.txt"
echo "# Watchflare Agent Checksums" > "$CHECKSUM_FILE"
echo "# Generated: $(date -u)" >> "$CHECKSUM_FILE"
echo "" >> "$CHECKSUM_FILE"

build() {
    local os=$1
    local arch=$2
    local platform="${os}_${arch}"
    local out="${OUTPUT_DIR}/${platform}/watchflare-agent"

    echo -e "${YELLOW}Building ${os}/${arch}...${NC}"
    mkdir -p "${OUTPUT_DIR}/${platform}"

    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
        -ldflags="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'" \
        -o "$out"

    if command -v sha256sum >/dev/null 2>&1; then
        (cd "$OUTPUT_DIR" && sha256sum "${platform}/watchflare-agent") >> "$CHECKSUM_FILE"
    elif command -v shasum >/dev/null 2>&1; then
        (cd "$OUTPUT_DIR" && shasum -a 256 "${platform}/watchflare-agent") >> "$CHECKSUM_FILE"
    fi

    echo "  → $out"
}

for target in "${BUILDS[@]}"; do
    IFS='/' read -r os arch <<< "$target"
    build "$os" "$arch"
done

echo ""
echo -e "${GREEN}Build complete!${NC}"
echo ""
tree "$OUTPUT_DIR" 2>/dev/null || find "$OUTPUT_DIR" -type f
echo ""
echo "Checksums: ${CHECKSUM_FILE}"
