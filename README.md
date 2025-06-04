# Hermes Peer Score Tool

A standalone tool for analyzing Ethereum network peer connection health and performance using Hermes as a gossipsub listener for beacon nodes.

## Prerequisites

The tool automatically downloads and builds the latest Hermes binary during CI runs, or you can build it locally:

```bash
git clone https://github.com/ethpandaops/hermes.git
cd hermes
go build -o hermes ./cmd/hermes
```

## 🛠️ Installation

```bash
git clone https://github.com/ethpandaops/hermes-peer-score
cd hermes-peer-score
go mod tidy
go build -o peer-score-tool
```

## Usage

### Basic Usage
```bash
./peer-score-tool \
  --prysm-host="username:password@prysm.example.com" \
  --duration=5m \
  --output=report.json
```

### Local Testing
```bash
./peer-score-tool \
  --prysm-host="localhost" \
  --prysm-http-port=5052 \
  --prysm-grpc-port=4000 \
  --duration=2m
```

### Command Line Options
- `--prysm-host` (required): Prysm beacon node connection string
- `--prysm-http-port` (default: 443): Prysm HTTP API port  
- `--prysm-grpc-port` (default: 443): Prysm gRPC API port
- `--duration` (default: 2m): Test duration
- `--output` (default: peer-score-report.json): Output file path

**Note**: TLS is automatically enabled when either port is 443.

## Report

```json
{
  "timestamp": "2025-06-04T11:49:34Z",
  "overall_score": 85.2,
  "total_connections": 25,
  "successful_handshakes": 22,
  "unique_clients": 4,
  "goodbye_messages": 3,
  "goodbye_reasons": {
    "client has too many peers": 2,
    "peer score too low": 1
  },
  "peers_by_client": {
    "lighthouse": 10,
    "prysm": 8,
    "nimbus": 3,
    "teku": 1
  },
  "summary": "Score: 85.2% | Connections: 25 | Handshakes: 22 | Clients: 4 | Goodbyes: 3"
}
```

### HTML Report

View live example: [GitHub Pages Report](https://ethpandaops.github.io/hermes-peer-score/)

## Score Calculation

The overall score is calculated as:

1. **Connection Score**: `(successful handshakes / total connections) × 100`
2. **Diversity Score**: `min(unique clients, 4) / 4 × 100` (max score for 4+ clients)
3. **Goodbye Penalty**: `ERROR-level goodbye count × 5 points`
4. **Final Score**: `max(0, ((Connection Score + Diversity Score) / 2) - Goodbye Penalty)`

### Score Classification
- **90-100%**: Excellent
- **80-89%**: Good  
- **60-79%**: Fair
- **40-59%**: Poor
- **0-39%**: Critical

