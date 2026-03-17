/**
 * User Slice
 * User profile state management (privacy-safe, no sensitive PII)
 */

import type { StateCreator } from 'zustand';
import { apiClient } from '../../lib/api-client';
import type { User } from '../authStore';
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
 * Create user slice with actions
 */
export const createUserSlice: StateCreator<
  EntityStore,
  [],
  [],
  Pick<EntityStore, 'user' | 'fetchUser' | 'setUser' | 'markUserStale' | 'clearUserError' | 'removeUser'>
> = (set, get) => ({
  user: {
    users: {},
  },

  fetchUser: async (did: string): Promise<User> => {
    return getOrCreateRequest(`user:${did}`, async () => {
      const state = get();
      const cached = state.user.users[did];

      // Set loading state
      set((state) => ({
        user: {
          ...state.user,
          users: {
            ...state.user.users,
            [did]: {
              data: cached?.data || ({} as User),
              metadata: setLoadingMetadata(cached?.metadata || createFreshMetadata()),
            },
          },
        },
      }));

      try {
        // Fetch from API - only basic profile info (no sensitive PII)
        const user = await apiClient.get<User>(`/users/${did}`);

        // Update cache with fresh data
        set((state) => ({
          user: {
            ...state.user,
            users: {
              ...state.user.users,
              [did]: {
                data: user,
                metadata: setSuccessMetadata(),
              },
            },
          },
        }));

        return user;
      } catch (error: unknown) {
        const errorMessage = (error as Error)?.message || 'Failed to fetch user';

        // Update cache with error
        set((state) => ({
          user: {
            ...state.user,
            users: {
              ...state.user.users,
              [did]: cached
                ? {
                    ...cached,
                    metadata: setErrorMetadata(cached.metadata, errorMessage),
                  }
                : {
                    data: {} as User,
                    metadata: setErrorMetadata(createFreshMetadata(), errorMessage),
                  },
            },
          },
        }));

        throw error;
      }
    });
  },

  setUser: (user: User) => {
    set((state) => ({
      user: {
        ...state.user,
        users: {
          ...state.user.users,
          [user.did]: {
            data: user,
            metadata: setSuccessMetadata(),
          },
        },
      },
    }));
  },

  markUserStale: (did: string) => {
    set((state) => {
      const cached = state.user.users[did];
      if (!cached) return state;

      return {
        user: {
          ...state.user,
          users: {
            ...state.user.users,
            [did]: {
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

  clearUserError: (did: string) => {
    set((state) => {
      const cached = state.user.users[did];
      if (!cached || !cached.metadata.error) return state;

      return {
        user: {
          ...state.user,
          users: {
            ...state.user.users,
            [did]: {
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

  removeUser: (did: string) => {
    set((state) => {
      const remainingUsers = { ...state.user.users };
      delete remainingUsers[did];

      return {
        user: {
          users: remainingUsers,
        },
      };
    });
  },
});
