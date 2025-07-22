#!/bin/bash

# Script to generate self-signed certificates for development

set -e

CERT_DIR="certs"
CERT_FILE="$CERT_DIR/server.crt"
KEY_FILE="$CERT_DIR/server.key"
DAYS=365

echo "🔐 Generating self-signed certificates for HTTPS..."

# Create certs directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Check if certificates already exist
if [[ -f "$CERT_FILE" && -f "$KEY_FILE" ]]; then
    echo "⚠️  Certificates already exist. Do you want to regenerate them? (y/N)"
    read -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        echo "✅ Using existing certificates."
        exit 0
    fi
    echo "🔄 Regenerating certificates..."
fi

# Generate private key
echo "📝 Generating private key..."
openssl genrsa -out "$KEY_FILE" 2048

# Generate certificate signing request
echo "📝 Generating certificate signing request..."
openssl req -new -key "$KEY_FILE" -out "$CERT_DIR/server.csr" -subj "/C=US/ST=Development/L=Local/O=PR-Documentator/OU=Development/CN=localhost"

# Generate self-signed certificate
echo "📝 Generating self-signed certificate..."
openssl x509 -req -days $DAYS -in "$CERT_DIR/server.csr" -signkey "$KEY_FILE" -out "$CERT_FILE" \
    -extensions v3_req -extfile <(cat <<EOF
[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF
)

# Clean up CSR file
rm "$CERT_DIR/server.csr"

# Set appropriate permissions
chmod 600 "$KEY_FILE"
chmod 644 "$CERT_FILE"

echo "✅ Certificates generated successfully!"
echo "📁 Certificate: $CERT_FILE"
echo "🔑 Private Key: $KEY_FILE"
echo "⏰ Valid for: $DAYS days"
echo ""
echo "⚠️  Note: These are self-signed certificates for development only."
echo "   Your browser will show a security warning. You can safely ignore it for development."
echo ""
echo "🔧 To trust the certificate in your system (optional):"
echo "   - macOS: sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain $CERT_FILE"
echo "   - Linux: sudo cp $CERT_FILE /usr/local/share/ca-certificates/pr-documentator.crt && sudo update-ca-certificates"