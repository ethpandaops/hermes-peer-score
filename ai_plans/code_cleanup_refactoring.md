# Hermes Peer Score Tool - Code Cleanup and Refactoring Implementation Plan

## Overview
> This plan outlines a comprehensive refactoring of the Hermes Peer Score Tool to improve code organization, maintainability, and clarity. The refactoring will eliminate code duplication, break down large files, improve separation of concerns, and establish better architectural patterns while preserving all existing functionality.

## Current State Assessment

### Existing Implementation
- Single-package Go CLI tool with 9 Go files in root directory
- `events.go` (993 lines) and `html_report.go` (1320 lines) are extremely large
- Mixed responsibilities throughout files with poor separation of concerns
- Complex functions with high cyclomatic complexity (some >150 lines)
- Manual deep copying and inefficient data operations
- Hardcoded magic numbers and strings throughout
- Embedded 600+ line HTML template in Go code
- Event handling with repetitive patterns and mixed business logic

### Limitations and Issues
- **Large Files**: `events.go` and `html_report.go` violate Single Responsibility Principle
- **Complex Functions**: `GenerateReport()` (171 lines), `StartHermes()` (145 lines)
- **Poor Separation**: CLI logic mixed with business logic in `main.go`
- **Code Duplication**: Similar patterns across event handlers and parsing logic
- **Hardcoded Values**: Magic numbers and strings without constants
- **No Abstractions**: Missing interfaces for extensibility and testing
- **Manual Operations**: Complex manual deep copying and data transformations

### Technical Debt and Constraints
- Must maintain existing CLI interface and functionality
- Must preserve all validation modes and report generation capabilities
- Must conform to `.golangci.yml` linting rules
- Must maintain backward compatibility with existing reports and configurations
- Dependencies on external Hermes and Prysm libraries cannot be changed

### What's Working Well
- Core peer scoring functionality is sound
- Comprehensive event handling covers all necessary cases
- Configuration management supports multiple validation modes
- Report generation produces detailed, useful output
- Good use of structured logging with logrus
- Proper mutex usage for concurrent access

## Goals

1. **Primary Goal**: Refactor codebase into maintainable, well-organized packages with clear separation of concerns
2. **Code Organization**: Break down large files into logical packages and modules
3. **Eliminate Duplication**: Remove repetitive patterns and consolidate common functionality
4. **Improve Testability**: Create interfaces and inject dependencies for better unit testing
5. **Performance Optimization**: Replace manual operations with more efficient implementations
6. **Code Quality**: Add constants, improve naming, and enhance documentation
7. **Template Management**: Extract HTML templates to separate files with proper template engine usage
8. **Error Handling**: Standardize error handling patterns with proper context and wrapping

## Refactoring Design Approach

### Architecture Overview
The refactored codebase will use a layered architecture with clear package boundaries:
- **CLI Layer**: Command-line interface and application orchestration
- **Core Layer**: Business logic for peer scoring and analysis  
- **Event Layer**: Event handling, parsing, and processing
- **Report Layer**: Report generation with template management
- **Config Layer**: Configuration management and validation
- **Internal Package Structure**: Organized by domain responsibility

### Component Breakdown

1. **CLI Package (`cmd/`)**
   - Purpose: Application entry point and command-line interface
   - Responsibilities: Argument parsing, application orchestration, graceful shutdown
   - Interfaces: Clean separation from business logic

2. **Core Package (`internal/core/`)**
   - Purpose: Main business logic and tool orchestration
   - Responsibilities: Peer score tool management, report coordination
   - Interfaces: Well-defined interfaces for extensibility

3. **Events Package (`internal/events/`)**
   - Purpose: Event handling and processing system
   - Responsibilities: Event routing, payload parsing, state management
   - Interfaces: Handler interfaces for different event types

4. **Reports Package (`internal/reports/`)**
   - Purpose: Report generation and template management
   - Responsibilities: JSON/HTML generation, template rendering, file operations
   - Interfaces: Template management and data serialization

5. **Config Package (`internal/config/`)**
   - Purpose: Configuration management and validation
   - Responsibilities: Setting defaults, validation, mode management
   - Interfaces: Configuration interfaces for different validation modes

6. **Peer Package (`internal/peer/`)**
   - Purpose: Peer state and statistics management
   - Responsibilities: Session tracking, statistics calculation, data operations
   - Interfaces: Repository pattern for peer data access

## Implementation Approach

### 1. **Setup and Preparation Phase**

#### Create Package Structure
- Create `internal/` directory with package subdirectories
- Create `templates/` directory for HTML/CSS/JS files
- Create `constants/` package for hardcoded values
- Set up interfaces and base types for each package

#### Extract Constants and Eliminate Magic Numbers
```go
// internal/constants/config.go
package constants

const (
    DefaultStatusReportInterval  = 15 * time.Second
    DefaultPeerScoreFreq        = 5 * time.Second
    DefaultPubSubLimit          = 200
    DefaultMaxPeers             = 30
    DefaultDialConcurrency      = 16
    ShortPeerIDLength          = 12
    MaxDisconnectReasons       = 5
    DefaultFilePermissions     = 0644
)
```

### 2. **Events Package Refactoring**

#### Create Event Handler Interface System
```go
// internal/events/handler.go
package events

type Handler interface {
    HandleEvent(ctx context.Context, event *host.TraceEvent) error
    EventType() string
}

type Manager struct {
    handlers map[string]Handler
    tool     ToolInterface
    logger   logrus.FieldLogger
}
```

#### Separate Event Handlers by Type
```go
// internal/events/handlers/connection.go
type ConnectionHandler struct {
    tool   ToolInterface
    logger logrus.FieldLogger
}

func (h *ConnectionHandler) HandleEvent(ctx context.Context, event *host.TraceEvent) error {
    // Handle only connection events
}

// internal/events/handlers/peer_score.go
type PeerScoreHandler struct {
    tool   ToolInterface
    logger logrus.FieldLogger
}

// internal/events/handlers/goodbye.go
type GoodbyeHandler struct {
    tool   ToolInterface
    logger logrus.FieldLogger
}
```

#### Create Payload Parsing System
```go
// internal/events/parsers/parser.go
package parsers

type PayloadParser interface {
    Parse(payload interface{}) (interface{}, error)
    SupportedType() string
}

type PeerScoreParser struct{}
type GoodbyeParser struct{}
type ConnectionParser struct{}
```

### 3. **Peer Package for State Management**

#### Create Peer Repository Pattern
```go
// internal/peer/repository.go
package peer

type Repository interface {
    GetPeer(peerID string) (*Stats, bool)
    CreatePeer(peerID string) *Stats
    UpdatePeer(peerID string, updateFn func(*Stats))
    GetAllPeers() map[string]*Stats
    GetPeerEventCounts() map[string]map[string]int
}

type InMemoryRepository struct {
    peers     map[string]*Stats
    events    map[string]map[string]int
    mu        sync.RWMutex
    eventsMu  sync.RWMutex
}
```

#### Separate Session Management
```go
// internal/peer/session.go
package peer

type SessionManager struct {
    repo Repository
}

func (sm *SessionManager) StartSession(peerID string) error
func (sm *SessionManager) EndSession(peerID string) error
func (sm *SessionManager) AddPeerScore(peerID string, score PeerScoreSnapshot) error
```

### 4. **Reports Package Restructuring**

#### Extract HTML Templates
```html
<!-- templates/report.html -->
<!DOCTYPE html>
<html>
<head>
    <title>Hermes Peer Score Report</title>
    <link rel="stylesheet" href="styles.css">
</head>
<body>
    <!-- Template content -->
    <script src="report.js"></script>
</body>
</html>
```

#### Create Template Manager
```go
// internal/reports/templates/manager.go
package templates

type Manager struct {
    templates map[string]*template.Template
}

func (m *Manager) LoadTemplates() error
func (m *Manager) RenderReport(data interface{}) (string, error)
```

#### Separate Report Generation Logic
```go
// internal/reports/generator.go
package reports

type Generator struct {
    templates TemplateManager
    logger    logrus.FieldLogger
}

func (g *Generator) GenerateJSON(report *core.Report) error
func (g *Generator) GenerateHTML(report *core.Report) error
```

### 5. **Core Package for Business Logic**

#### Refactor PeerScoreTool
```go
// internal/core/tool.go
package core

type Tool struct {
    ctx         context.Context
    logger      logrus.FieldLogger
    config      *config.Config
    peerRepo    peer.Repository
    eventMgr    *events.Manager
    reportGen   *reports.Generator
    startTime   time.Time
}

func (t *Tool) Start(ctx context.Context) error
func (t *Tool) Stop() error
func (t *Tool) GenerateReport() (*Report, error)
```

#### Simplify Report Generation
```go
// internal/core/report.go
package core

type ReportGenerator struct {
    peerRepo peer.Repository
}

func (rg *ReportGenerator) Generate(config *config.Config, startTime, endTime time.Time) (*Report, error) {
    // Simplified report generation without manual deep copying
    return &Report{
        // Use structured approach instead of manual copying
    }, nil
}
```

### 6. **CLI Package Simplification**

#### Clean Main Function
```go
// cmd/main.go
package main

func main() {
    cfg, err := config.LoadFromFlags()
    if err != nil {
        log.Fatal(err)
    }

    tool := core.NewTool(cfg)
    
    ctx, cancel := setupGracefulShutdown()
    defer cancel()

    if err := tool.Run(ctx); err != nil {
        log.Fatal(err)
    }
}
```

#### Separate Command Handlers
```go
// cmd/commands.go
package main

func handleHTMLOnlyMode(cfg *config.Config) error
func handleGoModValidation(cfg *config.Config) error
func handlePeerScoreTest(cfg *config.Config) error
```

## Testing Strategy

### Unit Testing
- **Event Handlers**: Test each handler independently with mock events
- **Payload Parsers**: Test parsing logic with various payload types
- **Peer Repository**: Test peer state management and concurrent access
- **Report Generation**: Test data transformation and template rendering
- **Configuration**: Test validation logic and mode switching

### Integration Testing
- **Event Processing Pipeline**: Test event flow from parsing to state updates
- **Report Generation End-to-End**: Test complete report generation process
- **Template Rendering**: Test HTML generation with sample data

### Validation Criteria
- All existing functionality preserved (no regressions)
- CLI interface maintains backward compatibility
- Performance improvements or no degradation
- Code coverage > 80% for new packages
- All linting rules pass
- Memory usage improvements in report generation

## Implementation Dependencies

### Phase 1: Foundation (Preparation)
- [ ] Create package structure and directories
- [ ] Extract constants from hardcoded values
- [ ] Create base interfaces for all packages
- [ ] Set up testing framework and initial tests
- Dependencies: None

### Phase 2: Events Refactoring (Core Refactoring)
- [ ] Break down `events.go` into handler packages
- [ ] Create payload parsing system
- [ ] Implement event manager with routing
- [ ] Migrate existing event handling logic
- Dependencies: Phase 1 completion

### Phase 3: Peer Management (State Management)
- [ ] Create peer repository interface and implementation
- [ ] Extract session management logic
- [ ] Implement concurrent-safe peer operations
- [ ] Migrate peer statistics calculation
- Dependencies: Phase 2 completion

### Phase 4: Report System (Templates and Generation)
- [ ] Extract HTML templates to separate files
- [ ] Create template management system
- [ ] Refactor report generation logic
- [ ] Implement efficient data serialization
- Dependencies: Phase 3 completion

### Phase 5: Core Tool and CLI (Application Layer)
- [ ] Refactor main tool orchestration
- [ ] Simplify CLI handling and command routing
- [ ] Clean up main.go and separate concerns
- [ ] Update configuration management
- Dependencies: Phase 4 completion

### Phase 6: Optimization and Polish (Final Improvements)
- [ ] Performance optimizations and benchmarking
- [ ] Documentation updates and code comments
- [ ] Final linting and code quality checks
- [ ] Integration testing and validation
- Dependencies: Phase 5 completion

## Risks and Considerations

### Implementation Risks
- **Regression Risk**: Large refactoring may introduce bugs in existing functionality
  - *Mitigation*: Comprehensive testing, incremental approach, extensive validation
- **Performance Risk**: New abstractions may impact performance
  - *Mitigation*: Benchmarking, performance testing, optimization focus
- **Complexity Risk**: Over-engineering may make code more complex
  - *Mitigation*: Keep interfaces simple, focus on clear responsibilities

### Performance Considerations
- **Memory Usage**: New package structure should not increase memory overhead
  - *Addressing*: Use efficient data structures, avoid unnecessary copying
- **Event Processing**: Handler dispatch should be fast for high-frequency events
  - *Addressing*: Use map-based dispatch, minimize allocations in hot paths

### Security Considerations
- **Template Security**: HTML template rendering must prevent injection attacks
  - *Addressing*: Use proper template escaping, validate all user inputs
- **File Permissions**: Ensure proper file permissions for generated reports
  - *Addressing*: Use constants for file permissions, validate paths

## Expected Outcomes

### Concrete Measurable Outcomes
- Reduce largest file size from 1320 lines to <300 lines per file
- Reduce function complexity: no functions >50 lines, cyclomatic complexity <10
- Eliminate all magic numbers and hardcoded strings
- Achieve >80% test coverage for new packages
- Maintain or improve performance benchmarks
- Pass all existing and new linting rules

### Success Metrics
- **Code Quality**: All functions <50 lines, clear single responsibilities
- **Test Coverage**: >80% coverage across all new packages
- **Performance**: Report generation time improvement or no regression
- **Maintainability**: Clear package boundaries, documented interfaces
- **Compatibility**: All existing CLI flags and functionality preserved
- **Documentation**: Complete package and function documentation