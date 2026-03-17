/**
 * Mock LiveKit Server for E2E Testing
 * 
 * This mock server simulates LiveKit WebRTC functionality without requiring
 * actual LiveKit infrastructure. It supports:
 * - Token generation
 * - Room lifecycle (create/join/leave/end)
 * - Participant management
 * - Organizer controls (mute/kick)
 * - Connection quality simulation
 * - Network condition simulation
 */

import { WebSocketServer, WebSocket } from 'ws';

export interface MockParticipant {
  identity: string;
  name: string;
  isOrganizer: boolean;
  isMuted: boolean;
  isSpeaking: boolean;
  connectionQuality: 'excellent' | 'good' | 'poor' | 'unknown';
  joinedAt: Date;
  reconnectCount: number;
}

export interface MockRoom {
  id: string;
  participants: Map<string, MockParticipant>;
  isLocked: boolean;
  createdAt: Date;
}

export class MockLiveKitServer {
  private wss: WebSocketServer | null = null;
  private rooms = new Map<string, MockRoom>();
  private connections = new Map<string, WebSocket>();
  
  constructor(private port: number = 7880) {}

  /**
   * Start the mock server
   */
  start(): Promise<void> {
    return new Promise((resolve) => {
      this.wss = new WebSocketServer({ port: this.port });
      
      this.wss.on('connection', (ws: WebSocket, req) => {
        const url = new URL(req.url || '', `http://localhost:${this.port}`);
        const roomId = url.searchParams.get('room') || 'default';
        const identity = url.searchParams.get('identity') || `user-${Date.now()}`;
        
        console.log(`[MockLiveKit] Client connected: ${identity} to room ${roomId}`);
        
        // Store connection
        this.connections.set(identity, ws);
        
        // Get or create room
        if (!this.rooms.has(roomId)) {
          this.rooms.set(roomId, {
            id: roomId,
            participants: new Map(),
            isLocked: false,
            createdAt: new Date(),
          });
        }
        
        const room = this.rooms.get(roomId)!;
        
        // Check if room is locked
        if (room.isLocked && !room.participants.has(identity)) {
          ws.send(JSON.stringify({
            type: 'error',
            message: 'Room is locked',
          }));
          ws.close();
          return;
        }
        
        // Add participant if new
        if (!room.participants.has(identity)) {
          room.participants.set(identity, {
            identity,
            name: identity,
            isOrganizer: room.participants.size === 0, // First participant is organizer
            isMuted: false,
            isSpeaking: false,
            connectionQuality: 'excellent',
            joinedAt: new Date(),
            reconnectCount: 0,
          });
          
          // Notify all participants
          this.broadcastToRoom(roomId, {
            type: 'participant_joined',
            participant: this.serializeParticipant(room.participants.get(identity)!),
          });
        } else {
          // Reconnection
          const participant = room.participants.get(identity)!;
          participant.reconnectCount++;
          
          this.broadcastToRoom(roomId, {
            type: 'participant_reconnected',
            participant: this.serializeParticipant(participant),
          });
        }
        
        // Send initial room state
        ws.send(JSON.stringify({
          type: 'room_state',
          room: this.serializeRoom(room),
        }));
        
        // Handle messages from client
        ws.on('message', (data: Buffer) => {
          try {
            const message = JSON.parse(data.toString());
            this.handleMessage(roomId, identity, message);
          } catch (err) {
            console.error('[MockLiveKit] Failed to parse message:', err);
          }
        });
        
        // Handle disconnection
        ws.on('close', () => {
          console.log(`[MockLiveKit] Client disconnected: ${identity}`);
          this.connections.delete(identity);
          
          if (this.rooms.has(roomId)) {
            const room = this.rooms.get(roomId)!;
            room.participants.delete(identity);
            
            this.broadcastToRoom(roomId, {
              type: 'participant_disconnected',
              identity,
            });
            
            // Clean up empty rooms after 5 seconds
            if (room.participants.size === 0) {
              setTimeout(() => {
                if (room.participants.size === 0) {
                  this.rooms.delete(roomId);
                  console.log(`[MockLiveKit] Room ${roomId} cleaned up`);
                }
              }, 5000);
            }
          }
        });
      });
      
      this.wss.on('listening', () => {
        console.log(`[MockLiveKit] Server started on port ${this.port}`);
        resolve();
      });
    });
  }

  /**
   * Stop the mock server
   */
  async stop(): Promise<void> {
    return new Promise((resolve) => {
      if (this.wss) {
        this.wss.close(() => {
          console.log('[MockLiveKit] Server stopped');
          resolve();
        });
      } else {
        resolve();
      }
    });
  }

  /**
   * Handle client messages
   */
  private handleMessage(roomId: string, identity: string, message: any): void {
    const room = this.rooms.get(roomId);
    if (!room) return;
    
    const participant = room.participants.get(identity);
    if (!participant) return;
    
    switch (message.type) {
      case 'mute':
        participant.isMuted = message.muted;
        this.broadcastToRoom(roomId, {
          type: 'participant_muted',
          identity,
          muted: message.muted,
        });
        break;
        
      case 'kick_participant':
        if (participant.isOrganizer) {
          const targetIdentity = message.targetIdentity;
          const targetWs = this.connections.get(targetIdentity);
          
          if (targetWs) {
            targetWs.send(JSON.stringify({
              type: 'kicked',
              message: 'You have been removed from the room',
            }));
            targetWs.close();
          }
          
          room.participants.delete(targetIdentity);
          this.broadcastToRoom(roomId, {
            type: 'participant_kicked',
            identity: targetIdentity,
          });
        }
        break;
        
      case 'mute_participant':
        if (participant.isOrganizer) {
          const targetIdentity = message.targetIdentity;
          const targetParticipant = room.participants.get(targetIdentity);
          
          if (targetParticipant) {
            targetParticipant.isMuted = true;
            
            const targetWs = this.connections.get(targetIdentity);
            if (targetWs) {
              targetWs.send(JSON.stringify({
                type: 'force_muted',
                message: 'You have been muted by the organizer',
              }));
            }
            
            this.broadcastToRoom(roomId, {
              type: 'participant_muted',
              identity: targetIdentity,
              muted: true,
            });
          }
        }
        break;
        
      case 'lock_room':
        if (participant.isOrganizer) {
          room.isLocked = message.locked;
          this.broadcastToRoom(roomId, {
            type: 'room_locked',
            locked: message.locked,
          });
        }
        break;
        
      case 'end_stream':
        if (participant.isOrganizer) {
          this.broadcastToRoom(roomId, {
            type: 'stream_ended',
            message: 'The stream has ended',
          });
          
          // Close all connections
          room.participants.forEach((_, participantId) => {
            const ws = this.connections.get(participantId);
            if (ws) {
              ws.close();
            }
          });
          
          // Remove room
          this.rooms.delete(roomId);
        }
        break;
        
      case 'simulate_quality_change':
        participant.connectionQuality = message.quality;
        this.broadcastToRoom(roomId, {
          type: 'quality_changed',
          identity,
          quality: message.quality,
        });
        break;
    }
  }

  /**
   * Broadcast message to all participants in a room
   */
  private broadcastToRoom(roomId: string, message: any): void {
    const room = this.rooms.get(roomId);
    if (!room) return;
    
    const messageStr = JSON.stringify(message);
    room.participants.forEach((_, identity) => {
      const ws = this.connections.get(identity);
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(messageStr);
      }
    });
  }

  /**
   * Serialize participant for transmission
   */
  private serializeParticipant(participant: MockParticipant) {
    return {
      identity: participant.identity,
      name: participant.name,
      isOrganizer: participant.isOrganizer,
      isMuted: participant.isMuted,
      isSpeaking: participant.isSpeaking,
      connectionQuality: participant.connectionQuality,
      reconnectCount: participant.reconnectCount,
    };
  }

  /**
   * Serialize room for transmission
   */
  private serializeRoom(room: MockRoom) {
    return {
      id: room.id,
      isLocked: room.isLocked,
      participants: Array.from(room.participants.values()).map(p => 
        this.serializeParticipant(p)
      ),
    };
  }

  /**
   * Generate a mock LiveKit token
   */
  generateToken(roomId: string, identity: string): string {
    const payload = {
      room: roomId,
      identity,
      exp: Math.floor(Date.now() / 1000) + 300, // 5 minutes
      iat: Math.floor(Date.now() / 1000),
    };
    
    // Simple base64 encoding (not secure, only for testing)
    return Buffer.from(JSON.stringify(payload)).toString('base64');
  }

  /**
   * Get room state for testing
   */
  getRoom(roomId: string): MockRoom | undefined {
    return this.rooms.get(roomId);
  }

  /**
   * Simulate network delay
   */
  simulateNetworkDelay(roomId: string, delayMs: number): void {
    const room = this.rooms.get(roomId);
    if (!room) return;
    
    room.participants.forEach((participant) => {
      setTimeout(() => {
        this.broadcastToRoom(roomId, {
          type: 'latency_spike',
          identity: participant.identity,
          delayMs,
        });
      }, delayMs);
    });
  }

  /**
   * Simulate packet loss
   */
  simulatePacketLoss(roomId: string, lossPercentage: number): void {
    const room = this.rooms.get(roomId);
    if (!room) return;
    
    room.participants.forEach((participant) => {
      participant.connectionQuality = lossPercentage > 20 ? 'poor' : 
                                     lossPercentage > 10 ? 'good' : 'excellent';
      
      this.broadcastToRoom(roomId, {
        type: 'quality_changed',
        identity: participant.identity,
        quality: participant.connectionQuality,
      });
    });
  }
}
