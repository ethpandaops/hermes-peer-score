#!/usr/bin/env bash
set -euo pipefail

# Simple script to run hermes-peer-score against the devnet

ENCLAVE_NAME="${ENCLAVE_NAME:-hermes-devnet}"
DURATION="${DURATION:-5m}"
VALIDATION_MODE="${VALIDATION_MODE:-delegated}"

echo "[INFO] Starting run-hermes-peer-score.sh script..."
echo "[INFO] Enclave name: $ENCLAVE_NAME"
echo "[INFO] Duration: $DURATION"
echo "[INFO] Validation mode: $VALIDATION_MODE"

# Check if enclave exists
echo "[INFO] Checking if enclave exists..."
if ! kurtosis enclave inspect $ENCLAVE_NAME >/dev/null 2>&1; then
    echo "[INFO] Enclave '$ENCLAVE_NAME' not found. Starting network..."
    
    # Get the directory where this script is located
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
    
    # Run spin-up-network.sh with matrix config
    if [ -f "$SCRIPT_DIR/spin-up-network.sh" ]; then
        echo "[INFO] Running spin-up-network.sh with matrix config..."
        if ! "$SCRIPT_DIR/spin-up-network.sh" matrix; then
            echo "[ERROR] Failed to spin up network"
            exit 1
        fi
        echo "[INFO] Network started successfully!"
        
        # Give it a moment to stabilize
        echo "[INFO] Waiting for network to stabilize..."
        sleep 5
    else
        echo "[ERROR] spin-up-network.sh not found in $SCRIPT_DIR"
        exit 1
    fi
else
    echo "[INFO] Enclave found!"
fi

# Get Apache URL (required for devnet)
echo "[INFO] Looking for Apache service..."
APACHE_INFO=$(kurtosis enclave inspect $ENCLAVE_NAME | grep -A3 "apache" | grep "http.*->" | head -1)
if [ -z "$APACHE_INFO" ]; then
    echo "[ERROR] Apache service not found in enclave (required for devnet)"
    exit 1
else
    APACHE_PORT=$(echo "$APACHE_INFO" | awk -F'-> ' '{print $2}' | awk '{print $1}')
    APACHE_URL="${APACHE_PORT}"
    echo "[INFO] Apache URL: $APACHE_URL"
    
    # Export for the tool to use if not passed via flag
    export DEVNET_APACHE_URL="$APACHE_URL"
fi

# Get Prysm ports
echo "[INFO] Getting Prysm service information..."
# Get the Prysm beacon service info specifically
PRYSM_INFO=$(kurtosis enclave inspect $ENCLAVE_NAME | grep -A5 "cl-.*prysm" | grep -E "(rpc:|http:)" | grep -v "metrics")
echo "[DEBUG] Prysm info raw:"
echo "$PRYSM_INFO"

PRYSM_GRPC=$(echo "$PRYSM_INFO" | grep "rpc:" | head -1 | awk -F'-> ' '{print $2}' | cut -d: -f2 | awk '{print $1}')
PRYSM_HTTP=$(echo "$PRYSM_INFO" | grep "http:" | head -1 | awk -F'-> ' '{print $2}' | sed 's/http:\/\///' | cut -d: -f2 | awk '{print $1}')

echo "[INFO] Prysm gRPC port: $PRYSM_GRPC"
echo "[INFO] Prysm HTTP port: $PRYSM_HTTP"

echo ""
echo "=== Connecting to devnet ==="
echo "  Prysm gRPC: 127.0.0.1:$PRYSM_GRPC"
echo "  Prysm HTTP: 127.0.0.1:$PRYSM_HTTP"
echo "  Duration: $DURATION"
echo "  Validation Mode: $VALIDATION_MODE"
echo ""

# First, update go.mod for the specified validation mode
echo "[INFO] Updating go.mod for $VALIDATION_MODE validation mode..."
if ! go run . --validation-mode=$VALIDATION_MODE --update-go-mod; then
    echo "[ERROR] Failed to update go.mod"
    exit 1
fi

# Run go mod tidy to ensure dependencies are resolved
echo "[INFO] Running go mod tidy..."
if ! go mod tidy; then
    echo "[ERROR] Failed to tidy go.mod"
    exit 1
fi

# Now build the peer-score tool with the updated dependencies
echo "[INFO] Building hermes-peer-score tool with updated dependencies..."

# Show which Hermes version we're using
echo "[INFO] Checking Hermes version from go.mod..."
HERMES_VERSION=$(grep -E "replace.*hermes.*=>" go.mod | grep -v "//" | head -1)
echo "[INFO] Using Hermes: $HERMES_VERSION"

if ! go build -o peer-score-tool .; then
    echo "[ERROR] Failed to build peer-score tool"
    exit 1
fi
echo "[INFO] Build successful!"

# Run hermes-peer-score
echo "[INFO] Starting hermes-peer-score..."
echo "[DEBUG] Command: ./peer-score-tool --prysm-host=127.0.0.1 --prysm-grpc-port=${PRYSM_GRPC} --prysm-http-port=${PRYSM_HTTP} --duration=${DURATION} --validation-mode=${VALIDATION_MODE} --network=devnet --devnet-apache-url=${APACHE_URL}"

go run . \
  --prysm-host=127.0.0.1 \
  --prysm-grpc-port=${PRYSM_GRPC} \
  --prysm-http-port=${PRYSM_HTTP} \
  --duration=${DURATION} \
  --validation-mode=${VALIDATION_MODE} \
  --network=devnet \
  --devnet-apache-url="${APACHE_URL}" \
  "$@"