ALTER SYSTEM SET wal_level = logical;
CREATE DATABASE servers;
\c servers
CREATE TABLE servers (
    id UUID PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid(),
    server_name TEXT UNIQUE,
    status TEXT,
    ipv4 TEXT,
    port INT,
    health_check_interval BIGINT,
    health_endpoint TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX servers_created_at_idx ON servers (created_at);
ALTER TABLE servers REPLICA IDENTITY FULL;
