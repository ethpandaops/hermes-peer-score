# Goodbye Event Reasons Summary Implementation Plan

## Overview

> This plan outlines the implementation of a goodbye event reasons summary feature for the HTML report in the Hermes Peer Score tool. The feature will provide a comprehensive view of why peers disconnect, displaying aggregated statistics and detailed breakdowns of goodbye events by reason (using string-based analysis), similar to how mesh events are currently displayed.

## Current State Assessment

- **Existing Goodbye Event Handling**:
  - Goodbye events are collected and stored in `ConnectionSession.GoodbyeEvents` array
  - Each event contains timestamp, code (uint64), and reason (string)
  - Events are displayed in the timeline view with orange badges
  - Summary count shown in peer list and session summary

- **Limitations**:
  - No dedicated section for goodbye events in peer detail modal
  - No aggregation of goodbye reasons across all peers
  - No analysis of reason strings to identify patterns
  - No visualization of goodbye event patterns or trends

- **Technical Infrastructure**:
  - HTML report generated from Go template in `html_report.go`
  - Dynamic data loading via JavaScript
  - Existing collapsible section pattern (used for mesh events)
  - Event data already serialized to JSON

## Goals

1. Primary goal: Add a dedicated "Goodbye Events" section to peer detail modals showing goodbye reasons breakdown
2. Provide aggregated goodbye event statistics in the main report summary
   - Total goodbye events across all peers
   - Group by unique reason strings (case-insensitive)
   - Most common disconnection reasons
3. Analyze goodbye event data to derive patterns
   - Group similar reason strings together
   - Preserve original data while providing analysis
   - Extract both code and reason text patterns
4. Non-functional requirements:
   - Maintain existing performance characteristics
   - Ensure backwards compatibility with existing reports
   - Follow established UI patterns and styling
   - Never drop or modify original data

## Design Approach

### Architecture Overview
The implementation will follow the existing pattern established by mesh events:
- Backend: Enhance data structures to include goodbye code mappings
- Frontend: Add new UI sections using existing collapsible panel patterns
- Data flow: Goodbye events → JSON serialization → JavaScript rendering → HTML display

### Component Breakdown

1. **Goodbye Reason Analysis Component**
   - Purpose: Analyze and group goodbye reasons from raw string data
   - Responsibilities:
     - Group reasons by exact string match (case-insensitive)
     - Preserve original reason strings
     - Count occurrences of each unique reason
   - Interfaces: Used during report generation

2. **Report Data Enhancement**
   - Purpose: Add goodbye event summaries to report data structure
   - Responsibilities:
     - Calculate goodbye event statistics during report generation
     - Group events by reason string
     - Derive common patterns from reason data
     - Add summary data to JSON output
   - Interfaces: Modifies `OptimizedHTMLTemplateData` and related structures

3. **UI Components**
   - Purpose: Display goodbye event information in the HTML report
   - Responsibilities:
     - Render goodbye events section in peer detail modal
     - Display summary statistics in main report
     - Show both raw data and derived patterns
     - Handle collapsible sections and interactions
   - Interfaces: JavaScript functions for rendering and event handling

## Implementation Approach

### 1. Create Goodbye Reason Analysis Functions

#### Specific Changes

- Create functions to analyze goodbye reasons from string data
- Group by exact reason string (case-insensitive)
- Derive patterns from reason text after collection

#### Sample Implementation

```go
// GoodbyeReasonStats tracks statistics for a specific goodbye reason
type GoodbyeReasonStats struct {
    Reason      string   `json:"reason"` // Original reason string
    Count       int      `json:"count"`
    Codes       []uint64 `json:"codes"` // All codes seen with this reason
    Examples    []string `json:"examples"` // First few examples of this reason
}

// AnalyzeGoodbyeReasons groups and analyzes goodbye reasons from events
func AnalyzeGoodbyeReasons(events []GoodbyeEvent) map[string]*GoodbyeReasonStats {
    stats := make(map[string]*GoodbyeReasonStats)

    for _, event := range events {
        // Use lowercase for grouping but preserve original
        key := strings.ToLower(event.Reason)
        if key == "" {
            key = "unknown"
        }

        if stat, exists := stats[key]; exists {
            stat.Count++
            // Track unique codes
            if !containsCode(stat.Codes, event.Code) {
                stat.Codes = append(stat.Codes, event.Code)
            }
        } else {
            stats[key] = &GoodbyeReasonStats{
                Reason:   event.Reason, // Preserve original casing
                Count:    1,
                Codes:    []uint64{event.Code},
                Examples: []string{event.Reason},
            }
        }
    }

    return stats
}
```

### 2. Enhance Report Data Structures

#### Specific Changes

- Add `GoodbyeEventsSummary` to `SummaryData` struct
- Create new struct for goodbye event statistics
- Store reason analysis results

#### Sample Implementation

```go
// GoodbyeEventsSummary contains aggregated goodbye event statistics
type GoodbyeEventsSummary struct {
    TotalEvents     int                           `json:"total_events"`
    ReasonStats     []*GoodbyeReasonStats        `json:"reason_stats"` // Sorted by count
    UniqueReasons   int                          `json:"unique_reasons"`
    TopReasons      []string                     `json:"top_reasons"` // Top 5 most common
    CodeFrequency   map[uint64]int               `json:"code_frequency"` // Code occurrence count
}

// Update SummaryData struct
type SummaryData struct {
    // ... existing fields ...
    GoodbyeEventsSummary GoodbyeEventsSummary `json:"goodbye_events_summary"`
}
```

### 3. Implement Goodbye Statistics Calculation

#### Specific Changes

- Add function to calculate goodbye event statistics
- Integrate into `extractSummaryData` function
- Analyze and group by reason strings

#### Sample Implementation

```go
func calculateGoodbyeEventsSummary(peers map[string]*PeerStats) GoodbyeEventsSummary {
    var allEvents []GoodbyeEvent
    
    codeFreq := make(map[uint64]int)

    // Collect all goodbye events
    for _, peer := range peers {
        for _, session := range peer.ConnectionSessions {
            for _, goodbye := range session.GoodbyeEvents {
                allEvents = append(allEvents, goodbye)
                codeFreq[goodbye.Code]++
            }
        }
    }

    // Analyze reasons
    reasonStats := AnalyzeGoodbyeReasons(allEvents)

    // Convert to sorted slice
    var statsList []*GoodbyeReasonStats
    for _, stat := range reasonStats {
        statsList = append(statsList, stat)
    }
    sort.Slice(statsList, func(i, j int) bool {
        return statsList[i].Count > statsList[j].Count
    })

    // Get top reasons
    var topReasons []string

    for i := 0; i < 5 && i < len(statsList); i++ {
        topReasons = append(topReasons, statsList[i].Reason)
    }

    return GoodbyeEventsSummary{
        TotalEvents:   len(allEvents),
        ReasonStats:   statsList,
        UniqueReasons: len(reasonStats),
        TopReasons:    topReasons,
        CodeFrequency: codeFreq,
    }
}
```

### 4. Add Goodbye Events Section to Peer Detail Modal

#### Specific Changes

- Add HTML template for goodbye events section
- Display both code and raw reason string
- Follow existing mesh events pattern

#### Sample Implementation

```javascript
// Add to peer detail modal rendering
const goodbyeEventsHtml = session.goodbye_events && session.goodbye_events.length > 0 ? `
<div>
    <div class="p-3 bg-gray-50 cursor-pointer border rounded-lg" onclick="toggleSection('${sessionId}-goodbye')">
        <div class="flex items-center justify-between">
            <h6 class="font-medium text-gray-800">Goodbye Events (${session.goodbye_events.length} events)</h6>
            <svg class="w-4 h-4 text-gray-500 transform transition-transform" id="${sessionId}-goodbye-arrow">
                <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
            </svg>
        </div>
    </div>
    <div class="hidden mt-2" id="${sessionId}-goodbye">
        <div class="max-h-64 overflow-y-auto">
            <table class="min-w-full bg-white border border-gray-200 rounded text-xs">
                <thead class="bg-gray-50">
                    <tr>
                        <th class="px-3 py-2 text-left">Time</th>
                        <th class="px-3 py-2 text-left">Code</th>
                        <th class="px-3 py-2 text-left">Reason</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-gray-100">
                    ${session.goodbye_events.map(event => `
                        <tr class="hover:bg-gray-50">
                            <td class="px-3 py-2 text-xs">${new Date(event.timestamp).toLocaleTimeString()}</td>
                            <td class="px-3 py-2 text-xs">
                                <span class="font-mono bg-gray-100 px-1 py-0.5 rounded">${event.code}</span>
                            </td>
                            <td class="px-3 py-2 text-xs text-gray-700">
                                ${event.reason || '<span class="text-gray-400 italic">no reason provided</span>'}
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        </div>
    </div>
</div>
` : '';
```
```

### 5. Add Goodbye Events Summary to Main Report

#### Specific Changes

- Add summary card showing total goodbye events
- Create breakdown section for goodbye reasons
- Show most common reason strings

#### Sample Implementation

```javascript
// Add to summary statistics grid
const goodbyeSummaryCard = `
<div class="bg-white rounded-lg shadow p-6">
    <div class="text-sm font-medium text-gray-500">Goodbye Events</div>
    <div class="text-2xl font-bold text-orange-600">${summary.goodbye_events_summary.total_events}</div>
    <div class="text-xs text-gray-500 mt-1">
        ${summary.goodbye_events_summary.unique_reasons} unique reasons
    </div>
</div>
`;

// Add goodbye events breakdown section
const goodbyeBreakdownSection = summary.goodbye_events_summary.total_events > 0 ? `
<div class="bg-white rounded-lg shadow p-6 mb-6">
    <h3 class="text-lg font-semibold mb-4">Top Goodbye Reasons</h3>
    <div class="space-y-3">
        ${summary.goodbye_events_summary.reason_stats
            .slice(0, 10) // Show top 10
            .map(stat => `
                <div class="flex items-center justify-between p-3 bg-gray-50 rounded hover:bg-gray-100 transition-colors">
                    <div class="flex-1">
                        <div class="font-medium text-gray-900">
                            ${stat.reason || '<span class="italic text-gray-500">no reason provided</span>'}
                        </div>
                        <div class="text-xs text-gray-500 mt-1">
                            Codes: ${stat.codes.join(', ')}
                        </div>
                    </div>
                    <div class="text-right">
                        <div class="text-lg font-semibold">${stat.count}</div>
                        <div class="text-xs text-gray-500">${((stat.count / summary.goodbye_events_summary.total_events) * 100).toFixed(1)}%</div>
                    </div>
                </div>
            `).join('')}
    </div>
    ${summary.goodbye_events_summary.unique_reasons > 10 ? `
        <div class="text-sm text-gray-500 mt-4 text-center">
            Showing top 10 of ${summary.goodbye_events_summary.unique_reasons} unique reasons
        </div>
    ` : ''}
</div>
` : '';
```

### 6. Add JavaScript Helper Functions

#### Specific Changes

- Add helper functions for formatting goodbye data
- Handle empty/missing reasons gracefully
- Support the string-based analysis approach

#### Sample Implementation

```javascript
// Helper function to format goodbye reason display
function formatGoodbyeReason(reason) {
    if (!reason || reason === "" || reason === "unknown") {
        return '<span class="text-gray-400 italic">no reason provided</span>';
    }
    return escapeHtml(reason);
}

// Helper to escape HTML for security
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Update the initial data loading to include goodbye summaries
function initializeReport() {
    // ... existing code ...

    // Render goodbye events summary if available
    if (window.reportData.summary.goodbye_events_summary) {
        renderGoodbyeEventsSummary(window.reportData.summary.goodbye_events_summary);
    }
}

// Function to render the goodbye events breakdown
function renderGoodbyeEventsSummary(summary) {
    if (!summary || summary.total_events === 0) return;

    // Add the breakdown section to the page
    const container = document.getElementById('goodbyeBreakdownContainer');
    if (container) {
        container.innerHTML = generateGoodbyeBreakdownHtml(summary);
    }
}
```

## Testing Strategy

### Unit Testing
- Test `GetGoodbyeCodeDescription` function with all known codes
- Test `calculateGoodbyeEventsSummary` with various peer configurations
- Verify JSON serialization includes new fields

### Integration Testing
- Verify goodbye events section appears in peer detail modal
- Test collapsible section functionality
- Ensure summary statistics are calculated correctly
- Validate backwards compatibility with existing reports

### Validation Criteria
- All goodbye events display with correct code descriptions
- Summary statistics match actual event counts
- UI elements are responsive and accessible
- No performance degradation with large numbers of events

## Implementation Dependencies

1. **Phase 1: Backend Implementation**
   - Add goodbye code constants and mapping function
   - Enhance data structures with goodbye summaries
   - Implement statistics calculation
   - Dependencies: None

2. **Phase 2: Frontend Implementation**
   - Update HTML template with new sections
   - Add JavaScript helper functions
   - Implement UI components
   - Dependencies: Phase 1 completion

3. **Phase 3: Testing and Polish**
   - Add comprehensive tests
   - Fix any UI/UX issues
   - Update documentation
   - Dependencies: Phase 2 completion

## Risks and Considerations

### Implementation Risks

- **Large Event Volumes**: Reports with many goodbye events could impact performance
  - Mitigation: Implement pagination or limit displayed events with "show more" option
- **Varied Reason Formats**: Different clients may format reasons differently
  - Mitigation: Use case-insensitive grouping and preserve original strings
- **Missing or Empty Reasons**: Some events may have no reason text
  - Mitigation: Handle empty reasons gracefully with clear indicators

### Performance Considerations

- **Report Generation**: String analysis and grouping operations
  - Addressing: Use efficient maps for grouping, single pass where possible
- **UI Rendering**: More DOM elements in peer detail modals
  - Addressing: Limit initial display to top N reasons, expandable for full list

### Security Considerations

- **XSS Prevention**: Raw reason strings from peers must be properly escaped
  - Addressing: Use proper HTML escaping for all user-provided strings
- **Data Integrity**: Preserve original data without modification
  - Addressing: Always work with copies when analyzing/grouping data

## Expected Outcomes

- Users can see a comprehensive breakdown of why peers disconnect
- Reason strings are grouped and analyzed to show patterns
- Original data is preserved while providing useful analysis
- Summary statistics provide quick insights into network health

### Success Metrics

- All goodbye reasons are properly grouped and counted
- No data is lost or modified during analysis
- Summary calculations complete in < 100ms for typical reports
- UI sections load without noticeable delay
- Zero XSS vulnerabilities in reason string display
