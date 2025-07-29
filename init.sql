-- Use these command as prerequisites for local start with separate Postgres Instance
-- these commands should invoked externally by Postgres admin (postgres/postgres usually)
--GRANT ALL PRIVILEGES ON SCHEMA public TO "user";
--ALTER SCHEMA public OWNER TO "user";
--GRANT CREATE, USAGE ON SCHEMA public TO "user";
--GRANT ALL ON ALL TABLES IN SCHEMA public TO "user";
--CREATE EXTENSION IF NOT EXISTS "pgcrypto";
--GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO "user";

CREATE TABLE IF NOT EXISTS quotes (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    currency TEXT NOT NULL,
    price DOUBLE PRECISION,
    updated_at TIMESTAMP,
    status TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS unique_currency_pending ON quotes(currency) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_quotes_currency_status ON quotes(currency, status, updated_at DESC);
