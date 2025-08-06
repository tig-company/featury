-- Drop audit_logs table and associated objects
DROP INDEX IF EXISTS idx_audit_logs_metadata_gin;
DROP INDEX IF EXISTS idx_audit_logs_changes_gin;
DROP INDEX IF EXISTS idx_audit_logs_entity_time;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP INDEX IF EXISTS idx_audit_logs_entity;
DROP TABLE IF EXISTS audit_logs;