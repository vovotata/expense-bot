CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE expense_type AS ENUM ('agentki', 'adpos', 'antique_service', 'other_service', 'setups');
CREATE TYPE payment_method AS ENUM ('usdt', 'trx', 'card', 'none');
CREATE TYPE request_status AS ENUM ('pending', 'approved', 'paid', 'rejected', 'cancelled');

CREATE TABLE users (
    id            BIGINT PRIMARY KEY,
    username      TEXT,
    first_name    TEXT NOT NULL,
    last_name     TEXT,
    is_blocked    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE requests (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         BIGINT NOT NULL REFERENCES users(id),
    expense_type    expense_type NOT NULL,
    payment_method  payment_method,
    address         TEXT,
    address_photo   TEXT,
    amount          NUMERIC(18, 6),
    antique_account TEXT,
    comment         TEXT NOT NULL,
    status          request_status NOT NULL DEFAULT 'pending',
    flow_type       CHAR(1) NOT NULL DEFAULT 'A',
    tg_message_id   BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_requests_user_id ON requests(user_id);
CREATE INDEX idx_requests_status ON requests(status);
CREATE INDEX idx_requests_created_at ON requests(created_at DESC);
CREATE INDEX idx_requests_expense_type ON requests(expense_type);

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_requests_updated_at BEFORE UPDATE ON requests FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at();
