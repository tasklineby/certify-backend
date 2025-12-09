-- +goose Up
-- +goose StatementBegin
CREATE TABLE verification_history (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    scanned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_history_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_history_document FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX idx_verification_history_user_id ON verification_history(user_id);
CREATE INDEX idx_verification_history_document_id ON verification_history(document_id);
CREATE INDEX idx_verification_history_scanned_at ON verification_history(scanned_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_verification_history_scanned_at;
DROP INDEX IF EXISTS idx_verification_history_document_id;
DROP INDEX IF EXISTS idx_verification_history_user_id;
DROP TABLE IF EXISTS verification_history;
-- +goose StatementEnd

