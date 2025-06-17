package peer

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/constants"
)

func TestInMemoryRepository(t *testing.T) {
	logger := logrus.New()
	repo := NewInMemoryRepository(logger)

	// Test creating a new peer
	peerID := "test-peer-123"
	peer := repo.CreatePeer(peerID)

	if peer.PeerID != peerID {
		t.Errorf("Expected peer ID %s, got %s", peerID, peer.PeerID)
	}

	if peer.FirstSeenAt == nil {
		t.Error("Expected FirstSeenAt to be set")
	}

	// Test getting the peer
	retrievedPeer, exists := repo.GetPeer(peerID)
	if !exists {
		t.Error("Expected peer to exist")
	}

	if retrievedPeer.PeerID != peerID {
		t.Errorf("Expected retrieved peer ID %s, got %s", peerID, retrievedPeer.PeerID)
	}

	// Test updating a peer
	updateCalled := false
	repo.UpdatePeer(peerID, func(p *Stats) {
		p.ClientType = constants.Lighthouse
		updateCalled = true
	})

	if !updateCalled {
		t.Error("Expected update function to be called")
	}

	// Verify the update
	updatedPeer, _ := repo.GetPeer(peerID)
	if updatedPeer.ClientType != constants.Lighthouse {
		t.Errorf("Expected client type 'lighthouse', got '%s'", updatedPeer.ClientType)
	}

	// Test event counting
	repo.IncrementEventCount(peerID, "CONNECTED")
	repo.IncrementEventCount(peerID, "CONNECTED")
	repo.IncrementEventCount(peerID, "PEERSCORE")

	eventCounts := repo.GetPeerEventCounts()
	if eventCounts[peerID]["CONNECTED"] != 2 {
		t.Errorf("Expected 2 CONNECTED events, got %d", eventCounts[peerID]["CONNECTED"])
	}

	if eventCounts[peerID]["PEERSCORE"] != 1 {
		t.Errorf("Expected 1 PEERSCORE event, got %d", eventCounts[peerID]["PEERSCORE"])
	}

	// Test getting all peers
	allPeers := repo.GetAllPeers()
	if len(allPeers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(allPeers))
	}

	// Test peer count
	count := repo.GetPeerCount()
	if count != 1 {
		t.Errorf("Expected peer count 1, got %d", count)
	}
}

func TestDeepCopy(t *testing.T) {
	logger := logrus.New()
	repo := NewInMemoryRepository(logger)

	peerID := "test-peer"
	_ = repo.CreatePeer(peerID)

	// Add a session with some data
	now := time.Now()
	session := ConnectionSession{
		ConnectedAt:   &now,
		IdentifiedAt:  &now,
		MessageCount:  10,
		Disconnected:  false,
		PeerScores:    []PeerScoreSnapshot{{Score: 1.5, Timestamp: now}},
		GoodbyeEvents: []GoodbyeEvent{{Code: 1, Timestamp: now}},
		MeshEvents:    []MeshEvent{{Type: "GRAFT", Timestamp: now}},
	}

	repo.UpdatePeer(peerID, func(p *Stats) {
		p.ConnectionSessions = append(p.ConnectionSessions, session)
	})

	// Get all peers (which should return a deep copy)
	allPeers := repo.GetAllPeers()
	copiedPeer := allPeers[peerID]

	// Modify the copied peer
	copiedPeer.ClientType = "modified"
	copiedPeer.ConnectionSessions[0].MessageCount = 999

	// Verify original is unchanged
	originalPeer, _ := repo.GetPeer(peerID)
	if originalPeer.ClientType == "modified" {
		t.Error("Deep copy failed: original peer was modified")
	}

	if originalPeer.ConnectionSessions[0].MessageCount == 999 {
		t.Error("Deep copy failed: original session was modified")
	}
}

func TestConcurrentAccess(t *testing.T) {
	logger := logrus.New()
	repo := NewInMemoryRepository(logger)

	peerID := "concurrent-peer"
	repo.CreatePeer(peerID)

	done := make(chan bool, 2)

	// Concurrent updates
	go func() {
		for i := 0; i < 100; i++ {
			repo.UpdatePeer(peerID, func(p *Stats) {
				p.TotalConnections++
			})
		}
		done <- true
	}()

	// Concurrent event counting
	go func() {
		for i := 0; i < 100; i++ {
			repo.IncrementEventCount(peerID, "TEST_EVENT")
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state
	peer, _ := repo.GetPeer(peerID)
	if peer.TotalConnections != 100 {
		t.Errorf("Expected 100 connections, got %d", peer.TotalConnections)
	}

	eventCounts := repo.GetPeerEventCounts()
	if eventCounts[peerID]["TEST_EVENT"] != 100 {
		t.Errorf("Expected 100 events, got %d", eventCounts[peerID]["TEST_EVENT"])
	}
}
