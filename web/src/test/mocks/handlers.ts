/**
 * MSW Request Handlers
 * Mock API endpoints for integration testing
 */

import { http, HttpResponse } from 'msw';

// Mock base URL for API
const API_BASE = 'http://localhost:8080/api';

/**
 * Mock authentication handlers
 */
const authHandlers = [
  // Login endpoint
  http.post(`${API_BASE}/auth/login`, async ({ request }) => {
    const body = await request.json() as { username: string; password: string };
    
    // Simulate auth failure for specific credentials
    if (body.password === 'wrongpassword') {
      return HttpResponse.json(
        { error: 'Invalid credentials' },
        { status: 401 }
      );
    }
    
    // Simulate successful login
    return HttpResponse.json({
      accessToken: 'mock-access-token',
      user: {
        did: `did:example:${body.username}`,
        role: body.username === 'admin' ? 'admin' : 'user',
      },
    });
  }),

  // Logout endpoint
  http.post(`${API_BASE}/auth/logout`, () => {
    return HttpResponse.json({ success: true });
  }),

  // Token refresh endpoint
  http.post(`${API_BASE}/auth/refresh`, () => {
    return HttpResponse.json({
      accessToken: 'mock-refreshed-access-token',
      user: {
        did: 'did:example:mock-user',
        role: 'user',
      },
    });
  }),
];

/**
 * Mock scene handlers
 */
const sceneHandlers = [
  // Get scene by ID
  http.get(`${API_BASE}/scenes/:id`, ({ params }) => {
    const { id } = params;
    
    return HttpResponse.json({
      id,
      name: `Test Scene ${id}`,
      description: 'A mock scene for testing',
      location: {
        latitude: 37.7749,
        longitude: -122.4194,
        city: 'San Francisco',
        country: 'US',
      },
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      organizer_did: 'did:example:organizer',
      allow_precise: true,
    });
  }),

  // List scenes
  http.get(`${API_BASE}/scenes`, ({ request }) => {
    const url = new URL(request.url);
    const search = url.searchParams.get('search') || '';
    
    const scenes = [
      {
        id: 'scene-1',
        name: search ? `${search} Scene 1` : 'Underground Beats',
        description: 'Electronic music scene',
        location: { latitude: 37.7749, longitude: -122.4194, city: 'San Francisco' },
      },
      {
        id: 'scene-2',
        name: search ? `${search} Scene 2` : 'Indie Rock Haven',
        description: 'Indie rock scene',
        location: { latitude: 40.7128, longitude: -74.0060, city: 'New York' },
      },
    ];

    return HttpResponse.json({ scenes, total: scenes.length });
  }),

  // Create scene (admin only)
  http.post(`${API_BASE}/scenes`, async ({ request }) => {
    const body = await request.json() as {
      name: string;
      description: string;
      location: { latitude: number; longitude: number };
    };
    
    // Check for authorization header
    const authHeader = request.headers.get('Authorization');
    if (!authHeader) {
      return HttpResponse.json(
        { error: 'Unauthorized' },
        { status: 401 }
      );
    }
    
    return HttpResponse.json({
      id: 'new-scene-123',
      ...body,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      organizer_did: 'did:example:admin',
      allow_precise: true,
    }, { status: 201 });
  }),

  // Update scene settings
  http.patch(`${API_BASE}/scenes/:id`, async ({ params, request }) => {
    const { id } = params;
    const body = await request.json() as Record<string, unknown>;
    
    return HttpResponse.json({
      id,
      ...body,
      updated_at: new Date().toISOString(),
    });
  }),
];

/**
 * Mock event handlers
 */
const eventHandlers = [
  // Get events for a scene
  http.get(`${API_BASE}/scenes/:sceneId/events`, ({ params }) => {
    const { sceneId } = params;
    
    // Use dynamic dates relative to current time
    const now = Date.now();
    const sevenDaysFromNow = new Date(now + 7 * 24 * 60 * 60 * 1000);
    const eightDaysFromNow = new Date(now + 8 * 24 * 60 * 60 * 1000);
    
    const events = [
      {
        id: 'event-1',
        scene_id: sceneId,
        name: 'Friday Night Sessions',
        description: 'Weekly electronic music night',
        start_time: new Date(sevenDaysFromNow.setHours(22, 0, 0, 0)).toISOString(),
        end_time: new Date(sevenDaysFromNow.setHours(28, 0, 0, 0)).toISOString(), // 4am next day
        location: {
          latitude: 37.7749,
          longitude: -122.4194,
          venue: 'Test Venue',
        },
      },
      {
        id: 'event-2',
        scene_id: sceneId,
        name: 'Saturday Showcase',
        description: 'Live performances',
        start_time: new Date(eightDaysFromNow.setHours(21, 0, 0, 0)).toISOString(),
        end_time: new Date(eightDaysFromNow.setHours(27, 0, 0, 0)).toISOString(), // 3am next day
        location: {
          latitude: 37.7749,
          longitude: -122.4194,
          venue: 'Test Venue',
        },
      },
    ];

    return HttpResponse.json({ events, total: events.length });
  }),

  // Get event by ID
  http.get(`${API_BASE}/events/:id`, ({ params }) => {
    const { id } = params;
    
    // Use dynamic dates - 7 days in the future
    const sevenDaysFromNow = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);
    
    return HttpResponse.json({
      id,
      scene_id: 'scene-123',
      name: `Test Event ${id}`,
      description: 'A mock event for testing',
      start_time: new Date(sevenDaysFromNow.setHours(22, 0, 0, 0)).toISOString(),
      end_time: new Date(sevenDaysFromNow.setHours(28, 0, 0, 0)).toISOString(), // 4am next day
    });
  }),
];

/**
 * Mock streaming/LiveKit handlers
 */
const streamHandlers = [
  // Get LiveKit token
  http.post(`${API_BASE}/livekit/token`, async ({ request }) => {
    const body = await request.json() as {
      room_id: string;
      scene_id?: string;
      event_id?: string;
    };
    
    if (!body.room_id) {
      return HttpResponse.json(
        { error: 'room_id is required' },
        { status: 400 }
      );
    }
    
    return HttpResponse.json({
      token: 'mock-livekit-token',
      expires_at: new Date(Date.now() + 300000).toISOString(),
    });
  }),

  // Get stream participants
  http.get(`${API_BASE}/streams/:roomId/participants`, ({ params }) => {
    const { roomId } = params;
    
    return HttpResponse.json({
      room_id: roomId,
      participants: [
        {
          identity: 'participant-1',
          name: 'Test User 1',
          joined_at: new Date(Date.now() - 60000).toISOString(),
        },
        {
          identity: 'participant-2',
          name: 'Test User 2',
          joined_at: new Date(Date.now() - 30000).toISOString(),
        },
      ],
    });
  }),
];

/**
 * Mock search handlers
 */
const searchHandlers = [
  // Search across scenes and events
  http.get(`${API_BASE}/search`, ({ request }) => {
    const url = new URL(request.url);
    const query = url.searchParams.get('q') || '';
    
    const results = {
      scenes: [
        {
          id: 'scene-1',
          type: 'scene',
          name: `${query} Scene`,
          description: 'Matching scene',
        },
      ],
      events: [
        {
          id: 'event-1',
          type: 'event',
          name: `${query} Event`,
          description: 'Matching event',
          start_time: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
        },
      ],
    };

    return HttpResponse.json(results);
  }),
];

/**
 * Mock settings handlers
 */
const settingsHandlers = [
  // Get user settings
  http.get(`${API_BASE}/users/me/settings`, () => {
    return HttpResponse.json({
      theme: 'dark',
      notifications: {
        email: true,
        push: false,
        scene_updates: true,
      },
      privacy: {
        allow_precise_location: false,
        show_profile: true,
      },
    });
  }),

  // Update user settings
  http.patch(`${API_BASE}/users/me/settings`, async ({ request }) => {
    const body = await request.json() as Record<string, unknown>;
    
    return HttpResponse.json({
      ...body,
      updated_at: new Date().toISOString(),
    });
  }),
];

/**
 * All request handlers
 */
export const handlers = [
  ...authHandlers,
  ...sceneHandlers,
  ...eventHandlers,
  ...streamHandlers,
  ...searchHandlers,
  ...settingsHandlers,
];
