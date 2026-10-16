-- +goose Up
CREATE TABLE app_user (
    id       BIGINT PRIMARY KEY,
    name     TEXT NOT NULL,
    username TEXT NOT NULL,
    email    TEXT NOT NULL,
    website  TEXT NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE IF EXISTS app_user;
