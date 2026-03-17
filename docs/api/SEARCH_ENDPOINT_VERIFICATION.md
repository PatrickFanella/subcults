# Scene Search Endpoint - Manual Verification

This document provides step-by-step instructions for manually verifying the scene search endpoint.

## Prerequisites

1. API server running on `http://localhost:8080`
2. `curl` or similar HTTP client
3. At least one scene created in the database

## Test Cases

### 1. Basic Text Search

**Description**: Search for scenes by name

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes?q=music" \
  -H "Content-Type: application/json"
```

**Expected Response**:
```json
{
  "results": [...],
  "next_cursor": "...",
  "count": 1
}
```

**Verification**:
- ✓ Status code: 200
- ✓ `count` matches number of results
- ✓ Each result has `jittered_centroid` (privacy check)
- ✓ Results are ordered by composite score

### 2. Geographic Bounding Box Search

**Description**: Search for scenes within a geographic area

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes?bbox=-74.1,40.6,-73.9,40.8" \
  -H "Content-Type: application/json"
```

**Expected Response**:
```json
{
  "results": [...],
  "next_cursor": "",
  "count": 2
}
```

**Verification**:
- ✓ Status code: 200
- ✓ Only scenes within bbox returned
- ✓ Coordinates are jittered (not exact)

### 3. Combined Text + Bbox Search

**Description**: Search with both text query and geographic filter

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes?q=electronic&bbox=-74.1,40.6,-73.9,40.8&limit=10" \
  -H "Content-Type: application/json"
```

**Expected Response**:
```json
{
  "results": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Electronic Music Scene",
      "description": "Underground techno parties",
      "jittered_centroid": {
        "lat": 40.7150,
        "lng": -74.0080
      },
      "coarse_geohash": "dr5regw",
      "tags": ["electronic", "techno"],
      "visibility": "public"
    }
  ],
  "next_cursor": "",
  "count": 1
}
```

**Verification**:
- ✓ Status code: 200
- ✓ Results match both text query AND bbox
- ✓ Text relevance evident (e.g., "electronic" in name/description/tags)

### 4. Pagination

**Description**: Verify cursor-based pagination works correctly

**Request 1** (First page):
```bash
curl -X GET "http://localhost:8080/search/scenes?q=scene&limit=2" \
  -H "Content-Type: application/json"
```

**Request 2** (Second page using cursor from Request 1):
```bash
CURSOR="<cursor_from_request_1>"
curl -X GET "http://localhost:8080/search/scenes?q=scene&limit=2&cursor=$CURSOR" \
  -H "Content-Type: application/json"
```

**Verification**:
- ✓ Request 1 returns `next_cursor` if more results exist
- ✓ Request 2 returns different scenes (no duplicates)
- ✓ Results maintain consistent ordering
- ✓ Final page has empty `next_cursor`

### 5. Validation: Missing Parameters

**Description**: Verify error when neither q nor bbox provided

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes" \
  -H "Content-Type: application/json"
```

**Expected Response**:
```json
{
  "error": {
    "code": "validation_error",
    "message": "At least one of 'q' or 'bbox' must be provided"
  }
}
```

**Verification**:
- ✓ Status code: 400
- ✓ Error code: `validation_error`

### 6. Validation: Invalid Bbox Format

**Description**: Verify error handling for malformed bbox

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes?bbox=-74.1,40.6,-73.9" \
  -H "Content-Type: application/json"
```

**Expected Response**:
```json
{
  "error": {
    "code": "validation_error",
    "message": "bbox must be in format: minLng,minLat,maxLng,maxLat"
  }
}
```

**Verification**:
- ✓ Status code: 400
- ✓ Error code: `validation_error`

### 7. Validation: Bbox Too Large

**Description**: Verify max area threshold enforcement

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes?bbox=-180,-90,180,90" \
  -H "Content-Type: application/json"
```

**Expected Response**:
```json
{
  "error": {
    "code": "validation_error",
    "message": "bbox area too large (max 10.0 square degrees)"
  }
}
```

**Verification**:
- ✓ Status code: 400
- ✓ Error code: `validation_error`

### 8. Validation: Limit Exceeds Max

**Description**: Verify limit is capped to max (50)

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes?q=scene&limit=100" \
  -H "Content-Type: application/json"
```

**Expected Response**:
```json
{
  "results": [...],
  "count": N  // where N <= 50
}
```

**Verification**:
- ✓ Status code: 200
- ✓ Returns at most 50 results (even though limit=100)

### 9. Privacy: Hidden Scenes Excluded

**Description**: Verify hidden/unlisted scenes don't appear in results

**Setup**:
1. Create a hidden scene (visibility="unlisted")
2. Create a public scene with similar attributes

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes?q=<matching_query>" \
  -H "Content-Type: application/json"
```

**Verification**:
- ✓ Only public scene in results
- ✓ Hidden scene NOT in results

### 10. Privacy: Jittered Coordinates

**Description**: Verify coordinates are privacy-protected

**Setup**:
1. Note precise coordinates of a scene
2. Search for the scene

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes?q=<scene_name>" \
  -H "Content-Type: application/json"
```

**Verification**:
- ✓ `jittered_centroid` differs from precise coordinates
- ✓ Offset is small but deterministic
- ✓ Same scene returns same jittered point (stability)

### 11. Trust Ranking: Disabled

**Description**: Verify trust scores excluded when flag disabled

**Setup**:
```bash
export RANK_TRUST_ENABLED=false
```

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes?q=music" \
  -H "Content-Type: application/json"
```

**Verification**:
- ✓ `trust_score` field NOT present in results
- ✓ Results ordered without trust influence

### 12. Trust Ranking: Enabled

**Description**: Verify trust scores included when flag enabled

**Setup**:
```bash
export RANK_TRUST_ENABLED=true
```

**Request**:
```bash
curl -X GET "http://localhost:8080/search/scenes?q=music" \
  -H "Content-Type: application/json"
```

**Verification**:
- ✓ `trust_score` field present in results
- ✓ High-trust scenes rank higher (given equal text/proximity)

## Automated Verification Script

```bash
#!/bin/bash
# Run all verification tests

BASE_URL="http://localhost:8080"
PASS=0
FAIL=0

# Test 1: Basic text search
echo "Test 1: Basic text search..."
RESPONSE=$(curl -s -w "%{http_code}" "$BASE_URL/search/scenes?q=music")
if [[ "$RESPONSE" == *"200" ]]; then
  echo "✓ PASS"
  PASS=$((PASS+1))
else
  echo "✗ FAIL"
  FAIL=$((FAIL+1))
fi

# Test 2: Bbox search
echo "Test 2: Bbox search..."
RESPONSE=$(curl -s -w "%{http_code}" "$BASE_URL/search/scenes?bbox=-74.1,40.6,-73.9,40.8")
if [[ "$RESPONSE" == *"200" ]]; then
  echo "✓ PASS"
  PASS=$((PASS+1))
else
  echo "✗ FAIL"
  FAIL=$((FAIL+1))
fi

# Test 3: Missing parameters
echo "Test 3: Missing parameters..."
RESPONSE=$(curl -s -w "%{http_code}" "$BASE_URL/search/scenes")
if [[ "$RESPONSE" == *"400" ]]; then
  echo "✓ PASS"
  PASS=$((PASS+1))
else
  echo "✗ FAIL"
  FAIL=$((FAIL+1))
fi

# Test 4: Invalid bbox
echo "Test 4: Invalid bbox..."
RESPONSE=$(curl -s -w "%{http_code}" "$BASE_URL/search/scenes?bbox=-74.1,40.6,-73.9")
if [[ "$RESPONSE" == *"400" ]]; then
  echo "✓ PASS"
  PASS=$((PASS+1))
else
  echo "✗ FAIL"
  FAIL=$((FAIL+1))
fi

# Test 5: Bbox too large
echo "Test 5: Bbox too large..."
RESPONSE=$(curl -s -w "%{http_code}" "$BASE_URL/search/scenes?bbox=-180,-90,180,90")
if [[ "$RESPONSE" == *"400" ]]; then
  echo "✓ PASS"
  PASS=$((PASS+1))
else
  echo "✗ FAIL"
  FAIL=$((FAIL+1))
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
```

## Notes

- All tests assume the API server is running and accessible
- Some tests require pre-existing scene data
- Coordinate jittering is deterministic (same input → same output)
- Trust ranking tests require appropriate environment variable configuration
