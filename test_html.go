package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

func testHTML() {
	log.Println("Generating test HTML report with sample data...")
	
	// Create sample report data with all the new timing features
	report := createSampleReport()
	
	// Save to JSON first
	jsonData, err := json.MarshalIndent(report, "", "    ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}
	
	err = os.WriteFile("test-peer-score-report.json", jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing JSON: %v\n", err)
		return
	}
	
	// Generate HTML report
	err = GenerateHTMLReport("test-peer-score-report.json", "test-peer-score-report.html")
	if err != nil {
		fmt.Printf("Error generating HTML: %v\n", err)
		return
	}
	
	fmt.Println("Test HTML report generated successfully: test-peer-score-report.html")
	fmt.Println("Test JSON data saved to: test-peer-score-report.json")
}

func createSampleReport() PeerScoreReport {
	now := time.Now()
	startTime := now.Add(-5 * time.Minute)
	
	// Create sample peers with timing data
	peers := make(map[string]*PeerStats)
	
	// Lighthouse peer with typical "too many peers" pattern
	peers["16Uiu2HAmGQR"] = &PeerStats{
		PeerID:      "16Uiu2HAmGQR",
		ClientType:  "lighthouse",
		ConnectedAt: startTime,
		Disconnected: true,
		DisconnectedAt: startTime.Add(60 * time.Second),
		HandshakeOK: true,
		GoodbyeCount: 1,
		LastGoodbye: "client has too many peers",
		ConnectionDuration: 60 * time.Second,
		FirstGoodbyeAt: startTime.Add(60 * time.Second),
		TimeToFirstGoodbye: 60 * time.Second,
		GoodbyeTimings: []GoodbyeTiming{
			{
				Reason: "client has too many peers",
				Timestamp: startTime.Add(60 * time.Second),
				DurationFromStart: 60 * time.Second,
				Sequence: 1,
			},
		},
		ReconnectionAttempts: 0,
	}
	
	// Prysm peer with peer score issue
	peers["16Uiu2HAkxyL"] = &PeerStats{
		PeerID:      "16Uiu2HAkxyL",
		ClientType:  "prysm",
		ConnectedAt: startTime.Add(30 * time.Second),
		Disconnected: true,
		DisconnectedAt: startTime.Add(75 * time.Second),
		HandshakeOK: true,
		GoodbyeCount: 1,
		LastGoodbye: "peer score too low",
		ConnectionDuration: 45 * time.Second,
		FirstGoodbyeAt: startTime.Add(75 * time.Second),
		TimeToFirstGoodbye: 45 * time.Second,
		GoodbyeTimings: []GoodbyeTiming{
			{
				Reason: "peer score too low",
				Timestamp: startTime.Add(75 * time.Second),
				DurationFromStart: 45 * time.Second,
				Sequence: 1,
			},
		},
		ReconnectionAttempts: 0,
	}
	
	// Nimbus peer with network verification issue
	peers["16Uiu2HAm8vX"] = &PeerStats{
		PeerID:      "16Uiu2HAm8vX",
		ClientType:  "nimbus",
		ConnectedAt: startTime.Add(60 * time.Second),
		Disconnected: true,
		DisconnectedAt: startTime.Add(90 * time.Second),
		HandshakeOK: true,
		GoodbyeCount: 1,
		LastGoodbye: "unable to verify network",
		ConnectionDuration: 30 * time.Second,
		FirstGoodbyeAt: startTime.Add(90 * time.Second),
		TimeToFirstGoodbye: 30 * time.Second,
		GoodbyeTimings: []GoodbyeTiming{
			{
				Reason: "unable to verify network",
				Timestamp: startTime.Add(90 * time.Second),
				DurationFromStart: 30 * time.Second,
				Sequence: 1,
			},
		},
		ReconnectionAttempts: 0,
	}
	
	// Teku peer still connected
	peers["16Uiu2HAm9dY"] = &PeerStats{
		PeerID:      "16Uiu2HAm9dY",
		ClientType:  "teku",
		ConnectedAt: startTime.Add(120 * time.Second),
		Disconnected: false,
		HandshakeOK: true,
		GoodbyeCount: 0,
		ConnectionDuration: 180 * time.Second,
		GoodbyeTimings: []GoodbyeTiming{},
		ReconnectionAttempts: 0,
	}
	
	// Create timing analysis
	timingAnalysis := ConnectionTiming{
		TotalConnections: 4,
		AverageConnectionDuration: 78 * time.Second,
		MedianConnectionDuration: 52 * time.Second,
		FastestDisconnect: 30 * time.Second,
		LongestConnection: 180 * time.Second,
		GoodbyeReasonTimings: map[string]GoodbyeReasonTiming{
			"client has too many peers": {
				Reason: "client has too many peers",
				Count: 1,
				AverageDuration: 60 * time.Second,
				MedianDuration: 60 * time.Second,
				MinDuration: 60 * time.Second,
				MaxDuration: 60 * time.Second,
				ClientBreakdown: map[string]int{"lighthouse": 1},
			},
			"peer score too low": {
				Reason: "peer score too low",
				Count: 1,
				AverageDuration: 45 * time.Second,
				MedianDuration: 45 * time.Second,
				MinDuration: 45 * time.Second,
				MaxDuration: 45 * time.Second,
				ClientBreakdown: map[string]int{"prysm": 1},
			},
			"unable to verify network": {
				Reason: "unable to verify network",
				Count: 1,
				AverageDuration: 30 * time.Second,
				MedianDuration: 30 * time.Second,
				MinDuration: 30 * time.Second,
				MaxDuration: 30 * time.Second,
				ClientBreakdown: map[string]int{"nimbus": 1},
			},
		},
		ClientSpecificTimings: map[string]map[string]GoodbyeReasonTiming{
			"lighthouse": {
				"client has too many peers": {
					Reason: "client has too many peers",
					Count: 1,
					AverageDuration: 60 * time.Second,
					MedianDuration: 60 * time.Second,
					MinDuration: 60 * time.Second,
					MaxDuration: 60 * time.Second,
					ClientBreakdown: map[string]int{"lighthouse": 1},
				},
			},
			"prysm": {
				"peer score too low": {
					Reason: "peer score too low",
					Count: 1,
					AverageDuration: 45 * time.Second,
					MedianDuration: 45 * time.Second,
					MinDuration: 45 * time.Second,
					MaxDuration: 45 * time.Second,
					ClientBreakdown: map[string]int{"prysm": 1},
				},
			},
			"nimbus": {
				"unable to verify network": {
					Reason: "unable to verify network",
					Count: 1,
					AverageDuration: 30 * time.Second,
					MedianDuration: 30 * time.Second,
					MinDuration: 30 * time.Second,
					MaxDuration: 30 * time.Second,
					ClientBreakdown: map[string]int{"nimbus": 1},
				},
			},
		},
		ClientTimingPatterns: []TimingPattern{
			{
				ClientType: "lighthouse",
				GoodbyeReason: "client has too many peers",
				AverageDuration: 60 * time.Second,
				Occurrences: 15,
				Pattern: "Timeout pattern: lighthouse peers disconnect with 'client has too many peers' around 60s mark (15 occurrences)",
			},
			{
				ClientType: "prysm",
				GoodbyeReason: "peer score too low",
				AverageDuration: 45 * time.Second,
				Occurrences: 5,
				Pattern: "Fast rejection: prysm peers send 'peer score too low' very quickly (avg 45s, 5 occurrences)",
			},
		},
		SuspiciousPatterns: []string{
			"Multiple peer score rejections: 5 peers rejected us for low score (avg after 45s)",
			"Network verification failures: 3 peers unable to verify network",
		},
	}
	
	return PeerScoreReport{
		Timestamp: now,
		Config: PeerScoreConfig{
			HermesPath: "./hermes",
			TestDuration: 5 * time.Minute,
			ReportInterval: 30 * time.Second,
			HermesArgs: []string{"--config", "config.yaml"},
		},
		StartTime: startTime,
		EndTime: now,
		Duration: 5 * time.Minute,
		TotalConnections: 4,
		SuccessfulHandshakes: 4,
		FailedHandshakes: 0,
		GoodbyeMessages: 3,
		GoodbyeReasons: map[string]int{
			"client has too many peers": 1,
			"peer score too low": 1,
			"unable to verify network": 1,
		},
		GoodbyesByClient: map[string]map[string]int{
			"lighthouse": {"client has too many peers": 1},
			"prysm": {"peer score too low": 1},
			"nimbus": {"unable to verify network": 1},
		},
		PeersByClient: map[string]int{
			"lighthouse": 1,
			"prysm": 1,
			"nimbus": 1,
			"teku": 1,
		},
		UniqueClients: 4,
		Peers: peers,
		OverallScore: 75.0,
		Summary: "Score: 75.0% | Connections: 4 | Handshakes: 4 | Clients: 4 | Goodbyes: 3",
		Errors: []string{},
		ConnectionFailed: false,
		TimingAnalysis: timingAnalysis,
		DownscoreIndicators: []string{
			"Direct peer score rejections: 1 peers rejected us for low peer score",
			"Network verification failures: 1 peers unable to verify network",
		},
	}
}