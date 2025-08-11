CREATE DATABASE scheduler;
\c scheduler
CREATE TABLE servers (
    id UUID PRIMARY KEY,
    ipv4 TEXT,
    port INT,
    health_check_interval BIGINT,
    health_endpoint TEXT,
    next_health_check_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

