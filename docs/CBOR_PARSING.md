# CBOR Record Parsing for AT Protocol

This package provides CBOR decoding and validation for AT Protocol records received from Jetstream.

## Overview

AT Protocol uses CBOR (Concise Binary Object Representation) for efficient data transmission. This implementation parses CBOR-encoded commits from Jetstream and converts them to validated JSON records.

## Usage

### Basic CBOR Parsing

```go
import "github.com/onnwee/subcults/internal/indexer"

// Raw CBOR data from Jetstream WebSocket
cborData := []byte{...}

// Parse the CBOR message
parsed, err := indexer.ParseRecord(cborData)
if err != nil {
    log.Printf("failed to parse CBOR: %v", err)
    return
}

// Access parsed fields
fmt.Printf("DID: %s\n", parsed.DID)
fmt.Printf("Collection: %s\n", parsed.Collection)
fmt.Printf("Operation: %s\n", parsed.Operation)
fmt.Printf("Record: %s\n", string(parsed.Record))
```

### Filtering and Validation

The `FilterCBOR` method provides end-to-end processing:

```go
import "github.com/onnwee/subcults/internal/indexer"

// Create filter with metrics
metrics := indexer.NewFilterMetrics()
filter := indexer.NewRecordFilter(metrics)

// Process CBOR message
result := filter.FilterCBOR(cborData)

if !result.Matched {
    // Not a subcult record, ignore
    return
}

if !result.Valid {
    log.Printf("validation failed: %v", result.Error)
    return
}

// Handle valid record
switch result.Collection {
case indexer.CollectionScene:
    handleScene(result)
case indexer.CollectionEvent:
    handleEvent(result)
case indexer.CollectionPost:
    handlePost(result)
}
```

## Supported Record Types

- **Scenes** (`app.subcult.scene`): Music venues and communities
- **Events** (`app.subcult.event`): Shows and performances
- **Posts** (`app.subcult.post`): Community updates
- **Alliances** (`app.subcult.alliance`): Scene relationships

## Operations

- **create**: New record creation
- **update**: Record modification
- **delete**: Record deletion (no payload validation)

## Data Flow

```
Jetstream WebSocket
    ↓
CBOR-encoded message
    ↓
DecodeCBORMessage()
    ↓
Extract commit + metadata
    ↓
Decode CBOR record → JSON
    ↓
FilterCBOR()
    ↓
Validate against schema
    ↓
Validated FilterResult
```

## Error Handling

Common errors:

- `ErrInvalidCBOR`: Malformed CBOR data
- `ErrMissingDID`: Commit missing DID field
- `ErrMissingPath`: Commit missing collection field
- `ErrMissingRecord`: Create/update without record payload
- `ErrNonMatchingLexicon`: Not an app.subcult.* collection
- `ErrMissingField`: Record missing required field
- `ErrInvalidFieldType`: Field has wrong type

## Testing

Run tests with:

```bash
go test ./internal/indexer/ -v
go test ./internal/indexer/ -race -cover
```

Test coverage for this package is monitored via the standard Go coverage tooling; ensure new changes preserve strong coverage, especially around parsing, filtering, and validation paths.

## Example Messages

### Scene Create

```go
msg := indexer.JetstreamMessage{
    DID:    "did:plc:user123",
    TimeUS: 1234567890,
    Kind:   "commit",
    Commit: &indexer.AtProtoCommit{
        DID:        "did:plc:user123",
        Rev:        "abc123",
        Operation:  "create",
        Collection: "app.subcult.scene",
        RKey:       "scene1",
        Record:     cborEncodedData, // {"name": "Underground Techno"}
    },
}
```

### Event Update

```go
msg := indexer.JetstreamMessage{
    DID:    "did:plc:user456",
    TimeUS: 1234567890,
    Kind:   "commit",
    Commit: &indexer.AtProtoCommit{
        DID:        "did:plc:user456",
        Rev:        "def456",
        Operation:  "update",
        Collection: "app.subcult.event",
        RKey:       "event1",
        Record:     cborEncodedData, // {"name": "Friday Night", "sceneId": "s1"}
    },
}
```

### Scene Delete

```go
msg := indexer.JetstreamMessage{
    DID:    "did:plc:user789",
    TimeUS: 1234567890,
    Kind:   "commit",
    Commit: &indexer.AtProtoCommit{
        DID:        "did:plc:user789",
        Rev:        "ghi789",
        Operation:  "delete",
        Collection: "app.subcult.scene",
        RKey:       "scene1",
        // No Record field for deletes
    },
}
```

## Implementation Notes

### CBOR to JSON Conversion

CBOR maps use `interface{}` keys by default, which JSON cannot encode. This implementation automatically converts `map[interface{}]interface{}` to `map[string]interface{}` for JSON compatibility.

### Privacy Compliance

After CBOR parsing, records should still enforce location consent rules via `EnforceLocationConsent()` before persistence.

### Performance

These parsing and validation routines are designed to be low-latency and suitable for real-time Jetstream consumption. Actual timings are highly dependent on hardware, load, and Go runtime configuration.

For up-to-date benchmark results, refer to the performance baselines and CI artifacts (see the `perf/` directory and associated tooling in this repository).

### Future Enhancements

- Streaming CBOR decoder for large payloads
- Schema caching for faster validation
- Binary record storage (keep CBOR, validate on read)
