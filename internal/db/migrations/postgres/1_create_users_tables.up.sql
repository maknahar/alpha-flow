CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION trigger_set_timestamp()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS users
(
    id                  bigserial PRIMARY KEY,
    email               citext                   NOT NULL UNIQUE,
    password            text                     NOT NULL,
    token               text,
    token_creation_time TIMESTAMP WITH TIME ZONE          DEFAULT now(),
    secret              text                              DEFAULT 'All your base are belong to us',
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE
);

CREATE TRIGGER set_updated_time
    BEFORE UPDATE
    ON users
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX IF NOT EXISTS idx_users_email_password ON users (email, password);
CREATE INDEX IF NOT EXISTS idx_users_secret ON users (secret);
