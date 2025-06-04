# Hermes Peer Score Tool

A standalone tool for analyzing peer connection health and network performance using Hermes as a gossipsub listener for Ethereum beacon nodes.

## Features

- **Peer Connection Monitoring**: Tracks successful/failed connections and handshakes
- **Client Diversity Analysis**: Identifies different consensus client implementations (Lighthouse, Prysm, Nimbus, etc.)
- **Goodbye Message Analysis**: Categorizes disconnection reasons by severity (NORMAL, ERROR, CRITICAL)
- **Real-time Error Detection**: Handles connection failures and provides diagnostic information
- **Prometheus Metrics**: Exports metrics for monitoring systems
- **JSON Reports**: Machine-readable results for CI/CD integration

## Prerequisites

You need the Hermes binary to run this tool. Install it from:
```bash
git clone https://github.com/ethpandaops/hermes
cd hermes
go build -o hermes ./cmd/hermes
```

Place the `hermes` binary in your PATH or in the same directory as the peer-score-tool.

## Installation

```bash
git clone https://github.com/ethpandaops/hermes-peer-score
cd hermes-peer-score
go mod tidy
go build -o peer-score-tool
```

## Usage

### Basic Usage (Secure Connection)
```bash
./peer-score-tool \
  --prysm-host="ethpandaops:key@prysm.example.com" \
  --duration=2m \
  --output=report.json
```

### Local Prysm Node (Insecure Connection)
```bash
./peer-score-tool \
  --prysm-host="localhost" \
  --prysm-http-port=5052 \
  --prysm-grpc-port=4000 \
  --duration=5m \
  --output=local-report.json
```

### Command Line Options
- `--prysm-host` (required): Prysm beacon node connection string
- `--prysm-http-port` (default: 443): Prysm HTTP API port  
- `--prysm-grpc-port` (default: 443): Prysm gRPC API port
- `--duration` (default: 2m): Test duration
- `--output` (default: peer-score-report.json): Output file path

**Note**: TLS is automatically enabled when either port is 443, disabled for custom ports.

## Metrics Endpoint

While running, the tool exposes Prometheus metrics at `http://localhost:8080/metrics`:

- `peer_score_connections_total`: Total peer connections
- `peer_score_handshakes_total`: Handshakes by result and client type
- `peer_score_goodbye_total`: Goodbye messages by reason and client
- `peer_score_attestations_total`: Total attestation messages received

## Report Format

The JSON report includes:

```json
{
  "timestamp": "2025-06-04T11:49:34Z",
  "start_time": "2025-06-04T11:47:34Z",
  "end_time": "2025-06-04T11:49:34Z",
  "duration": "2m0.002s",
  "total_connections": 30,
  "successful_handshakes": 30,
  "failed_handshakes": 0,
  "goodbye_messages": 3,
  "goodbye_reasons": {
    "client has too many peers": 3
  },
  "peers_by_client": {
    "lighthouse": 20,
    "prysm": 6,
    "nimbus": 1,
    "lodestar": 2,
    "rust-libp2p": 1
  },
  "unique_clients": 5,
  "overall_score": 100.0,
  "summary": "Score: 100.0% | Connections: 30 | Handshakes: 30 | Clients: 5 | Goodbyes: 3",
  "errors": [],
  "connection_failed": false
}
```

## Score Calculation

The overall score is calculated as:
1. **Connection Score**: `(successful handshakes / total connections) * 100`
2. **Diversity Score**: `min(unique clients, 4) / 4 * 100` (capped at 4 clients)
3. **Goodbye Penalty**: `ERROR goodbye count * 5 points`
4. **Overall Score**: `max(0, ((Connection Score + Diversity Score) / 2) - Goodbye Penalty)`

## GitHub CI Integration

### Example Workflow

```yaml
name: Peer Score Test
on:
  schedule:
    - cron: '0 6 * * *'  # Daily at 6 AM
  workflow_dispatch:

jobs:
  peer-score:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
          
      - name: Build Hermes
        run: |
          git clone https://github.com/ethpandaops/hermes
          cd hermes
          go build -o ../hermes ./cmd/hermes
          
      - name: Build Peer Score Tool
        run: |
          go mod tidy
          go build -o peer-score-tool
          
      - name: Run Peer Score Test
        run: |
          ./peer-score-tool \
            --prysm-host="${{ secrets.PRYSM_HOST }}" \
            --duration=5m \
            --output=peer-score-report.json
            
      - name: Upload Report
        uses: actions/upload-artifact@v3
        with:
          name: peer-score-report
          path: peer-score-report.json
          
      - name: Check Score Threshold
        run: |
          score=$(jq '.overall_score' peer-score-report.json)
          if (( $(echo "$score < 80" | bc -l) )); then
            echo "::error::Peer score below threshold: $score%"
            exit 1
          fi
          echo "::notice::Peer score: $score%"
```

## Testing

Run the test script to verify everything works:

```bash
./test.sh
```

This will test both secure (TLS) and insecure (local) connections.

## Architecture

The tool works by:
1. Starting Hermes as a subprocess with specified configuration
2. Parsing log output in real-time to track peer events
3. Categorizing goodbye messages by severity
4. Generating comprehensive report with scoring
5. Handling connection failures gracefully

## License

MIT License - see the Hermes project for details.