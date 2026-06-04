-- +goose Up
--
CREATE TYPE oauth_providers AS ENUM (
   'google',
   'github'
);

CREATE TABLE users (
  id text primary key,
  name text,
  email text not null,
  password_hash text,
  avatar_url text,
  email_verified_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

CREATE TABLE sessions (
  id text primary key,
  user_id text not null references users(id) on delete cascade,
  user_agent text,
  token_hash bytea not null,
  ip_address inet,
  expires_at timestamptz not null,
  revoked_at timestamptz,
  created_at timestamptz not null default now(),
  constraint expires_at_greater_than_created_at check (expires_at>created_at)
);

CREATE TABLE otps (
  id bigserial primary key,
  user_id text not null references users(id) on delete cascade,
  email text not null,
  otp_hash bytea not null,
  purpose text not null,
  attempts int not null default 0,
  consumed_at timestamptz,
  expires_at timestamptz not null,
  created_at timestamptz not null default now(),
  constraint otps_purpose_check check (purpose in ('signup', 'password_reset')),
  constraint otps_attempts_check check (attempts >= 0 and attempts <=5),
  constraint otps_expires_at_greater_than_created_at_check
    check (expires_at > created_at)
);

CREATE TABLE oauth_accounts (
  provider oauth_providers not null,
  provider_user_id text not null,
  user_id text not null references users(id) on delete cascade,
  created_at timestamptz not null default now(),
  primary key (provider, provider_user_id)
);

create unique index users_lower_email_unique_idx on users(lower(email));
create index sessions_user_idx on sessions(user_id);
create index otps_active_purpose_email_idx on otps(email,purpose,created_at desc) where consumed_at is null;
create index otps_expires_at_idx on otps(expires_at);

-- +goose Down
DROP INDEX IF EXISTS otps_expires_at_idx;
DROP INDEX IF EXISTS otps_active_purpose_email_idx;
DROP INDEX IF EXISTS sessions_user_idx;
DROP INDEX IF EXISTS users_lower_email_unique_idx;

DROP TABLE IF EXISTS oauth_accounts;
DROP TABLE IF EXISTS otps;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS oauth_providers;
