-- PostgreSQL initialization script for Listen Stream
-- Creates all databases needed by the services

-- auth-svc database
-- (created automatically if POSTGRES_DB=listen_stream)

-- Create additional schemas or enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
