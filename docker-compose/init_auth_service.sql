CREATE DATABASE auth;
\c auth

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE scopes (
    id UUID PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE role_scopes (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    scope_id UUID REFERENCES scopes(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, scope_id)
);

CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

INSERT INTO scopes (name, description, created_at, updated_at) VALUES
('users:create', 'Allow creating users', NOW(), NOW()),
('users:read', 'Allow reading user information', NOW(), NOW()),
('users:roles:update', 'Allow updating user roles', NOW(), NOW()),
('roles:create', 'Allow creating roles', NOW(), NOW()),
('roles:read', 'Allow reading role information', NOW(), NOW()),
('roles:update', 'Allow updating roles', NOW(), NOW()),
('roles:delete', 'Allow deleting roles', NOW(), NOW()),
('scopes:read', 'Allow reading scope information', NOW(), NOW()),
('servers:read', 'Allow reading server information', NOW(), NOW()),
('servers:create', 'Allow creating servers', NOW(), NOW()),
('servers:update', 'Allow updating servers', NOW(), NOW()),
('servers:delete', 'Allow deleting servers', NOW(), NOW());


INSERT INTO roles (name, description, created_at, updated_at)
VALUES ('admin', 'Administrator with all permissions', NOW(), NOW());

INSERT INTO role_scopes (role_id, scope_id)
SELECT
    (SELECT id FROM roles WHERE name = 'admin'),
    id
FROM scopes;

INSERT INTO users (email, password, first_name, last_name, created_at, updated_at)
VALUES ('admin@gmail.com', '$2a$04$CHxMEXL8vezb4FCk9BoHMu4isGPn.6Md.8GQfbwyGDF5UESazaPKq', 'admin', 'admin', NOW(), NOW());

INSERT INTO user_roles (user_id, role_id) VALUES
(
    (SELECT id FROM users WHERE email = 'admin@gmail.com'),
    (SELECT id FROM roles WHERE name = 'admin')
);
