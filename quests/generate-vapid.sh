#!/bin/bash
# VAPID key pair generator for web push notifications
# Outputs env vars ready to paste into .env file
#
# Usage: ./generate-vapid.sh

set -euo pipefail

# Generate private key to a temporary file (required by openssl ec)
# We use mktemp and immediately set up a trap to ensure cleanup
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

PRIVATE_KEY_FILE="$TEMP_DIR/vapid_private.pem"

# Generate the private key
openssl ecparam -name prime256v1 -genkey -noout -out "$PRIVATE_KEY_FILE" 2>/dev/null

# Extract and format the private key (32 bytes, base64url)
# The private key is the last 32 bytes of the DER encoding
VAPID_PRIVATE_KEY=$(openssl ec -in "$PRIVATE_KEY_FILE" -outform DER 2>/dev/null | tail -c 32 | base64 | tr '+/' '-_' | tr -d '=')

# Extract and format the public key (65 bytes uncompressed point, base64url)
# The public key is the last 65 bytes of the DER SubjectPublicKeyInfo
VAPID_PUBLIC_KEY=$(openssl ec -in "$PRIVATE_KEY_FILE" -pubout -outform DER 2>/dev/null | tail -c 65 | base64 | tr '+/' '-_' | tr -d '=')

# Output in .env format
echo "VAPID_PRIVATE_KEY=$VAPID_PRIVATE_KEY"
echo "VAPID_PUBLIC_KEY=$VAPID_PUBLIC_KEY"
echo ""
echo "# Add these to your .env or .env.local file"
