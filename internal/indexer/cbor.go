// Package indexer provides filtering and processing of AT Protocol records
// for the Subcults Jetstream indexer.
package indexer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// AT Protocol CBOR parsing errors.
var (
	ErrInvalidCBOR      = errors.New("invalid CBOR data")
	ErrMissingDID       = errors.New("missing DID in commit")
	ErrMissingPath      = errors.New("missing record path in commit")
	ErrMissingRecord    = errors.New("missing record data in commit")
	ErrInvalidRecordCID = errors.New("invalid record CID format")
)

// AtProtoCommit represents an AT Protocol commit object received from Jetstream.
// Commits contain metadata about record operations (create, update, delete).
type AtProtoCommit struct {
	// DID is the decentralized identifier of the repo owner
	DID string `cbor:"did"`

	// Rev is the revision string for this commit
	Rev string `cbor:"rev"`

	// Operation type: "create", "update", or "delete"
	Operation string `cbor:"operation"`

	// Collection is the lexicon type (e.g., "app.subcult.scene")
	Collection string `cbor:"collection"`

	// RKey is the record key (unique within the collection)
	RKey string `cbor:"rkey"`

	// Record contains the CBOR-encoded record data (for create/update)
	Record cbor.RawMessage `cbor:"record,omitempty"`

	// CID is the content identifier for the record
	CID interface{} `cbor:"cid,omitempty"`
}

// JetstreamMessage represents the top-level message structure from Jetstream.
// Jetstream sends messages containing commits, identity updates, and other events.
type JetstreamMessage struct {
	// DID of the repo that generated this event
	DID string `cbor:"did"`

	// TimeUS is the timestamp in microseconds
	TimeUS int64 `cbor:"time_us"`

	// Kind is the message type ("commit", "identity", "account")
	Kind string `cbor:"kind"`

	// Commit contains the commit data (when Kind == "commit")
	Commit *AtProtoCommit `cbor:"commit,omitempty"`
}

// ParsedRecord represents a successfully parsed AT Protocol record.
type ParsedRecord struct {
	DID        string // Decentralized identifier of the record owner
	Collection string // Lexicon collection type
	RKey       string // Record key
	Rev        string // Revision string for this commit
	Operation  string // Operation type: create, update, delete
	Record     []byte // JSON-encoded record payload (decoded from CBOR)
}

// DecodeCBORMessage decodes a CBOR-encoded Jetstream message.
// Returns the parsed message or an error if decoding fails.
func DecodeCBORMessage(data []byte) (*JetstreamMessage, error) {
	if len(data) == 0 {
		return nil, ErrInvalidCBOR
	}

	var msg JetstreamMessage
	dec := cbor.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&msg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCBOR, err)
	}

	return &msg, nil
}

// DecodeCBORCommit decodes a CBOR-encoded AT Protocol commit.
// Returns the parsed commit or an error if decoding fails.
func DecodeCBORCommit(data []byte) (*AtProtoCommit, error) {
	if len(data) == 0 {
		return nil, ErrInvalidCBOR
	}

	var commit AtProtoCommit
	dec := cbor.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&commit); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCBOR, err)
	}

	// Validate required fields
	if commit.DID == "" {
		return nil, ErrMissingDID
	}
	if commit.Collection == "" {
		return nil, ErrMissingPath
	}

	return &commit, nil
}

// ParseRecord extracts and validates a record from a Jetstream message.
// It decodes the CBOR commit and converts the record payload to JSON for validation.
func ParseRecord(data []byte) (*ParsedRecord, error) {
	msg, err := DecodeCBORMessage(data)
	if err != nil {
		return nil, err
	}

	// Only process commit messages
	if msg.Kind != "commit" {
		return nil, fmt.Errorf("unsupported message kind: %s", msg.Kind)
	}

	if msg.Commit == nil {
		return nil, ErrMissingRecord
	}

	commit := msg.Commit

	// Validate required fields
	if commit.DID == "" {
		return nil, ErrMissingDID
	}
	if commit.Collection == "" {
		return nil, ErrMissingPath
	}

	// For delete operations, record data is optional
	if commit.Operation == "delete" {
		return &ParsedRecord{
			DID:        commit.DID,
			Collection: commit.Collection,
			RKey:       commit.RKey,
			Rev:        commit.Rev,
			Operation:  commit.Operation,
			Record:     nil,
		}, nil
	}

	// For create/update, decode the CBOR record to JSON
	if len(commit.Record) == 0 {
		return nil, ErrMissingRecord
	}

	// Decode CBOR record with options to ensure JSON compatibility
	var recordData interface{}
	dm, err := cbor.DecOptions{
		// Allow byte-string keys in maps for JSON compatibility
		MapKeyByteString: cbor.MapKeyByteStringAllowed,
	}.DecMode()
	if err != nil {
		return nil, fmt.Errorf("failed to create CBOR decoder: %w", err)
	}

	if err := dm.Unmarshal(commit.Record, &recordData); err != nil {
		return nil, fmt.Errorf("failed to decode record CBOR: %w", err)
	}

	// Convert interface{} maps to string-keyed maps for JSON compatibility
	recordData = convertToStringKeyedMaps(recordData)

	// Re-encode as JSON for compatibility with existing validation
	jsonData, err := json.Marshal(recordData)
	if err != nil {
		return nil, fmt.Errorf("failed to encode record as JSON: %w", err)
	}

	return &ParsedRecord{
		DID:        commit.DID,
		Collection: commit.Collection,
		RKey:       commit.RKey,
		Rev:        commit.Rev,
		Operation:  commit.Operation,
		Record:     jsonData,
	}, nil
}

// EncodeCBOR encodes a value to CBOR bytes.
// This is useful for testing round-trip encoding/decoding.
func EncodeCBOR(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := cbor.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("failed to encode CBOR: %w", err)
	}
	return buf.Bytes(), nil
}

// convertToStringKeyedMaps recursively converts map[interface{}]interface{} to map[string]interface{}
// to make the data JSON-serializable.
func convertToStringKeyedMaps(data interface{}) interface{} {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		// Convert interface{} keyed map to string keyed map
		result := make(map[string]interface{}, len(v))
		for key, value := range v {
			// Convert key to string
			var strKey string
			switch k := key.(type) {
			case string:
				strKey = k
			case []byte:
				// Interpret CBOR byte-string keys as UTF-8/text keys
				strKey = string(k)
			default:
				// Fallback for other key types
				strKey = fmt.Sprintf("%v", k)
			}
			// Recursively convert nested values
			result[strKey] = convertToStringKeyedMaps(value)
		}
		return result
	case map[string]interface{}:
		// Already string-keyed, just recursively convert values
		result := make(map[string]interface{}, len(v))
		for key, value := range v {
			result[key] = convertToStringKeyedMaps(value)
		}
		return result
	case []interface{}:
		// Convert array elements
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertToStringKeyedMaps(item)
		}
		return result
	default:
		// Primitive types, return as-is
		return v
	}
}
