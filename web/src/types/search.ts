/**
 * Search Types
 * Type definitions for global search functionality
 */

import type { Scene, Event } from './scene';

/**
 * Post entity (simplified for search results).
 *
 * This type intentionally models only the subset of fields that are
 * relevant to global search and discovery. The canonical post model
 * (including full AT Protocol record shape) lives server-side.
 */
export interface Post {
  /**
   * Stable database identifier for the post within Subcults.
   */
  id: string;
  /**
   * Optional human-readable title for long-form or editorial posts.
   *
   * When `title` is present it should be preferred for display in
   * search results and detail views. For short text posts where no
   * explicit title exists, this field will be absent and UIs should
   * fall back to using the first part of `content` instead.
   */
  title?: string;
  /**
   * Main body text of the post.
   *
   * This field is always present and is used both for full-text search
   * indexing and as a display fallback when `title` is not provided.
   */
  content: string;
  /**
   * DID of the post author, when available.
   *
   * This is the AT Protocol DID associated with the user who created
   * the underlying record.
   */
  author_did?: string;
  /**
   * Identifier of the scene this post is associated with, if any.
   *
   * Use this when the post is about a scene in general (e.g. scene
   * updates, discussions) rather than a specific event. A post should
   * typically be linked to at most one of `scene_id` or `event_id`.
   */
  scene_id?: string;
  /**
   * Identifier of the event this post is associated with, if any.
   *
   * Use this when the post is tied to a particular event (e.g. set
   * announcements, live updates). In normal usage a post should not
   * belong to both a scene and an event at the same time; when it is
   * event-specific, prefer setting `event_id` and omit `scene_id`.
   */
  event_id?: string;
  /**
   * ISO-8601 timestamp of when the post was created, if known.
   */
  created_at?: string;
  /**
   * AT Protocol repository DID for the original record.
   *
   * Together with `record_rkey`, this uniquely identifies the AT
   * record backing this post (i.e. the `did` component of an
   * `at://did/collection/rkey` URI).
   */
  record_did?: string;
  /**
   * AT Protocol record key (rkey) for the original record.
   *
   * This is the `rkey` component of the AT URI. When combined with
   * `record_did`, it can be used to fetch or reconstruct the full
   * AT Protocol record associated with this search result.
   */
  record_rkey?: string;
}

/**
 * Search result types for each entity
 */
export type SceneSearchResult = Scene;
export type EventSearchResult = Event;
export type PostSearchResult = Post;

/**
 * Grouped search results
 */
export interface SearchResults {
  scenes: SceneSearchResult[];
  events: EventSearchResult[];
  posts: PostSearchResult[];
}

/**
 * Search result item with type discriminator
 */
export type SearchResultItem = 
  | { type: 'scene'; data: SceneSearchResult }
  | { type: 'event'; data: EventSearchResult }
  | { type: 'post'; data: PostSearchResult };

/**
 * Search query parameters
 */
export interface SearchParams {
  query: string;
  limit?: number;
}

/**
 * Search state for hook
 */
export interface SearchState {
  results: SearchResults;
  loading: boolean;
  error: string | null;
}
