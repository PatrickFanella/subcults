# Image Optimization Guide

This guide explains how to use the image optimization components in Subcults for optimal performance and user experience.

## Overview

Subcults implements modern image optimization techniques to ensure fast loading times and excellent user experience across all devices and network conditions:

- **WebP format** with JPEG fallback for broad browser support
- **Responsive images** using srcset and sizes attributes
- **Lazy loading** with Intersection Observer API
- **Cloudflare R2 CDN** integration for fast global delivery
- **Optimized components** for avatars and scene covers

## Components

### OptimizedImage

The base component for all optimized images. Provides WebP/JPEG fallback, lazy loading, and responsive sizing.

#### Basic Usage

```tsx
import { OptimizedImage } from '../components/OptimizedImage';

function MyComponent() {
  return (
    <OptimizedImage
      src="posts/123/image.jpg"
      alt="Concert photo"
      width={800}
      height={600}
      sizes="(max-width: 640px) 100vw, 50vw"
    />
  );
}
```

#### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | required | R2 object key or image URL |
| `alt` | string | required | Alternative text for accessibility |
| `width` | number | - | Image width (for aspect ratio) |
| `height` | number | - | Image height (for aspect ratio) |
| `lazy` | boolean | true | Enable lazy loading |
| `priority` | boolean | false | Disable lazy loading for above-fold images |
| `sizes` | string | '100vw' | Sizes attribute for responsive images |
| `objectFit` | string | 'cover' | CSS object-fit property |
| `className` | string | '' | Additional CSS classes |

#### Above-the-fold Images

For images that are visible on initial page load, use `priority` to disable lazy loading:

```tsx
<OptimizedImage
  src="hero.jpg"
  alt="Hero image"
  priority
  sizes="100vw"
/>
```

### Avatar

Optimized component for user avatars with automatic fallback to initials.

#### Basic Usage

```tsx
import { Avatar } from '../components/Avatar';

function UserProfile({ user }) {
  return (
    <Avatar
      src={user.avatarKey}
      name={user.name}
      size="lg"
      online={user.isOnline}
    />
  );
}
```

#### Size Variants

| Size | Dimension | Use Case |
|------|-----------|----------|
| `xs` | 32px | Tiny avatars in lists |
| `sm` | 48px | Small avatars in comments |
| `md` | 64px | Default avatar size |
| `lg` | 96px | Large avatars in profiles |
| `xl` | 128px | Extra large for headers |

#### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | - | R2 object key for avatar |
| `name` | string | required | User's display name |
| `size` | AvatarSize | 'md' | Size variant |
| `online` | boolean | false | Show online indicator |
| `onClick` | function | - | Click handler |
| `className` | string | '' | Additional CSS classes |

#### Fallback Behavior

If no image is provided or loading fails, the avatar displays:
- User's initials (first + last name)
- Consistent color based on name hash
- Maintains accessibility with proper ARIA labels

### SceneCover

Optimized component for scene cover images with responsive sizing and optional overlay.

#### Basic Usage

```tsx
import { SceneCover } from '../components/SceneCover';

function SceneCard({ scene }) {
  return (
    <SceneCover
      src={scene.coverImageKey}
      sceneName={scene.name}
      size="medium"
      overlay
      sizes="(max-width: 768px) 100vw, (max-width: 1024px) 50vw, 33vw"
    />
  );
}
```

#### Size Presets

| Size | Width | Use Case |
|------|-------|----------|
| `thumbnail` | 320px | Small previews |
| `small` | 640px | Card thumbnails |
| `medium` | 1024px | Standard covers |
| `large` | 1920px | Full-width covers |
| `xlarge` | 2560px | High-res displays |

#### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | string | - | R2 object key for cover |
| `sceneName` | string | required | Scene name (for alt text) |
| `size` | CoverSize | 'medium' | Size preset |
| `aspectRatio` | string | '16 / 9' | CSS aspect ratio |
| `priority` | boolean | false | Disable lazy loading |
| `overlay` | boolean | false | Add gradient overlay |
| `sizes` | string | '100vw' | Responsive sizes |
| `onClick` | function | - | Click handler |
| `className` | string | '' | Additional CSS classes |

## Image URL Utilities

The `imageUrl` module provides utilities for working with R2 CDN URLs.

### buildImageUrl

Generate transformed image URLs with resize, format, and quality parameters.

```tsx
import { buildImageUrl } from '../utils/imageUrl';

const url = buildImageUrl('posts/123/image.jpg', {
  width: 640,
  format: 'webp',
  quality: 80,
  fit: 'cover'
});
// Returns: /api/media/posts/123/image.jpg?w=640&f=webp&q=80&fit=cover
```

### generateSrcSet

Generate responsive srcset strings for multiple widths.

```tsx
import { generateSrcSet } from '../utils/imageUrl';

const srcset = generateSrcSet(
  'posts/123/image.jpg',
  [320, 640, 1024, 1920],
  'webp',
  80
);
// Returns: "...?w=320&f=webp&q=80 320w, ...?w=640&f=webp&q=80 640w, ..."
```

### getAvatarUrl / getCoverUrl

Convenience functions for avatar and cover images.

```tsx
import { getAvatarUrl, getCoverUrl } from '../utils/imageUrl';

const avatarUrl = getAvatarUrl('users/123/avatar.jpg', 'lg', 'webp');
const coverUrl = getCoverUrl('scenes/456/cover.jpg', 'large', 'webp');
```

## R2 CDN Configuration

### Environment Variables

Add to your `.env` file:

```bash
# Backend: R2 API credentials
R2_BUCKET_NAME=subcults-media
R2_ACCESS_KEY_ID=your_access_key
R2_SECRET_ACCESS_KEY=your_secret_key
R2_ENDPOINT=https://abc123.r2.cloudflarestorage.com

# Frontend: Public CDN URL
VITE_R2_CDN_URL=https://cdn.subcults.com
# or use R2 public URL: https://pub-xxxxx.r2.dev
```

### Setting up Cloudflare R2

1. **Create R2 Bucket**
   - Log into Cloudflare dashboard
   - Navigate to R2 Object Storage
   - Create a new bucket (e.g., `subcults-media`)

2. **Generate API Credentials**
   - Go to R2 → Manage R2 API Tokens
   - Create API token with read/write permissions
   - Save the Access Key ID and Secret Access Key

3. **Enable Public Access** (optional)
   - Navigate to your bucket settings
   - Enable "Public Access"
   - Note the public URL (e.g., `https://pub-xxxxx.r2.dev`)

4. **Custom Domain** (recommended)
   - Add a custom domain to your bucket
   - Configure DNS CNAME record
   - Use custom domain in `VITE_R2_CDN_URL`

### Image Transformations

**Note**: Cloudflare R2 itself does not provide built-in image transformation capabilities. The query parameters shown below are handled by your API proxy (`/api/media`) or Cloudflare Workers implementation. You have three options for implementing transformations:

1. **Cloudflare Images** (separate paid product) in front of R2
2. **API proxy** at `/api/media` that processes transformations
3. **Cloudflare Workers** with R2 bindings for on-the-fly transformations

The `imageUrl.ts` utilities generate transformation URLs with query parameters that your chosen implementation should handle:

```
https://cdn.subcults.com/posts/123/image.jpg?w=640&f=webp&q=80
```

Expected transformation parameters:
- `w` - Width in pixels
- `h` - Height in pixels
- `f` - Format (webp, jpeg, png)
- `q` - Quality (1-100)
- `fit` - Resize mode (cover, contain, fill)

## Performance Best Practices

### 1. Use Appropriate Sizes

Match image size to display size:

```tsx
// ❌ Bad: Loading full 2MB image for 100px thumbnail
<Avatar src="large-image.jpg" size="sm" />

// ✅ Good: Images are auto-optimized based on size prop
<Avatar src="image.jpg" size="sm" />
```

### 2. Specify Dimensions

Always provide width and height to prevent layout shift (CLS):

```tsx
<OptimizedImage
  src="image.jpg"
  alt="..."
  width={800}
  height={600} // Prevents CLS
/>
```

### 3. Use Priority for Above-fold Images

Hero images and critical content should load eagerly:

```tsx
<SceneCover
  src="hero.jpg"
  sceneName="Featured Scene"
  priority // Disables lazy loading
/>
```

### 4. Provide Accurate Sizes Attribute

Help the browser choose the right image size:

```tsx
<SceneCover
  src="cover.jpg"
  sceneName="Scene"
  sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 33vw"
/>
```

### 5. Lazy Load Below-fold Images

Let the browser defer loading off-screen images:

```tsx
{scenes.map(scene => (
  <SceneCover
    key={scene.id}
    src={scene.cover}
    sceneName={scene.name}
    lazy // Default behavior
  />
))}
```

## Lighthouse Performance

Target metrics:
- **Image score**: >90
- **Cumulative Layout Shift (CLS)**: <0.1
- **Largest Contentful Paint (LCP)**: <2.5s
- **Total Blocking Time (TBT)**: <200ms

### Optimization Checklist

- [ ] All images use OptimizedImage, Avatar, or SceneCover components
- [ ] Width and height specified to prevent CLS
- [ ] Lazy loading enabled for below-fold images
- [ ] Priority loading for above-fold images
- [ ] Appropriate sizes attribute for responsive images
- [ ] WebP format with JPEG fallback
- [ ] R2 CDN configured and working
- [ ] Image compression quality optimized (80 for most, 85 for avatars)

## Browser Support

| Feature | Support |
|---------|---------|
| WebP format | Chrome 23+, Firefox 65+, Safari 14+, Edge 18+ |
| JPEG fallback | All browsers |
| Lazy loading | Chrome 77+, Firefox 75+, Safari 15.4+, Edge 79+ |
| Intersection Observer | Chrome 51+, Firefox 55+, Safari 12.1+, Edge 15+ |

For older browsers:
- JPEG fallback ensures images still load
- IntersectionObserver polyfill can be added if needed
- Native lazy loading degrades gracefully

## Troubleshooting

### Images Not Loading

1. Check R2 credentials in `.env`
2. Verify bucket permissions
3. Check browser console for CORS errors
4. Ensure object keys are correct

### WebP Not Working

1. Verify browser supports WebP
2. Check that R2 is returning correct Content-Type
3. Ensure fallback JPEG is working

### Lazy Loading Issues

1. Check IntersectionObserver support
2. Verify `priority` prop for above-fold images
3. Ensure container has proper dimensions

### Performance Issues

1. Run Lighthouse audit
2. Check image file sizes
3. Verify CDN is being used (not API proxy)
4. Check network waterfall for optimization opportunities

## Examples

### Scene Card with Cover

```tsx
function SceneCard({ scene }) {
  return (
    <div className="card">
      <SceneCover
        src={scene.coverImageKey}
        sceneName={scene.name}
        size="medium"
        overlay
        onClick={() => navigate(`/scenes/${scene.id}`)}
        sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 33vw"
      />
      <div className="card-content">
        <h3>{scene.name}</h3>
        <p>{scene.description}</p>
      </div>
    </div>
  );
}
```

### User Profile Header

```tsx
function ProfileHeader({ user }) {
  return (
    <div className="profile-header">
      <SceneCover
        src={user.coverImageKey}
        sceneName={`${user.name}'s profile`}
        size="large"
        aspectRatio="21 / 9"
        priority
        overlay
      />
      <div className="profile-info">
        <Avatar
          src={user.avatarKey}
          name={user.name}
          size="xl"
          online={user.isOnline}
        />
        <h1>{user.name}</h1>
      </div>
    </div>
  );
}
```

### Image Gallery

```tsx
function ImageGallery({ images }) {
  return (
    <div className="grid grid-cols-3 gap-4">
      {images.map((image, index) => (
        <OptimizedImage
          key={image.id}
          src={image.key}
          alt={image.alt}
          width={400}
          height={300}
          priority={index < 3} // Prioritize first row
          sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 33vw"
        />
      ))}
    </div>
  );
}
```

## Related Documentation

- [Cloudflare R2 Documentation](https://developers.cloudflare.com/r2/)
- [Web.dev Image Optimization](https://web.dev/fast/#optimize-your-images)
- [MDN Picture Element](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/picture)
- [Lighthouse Performance](https://developer.chrome.com/docs/lighthouse/performance/)
