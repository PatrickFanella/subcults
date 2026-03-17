# Scene Palette Update Endpoint

## Overview

The palette update endpoint allows scene owners to customize the color scheme for their scene's visual identity on the map and detail panels.

## Endpoint

```
PATCH /scenes/{id}/palette
```

## Authentication

Requires valid JWT token with owner permissions for the specified scene.

## Request Body

```json
{
  "palette": {
    "primary": "#ff0000",
    "secondary": "#00ff00",
    "accent": "#0000ff",
    "background": "#ffffff",
    "text": "#000000"
  }
}
```

### Palette Fields

All fields are **required** and must be valid hex color codes in the format `#RRGGBB`:

- **primary**: Primary brand color for the scene
- **secondary**: Secondary accent color
- **accent**: Tertiary accent color for highlights
- **background**: Background color for panels and cards
- **text**: Text color for content displayed on the background

## Validation Rules

### 1. Hex Color Format

All colors must:
- Start with `#`
- Contain exactly 6 hexadecimal digits (0-9, A-F, case insensitive)
- Examples: `#ff0000`, `#00FF00`, `#0000Ff`

Invalid examples:
- `ff0000` (missing hash)
- `#fff` (too short)
- `#ff00000` (too long)
- `#gggggg` (invalid characters)

### 2. Contrast Ratio (WCAG AA)

The contrast ratio between **text** and **background** colors must meet **WCAG AA** standards:
- Minimum ratio: **4.5:1**
- This ensures readability for users with visual impairments

The contrast ratio is calculated using the WCAG 2.1 relative luminance formula.

### 3. Security

All color values are sanitized to prevent XSS attacks:
- HTML entities are escaped
- Script tags and other malicious content are rejected
- Only valid hex colors pass validation

## Response

### Success (200 OK)

Returns the updated scene object with the new palette and updated timestamp:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Underground Techno Scene",
  "owner_did": "did:plc:test123",
  "palette": {
    "primary": "#ff0000",
    "secondary": "#00ff00",
    "accent": "#0000ff",
    "background": "#ffffff",
    "text": "#000000"
  },
  "updated_at": "2024-12-09T01:58:55.959Z",
  ...
}
```

### Error Responses

#### 400 Bad Request - Invalid Palette

```json
{
  "error": {
    "code": "invalid_palette",
    "message": "primary color: invalid hex color format, expected #RRGGBB: got \"not-a-color\""
  }
}
```

#### 400 Bad Request - Insufficient Contrast

```json
{
  "error": {
    "code": "invalid_palette",
    "message": "Insufficient contrast between text and background colors (got 2.3:1, need 4.5:1 minimum for WCAG AA)"
  }
}
```

#### 404 Not Found

```json
{
  "error": {
    "code": "not_found",
    "message": "Scene not found"
  }
}
```

## Examples

### Valid Palette Update

**Request:**
```bash
curl -X PATCH https://api.subcults.com/scenes/550e8400-e29b-41d4-a716-446655440000/palette \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "palette": {
      "primary": "#8b0000",
      "secondary": "#ff4500",
      "accent": "#ffd700",
      "background": "#1a1a1a",
      "text": "#ffffff"
    }
  }'
```

**Response:** 200 OK with updated scene object

### Invalid Color Format

**Request:**
```bash
curl -X PATCH https://api.subcults.com/scenes/550e8400-e29b-41d4-a716-446655440000/palette \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "palette": {
      "primary": "red",
      "secondary": "#00ff00",
      "accent": "#0000ff",
      "background": "#ffffff",
      "text": "#000000"
    }
  }'
```

**Response:** 400 Bad Request
```json
{
  "error": {
    "code": "invalid_palette",
    "message": "primary color: invalid hex color format, expected #RRGGBB: got \"red\""
  }
}
```

### Insufficient Contrast

**Request:**
```bash
curl -X PATCH https://api.subcults.com/scenes/550e8400-e29b-41d4-a716-446655440000/palette \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "palette": {
      "primary": "#ff0000",
      "secondary": "#00ff00",
      "accent": "#0000ff",
      "background": "#ffffff",
      "text": "#cccccc"
    }
  }'
```

**Response:** 400 Bad Request
```json
{
  "error": {
    "code": "invalid_palette",
    "message": "Insufficient contrast between text and background colors (got 1.6:1, need 4.5:1 minimum for WCAG AA)"
  }
}
```

## Future Extensibility

The palette structure is designed to be extensible. Future versions may add:

- **hover**: Hover state colors for interactive elements
- **disabled**: Colors for disabled UI elements
- **error**: Color for error states
- **success**: Color for success states
- **warning**: Color for warning states

Additional validation may include:
- Color harmony checks
- Brand consistency validation
- Automatic palette generation suggestions

## Implementation Notes

### Contrast Calculation

The contrast ratio is calculated using the WCAG 2.1 formula:

```
ratio = (L1 + 0.05) / (L2 + 0.05)
```

Where L1 and L2 are the relative luminances of the lighter and darker colors respectively.

Relative luminance is calculated as:

```
L = 0.2126 * R + 0.7152 * G + 0.0722 * B
```

Where R, G, and B are gamma-corrected values derived from sRGB color components.

### WCAG AA Standards

- **Normal text**: 4.5:1 minimum contrast ratio
- **Large text** (18pt+ or 14pt+ bold): 3:1 minimum (not currently enforced)
- **WCAG AAA**: 7:1 minimum (future consideration)

### Security Considerations

- All color values are HTML-escaped before storage
- Invalid characters trigger immediate rejection
- The endpoint protects against:
  - XSS via script tag injection
  - HTML entity manipulation
  - Unicode normalization attacks

## Related Documentation

- [Scene Management API](./SCENE_API.md)
- [Color Validation Package](../../internal/color/validator.go)
- [WCAG 2.1 Contrast Guidelines](https://www.w3.org/WAI/WCAG21/Understanding/contrast-minimum.html)
