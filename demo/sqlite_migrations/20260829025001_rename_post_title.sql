-- +goose Up
ALTER TABLE post RENAME COLUMN title TO heading;

-- +goose Down
ALTER TABLE post RENAME COLUMN heading TO title;
