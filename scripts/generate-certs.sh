#!/bin/bash

# Watchflare Certificate Generation Script
# Generates self-signed CA and server certificates for gRPC TLS

set -e

# Configuration
CERT_DIR="${WATCHFLARE_CERT_DIR:-/etc/watchflare/certs}"
VALIDITY_DAYS=3650  # 10 years for homelab
KEY_SIZE=4096

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Watchflare Certificate Generator"
echo "================================"
echo ""

# Check if OpenSSL is installed
if ! command -v openssl &> /dev/null; then
    echo "Error: OpenSSL is not installed. Please install it first."
    exit 1
fi

# Create cert directory if it doesn't exist
if [ ! -d "$CERT_DIR" ]; then
    echo "Creating certificate directory: $CERT_DIR"
    sudo mkdir -p "$CERT_DIR"
fi

# Prompt for server hostname
read -p "Enter server hostname (e.g., watchflare.local): " SERVER_NAME
if [ -z "$SERVER_NAME" ]; then
    echo "Error: Server hostname cannot be empty"
    exit 1
fi

echo ""
echo "Generating certificates with the following parameters:"
echo "  Certificate directory: $CERT_DIR"
echo "  Server hostname: $SERVER_NAME"
echo "  Validity: $VALIDITY_DAYS days ($((VALIDITY_DAYS/365)) years)"
echo "  Key size: $KEY_SIZE bits"
echo ""

# Generate CA private key
echo -e "${YELLOW}[1/5]${NC} Generating CA private key..."
sudo openssl genrsa -out "$CERT_DIR/ca-key.pem" $KEY_SIZE 2>/dev/null

# Generate CA certificate
echo -e "${YELLOW}[2/5]${NC} Generating CA certificate..."
sudo openssl req -new -x509 -days $VALIDITY_DAYS -key "$CERT_DIR/ca-key.pem" \
    -out "$CERT_DIR/ca-cert.pem" \
    -subj "/C=US/ST=Homelab/L=Homelab/O=Watchflare/OU=CA/CN=Watchflare CA" \
    2>/dev/null

# Generate server private key
echo -e "${YELLOW}[3/5]${NC} Generating server private key..."
sudo openssl genrsa -out "$CERT_DIR/server-key.pem" $KEY_SIZE 2>/dev/null

# Generate server certificate signing request (CSR)
echo -e "${YELLOW}[4/5]${NC} Generating server certificate signing request..."
sudo openssl req -new -key "$CERT_DIR/server-key.pem" \
    -out "$CERT_DIR/server.csr" \
    -subj "/C=US/ST=Homelab/L=Homelab/O=Watchflare/OU=Server/CN=$SERVER_NAME" \
    2>/dev/null

# Sign server certificate with CA
echo -e "${YELLOW}[5/5]${NC} Signing server certificate with CA..."
sudo openssl x509 -req -days $VALIDITY_DAYS \
    -in "$CERT_DIR/server.csr" \
    -CA "$CERT_DIR/ca-cert.pem" \
    -CAkey "$CERT_DIR/ca-key.pem" \
    -CAcreateserial \
    -out "$CERT_DIR/server-cert.pem" \
    -extfile <(printf "subjectAltName=DNS:$SERVER_NAME,DNS:localhost,IP:127.0.0.1") \
    2>/dev/null

# Clean up CSR file
sudo rm "$CERT_DIR/server.csr"

# Set proper permissions
echo ""
echo "Setting file permissions..."
sudo chmod 600 "$CERT_DIR/ca-key.pem" "$CERT_DIR/server-key.pem"
sudo chmod 644 "$CERT_DIR/ca-cert.pem" "$CERT_DIR/server-cert.pem"

echo ""
echo -e "${GREEN}✓ Certificate generation complete!${NC}"
echo ""
echo "Generated files:"
echo "  CA Certificate:     $CERT_DIR/ca-cert.pem (public)"
echo "  CA Private Key:     $CERT_DIR/ca-key.pem (KEEP SECURE!)"
echo "  Server Certificate: $CERT_DIR/server-cert.pem (public)"
echo "  Server Private Key: $CERT_DIR/server-key.pem (KEEP SECURE!)"
echo ""
echo "Next steps:"
echo "  1. Backend: Configure GRPC_CERT_FILE and GRPC_KEY_FILE in .env"
echo "  2. Agents:  Copy $CERT_DIR/ca-cert.pem to each agent server"
echo "  3. Agents:  Configure ca_cert_file and server_name in agent.conf"
echo ""
echo -e "${YELLOW}IMPORTANT:${NC} Keep ca-key.pem secure and backed up!"
echo "            You'll need it if you want to generate more server certificates."
