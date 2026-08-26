-- +goose Up
CREATE TABLE IF NOT EXISTS user_profiles (
    id VARCHAR(36) PRIMARY KEY,
    auth_user_id TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    role VARCHAR(12) NOT NULL,
    name TEXT,
    image TEXT,
    active BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    modified_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(36) NOT NULL,
    modified_by VARCHAR(36) NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS user_profiles;
