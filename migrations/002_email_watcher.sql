-- +goose Up
CREATE TABLE email_accounts (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    email           TEXT NOT NULL,
    imap_server     TEXT NOT NULL,
    password_enc    BYTEA NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    last_connected  TIMESTAMPTZ,
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, email)
);

CREATE TABLE email_codes (
    id               BIGSERIAL PRIMARY KEY,
    email_account_id BIGINT NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    user_id          BIGINT NOT NULL REFERENCES users(id),
    sender           TEXT NOT NULL,
    subject          TEXT,
    code             TEXT NOT NULL,
    rule_name        TEXT,
    raw_body_hash    TEXT,
    tg_message_id    BIGINT,
    received_at      TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_codes_user_id ON email_codes(user_id);
CREATE INDEX idx_email_codes_received_at ON email_codes(received_at DESC);
CREATE INDEX idx_email_codes_body_hash ON email_codes(raw_body_hash);
CREATE INDEX idx_email_accounts_user_id ON email_accounts(user_id);

CREATE TRIGGER trg_email_accounts_updated_at BEFORE UPDATE ON email_accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS trg_email_accounts_updated_at ON email_accounts;
DROP TABLE IF EXISTS email_codes;
DROP TABLE IF EXISTS email_accounts;
