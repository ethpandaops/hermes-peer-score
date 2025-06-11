# Hermes Validation Comparison Implementation Plan

## Overview
> Extend the Hermes peer score tool to support both delegated validation (current approach using Prysm) and independent validation (new smart validation feature branch). This will be achieved through separate CI workflows running at different intervals, allowing side-by-side comparison of peer behavior and validation performance to evaluate the impact of the new in-process validation system.

## Current State Assessment

### Existing Implementation
- **Single Hermes Instance**: Currently runs one instance using `ethpandaops/hermes v0.0.4-0.20250513093811-320c1c3ee6e2`
- **Delegated Validation**: Relies on Prysm for message validation via gRPC
- **CI Integration**: Daily execution via GitHub Actions with 30-minute test duration
- **Redundant Binary Build**: CI builds Hermes binary but uses in-code integration instead
- **Report Generation**: Creates JSON/HTML reports with historical index page
- **GitHub Pages**: Publishes reports with 28-day retention policy

### Current Limitations
- No way to compare validation approaches side-by-side
- Cannot evaluate performance impact of new independent validation
- Single validation mode prevents comparative analysis
- No mechanism to differentiate reports by validation type
- Unnecessary binary build step adds CI overhead

### Dependencies and Constraints
- Must maintain backward compatibility with existing reports
- CI workflow execution time constraints (GitHub Actions limits)
- Go module dependency management for different Hermes versions
- GitHub Pages storage limits for expanded report archive
- Prysm dependency requirements for delegated validation mode

## Goals

1. **Primary Goal**: Enable separate CI workflows for delegated and independent validation approaches for comparative analysis
2. **Performance Comparison**: Measure validation latency, CPU usage, and memory consumption differences between approaches
3. **Behavioral Analysis**: Compare peer connection patterns, disconnect reasons, and score distributions
4. **Report Enhancement**: Generate distinguishable reports for both validation types with clear visual differentiation
5. **Historical Tracking**: Maintain separate historical trends for each validation approach
6. **Workflow Optimization**: Remove redundant binary builds and streamline CI execution

### Non-functional Requirements
- **Performance**: Each workflow should complete within existing CI time limits
- **Reliability**: Independent workflow execution prevents cross-contamination of failures
- **Maintainability**: Clean separation of validation-specific logic and configuration
- **Observability**: Clear logging and report identification for each validation approach

## Design Approach

### Architecture Overview
The solution introduces a **dual-workflow architecture** where separate CI workflows run different validation approaches:

1. **Delegated Validation Workflow**: Uses current version (320c1c3ee6e2) with Prysm delegation
2. **Independent Validation Workflow**: Uses feature branch (b9a99e7517d82cb9db55c9d534a168001d8b8418) with in-process validation
3. **Scheduled Execution**: Workflows run at different intervals (11am vs 12pm) for temporal separation
4. **Report Identification**: Clear validation type labeling in reports and filenames
5. **Unified Index**: Single GitHub Pages index distinguishing between validation approaches

### Component Breakdown

1. **Validation Mode Manager**
   - Purpose: Manages validation-specific configurations and Hermes versions
   - Responsibilities: Mode detection, configuration mapping, dependency resolution
   - Interfaces: StartHermes() enhancement, validation mode parameter handling

2. **Workflow Configuration System**
   - Purpose: Provides separate CI workflow configurations for each validation approach
   - Responsibilities: Schedule management, environment setup, artifact naming
   - Interfaces: GitHub Actions workflows, dependency management, build optimization

3. **Report Identification System**
   - Purpose: Generates reports with clear validation type identification
   - Responsibilities: Filename conventions, metadata tagging, visual differentiation
   - Interfaces: HTML templating, JSON structure, report metadata

4. **Enhanced Index Generator**
   - Purpose: Creates unified historical archive distinguishing between validation types
   - Responsibilities: Validation type categorization, trend analysis, filtering
   - Interfaces: Report metadata, template processing, search functionality

## Implementation Approach

### 1. Validation Mode Configuration System

#### Specific Changes
- Add validation mode parameter to CLI interface
- Implement validation-specific configuration logic
- Update go.mod to support both Hermes versions (conditional based on validation mode)
- Create mode detection and configuration mapping

#### Sample Implementation
```go
// types.go - Add validation mode configuration
type ValidationMode string

const (
    ValidationModeDelegated   ValidationMode = "delegated"   // Uses Prysm via gRPC
    ValidationModeIndependent ValidationMode = "independent" // In-process validation
)

type ValidationConfig struct {
    Mode           ValidationMode
    HermesVersion  string
    PrysmRequired  bool
    ConfigOverrides map[string]interface{}
}

// config.go - Validation-specific configurations
func GetValidationConfigs() map[ValidationMode]ValidationConfig {
    return map[ValidationMode]ValidationConfig{
        ValidationModeDelegated: {
            Mode:          ValidationModeDelegated,
            HermesVersion: "v0.0.4-0.20250513093811-320c1c3ee6e2",
            PrysmRequired: true,
            ConfigOverrides: map[string]interface{}{
                "validation-mode": "delegated",
            },
        },
        ValidationModeIndependent: {
            Mode:          ValidationModeIndependent,
            HermesVersion: "v0.0.4-0.20250513093811-b9a99e7517d82cb9db55c9d534a168001d8b8418",
            PrysmRequired: false,
            ConfigOverrides: map[string]interface{}{
                "validation-mode":                  "independent",
                "validation-attestation-threshold": 10,
                "validation-cache-size":           10000,
                "validation-state-sync-interval":  "30s",
            },
        },
    }
}
```

### 2. Enhanced StartHermes Function

#### Specific Changes
- Modify StartHermes() to support validation mode parameter
- Implement validation-specific configuration application
- Add validation type identification to events and reports
- Support conditional Prysm dependency based on validation mode

#### Sample Implementation
```go
// peer_score_tool.go - Enhanced StartHermes function
func StartHermes(ctx context.Context, config *Config, validationMode ValidationMode, eventHandler func(Event)) error {
    validationConfig := GetValidationConfigs()[validationMode]
    
    // Create validation-specific node configuration
    nodeConfig := createNodeConfig(config, validationConfig)
    
    // Add validation mode identifier to events
    wrappedHandler := func(event Event) {
        event.ValidationMode = string(validationMode)
        event.Timestamp = time.Now()
        eventHandler(event)
    }
    
    // Start Hermes with validation-specific config
    return startHermesNode(ctx, nodeConfig, wrappedHandler)
}

// main.go - CLI parameter for validation mode
func main() {
    var (
        duration       = flag.Duration("duration", 2*time.Minute, "Duration to run the test")
        prysmHost      = flag.String("prysm-host", "", "Prysm host")
        prysmGrpcPort  = flag.Int("prysm-grpc-port", 4000, "Prysm gRPC port")
        prysmHttpPort  = flag.Int("prysm-http-port", 3500, "Prysm HTTP port")
        outputFile     = flag.String("output-file", "peer-score-report.json", "Output file path")
        validationModeFlag = flag.String("validation-mode", "delegated", "Validation mode: delegated or independent")
        htmlOnly       = flag.Bool("html-only", false, "Generate HTML report only")
        openRouterKey  = flag.String("openrouter-key", "", "OpenRouter API key for AI analysis")
    )
    flag.Parse()
    
    // Parse validation mode
    validationMode := ValidationMode(*validationModeFlag)
    if validationMode != ValidationModeDelegated && validationMode != ValidationModeIndependent {
        log.Fatalf("Invalid validation mode: %s. Must be 'delegated' or 'independent'", *validationModeFlag)
    }
    
    // Validate Prysm requirement for delegated mode
    validationConfig := GetValidationConfigs()[validationMode]
    if validationConfig.PrysmRequired && *prysmHost == "" {
        log.Fatalf("Prysm host is required for delegated validation mode")
    }
    
    // Build configuration with validation mode
    config := &Config{
        Duration:       *duration,
        PrysmHost:      *prysmHost,
        PrysmGrpcPort:  *prysmGrpcPort,
        PrysmHttpPort:  *prysmHttpPort,
        OutputFile:     *outputFile,
        ValidationMode: validationMode,
        HTMLOnly:       *htmlOnly,
        OpenRouterKey:  *openRouterKey,
    }
    
    // Start monitoring with specified validation mode
    if err := StartHermes(ctx, config, validationMode, handleEvent); err != nil {
        log.Fatalf("Failed to start Hermes: %v", err)
    }
}
```

### 3. Enhanced Report Generation with Validation Mode Identification

#### Specific Changes
- Extend PeerScoreReport structure to include validation mode metadata
- Add validation-specific filename conventions for easy identification
- Include validation mode in report headers and titles
- Implement validation-specific performance metrics

#### Sample Implementation
```go
// types.go - Enhanced report structure with validation mode
type PeerScoreReport struct {
    Metadata            ReportMetadata              `json:"metadata"`
    ValidationMode      ValidationMode              `json:"validation_mode"`
    ValidationConfig    ValidationConfig            `json:"validation_config"`
    Configuration       Config                      `json:"configuration"`
    Summary             Summary                     `json:"summary"`
    Peers               []PeerInfo                  `json:"peers"`
    Events              []Event                     `json:"events"`
    Statistics          Statistics                  `json:"statistics"`
    AIAnalysis          *AIAnalysis                 `json:"ai_analysis,omitempty"`
}

// report.go - Enhanced report generation with validation mode
func GenerateReport(data *PeerData, validationMode ValidationMode) *PeerScoreReport {
    return &PeerScoreReport{
        Metadata:         generateMetadata(validationMode),
        ValidationMode:   validationMode,
        ValidationConfig: GetValidationConfigs()[validationMode],
        Configuration:    data.Config,
        Summary:          calculateSummary(data),
        Peers:           preparePeerInfo(data),
        Events:          data.Events,
        Statistics:      calculateStatistics(data, validationMode),
        AIAnalysis:      generateAIAnalysis(data, validationMode),
    }
}

// Enhanced filename generation with validation mode
func GenerateReportFilename(validationMode ValidationMode, timestamp time.Time) string {
    return fmt.Sprintf("peer-score-report-%s-%s.json", 
        string(validationMode),
        timestamp.Format("2006-01-02_15-04-05"))
}

func GenerateHTMLFilename(validationMode ValidationMode, timestamp time.Time) string {
    return fmt.Sprintf("peer-score-report-%s-%s.html", 
        string(validationMode),
        timestamp.Format("2006-01-02_15-04-05"))
}
```

### 4. HTML Template Enhancement with Validation Mode Branding

#### Specific Changes
- Add validation mode indicators in report headers and titles
- Create validation-specific color schemes and branding
- Include validation mode metadata in report summary
- Add validation-specific insights and performance metrics

#### Sample Implementation
```html
<!-- html_report.go template enhancement -->
<div class="validation-report-container">
    <!-- Header with Validation Mode Branding -->
    <div class="report-header bg-gradient-to-r {{if eq .ValidationMode "delegated"}}from-blue-600 to-blue-800{{else}}from-green-600 to-green-800{{end}} text-white p-6 rounded-t-lg">
        <div class="flex items-center justify-between">
            <div>
                <h1 class="text-3xl font-bold">Hermes Peer Score Report</h1>
                <div class="flex items-center mt-2">
                    <span class="validation-mode-badge bg-white bg-opacity-20 px-3 py-1 rounded-full text-sm font-medium">
                        {{if eq .ValidationMode "delegated"}}🔗 Delegated Validation{{else}}⚡ Independent Validation{{end}}
                    </span>
                    <span class="ml-4 text-sm opacity-90">
                        Generated: {{.Metadata.Timestamp.Format "2006-01-02 15:04:05 UTC"}}
                    </span>
                </div>
            </div>
            <div class="text-right">
                <div class="text-sm opacity-90">Validation Mode</div>
                <div class="text-xl font-bold">{{.ValidationMode | title}}</div>
            </div>
        </div>
    </div>
    
    <!-- Validation Configuration Summary -->
    <div class="validation-config-summary bg-gray-50 p-4 border-b">
        <h3 class="text-lg font-semibold mb-2">Validation Configuration</h3>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
                <span class="font-medium">Mode:</span> 
                <span class="{{if eq .ValidationMode "delegated"}}text-blue-600{{else}}text-green-600{{end}}">
                    {{.ValidationMode | title}}
                </span>
            </div>
            <div>
                <span class="font-medium">Hermes Version:</span> 
                <span class="font-mono text-xs">{{.ValidationConfig.HermesVersion}}</span>
            </div>
            <div>
                <span class="font-medium">Prysm Required:</span> 
                <span class="{{if .ValidationConfig.PrysmRequired}}text-orange-600{{else}}text-green-600{{end}}">
                    {{if .ValidationConfig.PrysmRequired}}Yes{{else}}No{{end}}
                </span>
            </div>
            <div>
                <span class="font-medium">Duration:</span> 
                <span>{{.Configuration.Duration}}</span>
            </div>
        </div>
    </div>
    
    <!-- Existing report content continues... -->
    <div class="report-content">
        <!-- Summary cards, peer tables, etc. -->
    </div>
</div>

<!-- CSS for validation-specific styling -->
<style>
.validation-mode-delegated {
    --primary-color: #2563eb;
    --primary-light: #dbeafe;
}

.validation-mode-independent {
    --primary-color: #059669;
    --primary-light: #d1fae5;
}
</style>
```

### 5. Separate CI Workflows for Each Validation Mode

#### Specific Changes
- Create separate workflow files for delegated and independent validation
- Remove redundant Hermes binary build step
- Update go.mod conditionally based on validation mode
- Add validation-specific scheduling and artifact naming

#### Sample Implementation
```yaml
# .github/workflows/delegated-validation.yml - New workflow for delegated validation
name: Delegated Validation Report
on:
  schedule:
    - cron: '0 11 * * *'  # Run at 11:00 AM UTC daily
  workflow_dispatch:
    inputs:
      duration:
        description: 'Test duration'
        required: false
        default: '30m'
      clear_history:
        description: 'Clear historical reports'
        type: boolean
        default: false

jobs:
  delegated-validation:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        prysm_host: 
          - 'bn-1-gcp-sfo.ethpandaops.io'
          - 'bn-1-gcp-syd.ethpandaops.io'
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          
      - name: Update go.mod for Delegated Validation
        run: |
          # Ensure using current Hermes version for delegated validation
          go mod edit -replace github.com/probe-lab/hermes=github.com/ethpandaops/hermes@320c1c3ee6e2
          go mod tidy
          
      - name: Build Peer Score Tool
        run: go build -o peer-score-tool .
        
      - name: Run Delegated Validation Test
        run: |
          ./peer-score-tool \
            --duration=${{ github.event.inputs.duration || '30m' }} \
            --prysm-host=${{ matrix.prysm_host }} \
            --prysm-grpc-port=4000 \
            --prysm-http-port=3500 \
            --validation-mode=delegated \
            --output-file=peer-score-report-delegated-$(date +%Y-%m-%d_%H-%M-%S).json \
            --html-only=true \
            --openrouter-key=${{ secrets.OPENROUTER_API_KEY }}

# .github/workflows/independent-validation.yml - New workflow for independent validation  
name: Independent Validation Report
on:
  schedule:
    - cron: '0 12 * * *'  # Run at 12:00 PM UTC daily (1 hour after delegated)
  workflow_dispatch:
    inputs:
      duration:
        description: 'Test duration'
        required: false
        default: '30m'
      
jobs:
  independent-validation:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        prysm_host: 
          - 'bn-1-gcp-sfo.ethpandaops.io'
          - 'bn-1-gcp-syd.ethpandaops.io'
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          
      - name: Update go.mod for Independent Validation
        run: |
          # Use feature branch for independent validation
          go mod edit -replace github.com/probe-lab/hermes=github.com/ethpandaops/hermes@b9a99e7517d82cb9db55c9d534a168001d8b8418
          go mod tidy
          
      - name: Build Peer Score Tool
        run: go build -o peer-score-tool .
        
      - name: Run Independent Validation Test
        run: |
          ./peer-score-tool \
            --duration=${{ github.event.inputs.duration || '30m' }} \
            --validation-mode=independent \
            --output-file=peer-score-report-independent-$(date +%Y-%m-%d_%H-%M-%S).json \
            --html-only=true \
            --openrouter-key=${{ secrets.OPENROUTER_API_KEY }}
```

### 6. Enhanced Index Page with Validation Mode Filtering

#### Specific Changes
- Add validation mode filter/toggle to historical archive
- Create separate sections for delegated and independent validation reports
- Implement validation-specific search and sorting
- Add comparative statistics and trend analysis

#### Sample Implementation
```python
# scripts/generate_index.py enhancements
def generate_enhanced_index():
    reports = collect_reports_with_validation_modes()
    
    delegated_reports = [r for r in reports if r.validation_mode == 'delegated']
    independent_reports = [r for r in reports if r.validation_mode == 'independent']
    
    template_data = {
        'reports': reports,
        'delegated_reports': delegated_reports,
        'independent_reports': independent_reports,
        'comparison_stats': calculate_validation_comparison(delegated_reports, independent_reports),
        'trend_data': generate_validation_trend_analysis(reports),
        'latest_delegated': delegated_reports[0] if delegated_reports else None,
        'latest_independent': independent_reports[0] if independent_reports else None,
    }
    
    return render_template('index-template.html', template_data)

def collect_reports_with_validation_modes():
    reports = []
    for file_path in glob.glob('peer-score-report-*.json'):
        try:
            with open(file_path, 'r') as f:
                report_data = json.load(f)
                
            # Extract validation mode from filename or report metadata
            validation_mode = extract_validation_mode(file_path, report_data)
            
            report_info = {
                'filename': file_path,
                'validation_mode': validation_mode,
                'timestamp': parse_timestamp_from_filename(file_path),
                'summary': report_data.get('summary', {}),
                'statistics': report_data.get('statistics', {}),
                'metadata': report_data.get('metadata', {}),
            }
            reports.append(report_info)
        except Exception as e:
            print(f"Error processing {file_path}: {e}")
            continue
    
    # Sort by timestamp, newest first
    reports.sort(key=lambda x: x['timestamp'], reverse=True)
    return reports

def extract_validation_mode(filename, report_data):
    # Try to extract from filename first
    if '-delegated-' in filename:
        return 'delegated'
    elif '-independent-' in filename:
        return 'independent'
    
    # Fallback to report metadata
    return report_data.get('validation_mode', 'unknown')

def calculate_validation_comparison(delegated_reports, independent_reports):
    if not delegated_reports or not independent_reports:
        return None
        
    return {
        'avg_peers_delegated': np.mean([r['summary'].get('total_peers', 0) for r in delegated_reports[:7]]),
        'avg_peers_independent': np.mean([r['summary'].get('total_peers', 0) for r in independent_reports[:7]]),
        'avg_success_rate_delegated': np.mean([r['statistics'].get('connection_success_rate', 0) for r in delegated_reports[:7]]),
        'avg_success_rate_independent': np.mean([r['statistics'].get('connection_success_rate', 0) for r in independent_reports[:7]]),
        'validation_performance_trend': calculate_performance_trend(delegated_reports, independent_reports),
    }
```

## Testing Strategy

### Unit Testing
- **Version Configuration**: Test version-specific configuration generation
- **Event Handling**: Validate event routing and version tagging
- **Report Generation**: Test comparative report structure and data integrity
- **Error Handling**: Verify graceful failure handling for each instance

### Integration Testing
- **Dual Instance Coordination**: Test parallel execution with resource contention
- **Network Behavior**: Validate both instances receive similar network events
- **Performance Impact**: Measure resource usage with dual execution
- **CI Pipeline**: End-to-end testing of enhanced workflow

### Validation Criteria
- Both instances successfully complete 30-minute test duration
- Generated reports contain valid comparative data
- Historical index correctly categorizes version-specific reports
- Performance overhead remains within acceptable limits (< 2x resource usage)
- No data corruption or event cross-contamination between instances

## Implementation Dependencies

### Phase 1: Core Infrastructure (Days 1-2)
- [ ] Add validation mode configuration system to types.go and config.go
- [ ] Implement validation mode CLI parameter handling in main.go
- [ ] Create validation-specific configuration mapping
- [ ] Add validation mode identification to event and report structures
- Dependencies: None

### Phase 2: Enhanced Tool Functionality (Days 3-4)
- [ ] Update StartHermes() function to support validation mode parameter
- [ ] Implement conditional go.mod dependency handling based on validation mode
- [ ] Add validation mode metadata to all events and reports
- [ ] Create validation-specific filename conventions
- Dependencies: Phase 1 completion

### Phase 3: Enhanced Reporting (Days 5-6)
- [ ] Extend PeerScoreReport structure with validation mode metadata
- [ ] Implement validation-specific HTML template enhancements
- [ ] Add validation mode branding and visual differentiation
- [ ] Create validation-specific performance metrics
- Dependencies: Phase 2 completion

### Phase 4: Separate CI Workflows (Days 7-8)
- [ ] Create delegated-validation.yml workflow file
- [ ] Create independent-validation.yml workflow file
- [ ] Remove redundant Hermes binary build steps
- [ ] Add validation-specific scheduling and artifact naming
- Dependencies: Phase 3 completion

### Phase 5: Index Enhancement (Days 9-10)
- [ ] Update scripts/generate_index.py for validation mode filtering
- [ ] Implement validation-specific report categorization
- [ ] Add comparative statistics and trend analysis
- [ ] Create validation mode filtering and search functionality
- Dependencies: Phase 4 completion, some historical data for testing

## Risks and Considerations

### Implementation Risks
- **Go Module Conflicts**: Dynamic go.mod updates may cause dependency conflicts → Mitigation: Use careful dependency management and testing for each validation mode
- **CI Workflow Complexity**: Separate workflows increase maintenance overhead → Mitigation: Share common steps and use reusable workflow components
- **Report Categorization**: Historical reports may not be properly categorized → Mitigation: Implement fallback detection and manual categorization options
- **Scheduling Conflicts**: Workflows running too close together may cause resource contention → Mitigation: Use appropriate time spacing and resource monitoring

### Performance Considerations
- **CI Runner Resources**: Independent validation may require more resources → Mitigation: Monitor resource usage and optimize configuration parameters
- **Build Time**: Dynamic go.mod updates may increase build time → Mitigation: Cache dependencies and optimize build process
- **Storage Usage**: Separate reports double storage requirements → Mitigation: Maintain existing 28-day retention policy

### Security Considerations
- **Dependency Security**: Feature branch may introduce security vulnerabilities → Mitigation: Regular security scanning and controlled rollout
- **Credential Management**: Separate workflows need proper secret management → Mitigation: Use shared secrets and proper access controls

## Expected Outcomes

### Concrete Measurable Outcomes
- **Separate Workflow Success**: Both validation workflows complete 100% of test runs within 30-minute window
- **Report Generation**: Generate validation-specific reports with clear identification and < 5% data loss
- **Performance Baseline**: Establish baseline metrics for each validation approach
- **Historical Tracking**: Maintain 28-day archive with validation mode categorization

### Success Metrics
- **Execution Reliability**: > 95% successful execution rate for each validation workflow
- **Report Identification**: 100% accurate validation mode identification in reports and index
- **Data Quality**: > 99% accurate validation mode metadata in all reports
- **User Experience**: Clear visual differentiation enables effective comparison analysis
- **CI Optimization**: Removal of redundant binary builds improves workflow efficiency

### Long-term Benefits
- **Validation Strategy**: Informed decision-making on independent validation adoption based on comparative data
- **Performance Analysis**: Clear understanding of performance differences between validation approaches
- **Network Insights**: Better understanding of peer behavior under different validation modes
- **Development Velocity**: Faster evaluation of Hermes improvements with separate validation tracking
- **Resource Optimization**: More efficient CI workflows without redundant build steps