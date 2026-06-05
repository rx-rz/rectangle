-- +goose Up
ALTER TABLE otps
    DROP CONSTRAINT otps_purpose_check;

ALTER TABLE otps
    ADD CONSTRAINT otps_purpose_check CHECK (purpose IN ('email_verification', 'password_reset'));

-- +goose Down
SELECT
    'down SQL query';

