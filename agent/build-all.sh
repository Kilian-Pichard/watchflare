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

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build matrix: OS/ARCH combinations
declare -a BUILDS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# Build function
build() {
    local os=$1
    local arch=$2
    local output_name="watchflare-agent-${os}-${arch}"

    if [ "$os" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo -e "${YELLOW}Building ${os}/${arch}...${NC}"

    GOOS=$os GOARCH=$arch go build \
        -ldflags="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'" \
        -o "${OUTPUT_DIR}/${output_name}"

    # Create checksum
    if command -v sha256sum >/dev/null 2>&1; then
        (cd "$OUTPUT_DIR" && sha256sum "$output_name" > "${output_name}.sha256")
    elif command -v shasum >/dev/null 2>&1; then
        (cd "$OUTPUT_DIR" && shasum -a 256 "$output_name" > "${output_name}.sha256")
    fi

    echo "  → ${OUTPUT_DIR}/${output_name}"
}

# Build all targets
for build_target in "${BUILDS[@]}"; do
    IFS='/' read -r os arch <<< "$build_target"
    build "$os" "$arch"
done

echo ""
echo -e "${GREEN}Build complete!${NC}"
echo ""
echo "Binaries:"
ls -lh "$OUTPUT_DIR" | grep -v ".sha256" | tail -n +2
echo ""
echo "Checksums:"
cat "$OUTPUT_DIR"/*.sha256
