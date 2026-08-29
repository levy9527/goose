-- +goose Up
CREATE TABLE post (
    user_id INTEGER NOT NULL,
    id      INTEGER NOT NULL,
    title   TEXT NOT NULL,
    body    TEXT NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE IF EXISTS post;
