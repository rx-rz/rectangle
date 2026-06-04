docker compose --env-file .env -f docker-compose.dev.yml up -d postgres

# load DB_URL from .env
set -a
source .env
set +a

# run migrations
goose -dir migrations postgres "$DB_URL" up