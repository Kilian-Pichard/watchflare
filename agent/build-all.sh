#!/bin/bash
set -e

# Watchflare Agent - Multi-architecture Build Script
# Builds agent binaries for multiple OS/architecture combinations

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Version (can be overridden with VERSION env var)
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Output directory
OUTPUT_DIR="./dist"

echo -e "${GREEN}Building Watchflare Agent v${VERSION}${NC}"
echo "Build time: ${BUILD_TIME}"
echo ""

# Clean and create output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Build matrix: OS/ARCH combinations
declare -a BUILDS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# Checksum file
CHECKSUM_FILE="${OUTPUT_DIR}/watchflare_checksums.txt"
echo "# Watchflare Agent Checksums" > "$CHECKSUM_FILE"
echo "# Generated: $(date -u)" >> "$CHECKSUM_FILE"
echo "" >> "$CHECKSUM_FILE"

# Build function
build() {
    local os=$1
    local arch=$2
    local platform="${os}_${arch}"
    local platform_dir="${OUTPUT_DIR}/${platform}"
    local binary_name="watchflare-agent"

    if [ "$os" = "windows" ]; then
        binary_name="${binary_name}.exe"
    fi

    echo -e "${YELLOW}Building ${os}/${arch}...${NC}"

    # Create platform directory
    mkdir -p "$platform_dir"

    # Build binary
    GOOS=$os GOARCH=$arch go build \
        -ldflags="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'" \
        -o "${platform_dir}/${binary_name}"

    # Generate checksum and append to main file
    echo "# ${os} ${arch}" >> "$CHECKSUM_FILE"
    if command -v sha256sum >/dev/null 2>&1; then
        (cd "$OUTPUT_DIR" && sha256sum "${platform}/${binary_name}") >> "$CHECKSUM_FILE"
    elif command -v shasum >/dev/null 2>&1; then
        (cd "$OUTPUT_DIR" && shasum -a 256 "${platform}/${binary_name}") >> "$CHECKSUM_FILE"
    fi
    echo "" >> "$CHECKSUM_FILE"

    echo "  → ${platform_dir}/${binary_name}"
}

# Build all targets
for build_target in "${BUILDS[@]}"; do
    IFS='/' read -r os arch <<< "$build_target"
    build "$os" "$arch"
done

echo ""
echo -e "${GREEN}Build complete!${NC}"
echo ""
echo "Structure:"
tree "$OUTPUT_DIR" 2>/dev/null || find "$OUTPUT_DIR" -type f
echo ""
echo "Checksums saved to: ${CHECKSUM_FILE}"
