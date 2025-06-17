package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/p2p/encoder"
	"github.com/OffchainLabs/prysm/v6/config/params"
	"github.com/OffchainLabs/prysm/v6/time/slots"
	"github.com/probe-lab/hermes/eth"
	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"github.com/ethpandaops/hermes-peer-score/internal/config"
)

// DefaultHermesController implements the HermesController interface.
type DefaultHermesController struct {
	config        config.Config
	logger        logrus.FieldLogger
	node          *eth.Node
	callback      func(ctx context.Context, event interface{}) error
	networkConfig *params.NetworkConfig
	beaconConfig  *params.BeaconChainConfig
}

// NewHermesController creates a new Hermes controller.
func NewHermesController(cfg config.Config, logger logrus.FieldLogger) *DefaultHermesController {
	return &DefaultHermesController{
		config: cfg,
		logger: logger.WithField("component", "hermes_controller"),
	}
}

// Start initializes and starts the Hermes node.
func (hc *DefaultHermesController) Start(ctx context.Context) error {
	hc.logger.Info("Starting Hermes node")

	// Derive network configuration
	c, err := eth.DeriveKnownNetworkConfig(ctx, params.MainnetName)
	if err != nil {
		return fmt.Errorf("get config for %s: %w", params.MainnetName, err)
	}

	hc.networkConfig = c.Network
	hc.beaconConfig = c.Beacon

	genesisRoot := c.Genesis.GenesisValidatorRoot
	genesisTime := c.Genesis.GenesisTime

	// Compute fork version and fork digest
	currentSlot := slots.Since(genesisTime)
	currentEpoch := slots.ToEpoch(currentSlot)

	currentForkVersion, err := eth.GetCurrentForkVersion(currentEpoch, hc.beaconConfig)
	if err != nil {
		return fmt.Errorf("compute fork version for epoch %d: %w", currentEpoch, err)
	}

	forkDigest, err := signing.ComputeForkDigest(currentForkVersion[:], genesisRoot)
	if err != nil {
		return fmt.Errorf("create fork digest (%s, %x): %w", genesisTime, genesisRoot, err)
	}

	// Override global configuration
	params.OverrideBeaconConfig(hc.beaconConfig)
	params.OverrideBeaconNetworkConfig(hc.networkConfig)

	// Create Hermes configuration
	hermesConfig := hc.createHermesConfig(forkDigest, currentForkVersion)
	hermesConfig.GenesisConfig = c.Genesis

	if err = hermesConfig.Validate(); err != nil {
		return fmt.Errorf("invalid Hermes node config: %w", err)
	}

	// Create the node
	node, err := eth.NewNode(hermesConfig)
	if err != nil {
		if strings.Contains(err.Error(), "in correct fork_digest") {
			return fmt.Errorf("invalid fork digest (config.ethereum.network and prysm network probably don't match): %w", err)
		}

		return err
	}

	hc.node = node

	// Register event callback
	hc.node.OnEvent(func(ctx context.Context, event *host.TraceEvent) {
		if hc.callback != nil {
			if err := hc.callback(ctx, event); err != nil {
				hc.logger.WithError(err).Error("Event callback failed")
			}
		}
	})

	// Start the node in a goroutine
	go func() {
		// Blackhole hermes logs by redirecting the default logger
		originalOutput := log.Writer()
		log.SetOutput(io.Discard)
		defer log.SetOutput(originalOutput)

		if err := hc.node.Start(ctx); err != nil {
			hc.logger.WithError(err).Fatal("Failed to start hermes")
		}
	}()

	hc.logger.Info("Hermes node started successfully")

	return nil
}

// Stop gracefully shuts down the Hermes node.
func (hc *DefaultHermesController) Stop() error {
	hc.logger.Info("Stopping Hermes node")

	if hc.node != nil {
		// Hermes doesn't have an explicit stop method, so we rely on context cancellation
		hc.logger.Info("Hermes node shutdown initiated")
	}

	return nil
}

// RegisterEventCallback sets the callback function for processing events.
func (hc *DefaultHermesController) RegisterEventCallback(callback func(ctx context.Context, event interface{}) error) {
	hc.callback = callback
	hc.logger.Debug("Event callback registered")
}

// GetNode returns the underlying Hermes node.
func (hc *DefaultHermesController) GetNode() interface{} {
	return hc.node
}

// createHermesConfig creates the Hermes node configuration.
func (hc *DefaultHermesController) createHermesConfig(forkDigest [4]byte, currentForkVersion [4]byte) *eth.NodeConfig {
	cfg := hc.config.AsHermesConfig()
	// Genesis config will be set by the caller
	cfg.NetworkConfig = hc.networkConfig
	cfg.BeaconConfig = hc.beaconConfig
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
	hc.applyValidationConfig(cfg)

	return cfg
}

// applyValidationConfig applies validation-specific configuration overrides.
func (hc *DefaultHermesController) applyValidationConfig(cfg *eth.NodeConfig) {
	validationMode := hc.config.GetValidationMode()

	hc.logger.WithField("validation_mode", validationMode).Info("Applying validation-specific configuration")

	switch validationMode {
	case "independent":
		hc.logger.Info("Configuring independent validation mode")
	case "delegated":
		hc.logger.Info("Configuring delegated validation mode")

		// GRPC port only used in delegated mode
		if cfg.PrysmPortGRPC == 0 {
			hc.logger.Warn("Prysm gRPC port not configured")
		}
	default:
		hc.logger.Warnf("Unknown validation mode: %s", validationMode)

		return
	}

	// Validate Prysm connection is properly configured (both modes need it)
	if cfg.PrysmHost == "" {
		hc.logger.Warn("Prysm host not configured - needed for validation mode")
	} else {
		hc.logger.WithField("prysm_host", cfg.PrysmHost).Info("Prysm connection configured")
	}

	// Validate connection parameters
	if cfg.PrysmPortHTTP == 0 {
		hc.logger.Warn("Prysm HTTP port not configured")
	}
}
