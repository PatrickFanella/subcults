-- Rollback initial schema

DROP TABLE IF EXISTS indexer_state;
DROP TABLE IF EXISTS stream_sessions;
DROP TABLE IF EXISTS alliances;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS scenes;

DROP EXTENSION IF EXISTS "uuid-ossp";
DROP EXTENSION IF EXISTS postgis;
