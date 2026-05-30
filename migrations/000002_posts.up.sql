BEGIN;

CREATE TABLE IF NOT EXISTS posts(
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    can_have_comm BOOLEAN NOT NULL DEFAULT true,
    text TEXT NOT NULL CHECK (char_length(text) <= 2000),
    status type_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

COMMIT;