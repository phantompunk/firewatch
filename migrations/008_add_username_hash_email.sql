ALTER TABLE admin_users
    ADD COLUMN username        TEXT UNIQUE NOT NULL DEFAULT '',
    ADD COLUMN email_hmac      TEXT UNIQUE NOT NULL DEFAULT '',
    ADD COLUMN email_encrypted BYTEA       NOT NULL DEFAULT '',
    DROP COLUMN email;

ALTER TABLE admin_users
    ALTER COLUMN username        DROP DEFAULT,
    ALTER COLUMN email_hmac      DROP DEFAULT,
    ALTER COLUMN email_encrypted DROP DEFAULT;

ALTER TABLE invitation_tokens
    ADD COLUMN email_encrypted BYTEA NOT NULL DEFAULT '',
    DROP COLUMN email;

ALTER TABLE invitation_tokens
    ALTER COLUMN email_encrypted DROP DEFAULT;
