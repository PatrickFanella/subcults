-- Rollback: Drop users table
-- WARNING: This is a destructive operation that will permanently delete all user data

DROP INDEX IF EXISTS idx_users_did;
DROP TABLE IF EXISTS users;
