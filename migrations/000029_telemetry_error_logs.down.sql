DROP TABLE IF EXISTS error_replay_events;
DROP TABLE IF EXISTS client_error_logs;
DROP TABLE IF EXISTS telemetry_events;

DELETE FROM schema_version WHERE version = 29;
