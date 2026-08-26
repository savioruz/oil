-- +goose Up
CREATE TABLE IF NOT EXISTS galleries (
    id VARCHAR(36) PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    description TEXT,
    images TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    modified_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(36) NOT NULL,
    modified_by VARCHAR(36) NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS galleries;
