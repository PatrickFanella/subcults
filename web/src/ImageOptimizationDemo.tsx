/**
 * Image Optimization Demo
 * Showcases optimized image components with examples
 * 
 * NOTE: This demo uses alert() for click handlers for simplicity.
 * Production implementations should use more user-friendly feedback mechanisms
 * such as toasts, modals, navigation, or other UI feedback instead of browser alerts.
 */

import React from 'react';
import { OptimizedImage } from './components/OptimizedImage';
import { Avatar } from './components/Avatar';
import { SceneCover } from './components/SceneCover';

export const ImageOptimizationDemo: React.FC = () => {
  // Demo data
  const users = [
    { name: 'Alice Johnson', avatarKey: '', isOnline: true },
    { name: 'Bob Smith', avatarKey: '', isOnline: false },
    { name: 'Charlie Brown', avatarKey: '', isOnline: true },
    { name: 'Diana Prince', avatarKey: '', isOnline: false },
  ];

  const scenes = [
    {
      id: '1',
      name: 'Underground Techno',
      coverKey: '',
    },
    {
      id: '2',
      name: 'Jazz & Blues',
      coverKey: '',
    },
    {
      id: '3',
      name: 'Indie Rock',
      coverKey: '',
    },
  ];

  return (
    <div className="max-w-7xl mx-auto p-8 space-y-12">
      {/* Header */}
      <div>
        <h1 className="text-4xl font-bold mb-2">Image Optimization Demo</h1>
        <p className="text-gray-600 dark:text-gray-400">
          Showcasing WebP format, responsive images, and lazy loading
        </p>
      </div>

      {/* Avatar Component Demo */}
      <section>
        <h2 className="text-2xl font-bold mb-6">Avatar Component</h2>
        
        <div className="space-y-8">
          {/* Size Variants */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Size Variants</h3>
            <div className="flex items-end gap-6">
              <div className="flex flex-col items-center gap-2">
                <Avatar name="Alice Johnson" size="xs" />
                <span className="text-xs text-gray-600">xs (32px)</span>
              </div>
              <div className="flex flex-col items-center gap-2">
                <Avatar name="Bob Smith" size="sm" />
                <span className="text-xs text-gray-600">sm (48px)</span>
              </div>
              <div className="flex flex-col items-center gap-2">
                <Avatar name="Charlie Brown" size="md" />
                <span className="text-xs text-gray-600">md (64px)</span>
              </div>
              <div className="flex flex-col items-center gap-2">
                <Avatar name="Diana Prince" size="lg" />
                <span className="text-xs text-gray-600">lg (96px)</span>
              </div>
              <div className="flex flex-col items-center gap-2">
                <Avatar name="Eve Anderson" size="xl" />
                <span className="text-xs text-gray-600">xl (128px)</span>
              </div>
            </div>
          </div>

          {/* Online Status */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Online Status</h3>
            <div className="flex items-center gap-6">
              {users.map((user) => (
                <div key={user.name} className="flex flex-col items-center gap-2">
                  <Avatar
                    name={user.name}
                    size="lg"
                    online={user.isOnline}
                  />
                  <span className="text-sm text-gray-600">
                    {user.isOnline ? 'Online' : 'Offline'}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Clickable Avatars */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Interactive Avatars</h3>
            <div className="flex items-center gap-4">
              {users.slice(0, 3).map((user) => (
                <Avatar
                  key={user.name}
                  name={user.name}
                  size="lg"
                  online={user.isOnline}
                  onClick={() => alert(`Clicked ${user.name}`)}
                  className="transition-transform hover:scale-110"
                />
              ))}
            </div>
            <p className="text-sm text-gray-600 mt-2">
              Click an avatar to trigger action
            </p>
          </div>
        </div>
      </section>

      {/* SceneCover Component Demo */}
      <section>
        <h2 className="text-2xl font-bold mb-6">SceneCover Component</h2>
        
        <div className="space-y-8">
          {/* Size Variants */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Size Variants</h3>
            <div className="space-y-4">
              <div>
                <p className="text-sm text-gray-600 mb-2">Thumbnail (320px)</p>
                <div className="w-64">
                  <SceneCover
                    sceneName="Underground Techno"
                    size="thumbnail"
                    overlay
                  />
                </div>
              </div>
              <div>
                <p className="text-sm text-gray-600 mb-2">Medium (1024px)</p>
                <div className="w-full max-w-2xl">
                  <SceneCover
                    sceneName="Jazz & Blues"
                    size="medium"
                    overlay
                  />
                </div>
              </div>
            </div>
          </div>

          {/* Aspect Ratios */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Aspect Ratios</h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <p className="text-sm text-gray-600 mb-2">16:9 (Default)</p>
                <SceneCover
                  sceneName="Scene 1"
                  aspectRatio="16 / 9"
                />
              </div>
              <div>
                <p className="text-sm text-gray-600 mb-2">4:3</p>
                <SceneCover
                  sceneName="Scene 2"
                  aspectRatio="4 / 3"
                />
              </div>
              <div>
                <p className="text-sm text-gray-600 mb-2">1:1 (Square)</p>
                <SceneCover
                  sceneName="Scene 3"
                  aspectRatio="1 / 1"
                />
              </div>
            </div>
          </div>

          {/* With Overlay */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Gradient Overlay</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-gray-600 mb-2">Without Overlay</p>
                <SceneCover
                  sceneName="Underground Techno"
                  overlay={false}
                />
              </div>
              <div>
                <p className="text-sm text-gray-600 mb-2">With Overlay</p>
                <SceneCover
                  sceneName="Underground Techno"
                  overlay={true}
                />
              </div>
            </div>
          </div>

          {/* Clickable Covers */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Interactive Scene Cards</h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {scenes.map((scene) => (
                <div
                  key={scene.id}
                  className="bg-white dark:bg-gray-800 rounded-lg overflow-hidden shadow-md hover:shadow-xl transition-shadow"
                >
                  <SceneCover
                    sceneName={scene.name}
                    size="small"
                    overlay
                    onClick={() => alert(`View ${scene.name}`)}
                  />
                  <div className="p-4">
                    <h4 className="font-semibold text-lg">{scene.name}</h4>
                    <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                      Click the cover to explore
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* OptimizedImage Component Demo */}
      <section>
        <h2 className="text-2xl font-bold mb-6">OptimizedImage Component</h2>
        
        <div className="space-y-8">
          {/* Responsive Images */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Responsive Srcset</h3>
            <p className="text-sm text-gray-600 mb-4">
              Images automatically adapt to viewport size and pixel density
            </p>
            <OptimizedImage
              src="example.jpg"
              alt="Responsive image example"
              width={1200}
              height={800}
              sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 33vw"
              className="rounded-lg shadow-md"
            />
          </div>

          {/* Object Fit Options */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Object Fit Options</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {(['cover', 'contain', 'fill', 'none'] as const).map((fit) => (
                <div key={fit}>
                  <p className="text-sm text-gray-600 mb-2">{fit}</p>
                  <div className="h-48 bg-gray-100 dark:bg-gray-900 rounded overflow-hidden">
                    <OptimizedImage
                      src="example.jpg"
                      alt={`Object fit: ${fit}`}
                      width={300}
                      height={200}
                      objectFit={fit}
                      priority
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* Performance Features */}
      <section className="bg-blue-50 dark:bg-blue-900/20 p-6 rounded-lg">
        <h2 className="text-2xl font-bold mb-4">Performance Features</h2>
        <ul className="space-y-2 text-sm">
          <li className="flex items-start gap-2">
            <span className="text-green-500 font-bold">✓</span>
            <span>
              <strong>WebP Format:</strong> Modern image format with 25-35% better compression
            </span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-green-500 font-bold">✓</span>
            <span>
              <strong>JPEG Fallback:</strong> Automatic fallback for older browsers
            </span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-green-500 font-bold">✓</span>
            <span>
              <strong>Lazy Loading:</strong> Images load only when entering viewport
            </span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-green-500 font-bold">✓</span>
            <span>
              <strong>Responsive Srcset:</strong> Multiple image sizes for different devices
            </span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-green-500 font-bold">✓</span>
            <span>
              <strong>Aspect Ratio:</strong> Prevents layout shift (CLS) during image load
            </span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-green-500 font-bold">✓</span>
            <span>
              <strong>R2 CDN:</strong> Global content delivery for fast loading
            </span>
          </li>
        </ul>
      </section>

      {/* Usage Guide */}
      <section>
        <h2 className="text-2xl font-bold mb-4">Quick Start</h2>
        <div className="bg-gray-50 dark:bg-gray-900 p-6 rounded-lg">
          <pre className="text-sm overflow-x-auto">
            <code>{`// Import components
import { Avatar } from './components/Avatar';
import { SceneCover } from './components/SceneCover';
import { OptimizedImage } from './components/OptimizedImage';

// Use in your components
<Avatar
  name="Alice Johnson"
  src="users/123/avatar.jpg"
  size="lg"
  online
/>

<SceneCover
  sceneName="Underground Techno"
  src="scenes/456/cover.jpg"
  size="medium"
  overlay
/>

<OptimizedImage
  src="posts/789/photo.jpg"
  alt="Concert photo"
  width={800}
  height={600}
  sizes="(max-width: 640px) 100vw, 50vw"
/>`}</code>
          </pre>
        </div>
      </section>
    </div>
  );
};
