-- Migration: 002_create_umf_user.sql
-- Description: Create umf_user and grant permissions for telemetry database
-- Date: 2024-01-01

-- Create umf_user if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'umf_user') THEN
        CREATE USER umf_user WITH PASSWORD 'umf_password';
    END IF;
END
$$;

-- Grant necessary permissions to umf_user
GRANT CONNECT ON DATABASE umf_db TO umf_user;
GRANT USAGE ON SCHEMA public TO umf_user;

-- Grant permissions on existing tables
GRANT SELECT, INSERT, UPDATE, DELETE ON switches TO umf_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON telemetry_metrics TO umf_user;

-- Grant permissions on sequences
GRANT USAGE, SELECT ON SEQUENCE telemetry_metrics_id_seq TO umf_user;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO umf_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO umf_user; 