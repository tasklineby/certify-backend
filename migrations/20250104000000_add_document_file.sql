-- +goose Up
-- +goose StatementBegin
ALTER TABLE documents
    ADD COLUMN file_name VARCHAR(255) NOT NULL,
    ADD COLUMN file_data BYTEA NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE documents
    DROP COLUMN IF EXISTS file_data,
    DROP COLUMN IF EXISTS file_name;
-- +goose StatementEnd

