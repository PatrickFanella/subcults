-- Drop stream quality metrics table
DROP INDEX IF EXISTS idx_quality_metrics_packet_loss;
DROP INDEX IF EXISTS idx_quality_metrics_session_participant_latest;
DROP INDEX IF EXISTS idx_quality_metrics_participant;
DROP INDEX IF EXISTS idx_quality_metrics_session;
DROP TABLE IF EXISTS stream_quality_metrics;
