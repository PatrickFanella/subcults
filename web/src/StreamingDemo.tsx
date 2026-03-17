/**
 * StreamingDemo
 * Demo page for streaming components with latency overlay
 */

import { useState } from 'react';
import { useLiveAudio } from './hooks/useLiveAudio';
import { useLatencyStore } from './stores/latencyStore';
import {
  JoinStreamButton,
  StreamLatencyOverlay,
  ParticipantList,
  AudioControls,
  ConnectionIndicator,
} from './components/streaming';

/**
 * StreamingDemo Component
 * Demonstrates LiveKit streaming with latency measurement overlay
 */
export function StreamingDemo() {
  const [roomName] = useState('demo-room');
  const [showOverlay, setShowOverlay] = useState(true);

  const {
    isConnected,
    isConnecting,
    participants,
    localParticipant,
    connectionQuality,
    error,
    connect,
    disconnect,
    toggleMute,
    setVolume,
  } = useLiveAudio(roomName);

  const { recordJoinClicked } = useLatencyStore();

  const handleJoin = async () => {
    // Record t0: Join button click
    recordJoinClicked();

    // Connect to room
    await connect();
  };

  return (
    <div className="min-h-screen bg-background p-8">
      <div className="max-w-4xl mx-auto space-y-8">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold text-foreground mb-2">
            🎵 Streaming Demo with Latency Tracking
          </h1>
          <p className="text-foreground-secondary">
            LiveKit audio streaming with real-time latency measurement overlay
          </p>
        </div>

        {/* Configuration */}
        <section className="space-y-4 p-6 bg-surface rounded-lg border border-border">
          <h2 className="text-xl font-semibold text-foreground">Configuration</h2>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-foreground-secondary">Room Name:</span>
              <code className="px-2 py-1 bg-background rounded text-sm">{roomName}</code>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-foreground-secondary">Connection Quality:</span>
              <span className="px-2 py-1 bg-background rounded text-sm capitalize">
                {connectionQuality}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <label htmlFor="overlay-toggle" className="text-foreground-secondary">
                Show Latency Overlay:
              </label>
              <input
                id="overlay-toggle"
                type="checkbox"
                checked={showOverlay}
                onChange={(e) => setShowOverlay(e.target.checked)}
                className="w-4 h-4"
              />
            </div>
          </div>
        </section>

        {/* Connection Status */}
        <section className="space-y-4 p-6 bg-surface rounded-lg border border-border">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-semibold text-foreground">Connection</h2>
            <ConnectionIndicator quality={connectionQuality} />
          </div>

          <div className="flex gap-4">
            <JoinStreamButton
              isConnected={isConnected}
              isConnecting={isConnecting}
              onJoin={handleJoin}
            />

            {isConnected && (
              <button
                onClick={disconnect}
                className="px-6 py-3 bg-status-error hover:opacity-90 text-white border border-status-error rounded-none font-bold uppercase tracking-[0.05em] transition-none"
              >
                Leave Room
              </button>
            )}
          </div>

          {error && (
            <div className="p-4 bg-background border border-status-error rounded-none">
              <p className="text-status-error text-sm">{error}</p>
            </div>
          )}
        </section>

        {/* Audio Controls */}
        {isConnected && localParticipant && (
          <section className="space-y-4 p-6 bg-surface rounded-lg border border-border">
            <h2 className="text-xl font-semibold text-foreground">Audio Controls</h2>
            <AudioControls
              isMuted={localParticipant.isMuted}
              onToggleMute={toggleMute}
              onLeave={disconnect}
              onVolumeChange={setVolume}
            />
          </section>
        )}

        {/* Participants */}
        {isConnected && (
          <section className="space-y-4 p-6 bg-surface rounded-lg border border-border">
            <h2 className="text-xl font-semibold text-foreground">
              Participants ({participants.length + (localParticipant ? 1 : 0)})
            </h2>
            <ParticipantList participants={participants} localParticipant={localParticipant} />
          </section>
        )}

        {/* Instructions */}
        <section className="space-y-4 p-6 bg-surface rounded-lg border border-border">
          <h2 className="text-xl font-semibold text-foreground">Instructions</h2>
          <ol className="list-decimal list-inside space-y-2 text-foreground-secondary">
            <li>Ensure VITE_LIVEKIT_WS_URL is set in your environment</li>
            <li>Click "Join Room" to connect to the audio room</li>
            <li>The latency overlay will appear showing join performance metrics</li>
            <li>
              Overlay shows:
              <ul className="list-disc list-inside ml-6 mt-1 space-y-1">
                <li>
                  <strong>Total:</strong> End-to-end join latency (target: &lt;2s)
                </li>
                <li>
                  <strong>Token fetch:</strong> Time to get token from backend
                </li>
                <li>
                  <strong>Room connect:</strong> Time to establish LiveKit connection
                </li>
                <li>
                  <strong>Audio sub:</strong> Time until first audio track is ready
                </li>
              </ul>
            </li>
            <li>Use the checkbox above to toggle overlay visibility</li>
            <li>Open browser console for detailed latency logs</li>
          </ol>
        </section>

        {/* Performance Notes */}
        <section className="space-y-4 p-6 bg-surface rounded-lg border border-border">
          <h2 className="text-xl font-semibold text-foreground">Performance Notes</h2>
          <div className="space-y-2 text-foreground-secondary text-sm">
            <p>
              <strong>Target:</strong> Total join latency &lt;2000ms
            </p>
            <p>
              <strong>Green indicator:</strong> Within target (&lt;2s)
            </p>
            <p>
              <strong>Red indicator:</strong> Above target (≥2s)
            </p>
            <p>
              <strong>Production:</strong> Overlay is automatically hidden in production builds
            </p>
            <p>
              <strong>Console logs:</strong> Only visible in development mode
            </p>
          </div>
        </section>
      </div>

      {/* Latency Overlay */}
      <StreamLatencyOverlay show={showOverlay} position="top-right" />
    </div>
  );
}

export default StreamingDemo;
