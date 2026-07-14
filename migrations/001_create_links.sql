CREATE TABLE IF NOT EXISTS links (
    code VARCHAR(10) PRIMARY KEY,
    original_url TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT links_code_length CHECK (char_length(code) = 10),
    CONSTRAINT links_code_alphabet CHECK (code ~ '^[A-Za-z0-9_]{10}$')
);
