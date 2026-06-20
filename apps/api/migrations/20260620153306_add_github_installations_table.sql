-- +goose Up
CREATE TYPE github_account_type AS ENUM(
    'user',
    'organization'
);

CREATE TYPE github_repository_selection AS ENUM(
    'all',
    'selected'
);

CREATE TABLE IF NOT EXISTS github_installations(
    id bigserial PRIMARY KEY,
    user_id text NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    installation_id bigint NOT NULL UNIQUE,
    account_login text NOT NULL,
    github_account_id bigint NOT NULL,
    account_type github_account_type NOT NULL,
    repository_selection github_repository_selection NOT NULL,
    suspended_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE github_installations;

DROP TYPE IF EXISTS github_repository_selection;

DROP TYPE IF EXISTS github_account_type;

