/**
 * Clustering components and utilities for map visualization
 * 
 * @module clustering
 */

export { ClusteredMapView } from './components/ClusteredMapView';
export type { ClusteredMapViewProps } from './components/ClusteredMapView';

export { useClusteredData, boundsToBox } from './hooks/useClusteredData';
export type {
  BBox,
  UseClusteredDataOptions,
  UseClusteredDataResult,
} from './hooks/useClusteredData';

export { buildGeoJSON, decodeGeohash, getDisplayCoordinates } from './utils/geojson';
export type { FeatureProperties } from './utils/geojson';

export type { Scene, Event, Point, Palette } from './types/scene';
export { isScene, isEvent } from './types/scene';
