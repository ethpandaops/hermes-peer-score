package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// EnhancedLogParser provides real-time parsing of Hermes log output
type EnhancedLogParser struct {
	tool            *PeerScoreTool
	patterns        map[string]*LogPattern
	connectionChan  chan ConnectionEvent
	goodbyeChan     chan GoodbyeEvent
	handshakeChan   chan HandshakeEvent
	errorChan       chan ErrorEvent
}

type LogPattern struct {
	Name    string
	Regex   *regexp.Regexp
	Handler func(matches []string) Event
}

type Event interface {
	Type() string
	Timestamp() time.Time
}

type ConnectionEvent struct {
	PeerID    string
	Connected bool
	Time      time.Time
}

func (e ConnectionEvent) Type() string         { return "connection" }
func (e ConnectionEvent) Timestamp() time.Time { return e.Time }

type HandshakeEvent struct {
	PeerID     string
	Success    bool
	Agent      string
	SeqNumber  uint64
	ForkDigest string
	Time       time.Time
}

func (e HandshakeEvent) Type() string         { return "handshake" }
func (e HandshakeEvent) Timestamp() time.Time { return e.Time }

type GoodbyeEvent struct {
	PeerID string
	Reason string
	Code   int
	Time   time.Time
}

func (e GoodbyeEvent) Type() string         { return "goodbye" }
func (e GoodbyeEvent) Timestamp() time.Time { return e.Time }


type ErrorEvent struct {
	Message string
	Time    time.Time
}

func (e ErrorEvent) Type() string         { return "error" }
func (e ErrorEvent) Timestamp() time.Time { return e.Time }

func NewEnhancedLogParser(tool *PeerScoreTool) *EnhancedLogParser {
	parser := &EnhancedLogParser{
		tool:            tool,
		patterns:        make(map[string]*LogPattern),
		connectionChan:  make(chan ConnectionEvent, 100),
		goodbyeChan:     make(chan GoodbyeEvent, 100),
		handshakeChan:   make(chan HandshakeEvent, 100),
		errorChan:       make(chan ErrorEvent, 100),
	}

	parser.setupPatterns()
	return parser
}

func (p *EnhancedLogParser) setupPatterns() {
	// Connection pattern: Connected with peer
	p.patterns["connection"] = &LogPattern{
		Name:  "connection",
		Regex: regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d+).*Connected with peer.*peer_id=(\w+)`),
		Handler: func(matches []string) Event {
			if len(matches) >= 3 {
				timestamp := p.parseTimestamp(matches[1])
				return ConnectionEvent{
					PeerID:    matches[2],
					Connected: true,
					Time:      timestamp,
				}
			}
			return nil
		},
	}

	// Handshake pattern: Performed successful handshake
	p.patterns["handshake"] = &LogPattern{
		Name:  "handshake",
		Regex: regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d+).*Performed successful handshake.*peer_id=(\w+).*seq=(\d+).*agent=([^\s]+).*fork-digest=(\w+)`),
		Handler: func(matches []string) Event {
			if len(matches) >= 6 {
				timestamp := p.parseTimestamp(matches[1])
				seq, _ := strconv.ParseUint(matches[3], 10, 64)
				return HandshakeEvent{
					PeerID:     matches[2],
					Success:    true,
					Agent:      matches[4],
					SeqNumber:  seq,
					ForkDigest: matches[5],
					Time:       timestamp,
				}
			}
			return nil
		},
	}

	// Goodbye pattern: Received goodbye message (from Hermes logs)
	p.patterns["goodbye"] = &LogPattern{
		Name:  "goodbye",
		Regex: regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d+).*Received goodbye message.*peer_id=(\w+).*msg="([^"]+)"`),
		Handler: func(matches []string) Event {
			if len(matches) >= 4 {
				timestamp := p.parseTimestamp(matches[1])
				return GoodbyeEvent{
					PeerID: matches[2],
					Reason: matches[3],
					Time:   timestamp,
				}
			}
			return nil
		},
	}

	// Disconnection pattern: Disconnected from handshaked peer
	p.patterns["disconnect"] = &LogPattern{
		Name:  "disconnect",
		Regex: regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d+).*Disconnected from handshaked peer.*peer_id=(\w+)`),
		Handler: func(matches []string) Event {
			if len(matches) >= 3 {
				timestamp := p.parseTimestamp(matches[1])
				return ConnectionEvent{
					PeerID:    matches[2],
					Connected: false,
					Time:      timestamp,
				}
			}
			return nil
		},
	}

	// Status request pattern: Perform status request
	p.patterns["status"] = &LogPattern{
		Name:  "status",
		Regex: regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d+).*Perform status request.*peer_id=(\w+)`),
		Handler: func(matches []string) Event {
			// This could be used to track status request frequency
			return nil
		},
	}

	// Error patterns for connection failures
	p.patterns["connection_failed"] = &LogPattern{
		Name:  "connection_failed",
		Regex: regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d+).*Connection to beacon node failed.*err="([^"]+)"`),
		Handler: func(matches []string) Event {
			if len(matches) >= 3 {
				timestamp := p.parseTimestamp(matches[1])
				return ErrorEvent{
					Message: "Connection to beacon node failed: " + matches[2],
					Time:    timestamp,
				}
			}
			return nil
		},
	}

	p.patterns["terminated_abnormally"] = &LogPattern{
		Name:  "terminated_abnormally",
		Regex: regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d+).*terminated abnormally.*err="([^"]+)"`),
		Handler: func(matches []string) Event {
			if len(matches) >= 3 {
				timestamp := p.parseTimestamp(matches[1])
				return ErrorEvent{
					Message: "Hermes terminated abnormally: " + matches[2],
					Time:    timestamp,
				}
			}
			return nil
		},
	}

	p.patterns["dialback_waiting"] = &LogPattern{
		Name:  "dialback_waiting",
		Regex: regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d+).*Waiting for dialback from Prysm node`),
		Handler: func(matches []string) Event {
			if len(matches) >= 2 {
				timestamp := p.parseTimestamp(matches[1])
				return ErrorEvent{
					Message: "Waiting for dialback from Prysm node",
					Time:    timestamp,
				}
			}
			return nil
		},
	}
}

func (p *EnhancedLogParser) parseTimestamp(timeStr string) time.Time {
	// Parse time format like "09:55:15.319669"
	today := time.Now().Format("2006-01-02")
	fullTimeStr := today + " " + timeStr

	t, err := time.Parse("2006-01-02 15:04:05.999999", fullTimeStr)
	if err != nil {
		// Fallback to current time if parsing fails
		return time.Now()
	}
	return t
}

func (p *EnhancedLogParser) StartParsing(ctx context.Context, reader io.Reader) {
	scanner := bufio.NewScanner(reader)

	go p.eventProcessor(ctx)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			line := scanner.Text()
			p.parseLine(line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading log stream: %v", err)
	}
}

func (p *EnhancedLogParser) parseLine(line string) {
	// Strip ANSI color codes
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	cleanLine := ansiRegex.ReplaceAllString(line, "")

	for _, pattern := range p.patterns {
		if matches := pattern.Regex.FindStringSubmatch(cleanLine); matches != nil {
			if event := pattern.Handler(matches); event != nil {
				p.dispatchEvent(event)
			}
			break // Only match first pattern
		}
	}
}

func (p *EnhancedLogParser) dispatchEvent(event Event) {
	switch e := event.(type) {
	case ConnectionEvent:
		select {
		case p.connectionChan <- e:
		default:
			// Channel full, drop event
		}
	case HandshakeEvent:
		select {
		case p.handshakeChan <- e:
		default:
		}
	case GoodbyeEvent:
		select {
		case p.goodbyeChan <- e:
		default:
		}
	case ErrorEvent:
		select {
		case p.errorChan <- e:
		default:
		}
	}
}

func (p *EnhancedLogParser) eventProcessor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-p.connectionChan:
			p.handleConnectionEvent(event)
		case event := <-p.handshakeChan:
			p.handleHandshakeEvent(event)
		case event := <-p.goodbyeChan:
			p.handleGoodbyeEvent(event)
		case event := <-p.errorChan:
			p.handleErrorEvent(event)
		}
	}
}

func (p *EnhancedLogParser) handleConnectionEvent(event ConnectionEvent) {
	p.tool.mu.Lock()
	defer p.tool.mu.Unlock()

	peer, exists := p.tool.peers[event.PeerID]
	if !exists && event.Connected {
		peer = &PeerStats{
			PeerID:      event.PeerID,
			ConnectedAt: event.Time,
		}
		p.tool.peers[event.PeerID] = peer
	} else if exists && !event.Connected {
		// Peer disconnected
		delete(p.tool.peers, event.PeerID)
	}
}

func (p *EnhancedLogParser) handleHandshakeEvent(event HandshakeEvent) {
	p.tool.mu.Lock()
	defer p.tool.mu.Unlock()

	peer, exists := p.tool.peers[event.PeerID]
	if !exists {
		peer = &PeerStats{
			PeerID:      event.PeerID,
			ConnectedAt: event.Time,
		}
		p.tool.peers[event.PeerID] = peer
	}

	peer.HandshakeOK = event.Success
	peer.ClientType = p.normalizeClientType(event.Agent)


	result := "success"
	if !event.Success {
		result = "failure"
	}
	log.Printf("Handshake %s with %s (%s)", result, event.PeerID[:12], peer.ClientType)
}

func (p *EnhancedLogParser) handleGoodbyeEvent(event GoodbyeEvent) {
	p.tool.mu.Lock()
	defer p.tool.mu.Unlock()

	// Always count goodbye messages globally
	p.tool.totalGoodbyes++

	// Track goodbye reasons
	p.tool.goodbyeReasons[event.Reason]++

	peer, exists := p.tool.peers[event.PeerID]
	clientType := "unknown"
	if exists {
		peer.GoodbyeCount++
		peer.LastGoodbye = event.Reason
		clientType = peer.ClientType
	}

	// Track goodbye by client type
	if p.tool.goodbyesByClient[clientType] == nil {
		p.tool.goodbyesByClient[clientType] = make(map[string]int)
	}
	p.tool.goodbyesByClient[clientType][event.Reason]++

	// Log with severity based on goodbye reason
	severity := classifyGoodbyeSeverity(event.Reason)
	log.Printf("Goodbye [%s] from %s: %s", severity, event.PeerID[:12], event.Reason)
}

// classifyGoodbyeSeverity categorizes goodbye reasons by severity
func classifyGoodbyeSeverity(reason string) string {
	switch reason {
	case "client has too many peers":
		return "NORMAL"
	case "client shutdown":
		return "NORMAL"
	case "peer score too low":
		return "ERROR"
	case "client banned this node":
		return "ERROR"
	case "irrelevant network":
		return "ERROR"
	case "unable to verify network":
		return "ERROR"
	case "fault/error":
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}


func (p *EnhancedLogParser) handleErrorEvent(event ErrorEvent) {
	p.tool.mu.Lock()
	defer p.tool.mu.Unlock()

	// Track errors in the tool
	p.tool.errors = append(p.tool.errors, event.Message)

	log.Printf("ERROR: %s", event.Message)

	// Set connection failure flag if this is a connection error
	if strings.Contains(event.Message, "Connection to beacon node failed") ||
		strings.Contains(event.Message, "terminated abnormally") {
		p.tool.connectionFailed = true
	}
}

func (p *EnhancedLogParser) normalizeClientType(agent string) string {
	agent = strings.ToLower(agent)

	clients := []string{"lighthouse", "prysm", "nimbus", "lodestar", "grandine", "teku", "erigon", "caplin"}
	for _, client := range clients {
		if strings.Contains(agent, client) {
			return client
		}
	}

	log.Printf("Unidentified User Agent: %s", agent)

	// Extract first part before slash if present
	parts := strings.Split(agent, "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return "unknown"
}

// GetPeerStats returns current peer statistics
func (p *EnhancedLogParser) GetPeerStats() map[string]*PeerStats {
	p.tool.mu.RLock()
	defer p.tool.mu.RUnlock()

	stats := make(map[string]*PeerStats)
	for id, peer := range p.tool.peers {
		// Create a copy to avoid race conditions
		peerCopy := *peer // Copy the struct
		stats[id] = &peerCopy
	}

	return stats
}

// GetClientDistribution returns the distribution of clients
func (p *EnhancedLogParser) GetClientDistribution() map[string]int {
	p.tool.mu.RLock()
	defer p.tool.mu.RUnlock()

	distribution := make(map[string]int)
	for _, peer := range p.tool.peers {
		if peer.HandshakeOK {
			distribution[peer.ClientType]++
		}
	}

	return distribution
}
