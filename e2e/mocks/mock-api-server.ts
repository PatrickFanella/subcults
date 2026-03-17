/**
 * Mock API Server for E2E Testing
 *
 * Simulates the Subcults API server endpoints needed for streaming tests.
 * Primarily handles LiveKit token generation.
 */

import express, { Express, Request, Response } from 'express';
import type { Server } from 'http';
import { MockLiveKitServer } from './mock-livekit-server';

export class MockAPIServer {
  private app: Express;
  private server: Server | null = null;
  private mockLiveKit: MockLiveKitServer;

  constructor(
    private port: number = 8080,
    private liveKitPort: number = 7880
  ) {
    this.app = express();
    this.mockLiveKit = new MockLiveKitServer(liveKitPort);
    this.setupMiddleware();
    this.setupRoutes();
  }

  /**
   * Setup Express middleware
   */
  private setupMiddleware(): void {
    this.app.use(express.json());

    // CORS for testing
    this.app.use((req, res, next) => {
      res.header('Access-Control-Allow-Origin', '*');
      res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
      res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization');

      if (req.method === 'OPTIONS') {
        res.sendStatus(200);
      } else {
        next();
      }
    });
  }

  /**
   * Setup API routes
   */
  private setupRoutes(): void {
    // Health check
    this.app.get('/health', (req: Request, res: Response) => {
      res.json({ status: 'ok', timestamp: new Date().toISOString() });
    });

    // ── Auth endpoints ──

    this.app.post('/api/auth/login', (req: Request, res: Response) => {
      const { handle, password } = req.body;
      if (!handle || !password) {
        res.status(400).json({ error: 'handle and password required' });
        return;
      }
      if (handle === 'invalid@test.com') {
        res.status(401).json({ error: 'invalid credentials' });
        return;
      }
      res.json({
        access_token: 'e2e-access-token',
        refresh_token: 'e2e-refresh-token',
        user: {
          did: 'did:plc:e2euser',
          handle: handle,
          display_name: 'E2E User',
        },
      });
    });

    this.app.post('/api/auth/refresh', (req: Request, res: Response) => {
      const { refresh_token } = req.body;
      if (refresh_token !== 'e2e-refresh-token') {
        res.status(401).json({ error: 'invalid refresh token' });
        return;
      }
      res.json({
        access_token: 'e2e-access-token-refreshed',
        refresh_token: 'e2e-refresh-token',
      });
    });

    this.app.get('/api/auth/me', (req: Request, res: Response) => {
      const auth = req.headers.authorization;
      if (!auth || !auth.startsWith('Bearer e2e-access-token')) {
        res.status(401).json({ error: 'unauthorized' });
        return;
      }
      res.json({
        did: 'did:plc:e2euser',
        handle: 'testuser.subcults.tv',
        display_name: 'E2E User',
      });
    });

    // ── Scenes endpoints ──

    this.app.get('/api/scenes', (req: Request, res: Response) => {
      res.json({
        scenes: [
          {
            id: 'scene-1',
            name: 'Brooklyn Underground',
            description: 'Deep house and techno in Brooklyn',
            genre: 'electronic',
            owner_did: 'did:plc:owner1',
            coarse_geohash: 'dr5ru7',
            allow_precise: false,
            latitude: 40.6782,
            longitude: -73.9442,
          },
          {
            id: 'scene-2',
            name: 'Berlin Beats',
            description: 'Berlin techno scene community',
            genre: 'techno',
            owner_did: 'did:plc:owner2',
            coarse_geohash: 'u33db8',
            allow_precise: false,
            latitude: 52.52,
            longitude: 13.405,
          },
        ],
        total: 2,
      });
    });

    this.app.get('/api/scenes/:id', (req: Request, res: Response) => {
      const id = req.params.id;
      res.json({
        id,
        name: 'Brooklyn Underground',
        description: 'Deep house and techno in Brooklyn',
        genre: 'electronic',
        owner_did: 'did:plc:owner1',
        coarse_geohash: 'dr5ru7',
        allow_precise: false,
        latitude: 40.6782,
        longitude: -73.9442,
        member_count: 42,
        created_at: '2024-01-15T00:00:00Z',
      });
    });

    // ── Events endpoints ──

    this.app.get('/api/events', (req: Request, res: Response) => {
      res.json({
        events: [
          {
            id: 'event-1',
            scene_id: 'scene-1',
            title: 'Friday Night Sessions',
            description: 'Weekly deep house night',
            starts_at: new Date(Date.now() + 86400000).toISOString(),
            ends_at: new Date(Date.now() + 100800000).toISOString(),
            status: 'upcoming',
          },
        ],
        total: 1,
      });
    });

    this.app.get('/api/events/:id', (req: Request, res: Response) => {
      const id = req.params.id;
      res.json({
        id,
        scene_id: 'scene-1',
        title: 'Friday Night Sessions',
        description: 'Weekly deep house night',
        starts_at: new Date(Date.now() + 86400000).toISOString(),
        ends_at: new Date(Date.now() + 100800000).toISOString(),
        status: 'upcoming',
      });
    });

    // ── Search endpoint ──

    this.app.get('/api/search', (req: Request, res: Response) => {
      const q = req.query.q as string;
      if (!q) {
        res.json({ results: [], total: 0 });
        return;
      }
      res.json({
        results: [
          {
            type: 'scene',
            id: 'scene-1',
            name: 'Brooklyn Underground',
            description: 'Deep house and techno in Brooklyn',
            score: 0.95,
          },
        ],
        total: 1,
      });
    });

    // ── Feed endpoint ──

    this.app.get('/api/feed', (req: Request, res: Response) => {
      res.json({
        posts: [
          {
            id: 'post-1',
            author_did: 'did:plc:author1',
            text: 'Great set last night!',
            scene_id: 'scene-1',
            created_at: new Date().toISOString(),
          },
        ],
        cursor: null,
      });
    });

    // LiveKit token generation (matches production API contract)
    this.app.post('/api/livekit/token', (req: Request, res: Response) => {
      const { room_id, scene_id, event_id } = req.body;

      if (!room_id) {
        res.status(400).json({
          error: 'room_id is required',
        });
        return;
      }

      // Validate room_id format (alphanumeric, hyphens, underscores, colons, max 128 chars)
      // Matches production: ^[a-zA-Z0-9_:-]{1,128}$
      if (!/^[a-zA-Z0-9_:-]{1,128}$/.test(room_id)) {
        res.status(400).json({
          error: 'Invalid room_id format',
        });
        return;
      }

      // Derive identity from scene/event IDs (deterministic for E2E tests)
      const identityParts: string[] = ['e2e'];
      if (scene_id) {
        identityParts.push(`scene-${scene_id}`);
      }
      if (event_id) {
        identityParts.push(`event-${event_id}`);
      }
      const identity = identityParts.join(':');

      const token = this.mockLiveKit.generateToken(room_id, identity);
      const expiresAt = new Date(Date.now() + 300000).toISOString(); // 5 minutes

      // Match production response structure (snake_case)
      res.json({
        token,
        expires_at: expiresAt,
      });
    });

    // Simulate latency for testing
    this.app.post('/api/test/simulate-latency', (req: Request, res: Response) => {
      const { roomId, delayMs } = req.body;

      if (roomId) {
        this.mockLiveKit.simulateNetworkDelay(roomId, delayMs || 1000);
        res.json({ success: true, roomId, delayMs });
      } else {
        res.status(400).json({ error: 'roomId required' });
      }
    });

    // Simulate packet loss for testing
    this.app.post('/api/test/simulate-packet-loss', (req: Request, res: Response) => {
      const { roomId, lossPercentage } = req.body;

      if (roomId) {
        this.mockLiveKit.simulatePacketLoss(roomId, lossPercentage || 10);
        res.json({ success: true, roomId, lossPercentage });
      } else {
        res.status(400).json({ error: 'roomId required' });
      }
    });

    // Get room state for testing
    this.app.get('/api/test/room/:roomId', (req: Request, res: Response) => {
      const roomId = Array.isArray(req.params.roomId) ? req.params.roomId[0] : req.params.roomId;
      const room = this.mockLiveKit.getRoom(roomId);

      if (room) {
        res.json({
          id: room.id,
          isLocked: room.isLocked,
          participantCount: room.participants.size,
          participants: Array.from(room.participants.values()).map((p) => ({
            identity: p.identity,
            name: p.name,
            isOrganizer: p.isOrganizer,
            isMuted: p.isMuted,
            connectionQuality: p.connectionQuality,
            reconnectCount: p.reconnectCount,
          })),
        });
      } else {
        res.status(404).json({ error: 'Room not found' });
      }
    });
  }

  /**
   * Start both the API server and mock LiveKit server
   */
  async start(): Promise<void> {
    // Start LiveKit mock first
    await this.mockLiveKit.start();

    // Then start API server
    return new Promise((resolve) => {
      this.server = this.app.listen(this.port, () => {
        console.log(`[MockAPI] Server started on port ${this.port}`);
        resolve();
      });
    });
  }

  /**
   * Stop both servers
   */
  async stop(): Promise<void> {
    // Stop API server
    if (this.server) {
      await new Promise<void>((resolve) => {
        this.server!.close(() => {
          console.log('[MockAPI] Server stopped');
          resolve();
        });
      });
    }

    // Stop LiveKit mock
    await this.mockLiveKit.stop();
  }

  /**
   * Get the Express app for testing
   */
  getApp(): Express {
    return this.app;
  }
}
