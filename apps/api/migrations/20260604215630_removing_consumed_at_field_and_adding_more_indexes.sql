-- +goose Up
ALTER TABLE otps
    DROP COLUMN consumed_at;

ALTER TABLE otps
    DROP CONSTRAINT otps_purpose_check;

ALTER TABLE otps
    ADD CONSTRAINT otps_purpose_check
        CHECK (purpose in ('email_verification', 'password_reset'));

CREATE INDEX otps_purpose_email_created_at_idx
    ON otps(email, purpose, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS otps_purpose_email_created_at_idx;

ALTER TABLE otps
    DROP CONSTRAINT otps_purpose_check;

ALTER TABLE otps
    ADD CONSTRAINT otps_purpose_check
        CHECK (purpose in ('signup', 'password_reset'));

ALTER TABLE otps
    ADD COLUMN consumed_at timestamptz;

CREATE INDEX otps_active_purpose_email_idx
    ON otps(email, purpose, created_at DESC)
    WHERE consumed_at IS NULL;
