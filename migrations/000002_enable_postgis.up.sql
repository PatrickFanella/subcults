-- Migration: Enable PostGIS extension for geo queries
-- NOTE: PostGIS is already enabled in 000000_initial_schema.up.sql
-- This migration exists for explicit documentation and verification

-- Enable PostGIS extension (idempotent - safe to run multiple times)
CREATE EXTENSION IF NOT EXISTS postgis;

-- Verify PostGIS is available by checking pg_extension
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'postgis') THEN
        RAISE EXCEPTION 'PostGIS extension is not available';
    END IF;
END $$;
