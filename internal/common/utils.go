package common

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/probe-lab/hermes/host"
	
	"github.com/ethpandaops/hermes-peer-score/constants"
)

const unknown = constants.Unknown


// GetPeerID extracts the peer ID from a trace event
func GetPeerID(event *host.TraceEvent) string {
	if event == nil || event.Payload == nil {
		return unknown
	}

	peerID := extractPeerIDFromStruct(event.Payload)
	if peerID == "" {
		return unknown
	}

	// Debug: log both the raw peerID and whether it looks like binary data
	if len(peerID) > 20 && strings.Contains(peerID, "\u0000") {
		// This looks like binary data, try to extract it properly using the old method
		if payload, ok := event.Payload.(map[string]any); ok {
			if remotePeerID, found := payload["PeerID"]; found {
				converted := fmt.Sprintf("%v", remotePeerID)
				if converted != peerID && len(converted) > 20 {
					return converted
				}
			}
			if remotePeerID, found := payload["RemotePeer"]; found {
				converted := fmt.Sprintf("%v", remotePeerID)
				if converted != peerID && len(converted) > 20 {
					return converted
				}
			}
		}
	}

	return peerID
}

// extractPeerIDFromStruct extracts peer ID from various payload structures using reflection
func extractPeerIDFromStruct(payload interface{}) string {
	if payload == nil {
		return ""
	}

	val := reflect.ValueOf(payload)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return ""
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		return extractFromStruct(val)
	case reflect.Map:
		return extractFromMap(val)
	default:
		return ""
	}
}

// extractFromStruct extracts peer ID from struct fields
func extractFromStruct(val reflect.Value) string {
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldName := typ.Field(i).Name

		// Check common peer ID field names
		if isPeerIDField(fieldName) {
			if peerID := extractPeerIDValue(field); peerID != "" {
				return peerID
			}
		}

		// Recursively check nested structs
		if field.Kind() == reflect.Struct || (field.Kind() == reflect.Ptr && !field.IsNil()) {
			if peerID := extractPeerIDFromStruct(field.Interface()); peerID != "" {
				return peerID
			}
		}
	}
	return ""
}

// extractFromMap extracts peer ID from map values
func extractFromMap(val reflect.Value) string {
	if val.Kind() != reflect.Map {
		return ""
	}

	for _, key := range val.MapKeys() {
		keyStr := ""
		if key.Kind() == reflect.String {
			keyStr = key.String()
		}

		if isPeerIDField(keyStr) {
			mapVal := val.MapIndex(key)
			if peerID := extractPeerIDValue(mapVal); peerID != "" {
				return peerID
			}
		}
	}
	return ""
}

// isPeerIDField checks if a field name indicates it contains a peer ID
func isPeerIDField(fieldName string) bool {
	lowerName := strings.ToLower(fieldName)
	return lowerName == "peerid" || 
		   lowerName == "peer_id" || 
		   lowerName == "remotepeer" || 
		   lowerName == "remote_peer"
}

// extractPeerIDValue extracts the actual peer ID string from a field value
func extractPeerIDValue(field reflect.Value) string {
	if !field.IsValid() {
		return ""
	}

	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Ptr:
		if !field.IsNil() {
			return extractPeerIDValue(field.Elem())
		}
	case reflect.Interface:
		if !field.IsNil() {
			return extractPeerIDValue(reflect.ValueOf(field.Interface()))
		}
	case reflect.Struct:
		// Check if struct has a GetValue() method (common in protobuf)
		if method := field.MethodByName("GetValue"); method.IsValid() {
			results := method.Call(nil)
			if len(results) > 0 && results[0].Kind() == reflect.String {
				return results[0].String()
			}
		}
		// Check if struct has a Value field
		if valueField := field.FieldByName("Value"); valueField.IsValid() {
			return extractPeerIDValue(valueField)
		}
	}

	return ""
}

// NormalizeClientType normalizes client agent strings to standard types
func NormalizeClientType(clientAgent string) string {
	if clientAgent == "" {
		return unknown
	}

	agent := strings.ToLower(clientAgent)
	
	switch {
	case strings.Contains(agent, constants.Lighthouse):
		return constants.Lighthouse
	case strings.Contains(agent, constants.Prysm):
		return constants.Prysm
	case strings.Contains(agent, constants.Teku):
		return constants.Teku
	case strings.Contains(agent, constants.Nimbus):
		return constants.Nimbus
	case strings.Contains(agent, constants.Lodestar):
		return constants.Lodestar
	case strings.Contains(agent, constants.Grandine):
		return constants.Grandine
	default:
		// Try to extract the first word as client type
		parts := strings.Fields(agent)
		if len(parts) > 0 {
			return parts[0]
		}
		return unknown
	}
}

// FormatShortPeerID returns a shortened version of the peer ID for logging
func FormatShortPeerID(peerID string) string {
	if len(peerID) <= 12 {
		return peerID
	}
	return peerID[:12]
}