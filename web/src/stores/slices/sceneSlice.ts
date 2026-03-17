/**
 * Scene Slice
 * Scene-specific state management with optimistic updates
 */

import type { StateCreator } from 'zustand';
import { apiClient } from '../../lib/api-client';
import type { Scene } from '../../types/scene';
import type { EntityStore } from '../entityStore';
import {
  createFreshMetadata,
  setLoadingMetadata,
  setSuccessMetadata,
  setErrorMetadata,
  getOrCreateRequest,
} from '../entityStore';

/**
 * Create scene slice with actions
 */
export const createSceneSlice: StateCreator<
  EntityStore,
  [],
  [],
  Pick<EntityStore, 'scene' | 'fetchScene' | 'setScene' | 'markSceneStale' | 'optimisticJoinScene' | 'rollbackSceneUpdate' | 'commitSceneUpdate' | 'clearSceneError' | 'removeScene' | 'updateScene'>
> = (set, get) => ({
  scene: {
    scenes: {},
    optimisticUpdates: {},
  },

  fetchScene: async (id: string): Promise<Scene> => {
    return getOrCreateRequest(`scene:${id}`, async () => {
      const state = get();
      const cached = state.scene.scenes[id];

      // Set loading state
      set((state) => ({
        scene: {
          ...state.scene,
          scenes: {
            ...state.scene.scenes,
            [id]: {
              data: cached?.data || ({} as Scene),
              metadata: setLoadingMetadata(cached?.metadata || createFreshMetadata()),
            },
          },
        },
      }));

      try {
        // Fetch from API
        const scene = await apiClient.get<Scene>(`/scenes/${id}`);

        // Update cache with fresh data
        set((state) => ({
          scene: {
            ...state.scene,
            scenes: {
              ...state.scene.scenes,
              [id]: {
                data: scene,
                metadata: setSuccessMetadata(),
              },
            },
          },
        }));

        return scene;
      } catch (error: unknown) {
        const errorMessage = (error as Error)?.message || 'Failed to fetch scene';

        // Update cache with error
        set((state) => ({
          scene: {
            ...state.scene,
            scenes: {
              ...state.scene.scenes,
              [id]: cached
                ? {
                    ...cached,
                    metadata: setErrorMetadata(cached.metadata, errorMessage),
                  }
                : {
                    data: {} as Scene,
                    metadata: setErrorMetadata(createFreshMetadata(), errorMessage),
                  },
            },
          },
        }));

        throw error;
      }
    });
  },

  setScene: (scene: Scene) => {
    set((state) => ({
      scene: {
        ...state.scene,
        scenes: {
          ...state.scene.scenes,
          [scene.id]: {
            data: scene,
            metadata: setSuccessMetadata(),
          },
        },
      },
    }));
  },

  markSceneStale: (id: string) => {
    set((state) => {
      const cached = state.scene.scenes[id];
      if (!cached) return state;

      return {
        scene: {
          ...state.scene,
          scenes: {
            ...state.scene.scenes,
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

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  optimisticJoinScene: (sceneId: string, _userId: string) => {
    // Note: _userId parameter reserved for future use in optimistic update logic
    const state = get();
    const cached = state.scene.scenes[sceneId];

    if (!cached) {
      return;
    }

    // Save current state for rollback
    set((state) => ({
      scene: {
        ...state.scene,
        optimisticUpdates: {
          ...state.scene.optimisticUpdates,
          [sceneId]: cached.data,
        },
      },
    }));

    // Apply optimistic update
    // Add a temporary marker field that UI can use for optimistic feedback
    set((state) => ({
      scene: {
        ...state.scene,
        scenes: {
          ...state.scene.scenes,
          [sceneId]: {
            ...cached,
            data: {
              ...cached.data,
              // Temporary marker for optimistic UI feedback
              // Cast to any to allow adding non-standard field
              _optimisticJoin: true,
            } as Scene,
          },
        },
      },
    }));
  },

  rollbackSceneUpdate: (sceneId: string) => {
    const state = get();
    const backup = state.scene.optimisticUpdates[sceneId];

    if (!backup) {
      return;
    }

    // Restore from backup
    set((state) => {
      const cached = state.scene.scenes[sceneId];
      // Remove from optimistic updates
      const remainingUpdates = { ...state.scene.optimisticUpdates };
      delete remainingUpdates[sceneId];

      return {
        scene: {
          ...state.scene,
          scenes: {
            ...state.scene.scenes,
            [sceneId]: {
              data: backup,
              metadata: cached?.metadata || createFreshMetadata(),
            },
          },
          optimisticUpdates: remainingUpdates,
        },
      };
    });
  },

  commitSceneUpdate: (sceneId: string) => {
    // Remove backup after successful commit
    set((state) => {
      const remainingUpdates = { ...state.scene.optimisticUpdates };
      delete remainingUpdates[sceneId];

      return {
        scene: {
          ...state.scene,
          optimisticUpdates: remainingUpdates,
        },
      };
    });
  },

  clearSceneError: (id: string) => {
    set((state) => {
      const cached = state.scene.scenes[id];
      if (!cached || !cached.metadata.error) return state;

      return {
        scene: {
          ...state.scene,
          scenes: {
            ...state.scene.scenes,
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

  removeScene: (id: string) => {
    set((state) => {
      const remainingScenes = { ...state.scene.scenes };
      delete remainingScenes[id];
      
      const remainingUpdates = { ...state.scene.optimisticUpdates };
      delete remainingUpdates[id];

      return {
        scene: {
          scenes: remainingScenes,
          optimisticUpdates: remainingUpdates,
        },
      };
    });
  },

  updateScene: async (id: string, updates: Partial<Scene>): Promise<Scene> => {
    const state = get();
    const cached = state.scene.scenes[id];

    // Set loading state
    set((state) => ({
      scene: {
        ...state.scene,
        scenes: {
          ...state.scene.scenes,
          [id]: {
            data: cached?.data || ({} as Scene),
            metadata: setLoadingMetadata(cached?.metadata || createFreshMetadata()),
          },
        },
      },
    }));

    try {
      // Update scene via API
      const updatedScene = await apiClient.patch<Scene>(`/scenes/${id}`, updates);

      // Update cache with fresh data
      set((state) => ({
        scene: {
          ...state.scene,
          scenes: {
            ...state.scene.scenes,
            [id]: {
              data: updatedScene,
              metadata: setSuccessMetadata(),
            },
          },
        },
      }));

      return updatedScene;
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : String(error || 'Failed to update scene');

      // Update cache with error
      set((state) => ({
        scene: {
          ...state.scene,
          scenes: {
            ...state.scene.scenes,
            [id]: cached
              ? {
                  ...cached,
                  metadata: setErrorMetadata(cached.metadata, errorMessage),
                }
              : {
                  data: {} as Scene,
                  metadata: setErrorMetadata(createFreshMetadata(), errorMessage),
                },
          },
        },
      }));

      throw error;
    }
  },
});
