package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/p2p/encoder"
	"github.com/OffchainLabs/prysm/v6/config/params"
	"github.com/OffchainLabs/prysm/v6/time/slots"
	"github.com/probe-lab/hermes/eth"
	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

// PeerScoreTool manages the peer scoring test execution and data collection.
// It orchestrates the Hermes process, parses logs in real-time, and aggregates
// peer connection statistics for scoring and analysis.
type PeerScoreTool struct {
	ctx               context.Context //nolint:containedctx // ok.
	log               logrus.FieldLogger
	config            PeerScoreConfig           // Test configuration and parameters.
	peers             map[string]*PeerStats     // Individual peer statistics indexed by peer ID.
	peersMu           sync.RWMutex              // Protects concurrent access to peer data.
	peerEventCounts   map[string]map[string]int // Count of event types by peers.
	peerEventCountsMu sync.RWMutex              // Protects concurrent access to peer data.
	startTime         time.Time                 // When the test execution began.

	// Hermes.
	node          *eth.Node
	networkConfig *params.NetworkConfig
	beaconConfig  *params.BeaconChainConfig
}

// NewPeerScoreTool creates a new peer score tool instance with the given configuration.
func NewPeerScoreTool(
	ctx context.Context,
	log logrus.FieldLogger,
	config PeerScoreConfig,
) *PeerScoreTool {
	return &PeerScoreTool{
		ctx:             ctx,
		log:             log,
		config:          config,
		peers:           make(map[string]*PeerStats),
		peerEventCounts: make(map[string]map[string]int),
	}
}

// StartHermes initiates connection to Hermes and begins listening for events.
func (pst *PeerScoreTool) StartHermes(ctx context.Context) error {
	// Record start time
	pst.startTime = time.Now()

	// Direct connect.
	c, err := eth.DeriveKnownNetworkConfig(ctx, params.MainnetName)
	if err != nil {
		return fmt.Errorf("get config for %s: %w", params.MainnetName, err)
	}

	pst.networkConfig = c.Network
	pst.beaconConfig = c.Beacon

	genesisRoot := c.Genesis.GenesisValidatorRoot
	genesisTime := c.Genesis.GenesisTime

	// compute fork version and fork digest
	currentSlot := slots.Since(genesisTime)
	currentEpoch := slots.ToEpoch(currentSlot)

	currentForkVersion, err := eth.GetCurrentForkVersion(currentEpoch, pst.beaconConfig)
	if err != nil {
		return fmt.Errorf("compute fork version for epoch %d: %w", currentEpoch, err)
	}

	forkDigest, err := signing.ComputeForkDigest(currentForkVersion[:], genesisRoot)
	if err != nil {
		return fmt.Errorf("create fork digest (%s, %x): %w", genesisTime, genesisRoot, err)
	}

	// Overriding configuration so that functions like ComputForkDigest take the
	// correct input data from the global configuration.
	params.OverrideBeaconConfig(pst.beaconConfig)
	params.OverrideBeaconNetworkConfig(pst.networkConfig)

	// Hermes config.
	cfg := pst.config.ToolConfig.AsHermesConfig()
	cfg.GenesisConfig = c.Genesis
	cfg.NetworkConfig = pst.networkConfig
	cfg.BeaconConfig = pst.beaconConfig
	cfg.ForkDigest = forkDigest
	cfg.ForkVersion = currentForkVersion
	cfg.PubSubSubscriptionRequestLimit = 200
	cfg.PubSubQueueSize = 200
	cfg.Libp2pPeerscoreSnapshotFreq = 5 * time.Second
	cfg.GossipSubMessageEncoder = encoder.SszNetworkEncoder{}
	cfg.RPCEncoder = encoder.SszNetworkEncoder{}
	cfg.Tracer = otel.GetTracerProvider().Tracer("hermes")
	cfg.Meter = otel.GetMeterProvider().Meter("hermes")

	// Apply validation-specific configuration overrides
	validationConfig := GetValidationConfigs()[pst.config.ValidationMode]
	pst.applyValidationConfig(cfg, validationConfig)

	if err = cfg.Validate(); err != nil {
		return fmt.Errorf("invalid Hermes node config: %w", err)
	}

	node, err := eth.NewNode(cfg)
	if err != nil {
		if strings.Contains(err.Error(), "in correct fork_digest") {
			return fmt.Errorf("invalid fork digest (config.ethereum.network and prysm network probably don't match): %w", err)
		}

		return err
	}

	pst.node = node

	pst.node.OnEvent(func(ctx context.Context, event *host.TraceEvent) {
		if err := pst.handleHermesEvent(ctx, event); err != nil {
			pst.log.WithError(err).Error("Failed to handle hermes event")
		}
	})

	go func() {
		// Blackhole hermes logs by redirecting the default logger. It's noise, and we'll
		// log our own tracking of connections/events.
		originalOutput := log.Writer()
		log.SetOutput(io.Discard)
		defer log.SetOutput(originalOutput) // Restore original output when done.

		if err := pst.node.Start(ctx); err != nil {
			pst.log.WithError(err).Fatal("Failed to start hermes")
		}
	}()

	return nil
}

// applyValidationConfig applies validation-specific configuration overrides to the Hermes config.
func (pst *PeerScoreTool) applyValidationConfig(cfg *eth.NodeConfig, validationConfig ValidationConfig) {
	// Log the validation mode being applied
	pst.log.WithFields(logrus.Fields{
		"validation_mode":  validationConfig.Mode,
		"hermes_version":   validationConfig.HermesVersion,
		"config_overrides": validationConfig.ConfigOverrides,
	}).Info("Applying validation-specific configuration")

	// Apply mode-specific configuration
	pst.applyModeSpecificValidationConfig(cfg, validationConfig)
}

// applyModeSpecificValidationConfig applies configuration specific to the validation mode.
func (pst *PeerScoreTool) applyModeSpecificValidationConfig(cfg *eth.NodeConfig, validationConfig ValidationConfig) {
	switch validationConfig.Mode {
	case ValidationModeIndependent:
		pst.log.Info("Configuring independent validation mode - Hermes will use Prysm for beacon state but perform validation internally")
		pst.log.Info("Independent validation configuration applied - will fetch beacon state via HTTP and validate internally")
	case ValidationModeDelegated:
		pst.log.Info("Configuring delegated validation mode - Hermes will send validation requests to Prysm for processing")
		pst.log.Info("Delegated validation configuration applied - will delegate validation processing to Prysm")

		// GRPC port only used in delegated mode.
		if cfg.PrysmPortGRPC == 0 {
			pst.log.Warn("Prysm gRPC port not configured")
		}
	default:
		pst.log.Warnf("Unknown validation mode: %s", validationConfig.Mode)

		return
	}

	// Validate Prysm connection is properly configured (both modes need it).
	if cfg.PrysmHost == "" {
		pst.log.Warn("Prysm host not configured - needed for validation mode")
	} else {
		pst.log.WithField("prysm_host", cfg.PrysmHost).Info("Prysm connection configured")
	}

	// Validate connection parameters.
	if cfg.PrysmPortHTTP == 0 {
		pst.log.Warn("Prysm HTTP port not configured")
	}
}

// Stop terminates the tool gracefully.
func (pst *PeerScoreTool) Stop() error {
	return nil
}

// GenerateReport creates the final peer score report with comprehensive analysis.
// It aggregates all collected data, calculates scores, and produces a detailed
// report suitable for both JSON serialization and further analysis.
func (pst *PeerScoreTool) GenerateReport() PeerScoreReport {
	pst.log.Info("Generating peer score report...")

	pst.peersMu.RLock()
	pst.peerEventCountsMu.RLock()
	defer pst.peersMu.RUnlock()
	defer pst.peerEventCountsMu.RUnlock()

	pst.log.Infof("Starting deep copy of %d peers...", len(pst.peers))

	now := time.Now()
	endTime := now
	duration := endTime.Sub(pst.startTime)

	// Create a deep copy of peers to avoid data races
	peers := make(map[string]*PeerStats)
	for peerID, peer := range pst.peers {
		// Deep copy connection sessions
		sessionsCopy := make([]ConnectionSession, len(peer.ConnectionSessions))

		for i, session := range peer.ConnectionSessions {
			// Deep copy peer scores
			peerScoresCopy := make([]PeerScoreSnapshot, len(session.PeerScores))

			for j, score := range session.PeerScores {
				// Deep copy topics
				topicsCopy := make([]TopicScore, len(score.Topics))
				copy(topicsCopy, score.Topics)

				peerScoresCopy[j] = PeerScoreSnapshot{
					Timestamp:          score.Timestamp,
					Score:              score.Score,
					AppSpecificScore:   score.AppSpecificScore,
					IPColocationFactor: score.IPColocationFactor,
					BehaviourPenalty:   score.BehaviourPenalty,
					Topics:             topicsCopy,
				}
			}

			// Deep copy goodbye events
			goodbyeEventsCopy := make([]GoodbyeEvent, len(session.GoodbyeEvents))
			copy(goodbyeEventsCopy, session.GoodbyeEvents)

			// Deep copy mesh events
			meshEventsCopy := make([]MeshEvent, len(session.MeshEvents))
			copy(meshEventsCopy, session.MeshEvents)

			sessionCopy := ConnectionSession{
				ConnectedAt:        session.ConnectedAt,
				IdentifiedAt:       session.IdentifiedAt,
				DisconnectedAt:     session.DisconnectedAt,
				MessageCount:       session.MessageCount,
				ConnectionDuration: session.ConnectionDuration,
				Disconnected:       session.Disconnected,
				PeerScores:         peerScoresCopy,
				GoodbyeEvents:      goodbyeEventsCopy,
				MeshEvents:         meshEventsCopy,
			}

			// Calculate connection duration for active sessions
			if !session.Disconnected && session.ConnectedAt != nil {
				sessionCopy.ConnectionDuration = endTime.Sub(*session.ConnectedAt)
			}

			sessionsCopy[i] = sessionCopy
		}

		// Calculate total message count across all sessions
		totalMessageCount := 0
		for _, session := range sessionsCopy {
			totalMessageCount += session.MessageCount
		}

		// Create a copy of the peer stats
		peerCopy := &PeerStats{
			PeerID:             peer.PeerID,
			ClientType:         peer.ClientType,
			ClientAgent:        peer.ClientAgent,
			ConnectionSessions: sessionsCopy,
			TotalConnections:   peer.TotalConnections,
			TotalMessageCount:  totalMessageCount,
			FirstSeenAt:        peer.FirstSeenAt,
			LastSeenAt:         peer.LastSeenAt,
		}

		// Update PeerID if it's empty (use the map key)
		if peerCopy.PeerID == "" {
			peerCopy.PeerID = peerID
		}

		peers[peerID] = peerCopy
	}

	pst.log.Info("Deep copy completed, calculating summary statistics...")

	// Calculate summary statistics
	totalConnections := 0
	successfulHandshakes := 0
	failedHandshakes := 0
	connectedPeers := 0
	clientCounts := make(map[string]int)

	for _, peer := range peers {
		// Count all connection sessions
		totalConnections += peer.TotalConnections

		// Count successful/failed handshakes per session
		hasActiveSession := false

		for _, session := range peer.ConnectionSessions {
			if session.IdentifiedAt != nil {
				successfulHandshakes++
			} else if session.ConnectedAt != nil {
				// Connected but never identified = failed handshake
				failedHandshakes++
			}

			// Check if peer has an active (non-disconnected) session
			if !session.Disconnected {
				hasActiveSession = true
			}
		}

		if hasActiveSession {
			connectedPeers++
		}

		if peer.ClientType != "" {
			clientCounts[peer.ClientType]++
		}
	}

	// Create a deep copy of peerEventCounts to avoid data races
	peerEventCounts := make(map[string]map[string]int)
	for peerID, eventCounts := range pst.peerEventCounts {
		peerEventCounts[peerID] = make(map[string]int)
		for eventType, count := range eventCounts {
			peerEventCounts[peerID][eventType] = count
		}
	}

	// Get validation configuration for this mode
	validationConfig := GetValidationConfigs()[pst.config.ValidationMode]

	report := PeerScoreReport{
		Config:               pst.config,
		ValidationMode:       pst.config.ValidationMode,
		ValidationConfig:     validationConfig,
		Timestamp:            now,
		StartTime:            pst.startTime,
		EndTime:              endTime,
		Duration:             duration,
		TotalConnections:     totalConnections,
		SuccessfulHandshakes: successfulHandshakes,
		FailedHandshakes:     failedHandshakes,
		Peers:                peers,
		PeerEventCounts:      peerEventCounts,
	}

	pst.log.WithFields(map[string]interface{}{
		"validation_mode":       pst.config.ValidationMode,
		"total_connections":     totalConnections,
		"successful_handshakes": successfulHandshakes,
		"failed_handshakes":     failedHandshakes,
		"connected_peers":       connectedPeers,
		"test_duration":         duration,
		"client_types":          clientCounts,
	}).Info("Report generation complete")

	return report
}
