# Moderation Labels & Content Filtering

## Overview

Subcults implements a privacy-first content moderation system using standardized labels. This approach provides fine-grained control over content visibility without permanent deletion, enabling appeals, audits, and user preference-based filtering.

## Core Principles

1. **No Permanent Deletion**: Moderation labels control visibility rather than deleting content
2. **User Preference Respect**: Content filtering adapts to individual user preferences
3. **Owner Visibility**: Post owners always see their own content regardless of labels
4. **Transparent Enforcement**: Clear rules for each label type

## Allowed Moderation Labels

The system supports four standardized moderation labels:

### `hidden`
**Purpose**: Complete removal from public visibility  
**Behavior**:
- Excluded from all public feeds and search results
- Never visible to anyone except the post owner
- Use for content that violates community guidelines

**Example Use Cases**:
- Policy violations
- Harmful content removal
- Privacy-sensitive takedowns

### `nsfw`
**Purpose**: Adult/mature content that requires explicit opt-in  
**Behavior**:
- Hidden by default for all users
- Visible only to users with `ShowNSFW` preference enabled
- Always visible to the post owner

**Example Use Cases**:
- Explicit or sexual content
- Graphic violence
- Other adult-oriented material

### `spam`
**Purpose**: Unwanted commercial or repetitive content  
**Behavior**:
- Excluded from search results (`includeModerated=false`)
- Visible in general feeds (`includeModerated=true`)
- Always visible to the post owner

**Example Use Cases**:
- Unsolicited advertisements
- Repetitive promotional content
- Bot-generated spam

### `flagged`
**Purpose**: Content under review or reported by users  
**Behavior**:
- Excluded from search results (`includeModerated=false`)
- Visible in general feeds (`includeModerated=true`)
- Always visible to the post owner

**Example Use Cases**:
- User-reported content pending review
- Suspicious or potentially violating content
- Content awaiting moderator decision

## Label Validation

All labels must be validated before being applied to posts. The system rejects any labels not in the allowed list.

### Validation Rules
- Labels are case-sensitive (must be lowercase)
- Multiple labels can be applied to a single post
- Empty label arrays are valid (no moderation)
- Invalid labels return HTTP 400 with error code `validation_error`

### API Behavior

**Create Post with Invalid Label**:
```json
POST /posts
{
  "scene_id": "123",
  "text": "Test post",
  "labels": ["invalid_label"]
}

Response: 400 Bad Request
{
  "error": {
    "code": "validation_error",
    "message": "Invalid moderation label"
  }
}
```

**Create Post with Valid Labels**:
```json
POST /posts
{
  "scene_id": "123",
  "text": "Test post",
  "labels": ["nsfw", "flagged"]
}

Response: 201 Created
```

## Content Filtering

### FilterPostsForUser Function

The `FilterPostsForUser` helper function applies moderation rules based on user preferences and context.

**Function Signature**:
```go
func FilterPostsForUser(
    posts []*Post,
    prefs *UserPreferences,
    viewerDID string,
    includeModerated bool
) []*Post
```

**Parameters**:
- `posts`: List of posts to filter
- `prefs`: User preferences (includes `ShowNSFW` flag)
- `viewerDID`: DID of the viewing user (empty for anonymous)
- `includeModerated`: Whether to include spam/flagged content (false for search, true for feeds)

### User Preferences

```go
type UserPreferences struct {
    ShowNSFW bool  // Default: false
}
```

Users must explicitly opt-in to view NSFW content. The default preference hides all NSFW-labeled posts.

### Filtering Rules Table

| Label      | Anonymous User | User (ShowNSFW=false) | User (ShowNSFW=true) | Post Owner | Search (includeModerated=false) | Feed (includeModerated=true) |
|------------|----------------|----------------------|---------------------|------------|-------------------------------|----------------------------|
| `hidden`   | ❌ Hidden       | ❌ Hidden             | ❌ Hidden            | ✅ Visible  | ❌ Hidden                      | ❌ Hidden                   |
| `nsfw`     | ❌ Hidden       | ❌ Hidden             | ✅ Visible           | ✅ Visible  | Depends on ShowNSFW           | Depends on ShowNSFW        |
| `spam`     | ✅ Visible      | ✅ Visible            | ✅ Visible           | ✅ Visible  | ❌ Hidden                      | ✅ Visible                  |
| `flagged`  | ✅ Visible      | ✅ Visible            | ✅ Visible           | ✅ Visible  | ❌ Hidden                      | ✅ Visible                  |

### Multiple Labels

When multiple labels are applied, the most restrictive rule wins:
- `hidden` + `nsfw` → Always hidden (except to owner)
- `nsfw` + `spam` → Hidden in search OR if ShowNSFW=false
- Any label + owner → Always visible

## Implementation Examples

### Applying Labels on Post Creation

```go
// Handler implementation
if err := post.ValidateLabels(sanitizedLabels); err != nil {
    ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
    WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Invalid moderation label")
    return
}

newPost := &post.Post{
    Text:   "Example post",
    Labels: sanitizedLabels, // []string{"nsfw"}
}
```

### Filtering Posts for a User

```go
// Search context (exclude spam/flagged)
userPrefs := &post.UserPreferences{ShowNSFW: true}
filteredPosts := post.FilterPostsForUser(allPosts, userPrefs, userDID, false)

// Feed context (include spam/flagged)
filteredPosts := post.FilterPostsForUser(allPosts, userPrefs, userDID, true)
```

### Checking Labels on Posts

```go
if post.IsHidden() {
    // Handle hidden post
}

if post.IsNSFW() && !userPrefs.ShowNSFW {
    // Hide NSFW content
}

if post.HasLabel(post.LabelSpam) {
    // Handle spam
}
```

## Security Considerations

### Privacy Protection
- Labels should not leak to unauthorized users
- Only include necessary label information in API responses
- Consider showing simplified labels to non-moderators

### Audit Trail
- Log all label applications with:
  - Timestamp
  - Moderator DID
  - Label type
  - Reason (when applicable)

### Rate Limiting
- Apply rate limits to label modification endpoints
- Prevent abuse of flagging mechanisms
- Monitor for patterns of malicious labeling

## Future Enhancements

### Planned Features
1. **Label Expiry**: Time-based label removal for temporary restrictions
2. **Appeal System**: Allow users to contest labels on their content
3. **Granular Permissions**: Role-based label application (moderators only)
4. **Label History**: Track label changes over time
5. **Custom Labels**: Scene-specific moderation labels

### Under Consideration
- Machine learning-based auto-labeling
- Community-driven moderation (voting)
- Geographic-specific content rules
- Age-gated content beyond NSFW

## Testing

Comprehensive test coverage ensures correct filtering behavior:

### Test Categories
1. **Label Validation**: Valid/invalid label acceptance
2. **Single Label Filtering**: Each label type in isolation
3. **Multiple Labels**: Combined label behavior
4. **User Preferences**: ShowNSFW flag interactions
5. **Context Switching**: Search vs feed filtering
6. **Owner Visibility**: Owner always sees their content
7. **Edge Cases**: Nil/empty inputs, nil posts

### Running Tests
```bash
# Run all moderation tests
go test -v ./internal/post/... -run "TestValidateLabels|TestFilterPostsForUser"

# Run API handler tests
go test -v ./internal/api/... -run "TestCreatePost.*Label|TestUpdatePost.*Label"
```

## Migration Guide

### Existing Posts
Posts created before moderation labels will have `labels: []` by default. No migration needed.

### Database Schema
Labels are stored as a JSON array in the `labels` column:
```sql
ALTER TABLE posts ADD COLUMN labels TEXT[] DEFAULT '{}';
```

### API Clients
Clients should handle label validation errors gracefully:
```typescript
try {
  await createPost({ text: "Example", labels: ["nsfw"] });
} catch (error) {
  if (error.code === "validation_error") {
    // Show user-friendly error about invalid labels
  }
}
```

## References

- **Issue**: [#90 - Task: Moderation Label Application & Filtering](https://github.com/subculture-collective/subcults/issues/90)
- **Implementation**: `internal/post/moderation.go`
- **Tests**: `internal/post/moderation_test.go`
- **API Integration**: `internal/api/post_handlers.go`
