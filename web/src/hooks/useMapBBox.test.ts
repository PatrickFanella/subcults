import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useMapBBox } from './useMapBBox';
import type { Map as MapLibreMap, LngLatBounds } from 'maplibre-gl';

type EventHandler = (...args: unknown[]) => void;

/**
 * Mock MapLibre Map with event listeners
 */
class MockMap {
  private listeners = new Map<string, Array<EventHandler>>();
  private mockBounds: LngLatBounds | null = null;
  
  constructor(bounds?: LngLatBounds) {
    this.mockBounds = bounds || null;
  }
  
  on(event: string, handler: EventHandler) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event)!.push(handler);
  }
  
  off(event: string, handler: EventHandler) {
    const handlers = this.listeners.get(event);
    if (handlers) {
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    }
  }
  
  emit(event: string, ...args: unknown[]) {
    const handlers = this.listeners.get(event);
    if (handlers) {
      handlers.forEach((handler: EventHandler) => handler(...args));
    }
  }
  
  getBounds(): LngLatBounds | null {
    return this.mockBounds;
  }
  
  setBounds(bounds: LngLatBounds) {
    this.mockBounds = bounds;
  }
}

/**
 * Create mock bounds object
 */
function createMockBounds(west: number, south: number, east: number, north: number): LngLatBounds {
  return {
    getWest: () => west,
    getSouth: () => south,
    getEast: () => east,
    getNorth: () => north,
    getNorthWest: () => ({ lng: west, lat: north }),
    getNorthEast: () => ({ lng: east, lat: north }),
    getSouthWest: () => ({ lng: west, lat: south }),
    getSouthEast: () => ({ lng: east, lat: south }),
    getCenter: () => ({ lng: (west + east) / 2, lat: (south + north) / 2 }),
    toArray: () => [[west, south], [east, north]],
    toString: () => `LngLatBounds(${west}, ${south}, ${east}, ${north})`,
  } as LngLatBounds;
}

describe('useMapBBox', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  
  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });
  
  it('returns null bbox when map is null', () => {
    const callback = vi.fn();
    const { result } = renderHook(() => useMapBBox(null, callback));
    
    expect(result.current.bbox).toBeNull();
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });
  
  it('returns null bbox when map has no bounds', () => {
    const mockMap = new MockMap();
    const callback = vi.fn();
    const { result } = renderHook(() => useMapBBox(mockMap as unknown as MapLibreMap, callback));
    
    expect(result.current.bbox).toBeNull();
    expect(result.current.loading).toBe(false);
  });
  
  it('computes bbox in correct [minLng, minLat, maxLng, maxLat] format', async () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback = vi.fn();
    
    renderHook(() => useMapBBox(mockMap as unknown as MapLibreMap, callback, { immediate: true }));
    
    await act(async () => {
      vi.runAllTimers();
    });
    
    expect(callback).toHaveBeenCalledWith([-122.5, 37.7, -122.4, 37.8]);
  });
  
  it('calls callback after debounce on moveend', async () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback = vi.fn();
    
    renderHook(() => useMapBBox(mockMap as unknown as MapLibreMap, callback, { debounceMs: 500 }));
    
    // Simulate moveend event
    act(() => {
      mockMap.emit('movestart');
      mockMap.emit('move');
      mockMap.emit('moveend');
    });
    
    // Callback should not be called immediately
    expect(callback).not.toHaveBeenCalled();
    
    // Fast forward debounce timer
    await act(async () => {
      vi.advanceTimersByTime(500);
    });
    
    expect(callback).toHaveBeenCalledTimes(1);
    expect(callback).toHaveBeenCalledWith([-122.5, 37.7, -122.4, 37.8]);
  });
  
  it('debounces rapid pan movements - only one callback after settling', async () => {
    const bounds1 = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const bounds2 = createMockBounds(-122.6, 37.6, -122.5, 37.7);
    const bounds3 = createMockBounds(-122.7, 37.5, -122.6, 37.6);
    
    const mockMap = new MockMap(bounds1);
    const callback = vi.fn();
    
    renderHook(() => useMapBBox(mockMap as unknown as MapLibreMap, callback, { debounceMs: 500 }));
    
    // First movement
    act(() => {
      mockMap.emit('movestart');
      mockMap.emit('move');
      mockMap.emit('moveend');
    });
    
    // Advance 200ms (less than debounce)
    await act(async () => {
      vi.advanceTimersByTime(200);
    });
    
    // Second movement (should cancel first timer)
    act(() => {
      mockMap.setBounds(bounds2);
      mockMap.emit('movestart');
      mockMap.emit('move');
      mockMap.emit('moveend');
    });
    
    // Advance 300ms (still less than debounce from second movement)
    await act(async () => {
      vi.advanceTimersByTime(300);
    });
    
    // Third movement (should cancel second timer)
    act(() => {
      mockMap.setBounds(bounds3);
      mockMap.emit('movestart');
      mockMap.emit('move');
      mockMap.emit('moveend');
    });
    
    // Callback should still not be called
    expect(callback).not.toHaveBeenCalled();
    
    // Advance final 500ms
    await act(async () => {
      vi.advanceTimersByTime(500);
    });
    
    // Should only be called once with final bounds
    expect(callback).toHaveBeenCalledTimes(1);
    expect(callback).toHaveBeenCalledWith([-122.7, 37.5, -122.6, 37.6]);
  });
  
  it('sets loading state during movement', async () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback = vi.fn();
    
    const { result } = renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback, { debounceMs: 500 })
    );
    
    expect(result.current.loading).toBe(false);
    
    // Start movement
    act(() => {
      mockMap.emit('movestart');
    });
    
    expect(result.current.loading).toBe(true);
    
    // End movement
    act(() => {
      mockMap.emit('moveend');
    });
    
    // Still loading during debounce
    expect(result.current.loading).toBe(true);
    
    // Complete debounce
    await act(async () => {
      vi.advanceTimersByTime(500);
    });
    
    expect(result.current.loading).toBe(false);
  });
  
  it('updates bbox state after debounce', async () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback = vi.fn();
    
    const { result } = renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback, { debounceMs: 300 })
    );
    
    expect(result.current.bbox).toBeNull();
    
    // Trigger movement
    act(() => {
      mockMap.emit('movestart');
      mockMap.emit('moveend');
    });
    
    // Advance debounce timer
    await act(async () => {
      vi.advanceTimersByTime(300);
    });
    
    expect(result.current.bbox).toEqual([-122.5, 37.7, -122.4, 37.8]);
  });
  
  it('provides consistent bbox format after zoom', async () => {
    const bounds1 = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const bounds2 = createMockBounds(-122.45, 37.74, -122.44, 37.76); // Zoomed in
    
    const mockMap = new MockMap(bounds1);
    const callback = vi.fn();
    
    const { result } = renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback, { debounceMs: 300 })
    );
    
    // First pan/zoom
    act(() => {
      mockMap.emit('movestart');
      mockMap.emit('moveend');
    });
    
    await act(async () => {
      vi.advanceTimersByTime(300);
    });
    
    const firstBBox = result.current.bbox;
    expect(firstBBox).toEqual([-122.5, 37.7, -122.4, 37.8]);
    
    // Zoom in
    act(() => {
      mockMap.setBounds(bounds2);
      mockMap.emit('movestart');
      mockMap.emit('moveend');
    });
    
    await act(async () => {
      vi.advanceTimersByTime(300);
    });
    
    const secondBBox = result.current.bbox;
    expect(secondBBox).toEqual([-122.45, 37.74, -122.44, 37.76]);
    
    // Verify format consistency: [minLng, minLat, maxLng, maxLat]
    expect(secondBBox![0]).toBeLessThan(secondBBox![2]); // minLng < maxLng
    expect(secondBBox![1]).toBeLessThan(secondBBox![3]); // minLat < maxLat
  });
  
  it('calls callback immediately when immediate option is true', async () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback = vi.fn();
    
    renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback, { immediate: true })
    );
    
    // Should be called immediately without waiting for debounce
    await act(async () => {
      vi.runAllTimers();
    });
    
    expect(callback).toHaveBeenCalledTimes(1);
    expect(callback).toHaveBeenCalledWith([-122.5, 37.7, -122.4, 37.8]);
  });
  
  it('does not call callback immediately when immediate is false', async () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback = vi.fn();
    
    renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback, { immediate: false })
    );
    
    await act(async () => {
      vi.runAllTimers();
    });
    
    // Should not be called until movement occurs
    expect(callback).not.toHaveBeenCalled();
  });
  
  it('uses custom debounce delay', async () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback = vi.fn();
    
    renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback, { debounceMs: 1000 })
    );
    
    act(() => {
      mockMap.emit('movestart');
      mockMap.emit('moveend');
    });
    
    // Should not be called after 500ms
    await act(async () => {
      vi.advanceTimersByTime(500);
    });
    expect(callback).not.toHaveBeenCalled();
    
    // Should be called after full 1000ms
    await act(async () => {
      vi.advanceTimersByTime(500);
    });
    expect(callback).toHaveBeenCalledTimes(1);
  });
  
  it('cancels timer during ongoing movement', async () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback = vi.fn();
    
    renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback, { debounceMs: 500 })
    );
    
    // Start movement
    act(() => {
      mockMap.emit('movestart');
      mockMap.emit('moveend');
    });
    
    // Advance partway through debounce
    await act(async () => {
      vi.advanceTimersByTime(300);
    });
    
    // Continue moving (should cancel timer)
    act(() => {
      mockMap.emit('move');
    });
    
    // Advance past original debounce time
    await act(async () => {
      vi.advanceTimersByTime(300);
    });
    
    // Should not have been called yet
    expect(callback).not.toHaveBeenCalled();
    
    // End movement again
    act(() => {
      mockMap.emit('moveend');
    });
    
    // Advance full debounce time
    await act(async () => {
      vi.advanceTimersByTime(500);
    });
    
    // Now should be called
    expect(callback).toHaveBeenCalledTimes(1);
  });
  
  it('cleans up event listeners on unmount', () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback = vi.fn();
    
    const offSpy = vi.spyOn(mockMap, 'off');
    
    const { unmount } = renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback)
    );
    
    unmount();
    
    expect(offSpy).toHaveBeenCalledWith('movestart', expect.any(Function));
    expect(offSpy).toHaveBeenCalledWith('move', expect.any(Function));
    expect(offSpy).toHaveBeenCalledWith('moveend', expect.any(Function));
  });
  
  it('clears pending timer on unmount', async () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback = vi.fn();
    
    const { unmount } = renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback, { debounceMs: 500 })
    );
    
    // Start movement
    act(() => {
      mockMap.emit('movestart');
      mockMap.emit('moveend');
    });
    
    // Unmount before debounce completes
    unmount();
    
    // Advance past debounce time
    await act(async () => {
      vi.advanceTimersByTime(500);
    });
    
    // Callback should not be called after unmount
    expect(callback).not.toHaveBeenCalled();
  });
  
  it('handles rapid pans within 300ms - acceptance criteria test', async () => {
    const bounds1 = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const bounds2 = createMockBounds(-122.6, 37.6, -122.5, 37.7);
    const bounds3 = createMockBounds(-122.7, 37.5, -122.6, 37.6);
    
    const mockMap = new MockMap(bounds1);
    const callback = vi.fn();
    
    renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback, { debounceMs: 500 })
    );
    
    // Pan 1
    act(() => {
      mockMap.emit('movestart');
      mockMap.emit('move');
      mockMap.emit('moveend');
    });
    
    // 200ms later (< 300ms)
    await act(async () => {
      vi.advanceTimersByTime(200);
    });
    
    // Pan 2
    act(() => {
      mockMap.setBounds(bounds2);
      mockMap.emit('movestart');
      mockMap.emit('move');
      mockMap.emit('moveend');
    });
    
    // 250ms later (< 300ms from Pan 2)
    await act(async () => {
      vi.advanceTimersByTime(250);
    });
    
    // Pan 3
    act(() => {
      mockMap.setBounds(bounds3);
      mockMap.emit('movestart');
      mockMap.emit('move');
      mockMap.emit('moveend');
    });
    
    // Still no callback
    expect(callback).not.toHaveBeenCalled();
    
    // Wait for final debounce (500ms from Pan 3)
    await act(async () => {
      vi.advanceTimersByTime(500);
    });
    
    // Should only have one network call after settling
    expect(callback).toHaveBeenCalledTimes(1);
    expect(callback).toHaveBeenCalledWith([-122.7, 37.5, -122.6, 37.6]);
  });
  
  it('handles error when bounds computation fails', async () => {
    const mockMap = new MockMap();
    // Override getBounds to throw error
    mockMap.getBounds = () => {
      throw new Error('Bounds error');
    };
    
    const callback = vi.fn();
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    
    const { result } = renderHook(() => 
      useMapBBox(mockMap as unknown as MapLibreMap, callback, { immediate: true })
    );
    
    await act(async () => {
      vi.runAllTimers();
    });
    
    expect(result.current.error).toBe('Bounds error');
    expect(result.current.bbox).toBeNull();
    expect(callback).not.toHaveBeenCalled();
    expect(consoleErrorSpy).toHaveBeenCalled();
    
    consoleErrorSpy.mockRestore();
  });
  
  it('updates callback ref when callback changes', async () => {
    const bounds = createMockBounds(-122.5, 37.7, -122.4, 37.8);
    const mockMap = new MockMap(bounds);
    const callback1 = vi.fn();
    const callback2 = vi.fn();
    
    const { rerender } = renderHook(
      ({ cb }) => useMapBBox(mockMap as unknown as MapLibreMap, cb, { debounceMs: 300 }),
      { initialProps: { cb: callback1 } }
    );
    
    // Trigger movement
    act(() => {
      mockMap.emit('movestart');
      mockMap.emit('moveend');
    });
    
    // Change callback before debounce completes
    rerender({ cb: callback2 });
    
    // Complete debounce
    await act(async () => {
      vi.advanceTimersByTime(300);
    });
    
    // Should call the new callback
    expect(callback1).not.toHaveBeenCalled();
    expect(callback2).toHaveBeenCalledTimes(1);
  });
});
