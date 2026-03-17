// Package indexer provides filtering and processing of AT Protocol records
// for the Subcults Jetstream indexer.
package indexer

import (
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
)

// LexiconPrefix is the namespace prefix for Subcults domain records.
const LexiconPrefix = "app.subcult."

// Supported lexicon collection types within the app.subcult namespace.
const (
	CollectionScene    = "app.subcult.scene"
	CollectionEvent    = "app.subcult.event"
	CollectionPost     = "app.subcult.post"
	CollectionAlliance = "app.subcult.alliance"
)

// Errors for record filtering and validation.
var (
	ErrNonMatchingLexicon = errors.New("lexicon does not match app.subcult.* namespace")
	ErrMalformedJSON      = errors.New("malformed JSON payload")
	ErrMissingField       = errors.New("required field missing")
	ErrInvalidFieldType   = errors.New("field has invalid type")
)

// FilterMetrics tracks counts for record filtering operations.
// All operations are thread-safe using atomic counters.
type FilterMetrics struct {
	processed int64 // Total records received
	matched   int64 // Records matching app.subcult.* prefix
	discarded int64 // Records discarded due to validation errors
}

// NewFilterMetrics creates a new FilterMetrics instance.
func NewFilterMetrics() *FilterMetrics {
	return &FilterMetrics{}
}

// Processed returns the total number of records received.
func (m *FilterMetrics) Processed() int64 {
	return atomic.LoadInt64(&m.processed)
}

// Matched returns the number of records matching the lexicon prefix.
func (m *FilterMetrics) Matched() int64 {
	return atomic.LoadInt64(&m.matched)
}

// Discarded returns the number of records discarded due to validation errors.
func (m *FilterMetrics) Discarded() int64 {
	return atomic.LoadInt64(&m.discarded)
}

// incProcessed atomically increments the processed counter.
func (m *FilterMetrics) incProcessed() {
	atomic.AddInt64(&m.processed, 1)
}

// incMatched atomically increments the matched counter.
func (m *FilterMetrics) incMatched() {
	atomic.AddInt64(&m.matched, 1)
}

// incDiscarded atomically increments the discarded counter.
func (m *FilterMetrics) incDiscarded() {
	atomic.AddInt64(&m.discarded, 1)
}

// RecordFilter filters and validates AT Protocol records for the app.subcult.* namespace.
type RecordFilter struct {
	metrics *FilterMetrics
}

// NewRecordFilter creates a new RecordFilter with the given metrics collector.
func NewRecordFilter(metrics *FilterMetrics) *RecordFilter {
	return &RecordFilter{
		metrics: metrics,
	}
}

// MatchesLexicon checks if the given collection path matches the app.subcult.* namespace.
func MatchesLexicon(collection string) bool {
	return strings.HasPrefix(collection, LexiconPrefix)
}

// FilterResult represents the result of filtering and validating a record.
type FilterResult struct {
	Matched    bool            // Whether the record matched the lexicon prefix
	Valid      bool            // Whether the record passed validation
	Collection string          // The collection type (e.g., "app.subcult.scene")
	Record     json.RawMessage // The validated record JSON
	DID        string          // Decentralized identifier of the record owner (from CBOR parsing)
	RKey       string          // Record key (from CBOR parsing)
	Rev        string          // Revision string for this commit (from CBOR parsing)
	Operation  string          // Operation type: create, update, delete (from CBOR parsing)
	Error      error           // Validation error, if any
}

// Filter processes a record, checking if it matches the app.subcult.* namespace
// and validating the JSON payload if it matches.
func (f *RecordFilter) Filter(collection string, payload []byte) FilterResult {
	f.metrics.incProcessed()

	result := FilterResult{
		Collection: collection,
	}

	// Check if collection matches our lexicon namespace
	if !MatchesLexicon(collection) {
		result.Matched = false
		result.Error = ErrNonMatchingLexicon
		return result
	}

	result.Matched = true
	f.metrics.incMatched()

	// Validate the JSON payload based on collection type
	if err := f.validateRecord(collection, payload); err != nil {
		result.Valid = false
		result.Error = err
		f.metrics.incDiscarded()
		return result
	}

	result.Valid = true
	result.Record = payload
	return result
}

// FilterCBOR processes a CBOR-encoded AT Protocol message from Jetstream.
// It parses the CBOR, extracts the record, validates it, and returns a FilterResult.
// This is the primary method for processing real Jetstream messages.
func (f *RecordFilter) FilterCBOR(cborData []byte) FilterResult {
	f.metrics.incProcessed()

	result := FilterResult{}

	// Parse CBOR message
	parsed, err := ParseRecord(cborData)
	if err != nil {
		result.Matched = false
		result.Valid = false
		result.Error = err
		return result
	}

	// Populate result with parsed data
	result.DID = parsed.DID
	result.Collection = parsed.Collection
	result.RKey = parsed.RKey
	result.Rev = parsed.Rev
	result.Operation = parsed.Operation

	// Check if collection matches our lexicon namespace
	if !MatchesLexicon(parsed.Collection) {
		result.Matched = false
		result.Error = ErrNonMatchingLexicon
		return result
	}

	result.Matched = true
	f.metrics.incMatched()

	// For delete operations, no validation needed
	if parsed.Operation == "delete" {
		result.Valid = true
		result.Record = nil
		return result
	}

	// Validate the record payload
	if err := f.validateRecord(parsed.Collection, parsed.Record); err != nil {
		result.Valid = false
		result.Error = err
		f.metrics.incDiscarded()
		return result
	}

	result.Valid = true
	result.Record = parsed.Record
	return result
}

// validateRecord validates the JSON payload based on the collection type.
func (f *RecordFilter) validateRecord(collection string, payload []byte) error {
	switch collection {
	case CollectionScene:
		return validateSceneRecord(payload)
	case CollectionEvent:
		return validateEventRecord(payload)
	case CollectionPost:
		return validatePostRecord(payload)
	case CollectionAlliance:
		return validateAllianceRecord(payload)
	default:
		// For unknown app.subcult.* collections, just validate JSON syntax
		return validateJSONSyntax(payload)
	}
}

// validateJSONSyntax checks if the payload is valid JSON.
func validateJSONSyntax(payload []byte) error {
	var js json.RawMessage
	if err := json.Unmarshal(payload, &js); err != nil {
		return ErrMalformedJSON
	}
	return nil
}

// validateStringField checks if a required string field exists and has the correct type.
func validateStringField(record map[string]interface{}, fieldName string) error {
	value, exists := record[fieldName]
	if !exists {
		return ErrMissingField
	}
	if _, ok := value.(string); !ok {
		return ErrInvalidFieldType
	}
	return nil
}

// SceneRecord documents the expected structure of an app.subcult.scene record.
// Note: This type is for documentation only. Validation uses map-based checking
// to allow extra fields while enforcing required field presence and types.
type SceneRecord struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// validateSceneRecord validates a scene record's required fields.
func validateSceneRecord(payload []byte) error {
	var record map[string]interface{}
	if err := json.Unmarshal(payload, &record); err != nil {
		return ErrMalformedJSON
	}
	return validateStringField(record, "name")
}

// EventRecord documents the expected structure of an app.subcult.event record.
// Note: This type is for documentation only. Validation uses map-based checking
// to allow extra fields while enforcing required field presence and types.
type EventRecord struct {
	Name    string `json:"name"`
	SceneID string `json:"sceneId"`
}

// validateEventRecord validates an event record's required fields.
func validateEventRecord(payload []byte) error {
	var record map[string]interface{}
	if err := json.Unmarshal(payload, &record); err != nil {
		return ErrMalformedJSON
	}
	if err := validateStringField(record, "name"); err != nil {
		return err
	}
	return validateStringField(record, "sceneId")
}

// PostRecord documents the expected structure of an app.subcult.post record.
// Note: This type is for documentation only. Validation uses map-based checking
// to allow extra fields while enforcing required field presence and types.
// At least one of SceneID or EventID must be present.
type PostRecord struct {
	Text    string  `json:"text"`
	SceneID *string `json:"sceneId,omitempty"`
	EventID *string `json:"eventId,omitempty"`
}

// validatePostRecord validates a post record's required fields.
// Requires: text field (mandatory)
// Requires: at least one of sceneId or eventId (not both can be missing)
func validatePostRecord(payload []byte) error {
	var record map[string]interface{}
	if err := json.Unmarshal(payload, &record); err != nil {
		return ErrMalformedJSON
	}
	
	// Validate required text field
	if err := validateStringField(record, "text"); err != nil {
		return err
	}
	
	// At least one of sceneId or eventId must be present
	hasSceneID := false
	hasEventID := false
	
	if sceneID, exists := record["sceneId"]; exists {
		if _, ok := sceneID.(string); ok {
			hasSceneID = true
		}
	}
	
	if eventID, exists := record["eventId"]; exists {
		if _, ok := eventID.(string); ok {
			hasEventID = true
		}
	}
	
	if !hasSceneID && !hasEventID {
		return ErrMissingField
	}
	
	return nil
}

// AllianceRecord documents the expected structure of an app.subcult.alliance record.
// Note: This type is for documentation only. Validation uses map-based checking
// to allow extra fields while enforcing required field presence and types.
type AllianceRecord struct {
	FromSceneID string `json:"fromSceneId"`
	ToSceneID   string `json:"toSceneId"`
}

// validateAllianceRecord validates an alliance record's required fields.
func validateAllianceRecord(payload []byte) error {
	var record map[string]interface{}
	if err := json.Unmarshal(payload, &record); err != nil {
		return ErrMalformedJSON
	}
	if err := validateStringField(record, "fromSceneId"); err != nil {
		return err
	}
	return validateStringField(record, "toSceneId")
}
