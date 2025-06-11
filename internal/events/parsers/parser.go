package parsers

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// DefaultParser provides common parsing functionality
type DefaultParser struct{}

// ParsePeerScoreFromMap parses peer score data from a map payload
func (p *DefaultParser) ParsePeerScoreFromMap(payload map[string]interface{}) (*PeerScoreData, error) {
	score := &PeerScoreData{
		Timestamp: time.Now(),
		Topics:    make([]TopicScore, 0),
	}

	// Parse main score fields
	if val, ok := payload["Score"]; ok {
		if floatVal, err := parseFloat64(val); err == nil {
			score.Score = floatVal
		}
	}

	if val, ok := payload["AppSpecificScore"]; ok {
		if floatVal, err := parseFloat64(val); err == nil {
			score.AppSpecificScore = floatVal
		}
	}

	if val, ok := payload["IPColocationFactor"]; ok {
		if floatVal, err := parseFloat64(val); err == nil {
			score.IPColocationFactor = floatVal
		}
	}

	if val, ok := payload["BehaviourPenalty"]; ok {
		if floatVal, err := parseFloat64(val); err == nil {
			score.BehaviourPenalty = floatVal
		}
	}

	// Parse topic scores
	if topicsVal, ok := payload["Topics"]; ok {
		if topics, err := p.parseTopicScores(topicsVal); err == nil {
			score.Topics = topics
		}
	}

	return score, nil
}

// ParseGoodbyeFromMap parses goodbye event data from a map payload
func (p *DefaultParser) ParseGoodbyeFromMap(payload map[string]interface{}) (*GoodbyeData, error) {
	goodbye := &GoodbyeData{
		Timestamp: time.Now(),
	}

	if val, ok := payload["Code"]; ok {
		if code, err := parseUint64(val); err == nil {
			goodbye.Code = code
		}
	}

	if val, ok := payload["Reason"]; ok {
		if reason, ok := val.(string); ok {
			goodbye.Reason = reason
		}
	}

	return goodbye, nil
}

// ParseMeshFromMap parses mesh event data from a map payload
func (p *DefaultParser) ParseMeshFromMap(payload map[string]interface{}, eventType string) (*MeshData, error) {
	mesh := &MeshData{
		Timestamp: time.Now(),
		Type:      eventType,
	}

	if val, ok := payload["Direction"]; ok {
		if direction, ok := val.(string); ok {
			mesh.Direction = direction
		}
	}

	if val, ok := payload["Topic"]; ok {
		if topic, ok := val.(string); ok {
			mesh.Topic = topic
		}
	}

	if val, ok := payload["Reason"]; ok {
		if reason, ok := val.(string); ok {
			mesh.Reason = reason
		}
	}

	return mesh, nil
}

// parseTopicScores parses topic score data from various formats
func (p *DefaultParser) parseTopicScores(topicsVal interface{}) ([]TopicScore, error) {
	var topics []TopicScore

	val := reflect.ValueOf(topicsVal)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return topics, nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Map:
		return p.parseTopicScoresFromMap(val)
	case reflect.Slice, reflect.Array:
		return p.parseTopicScoresFromSlice(val)
	default:
		return topics, fmt.Errorf("unsupported topics format: %T", topicsVal)
	}
}

// parseTopicScoresFromMap parses topic scores from a map structure
func (p *DefaultParser) parseTopicScoresFromMap(val reflect.Value) ([]TopicScore, error) {
	var topics []TopicScore

	for _, key := range val.MapKeys() {
		topicName := key.String()
		topicVal := val.MapIndex(key)

		if topicVal.Kind() == reflect.Interface {
			topicVal = topicVal.Elem()
		}

		if topicVal.Kind() == reflect.Map {
			topic := TopicScore{Topic: topicName}
			
			// Parse topic score fields
			for _, topicKey := range topicVal.MapKeys() {
				fieldName := topicKey.String()
				fieldVal := topicVal.MapIndex(topicKey)
				
				if err := p.setTopicScoreField(&topic, fieldName, fieldVal.Interface()); err != nil {
					continue // Skip invalid fields
				}
			}
			
			topics = append(topics, topic)
		}
	}

	return topics, nil
}

// parseTopicScoresFromSlice parses topic scores from a slice structure
func (p *DefaultParser) parseTopicScoresFromSlice(val reflect.Value) ([]TopicScore, error) {
	var topics []TopicScore

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)
		if item.Kind() == reflect.Interface {
			item = item.Elem()
		}

		if item.Kind() == reflect.Map {
			topic := TopicScore{}
			
			for _, key := range item.MapKeys() {
				fieldName := key.String()
				fieldVal := item.MapIndex(key)
				
				if err := p.setTopicScoreField(&topic, fieldName, fieldVal.Interface()); err != nil {
					continue // Skip invalid fields
				}
			}
			
			topics = append(topics, topic)
		}
	}

	return topics, nil
}

// setTopicScoreField sets a field on the TopicScore struct
func (p *DefaultParser) setTopicScoreField(topic *TopicScore, fieldName string, value interface{}) error {
	switch fieldName {
	case "Topic":
		if str, ok := value.(string); ok {
			topic.Topic = str
		}
	case "TimeInMesh":
		if duration, err := parseDuration(value); err == nil {
			topic.TimeInMesh = duration
		}
	case "FirstMessageDeliveries":
		if float, err := parseFloat64(value); err == nil {
			topic.FirstMessageDeliveries = float
		}
	case "MeshMessageDeliveries":
		if float, err := parseFloat64(value); err == nil {
			topic.MeshMessageDeliveries = float
		}
	case "InvalidMessageDeliveries":
		if float, err := parseFloat64(value); err == nil {
			topic.InvalidMessageDeliveries = float
		}
	}
	return nil
}

// parseFloat64 safely converts various types to float64
func parseFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

// parseUint64 safely converts various types to uint64
func parseUint64(val interface{}) (uint64, error) {
	switch v := val.(type) {
	case uint64:
		return v, nil
	case uint32:
		return uint64(v), nil
	case uint:
		return uint64(v), nil
	case int:
		if v >= 0 {
			return uint64(v), nil
		}
		return 0, fmt.Errorf("negative int cannot be converted to uint64")
	case int32:
		if v >= 0 {
			return uint64(v), nil
		}
		return 0, fmt.Errorf("negative int32 cannot be converted to uint64")
	case int64:
		if v >= 0 {
			return uint64(v), nil
		}
		return 0, fmt.Errorf("negative int64 cannot be converted to uint64")
	case string:
		return strconv.ParseUint(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to uint64", val)
	}
}

// parseDuration safely converts various types to time.Duration
func parseDuration(val interface{}) (time.Duration, error) {
	switch v := val.(type) {
	case time.Duration:
		return v, nil
	case int64:
		return time.Duration(v), nil
	case float64:
		return time.Duration(v), nil
	case string:
		return time.ParseDuration(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to time.Duration", val)
	}
}