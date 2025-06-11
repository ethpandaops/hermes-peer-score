# Hermes Peer Score Tool

A standalone Go CLI tool for analyzing Ethereum network peer connection health and performance using Hermes as a gossipsub listener for beacon nodes. This tool monitors peer connections, analyzes network behavior, and generates comprehensive reports with optional AI-powered insights.

## Features

- **Dual Validation Modes**: Support for delegated validation (via Prysm) and independent validation (internal processing)
- **Real-time Monitoring**: Live peer discovery and connection tracking with detailed session analysis
- **Comprehensive Reporting**: Generates both JSON and HTML reports with interactive visualizations
- **AI-Powered Analysis**: Optional OpenRouter integration for intelligent network behavior insights
- **Historical Tracking**: Automated report archival with 28-day retention and GitHub Pages deployment
- **Client Detection**: Automatic identification of Ethereum client types and versions
- **Event Analytics**: Detailed tracking of peer events, handshakes, and connection patterns

## Installation

### Prerequisites

- Go 1.21 or later
- Access to an Ethereum beacon node (Prysm required for both validation modes)

### Building

```bash
git clone https://github.com/ethpandaops/hermes-peer-score.git
cd hermes-peer-score
go build -o peer-score-tool
```

## Usage

### Basic Usage

```bash
# Delegated validation (default)
./peer-score-tool --validation-mode=delegated --prysm-host=<host> --duration=30m

# Independent validation
./peer-score-tool --validation-mode=independent --prysm-host=<host> --duration=30m
```

### Command Line Options

```
--validation-mode string     Validation mode: 'delegated' or 'independent' (default "delegated")
--prysm-host string          Prysm host connection string (required for both modes)
--prysm-http-port int        Prysm HTTP port (default 443)
--prysm-grpc-port int        Prysm gRPC port (default 443)
--duration duration          Test duration for peer scoring (default 2m)
--html-only                  Generate HTML report from existing JSON without running test
--input-json string          Input JSON file for HTML-only mode (default "peer-score-report.json")
--openrouter-api-key string  OpenRouter API key for AI analysis
--skip-ai                    Skip AI analysis even if API key is available
--update-go-mod              Update go.mod for specified validation mode and exit
--validate-go-mod            Validate go.mod configuration for specified validation mode and exit
```

### Environment Variables

```bash
export OPENROUTER_API_KEY="your-api-key"  # For AI-powered analysis
```

## Validation Modes

### Delegated Validation
- Delegates validation requests to Prysm for processing
- Requires Prysm connection for both beacon state access and validation logic
- Lower resource usage on the tool itself
- Uses Hermes version: `v0.0.4-0.20250513093811-320c1c3ee6e2`

### Independent Validation
- Uses Prysm only for beacon state and validator data access
- Performs validation logic internally within the tool
- Higher resource usage but more control over validation process
- Uses Hermes version: `v0.0.4-0.20250611164742-0abea7d82cb4`

## Go Module Management

The tool requires different Hermes versions for each validation mode. Use the built-in commands to manage dependencies:

```bash
# Update go.mod for delegated mode
./peer-score-tool --validation-mode=delegated --update-go-mod

# Update go.mod for independent mode
./peer-score-tool --validation-mode=independent --update-go-mod

# Validate current configuration
./peer-score-tool --validation-mode=delegated --validate-go-mod
```

## Report Generation

### Output Files

The tool generates timestamped files to prevent conflicts:

- `peer-score-report-<mode>-<timestamp>.json` - Raw data in JSON format
- `peer-score-report-<mode>-<timestamp>.html` - Interactive HTML report
- `peer-score-report-<mode>-<timestamp>-data.js` - JavaScript data for HTML report

### HTML-Only Mode

Generate HTML reports from existing JSON data:

```bash
./peer-score-tool --html-only --input-json=peer-score-report-delegated-2024-01-15_14-30-00.json
```

## CI/CD Integration

### GitHub Actions Workflows

The project includes automated CI workflows:

- **ci-delegated.yml**: Daily delegated validation tests at 11 AM UTC
- **ci-independent.yml**: Daily independent validation tests at 12 PM UTC
- **clear-reports.yml**: Manual workflow for clearing historical reports

### GitHub Pages Deployment

Reports are automatically deployed to GitHub Pages with:
- Interactive historical report browser
- Validation mode filtering
- 28-day automatic retention
- Search and sorting capabilities

## Configuration Examples

### Local Development
```bash
./peer-score-tool \
  --validation-mode=delegated \
  --prysm-host=localhost \
  --prysm-http-port=3500 \
  --prysm-grpc-port=4000 \
  --duration=5m
```

### Production Monitoring
```bash
./peer-score-tool \
  --validation-mode=independent \
  --prysm-host=beacon.example.com \
  --duration=1h \
  --openrouter-api-key=$OPENROUTER_API_KEY
```

## Report Analysis

### Key Metrics Tracked

- **Connection Statistics**: Total connections, successful/failed handshakes, success rates
- **Peer Discovery**: Unique peers, client type distribution, geographic diversity
- **Event Analytics**: Peer events by type, connection session details, timing analysis
- **Network Health**: Connection stability, handshake patterns, client version spread

### AI Analysis Features

When OpenRouter API key is provided:
- Intelligent pattern recognition in peer behavior
- Anomaly detection in connection patterns
- Network health insights and recommendations
- Trend analysis across historical data

## Architecture

### Core Components

- **PeerScoreTool**: Main orchestration and peer management
- **ValidationConfig**: Mode-specific configuration and Hermes version management
- **Report Generation**: JSON/HTML output with optional AI enhancement
- **Event Processing**: Real-time peer event capture and analysis

### Dependencies

- **Hermes**: Gossipsub listener and peer discovery (version varies by mode)
- **Prysm**: Beacon chain data access and validation services
- **OpenRouter**: Optional AI analysis integration
- **Logrus**: Structured logging throughout the application

## Development

### Project Structure
```
├── main.go              # CLI entry point and configuration
├── peer_score_tool.go   # Core peer scoring logic
├── config.go            # Configuration management and validation
├── types.go             # Data structures and type definitions
├── report.go            # Report generation and file I/O
├── html_report.go       # HTML template rendering
├── events.go            # Event processing and analysis
├── analyze.go           # AI integration and analysis
└── scripts/             # Python utilities for report management
    ├── generate_index.py    # Historical report index generation
    └── download_reports.py  # Report synchronization utilities
```

### Testing

The CI workflows provide comprehensive testing across both validation modes with real network conditions.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Ensure both validation modes work correctly
4. Test report generation with and without AI analysis
5. Submit a pull request

## License

This project is part of the Ethereum ecosystem tooling maintained by EthPandaOps.
