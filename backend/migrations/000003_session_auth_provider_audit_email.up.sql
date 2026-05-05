-- Add auth_provider to sessions table
ALTER TABLE sessions ADD COLUMN auth_provider VARCHAR(20) NOT NULL DEFAULT 'local';

-- Add user_email to audit_logs table for immutable email tracking
ALTER TABLE audit_logs ADD COLUMN user_email VARCHAR(255) NOT NULL DEFAULT '';
