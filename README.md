# Alert Correlation Engine

Status: **Week 1** — ingestion endpoint + idempotency. No correlation logic yet
(dedup, flapping, suppression) — that is scheduled for Week 2.

See `SPEC.md` (not yet added to this repo) for problem background, use cases,
and full timeline.

## Quick start (local dev)

```bash
# 1. Start Postgres + Redis
docker compose up -d

# 2. Run migrations (example uses golang-migrate, adjust if using goose)
make db/migrations/up

# 3. Resolve Go dependencies (go.mod already has the requirements, but go.sum isn't
#    generated here yet because this environment lacks access to proxy.golang.org)
go mod tidy

# 4. Run the API
go run ./cmd/api

# 5. In another terminal, send synthetic alerts
pip install requests
python scripts/generate_alerts.py
```
## Struktur folder

```
cmd/api/                entrypoint, wiring only (connect DB, register routes, start server)
internal/model/          domain structs (Alert, later Incident, etc.)
internal/repository/     pure Postgres access, no business logic
internal/service/        business logic: fingerprinting, ingestion, later correlation & suppression
internal/handler/        HTTP layer (Fiber): parse request, format response
internal/httpserver/     route declarations
migrations/              SQL schema
scripts/                 synthetic alert generator for testing
```


## Completed in Week 1
- [x] Initial Postgres schema + composite index for future incident queries
- [x] POST /alerts endpoint accepting generic payload
- [x] Idempotency: raw_alert_id UNIQUE constraint, webhook retries do not create duplicate rows
- [ ] Correlation engine (Redis, active fingerprint dedup, flapping) — Week 2
- [ ] Known-pattern suppression — Week 2
- [ ] RCA knowledge base — Week 3
- [ ] Dashboard — Week 3
