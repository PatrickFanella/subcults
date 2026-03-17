# AT Protocol Record Schema for Subcults

This document defines the AT Protocol lexicon schema for Subcults records. These schemas define the structure and validation rules for records ingested by the Jetstream indexer.

## Overview

Subcults uses the AT Protocol namespace `app.subcult.*` for all domain-specific records. The indexer filters and validates these records before persisting them to PostgreSQL.

**Related Components:**
- Jetstream Indexer: `internal/indexer/` ([README](../internal/indexer/README.md))
- Record Filter: `internal/indexer/filter.go`
- Record Validation: `internal/indexer/filter.go` (validation functions)

**Related Issues:**
- #305 (Jetstream Indexer epic)
- #416 (Canonical Roadmap)

## Namespace

All Subcults records use the lexicon prefix:

```
app.subcult.*
```

The indexer filters incoming AT Protocol records and only processes those matching this namespace.

## Supported Collections

### 1. `app.subcult.scene`

Represents a music scene or venue where events take place.

**Required Fields:**
- `name` (string): Scene name

**Optional Fields:**
- `description` (string): Scene description
- Additional fields allowed for forward compatibility

**Example:**
```json
{
  "name": "Underground Warehouse",
  "description": "Industrial techno venue in East Berlin"
}
```

**Validation:**
- `name` must be present and be a string type
- Additional fields are allowed but not validated

**Implementation:**
- Validator: `validateSceneRecord()` in `internal/indexer/filter.go`
- Type documentation: `SceneRecord` struct

---

### 2. `app.subcult.event`

Represents an event happening at a scene.

**Required Fields:**
- `name` (string): Event name
- `sceneId` (string): Reference to the scene ID where event occurs

**Optional Fields:**
- Additional fields allowed for forward compatibility

**Example:**
```json
{
  "name": "Techno Night Vol. 42",
  "sceneId": "scene_abc123"
}
```

**Validation:**
- `name` must be present and be a string type
- `sceneId` must be present and be a string type
- Additional fields are allowed but not validated

**Implementation:**
- Validator: `validateEventRecord()` in `internal/indexer/filter.go`
- Type documentation: `EventRecord` struct

---

### 3. `app.subcult.post`

Represents a post or announcement related to a scene.

**Required Fields:**
- `text` (string): Post content
- `sceneId` (string): Reference to the scene ID this post is about

**Optional Fields:**
- `embed` (object): Optional embedded content (images, links, etc.)
- Additional fields allowed for forward compatibility

**Example:**
```json
{
  "text": "Great show tonight!",
  "sceneId": "scene_abc123",
  "embed": {
    "uri": "at://..."
  }
}
```

**Validation:**
- `text` must be present and be a string type
- `sceneId` must be present and be a string type
- Additional fields are allowed but not validated

**Implementation:**
- Validator: `validatePostRecord()` in `internal/indexer/filter.go`
- Type documentation: `PostRecord` struct

---

### 4. `app.subcult.alliance`

Represents a trust relationship between scenes in the alliance network.

**Required Fields:**
- `fromSceneId` (string): Source scene ID
- `toSceneId` (string): Target scene ID

**Optional Fields:**
- Additional fields allowed for forward compatibility

**Example:**
```json
{
  "fromSceneId": "scene_abc123",
  "toSceneId": "scene_def456"
}
```

**Validation:**
- `fromSceneId` must be present and be a string type
- `toSceneId` must be present and be a string type
- Additional fields are allowed but not validated

**Implementation:**
- Validator: `validateAllianceRecord()` in `internal/indexer/filter.go`
- Type documentation: `AllianceRecord` struct

---

## Record Operations

The indexer supports three AT Protocol operations:

### Create
Creates a new record in the repository.

**Processing:**
1. Validate record against collection schema
2. Check idempotency key (prevents duplicates)
3. Insert record into database

### Update
Updates an existing record (identified by DID + rkey).

**Processing:**
1. Validate record against collection schema
2. Check idempotency key
3. Upsert record (INSERT or UPDATE based on existence)

### Delete
Removes a record from the repository.

**Processing:**
1. No validation required (payload is empty)
2. Delete record by DID + rkey

## Validation Rules

### General Rules

1. **JSON Syntax**: All records must be valid JSON
2. **Required Fields**: Must be present with correct type (string)
3. **Forward Compatibility**: Additional fields are allowed and preserved
4. **Case Sensitivity**: Collection names are case-sensitive (must be lowercase)

### Error Types

| Error | Description | Action |
|-------|-------------|--------|
| `ErrNonMatchingLexicon` | Collection not in `app.subcult.*` namespace | Record skipped (not matched) |
| `ErrMalformedJSON` | Invalid JSON syntax | Record discarded, error logged |
| `ErrMissingField` | Required field missing | Record discarded, error logged |
| `ErrInvalidFieldType` | Field has wrong type (e.g., number instead of string) | Record discarded, error logged |

### Validation Flow

```
Incoming CBOR → Parse → Check Lexicon → Validate Schema → Process
                  ↓          ↓              ↓               ↓
                Error    Not Matched    Discarded      Success
```

## AT Protocol Integration

### Record Identifier

Each record is uniquely identified by:

- **DID** (Decentralized Identifier): Owner of the record (e.g., `did:plc:abc123`)
- **Collection**: Record type (e.g., `app.subcult.scene`)
- **RKey**: Record key within collection (e.g., `scene1`)
- **Rev**: Revision string for this commit (e.g., `abc123`)

### Idempotency

The indexer uses SHA256 hashing for idempotency:

```go
idempotency_key = SHA256(DID:collection:rkey:rev)
```

This ensures:
- Same revision → same key (duplicate prevented)
- Different revision → different key (update processed)

See: `internal/indexer/README.md` for more details on idempotency implementation.

## Adding New Collections

To add a new collection to the `app.subcult.*` namespace:

1. **Define constant** in `internal/indexer/filter.go`:
   ```go
   const CollectionYourType = "app.subcult.yourtype"
   ```

2. **Create record struct** (documentation only):
   ```go
   type YourTypeRecord struct {
       RequiredField string `json:"requiredField"`
   }
   ```

3. **Implement validator**:
   ```go
   func validateYourTypeRecord(payload []byte) error {
       var record map[string]interface{}
       if err := json.Unmarshal(payload, &record); err != nil {
           return ErrMalformedJSON
       }
       return validateStringField(record, "requiredField")
   }
   ```

4. **Add to switch** in `validateRecord()`:
   ```go
   case CollectionYourType:
       return validateYourTypeRecord(payload)
   ```

5. **Write tests** in `internal/indexer/filter_test.go`

6. **Update this documentation** with the new collection schema

## Testing

The validation logic is extensively tested in:

- `internal/indexer/filter_test.go` - JSON validation tests
- `internal/indexer/filter_cbor_test.go` - CBOR parsing and integration tests

Run tests:
```bash
go test ./internal/indexer/... -run Filter
```

## References

- **AT Protocol Specification**: https://atproto.com/specs/repository
- **Jetstream Documentation**: https://github.com/ericvolp12/jetstream
- **Indexer Implementation**: `internal/indexer/README.md`
- **Filter Implementation**: `internal/indexer/filter.go`
- **Issue #305**: Jetstream Indexer epic
- **Issue #416**: Canonical Roadmap

## Schema Versioning

Currently, the schema uses an implicit version (v1) with forward compatibility:
- New optional fields can be added without breaking existing records
- Required fields cannot be removed or made optional
- Field types cannot change

Future versions may use explicit versioning in the lexicon name (e.g., `app.subcult.v2.scene`).

## Privacy Considerations

Record schemas should:
- Avoid storing sensitive personal information in public records
- Use references (IDs) instead of embedding full user data
- Follow location consent rules (see `docs/PRIVACY.md`)
- Minimize data collection to essential fields only

All location data in scene records must respect the `allow_precise` flag as documented in the privacy guide.
