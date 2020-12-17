DROP INDEX IF EXISTS idx_users_secret;
DROP INDEX IF EXISTS idx_users_email_password;
DROP TRIGGER IF EXISTS set_updated_time ON users CASCADE;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS trigger_set_timestamp;
DROP EXTENSION IF EXISTS pgcrypto;
DROP EXTENSION IF EXISTS citext;
