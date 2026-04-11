-- +goose Up
-- Migration 008: SMTP settings (singleton row, id=1)
CREATE TABLE IF NOT EXISTS smtp_settings (
    singleton          BOOLEAN      PRIMARY KEY DEFAULT TRUE CHECK (singleton = TRUE),
    host               VARCHAR(255) NOT NULL DEFAULT '',
    port               INTEGER      NOT NULL DEFAULT 587,
    username           VARCHAR(255) NOT NULL DEFAULT '',
    encrypted_password TEXT         NOT NULL DEFAULT '',
    from_address       VARCHAR(255) NOT NULL DEFAULT '',
    from_name          VARCHAR(255) NOT NULL DEFAULT '',
    tls_mode           VARCHAR(10)  NOT NULL DEFAULT 'starttls'
                           CHECK (tls_mode IN ('none', 'starttls', 'tls')),
    auth_type          VARCHAR(10)  NOT NULL DEFAULT 'plain'
                           CHECK (auth_type IN ('plain', 'login')),
    helo_name          VARCHAR(255) NOT NULL DEFAULT '',
    enabled            BOOLEAN      NOT NULL DEFAULT FALSE,
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

INSERT INTO smtp_settings (singleton) VALUES (TRUE) ON CONFLICT (singleton) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS smtp_settings;
