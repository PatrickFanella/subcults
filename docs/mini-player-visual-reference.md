# Streaming Mini-Player Visual Reference

## Layout Overview

The mini-player appears as a fixed bar at the bottom of the screen when a user is connected to a stream.

```
┌─────────────────────────────────────────────────────────────────┐
│                        Main Application                          │
│                                                                   │
│  Content scrolls here...                                         │
│                                                                   │
│                                                                   │
├─────────────────────────────────────────────────────────────────┤
│ 🟢  Now Playing                   🎤   🔊    [Leave]            │
│     live-session-2024                                            │
└─────────────────────────────────────────────────────────────────┘
```

## Component Breakdown

### 1. Connection Quality Indicator (Left)
```
🟢 Green  - Excellent connection
🟡 Yellow - Good connection  
🔴 Red    - Poor connection
⚪ Gray   - Unknown connection
```
- 8x8px circular indicator
- Color-coded based on LiveKit connection quality
- Positioned at far left

### 2. Stream Information (Center-Left)
```
┌─────────────────┐
│ Now Playing     │ ← Primary text (0.875rem, bold)
│ room-name-here  │ ← Secondary text (0.75rem, gray)
└─────────────────┘
```
- Flex: 1 (expands to fill available space)
- Text truncates with ellipsis on overflow
- White title, gray subtitle

### 3. Mute Button (Center)
```
┌─────┐
│ 🎤  │  Unmuted (green background)
└─────┘

┌─────┐
│ 🔇  │  Muted (red background)
└─────┘
```
- 2.5rem circular button
- Green when unmuted, red when muted
- Hover effect: slight scale up
- Keyboard shortcut: Space

### 4. Volume Control (Center-Right)
```
┌─────┐
│ 🔊  │  Volume button (gray background)
└─────┘

When clicked, shows popup above:
┌───────────────┐
│ Volume: 75%   │
│ ▓▓▓▓▓▓▓▓░░░░░ │ ← Slider (0-100)
└───────────────┘
```
- 2.5rem button
- Icon changes based on volume:
  - 🔈 Volume 0
  - 🔉 Volume 1-49
  - 🔊 Volume 50-100
- Slider appears in popup above button
- Popup closes on Escape or click outside

### 5. Leave Button (Right)
```
┌────────────┐
│   Leave    │  Red button
└────────────┘
```
- Rectangular button with padding
- Red background (#dc2626)
- Darker red on hover (#b91c1c)
- Disconnects from stream when clicked

## Styling Details

### Colors
- Background: #1f2937 (dark gray)
- Border top: #374151 (medium gray)
- Shadow: 0 -2px 10px rgba(0, 0, 0, 0.3)

### Positioning
- Position: fixed
- Bottom: 0
- Left: 0
- Right: 0
- Z-index: 1000

### Spacing
- Padding: 0.75rem 1rem
- Gap between elements: 1rem

### Typography
- Font family: System font stack
- Title: 0.875rem, weight 600
- Subtitle: 0.75rem, color #9ca3af

## Responsive Behavior

### Mobile (<768px)
- Same layout, elements stack responsively
- Text truncates more aggressively
- Touch-friendly hit targets maintained

### Desktop (≥768px)
- Full layout as shown
- Hover effects enabled
- Keyboard navigation enhanced

## Accessibility Features

### ARIA Labels
```html
<div role="region" aria-label="Mini player">
  <div aria-label="Excellent connection"></div>
  <button aria-label="Mute microphone"></button>
  <button aria-label="Volume control" aria-expanded="false"></button>
  <button aria-label="Leave stream"></button>
</div>
```

### Volume Slider
```html
<input
  type="range"
  aria-label="Volume slider"
  aria-valuemin="0"
  aria-valuemax="100"
  aria-valuenow="75"
  aria-valuetext="75%"
/>
```

### Keyboard Navigation
- Tab: Move between controls
- Space: Toggle mute (when mini-player focused)
- Escape: Close volume slider (when open)
- Arrow keys: Adjust volume slider

### Focus Indicators
- 2px solid blue (#60a5fa) outline
- 2px offset from element
- Visible on all interactive elements

## Integration Points

### Shows When
- `isConnected === true` AND
- `roomName !== null`

### Hides When
- User explicitly clicks "Leave"
- Connection is lost and reconnection fails
- Component unmounts

### Persists Across
- All route changes
- Page refreshes (reconnects automatically)
- App state changes

## Performance Considerations

- Component wrapped in `React.memo`
- Selective store subscriptions
- Stable function references from store
- Conditional rendering (only when connected)
- No unnecessary re-renders on unrelated state changes
