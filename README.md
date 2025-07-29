# Fin Quotes Service

Service for fetching and saving actual financial quotes values.

---

## About

This service allows you to asynchronously request and update currency pair quotes (exchange rates).
Updating a quote is performed in the background: the user receives an update request ID, then can later retrieve the result by this ID.

Quote data is stored in PostgreSQL.  
A background worker picks up update tasks from the queue, fetches rates from an external API (with emulated delay for 30s for testing), and saves the result.

Only 4 currencies are supported: USD/EUR, EUR/USD, USD/MXN, EUR/MXN (as test examples)

---

## Requirements

- **Go** 1.24+
- **PostgreSQL** 15+
- **Docker** (for quick start for dev env)
- **curl** (for execute and check API)

---

## Quick start (Docker Compose)

1. **Сlone from git repo and go to the target dir:**
   ```bash
   git clone https://github.com/ArnoldSh/FinQuotesService
   cd FinQuotesService
   ```
2. **Exec command:**
   ```bash
   docker compose up ## or docker-compose up
   ```
3. **If you want to change the code -> use --build option to re-create docker images:**
   ```bash
   docker compose up --build
   ```
   
---

## Local start

**Postgres prerequisites: Please read the init.sql comments - before local launch ensure admin commands (grants & extension) executed**

1. Install Go 1.24+ -> https://go.dev/dl/
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Set the environment variable for the database (Linux/Mac example):
   ```bash
   export DB_DSN="postgres://user:pass@localhost:5432/quotes?sslmode=disable"
   ```
4. Run the server:
   ```bash
   go run ./cmd/server/main.go
   ```

--- 

## Check server API

You can use curl for invoke server api:
```bash
curl -X POST -d '{"currency":"USD/EUR"}' http://localhost:8080/quotes/update
curl -X GET http://localhost:8080/quotes/update/<REQUEST_ID>
curl -X GET http://localhost:8080/quotes/last/<CURRENCY_PAIR>
```