# AGENTS.md — crave-and-glaze

Go 1.25 monolith (stdlib `net/http` + `html/template`, Postgres `lib/pq`, M-Pesa Daraja sandbox). No tests, linter, or Makefile.

## Entrypoints & layout

- `cmd/server/main.go` — the whole app: routes, all handlers, `render()` helper (~1100 lines). `internal/handlers/` exists but is **empty** — do not add code there unless refactoring deliberately.
- `cmd/seed/main.go` — seeds admin user from env (bcrypt, upsert).
- `internal/{database,repository,models,cart,mailer}/` — support packages. No package boundaries beyond these dirs. (M-Pesa `internal/daraja/` was removed; `orders.mpesa_receipt` column remains in schema but is unused.)
- `web/templates/` — `base.layout.html` (`{{define "base"}}`) + `*.page.html`; admin uses `admin/admin.layout.html` (`{{define "admin_base"}}`); mail bodies in `web/templates/email/`. `web/static/` served at `/static/`.
- Root files literally named `struct` and `type` are empty leftovers — ignore them.

## Run from repo root (paths are relative)

Handlers resolve `./web/templates/...`, `./web/static/uploads/`, and `schema.sql` via CWD/`os.Getwd()`. Always run from repo root or templates/static/uploads break. Same applies inside Docker (Dockerfile copies `web/` and `schema.sql` next to the binary for this reason).

```bash
go run ./cmd/server/main.go        # :8080, needs Postgres reachable
go run ./cmd/seed/main.go          # requires ADMIN_USERNAME + ADMIN_PASSWORD in env
go build ./...                     # verification (no test suite exists)
docker compose up -d --build       # app + db; needs `docker network create proxy` first (external net, see compose)
```

## Env & database

- Config via `godotenv.Load()` + `os.Getenv`; `.env` is gitignored. Required keys: `DB_HOST DB_PORT DB_USER DB_PASSWORD DB_NAME ADMIN_USERNAME ADMIN_PASSWORD SMTP_HOST SMTP_PORT SMTP_EMAIL SMTP_PASSWORD ADMIN_EMAIL ADMIN_WHATSAPP` (`MPESA_*` keys may linger in `.env` but are no longer read).
- `internal/database/db.go`: if `DATABASE_URL` set (Render path) it wins; else `DB_HOST/PORT/USER/PASSWORD/NAME` with defaults (`localhost:5432 / admin / password123 / crave_glaze`).
- Schema: `database.InitDB()` executes `schema.sql` on every boot **plus** inline `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` migrations in `db.go` (orders: `first_name, last_name, email, whatsapp_number, mpesa_receipt, customer_phone`). `models.Order` already expects these — if adding an order column, update **both** `schema.sql` and the migration list. Fresh containers also seed via `schema.sql` mounted at `/docker-entrypoint-initdb.d/`.
- Compose ports: host `${DB_PORT}` (loopback-only) → container 5432; `DB_HOST=db` inside compose, `localhost` outside. App waits on `db` `service_healthy` (`pg_isready`).
- Uploads land in `./web/static/uploads/` and persist via the `uploads_data` volume.

## Conventions & gotchas

- Router is Go 1.22+ `ServeMux` with method prefixes (`"GET /static/"`, `"POST /cart/add"`). `GET /` handler 404s non-`/` paths itself. Admin guard `requireAdmin` only checks the `admin_session` cookie exists (user id as value, `HttpOnly`, no signature/expiry check).
- Cart = base64-JSON `crave_cart` cookie (24h, `cart` package); prices are re-derived from DB variants at add-time, totals via `cart.Total`. Listing cards show name + 2-line description + an `Order` button that goes to the product page, where size, message, and icing are chosen before `POST /cart/add`.
- Orders (no online payment): `POST /checkout` renders a review page (`confirm.page.html`, nothing saved yet); `POST /checkout/confirm` saves a `PENDING` order + items transactionally, fires `customer_receipt.html` + `admin_alert.html` emails in goroutines, clears the cart, and redirects to `GET /order-confirmed?id=` which shows the summary plus a `wa.me` link (`whatsAppOrderURL` in `main.go`, admin number from `ADMIN_WHATSAPP`, default `254735792437`, `07...` normalized to `254...`). Customer taps send in WhatsApp to deliver the order. Admin confirms/follows up from the dashboard (status update stays manual) — order-details page already has a `wa.me` chat link to the customer.
- Deploy: push to `main` → `.github/workflows/deploy.yml` SSH-deploys at `/opt/crave-and-glaze` (`git reset --hard`, `docker compose up -d --build`), health-checks `http://localhost:8080/`, then warns on public `https://craveandglaze.duckdns.org/`.
