// Package db provides database utilities and connection handling for Subcults.
package db

// PostGISRequirement documents that the application requires PostgreSQL with PostGIS.
// PostGIS enables geographic queries for scene and event location data.
const PostGISRequirement = "PostGIS extension is required for geo queries"

// VersionQuery is the SQL query to verify PostGIS is available.
const VersionQuery = "SELECT PostGIS_Version()"
