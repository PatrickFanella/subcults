/**
 * StreamUIComponentsDemo
 * Demo page showcasing all the new streaming UI components
 */

import { useState } from 'react';
import {
  StreamHeader,
  AudioLevelVisualization,
  ShareInviteButtons,
  ChatSidebar,
  ConnectionIndicator,
  AudioControls,
  ParticipantList,
  JoinStreamButton,
} from './components/streaming';
import type { ChatMessage } from './components/streaming';
import type { Participant, ConnectionQuality } from './types/streaming';

export function StreamUIComponentsDemo() {
  // Stream header state
  const [listenerCount, setListenerCount] = useState(42);
  const [isLive, setIsLive] = useState(true);

  // Audio level state
  const [audioLevel, setAudioLevel] = useState(0.5);
  const [isMuted, setIsMuted] = useState(false);

  // Connection state
  const [connectionQuality, setConnectionQuality] = useState<ConnectionQuality>('good');
  const [isConnected, setIsConnected] = useState(true);
  const [isConnecting, setIsConnecting] = useState(false);

  // Chat state
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([
    {
      id: '1',
      sender: 'Alice',
      message: 'Hey everyone! Great stream!',
      timestamp: Date.now() - 120000,
      isLocal: false,
    },
    {
      id: '2',
      sender: 'Bob',
      message: 'Loving the vibes 🎵',
      timestamp: Date.now() - 90000,
      isLocal: false,
    },
    {
      id: '3',
      sender: 'You',
      message: 'Thanks for joining!',
      timestamp: Date.now() - 60000,
      isLocal: true,
    },
  ]);

  // Mock participants
  const localParticipant: Participant = {
    identity: 'local-user',
    name: 'You',
    isLocal: true,
    isMuted: isMuted,
    isSpeaking: audioLevel > 0.3,
  };

  const participants: Participant[] = [
    {
      identity: 'user-1',
      name: 'Alice',
      isLocal: false,
      isMuted: false,
      isSpeaking: false,
    },
    {
      identity: 'user-2',
      name: 'Bob',
      isLocal: false,
      isMuted: false,
      isSpeaking: true,
    },
    {
      identity: 'user-3',
      name: 'Charlie',
      isLocal: false,
      isMuted: true,
      isSpeaking: false,
    },
  ];

  const handleSendMessage = (message: string) => {
    const newMessage: ChatMessage = {
      id:
        typeof crypto !== 'undefined' && 'randomUUID' in crypto
          ? crypto.randomUUID()
          : `${Date.now()}-${Math.random().toString(36).slice(2)}`,
      sender: 'You',
      message,
      timestamp: Date.now(),
      isLocal: true,
    };
    setChatMessages((prev) => [...prev, newMessage]);
  };

  const handleJoin = () => {
    setIsConnecting(true);
    setTimeout(() => {
      setIsConnected(true);
      setIsConnecting(false);
    }, 1500);
  };

  const handleLeave = () => {
    setIsConnected(false);
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white p-8">
      <div className="max-w-7xl mx-auto space-y-8">
        {/* Page Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold mb-2">🎵 Streaming UI Components Demo</h1>
          <p className="text-gray-400">
            Interactive showcase of all new streaming UI components
          </p>
        </div>

        {/* Controls Panel */}
        <section className="p-6 bg-gray-800 rounded-lg border border-gray-700">
          <h2 className="text-2xl font-semibold mb-4">Demo Controls</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm text-gray-400 mb-2">
                Listener Count: {listenerCount}
              </label>
              <input
                type="range"
                min="0"
                max="100"
                value={listenerCount}
                onChange={(e) => setListenerCount(parseInt(e.target.value))}
                className="w-full"
              />
            </div>

            <div>
              <label className="block text-sm text-gray-400 mb-2">
                Audio Level: {audioLevel.toFixed(2)}
              </label>
              <input
                type="range"
                min="0"
                max="1"
                step="0.01"
                value={audioLevel}
                onChange={(e) => setAudioLevel(parseFloat(e.target.value))}
                className="w-full"
              />
            </div>

            <div>
              <label className="block text-sm text-gray-400 mb-2">Connection Quality</label>
              <select
                value={connectionQuality}
                onChange={(e) => setConnectionQuality(e.target.value as ConnectionQuality)}
                className="w-full px-3 py-2 bg-gray-700 rounded border border-gray-600"
              >
                <option value="excellent">Excellent</option>
                <option value="good">Good</option>
                <option value="poor">Poor</option>
                <option value="unknown">Unknown</option>
              </select>
            </div>

            <div className="flex items-center gap-4">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={isLive}
                  onChange={(e) => setIsLive(e.target.checked)}
                  className="w-4 h-4"
                />
                <span className="text-sm text-gray-400">Is Live</span>
              </label>

              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={isMuted}
                  onChange={(e) => setIsMuted(e.target.checked)}
                  className="w-4 h-4"
                />
                <span className="text-sm text-gray-400">Muted</span>
              </label>
            </div>
          </div>
        </section>

        {/* Stream Header Component */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-semibold">StreamHeader Component</h2>
            <code className="px-3 py-1 bg-gray-800 rounded text-sm">
              {'<StreamHeader />'}
            </code>
          </div>
          <StreamHeader
            title="Underground Beats Session"
            organizer="DJ Collective"
            listenerCount={listenerCount}
            isLive={isLive}
          />
        </section>

        {/* Audio Level Visualization Component */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-semibold">AudioLevelVisualization Component</h2>
            <code className="px-3 py-1 bg-gray-800 rounded text-sm">
              {'<AudioLevelVisualization />'}
            </code>
          </div>
          <div className="p-6 bg-gray-800 rounded-lg border border-gray-700">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
              <div className="flex flex-col items-center gap-4">
                <span className="text-gray-400">Small Size</span>
                <AudioLevelVisualization
                  level={audioLevel}
                  isMuted={isMuted}
                  size="small"
                  showSpeaking={true}
                />
              </div>
              <div className="flex flex-col items-center gap-4">
                <span className="text-gray-400">Medium Size (Default)</span>
                <AudioLevelVisualization
                  level={audioLevel}
                  isMuted={isMuted}
                  size="medium"
                  showSpeaking={true}
                />
              </div>
              <div className="flex flex-col items-center gap-4">
                <span className="text-gray-400">Large Size</span>
                <AudioLevelVisualization
                  level={audioLevel}
                  isMuted={isMuted}
                  size="large"
                  showSpeaking={true}
                />
              </div>
            </div>
          </div>
        </section>

        {/* Share/Invite Buttons Component */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-semibold">ShareInviteButtons Component</h2>
            <code className="px-3 py-1 bg-gray-800 rounded text-sm">
              {'<ShareInviteButtons />'}
            </code>
          </div>
          <div className="p-6 bg-gray-800 rounded-lg border border-gray-700">
            <ShareInviteButtons
              streamUrl={window.location.href}
              streamTitle="Underground Beats Session"
              onShare={() => console.log('Share clicked')}
            />
          </div>
        </section>

        {/* Connection Indicator Component */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-semibold">ConnectionIndicator Component</h2>
            <code className="px-3 py-1 bg-gray-800 rounded text-sm">
              {'<ConnectionIndicator />'}
            </code>
          </div>
          <div className="p-6 bg-gray-800 rounded-lg border border-gray-700">
            <div className="flex gap-4 flex-wrap">
              <ConnectionIndicator quality="excellent" />
              <ConnectionIndicator quality="good" />
              <ConnectionIndicator quality="poor" />
              <ConnectionIndicator quality="unknown" />
            </div>
          </div>
        </section>

        {/* Join Stream Button Component */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-semibold">JoinStreamButton Component</h2>
            <code className="px-3 py-1 bg-gray-800 rounded text-sm">
              {'<JoinStreamButton />'}
            </code>
          </div>
          <div className="p-6 bg-gray-800 rounded-lg border border-gray-700">
            <JoinStreamButton
              isConnected={isConnected}
              isConnecting={isConnecting}
              onJoin={handleJoin}
            />
          </div>
        </section>

        {/* Full Layout Example */}
        <section className="space-y-4">
          <h2 className="text-2xl font-semibold">Full Stream Page Layout</h2>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Left Column: Controls and Participants */}
            <div className="lg:col-span-2 space-y-6">
              <AudioControls
                isMuted={isMuted}
                onToggleMute={() => setIsMuted(!isMuted)}
                onLeave={handleLeave}
                onVolumeChange={(vol) => console.log('Volume:', vol)}
              />

              <div>
                <h3 className="text-xl font-semibold mb-4">
                  Participants ({participants.length + 1})
                </h3>
                <ParticipantList
                  participants={participants}
                  localParticipant={localParticipant}
                />
              </div>
            </div>

            {/* Right Column: Chat */}
            <div>
              <ChatSidebar
                messages={chatMessages}
                onSendMessage={handleSendMessage}
                maxHeight="600px"
              />
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}

export default StreamUIComponentsDemo;
