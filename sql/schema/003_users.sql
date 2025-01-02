-- +goose Up
ALTER TABLE users
ADD COLUMN hashed_password text NOT NULL DEFAULT 'unset';

-- +goose Down
ALTER TABLE users
REMOVE COLUMN hashed_password;
