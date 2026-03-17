/**
 * Event Slice
 * Event-specific state management
 */

import type { StateCreator } from 'zustand';
import { apiClient } from '../../lib/api-client';
import type { Event } from '../../types/scene';
import type {
  EntityStore,
} from '../entityStore';
import {
  createFreshMetadata,
  setLoadingMetadata,
  setSuccessMetadata,
  setErrorMetadata,
  getOrCreateRequest,
} from '../entityStore';

/**
 * Create event slice with actions
 */
export const createEventSlice: StateCreator<
  EntityStore,
  [],
  [],
  Pick<EntityStore, 'event' | 'fetchEvent' | 'setEvent' | 'markEventStale' | 'clearEventError' | 'removeEvent'>
> = (set, get) => ({
  event: {
    events: {},
  },

  fetchEvent: async (id: string): Promise<Event> => {
    return getOrCreateRequest(`event:${id}`, async () => {
      const state = get();
      const cached = state.event.events[id];

      // Set loading state
      set((state) => ({
        event: {
          ...state.event,
          events: {
            ...state.event.events,
            [id]: {
              data: cached?.data || ({} as Event),
              metadata: setLoadingMetadata(cached?.metadata || createFreshMetadata()),
            },
          },
        },
      }));

      try {
        // Fetch from API
        const event = await apiClient.get<Event>(`/events/${id}`);

        // Update cache with fresh data
        set((state) => ({
          event: {
            ...state.event,
            events: {
              ...state.event.events,
              [id]: {
                data: event,
                metadata: setSuccessMetadata(),
              },
            },
          },
        }));

        return event;
      } catch (error: unknown) {
        const errorMessage = (error as Error)?.message || 'Failed to fetch event';

        // Update cache with error
        set((state) => ({
          event: {
            ...state.event,
            events: {
              ...state.event.events,
              [id]: cached
                ? {
                    ...cached,
                    metadata: setErrorMetadata(cached.metadata, errorMessage),
                  }
                : {
                    data: {} as Event,
                    metadata: setErrorMetadata(createFreshMetadata(), errorMessage),
                  },
            },
          },
        }));

        throw error;
      }
    });
  },

  setEvent: (event: Event) => {
    set((state) => ({
      event: {
        ...state.event,
        events: {
          ...state.event.events,
          [event.id]: {
            data: event,
            metadata: setSuccessMetadata(),
          },
        },
      },
    }));
  },

  markEventStale: (id: string) => {
    set((state) => {
      const cached = state.event.events[id];
      if (!cached) return state;

      return {
        event: {
          ...state.event,
          events: {
            ...state.event.events,
            [id]: {
              ...cached,
              metadata: {
                ...cached.metadata,
                stale: true,
              },
            },
          },
        },
      };
    });
  },

  clearEventError: (id: string) => {
    set((state) => {
      const cached = state.event.events[id];
      if (!cached || !cached.metadata.error) return state;

      return {
        event: {
          ...state.event,
          events: {
            ...state.event.events,
            [id]: {
              ...cached,
              metadata: {
                ...cached.metadata,
                error: null,
              },
            },
          },
        },
      };
    });
  },

  removeEvent: (id: string) => {
    set((state) => {
      const remainingEvents = { ...state.event.events };
      delete remainingEvents[id];

      return {
        event: {
          events: remainingEvents,
        },
      };
    });
  },
});
