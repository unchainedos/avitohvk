# avitohvk — agent guide

Barter/exchange marketplace ("avito exchange chain"). Go backend (chi + pgx/v5 +
Postgres), React/TS frontend in `web/`. This file orients any coding agent working
in this repo; see `README.md` for the full (Russian) product write-up.

## Stack

- Go 1.25, `chi` router, `pgx/v5` + `pgxpool`, PostgreSQL 17
- Auth: `golang-jwt/v5` (HS256), delivered via an httpOnly `access_token` cookie —
  not an `Authorization: Bearer` header
- Config: `viper`, loaded from `config/config.yaml` (see `config/config.go`)
- `web/`: React 19 + TypeScript + Vite, MobX, React Router, React Hook Form, Zod,
  Tailwind
- Local stack: `docker-compose.yml` (postgres + one-step migrator + app + frontend)

## Layout

```
cmd/main.go            entrypoint: wires repositories -> services -> handlers -> router
config/                viper config loading
migrations/             numbered SQL migrations, each with an _up/_down pair
web/                    React/TS frontend (separate npm project, see web/package.json)
internal/
  domain/                core entities: item, user, wish, deal, proposal
  dto/                    HTTP request/response structs, kept separate from domain
  errors/                 sentinel errors (ErrBadRequest/Unauthorized/NotFound/
                          Conflict/BadGateway) that utilhttp maps to status codes
  server/                 http.Server start/stop, graceful shutdown on signal/timeout
  repository/             one package per entity, pgx/pgxpool against Postgres:
    chown/, deal/, proposal/, item/, user/, wish/, search/, dbtest/
  service/                business logic, mirrors repository/ package-for-package:
    chown/, item/, proposal/, search/, user/, wish/
    (there is no separate service/deal package — deal read/update lives in
    service/proposal, see below)
  transport/
    router/                assembles route groups (public vs JWT-protected) from
                           each handler's RegisterRoutes
    middleware/             JWT auth middleware, reads/validates the access_token
                           cookie and puts the user id in request context
    utilhttp/                shared JSON read/write + error->HTTP status mapping
    handler/                 one package per resource: auth, user, users, item,
                           wish, search, chown, deal, props
```

`users` (plural) is a read-only package for viewing *another* user's public items
and deals; `user` (singular) is the authenticated user's own profile CRUD. Don't
confuse the two when searching for "user" code.

`deal` and `props` are two thin handler packages over the **same** underlying
`proposal` service (`ProposalService` — see `cmd/main.go`): `deal` only exposes
`GET /deal/{deal_id}`, `props` exposes create/join/update/withdraw/approve. There
is no `internal/service/deal` package despite `internal/repository/deal` existing —
the deal repository is a dependency of the proposal service, not a peer of it.

## Domain model

- `chain_deals` — a formal, negotiated barter deal.
- `chain_deal_transactions` — one participant's proposal within a deal.
- `transactions` — a ledger row (item / from_user / to_user / quantity).
- `items` — `holder_id` is the current holder, mutated only by chown's transfer
  mechanics; `author_id` is the true, immutable owner, never touched by any code
  path; `is_locked` / `locked_by_deal_id` is exclusivity state.
- `wishes` / `wish_items` — wishlist, used to resolve recipients. **Not**
  deal-scoped: whoever's wish for an item was created earliest always wins the
  match, across every deal/chown that ever references that item.

### `props` vs `chown`

- `props` — the formal, negotiated chain-deal flow: `CreateDeal`, `CreateProposal`
  (join), `Update`, `Withdraw`, `Approve`.
- `chown` — the immediate, one-sided "relay" mechanism: claims exclusive rights on
  a target item (`is_locked=true`, holder unchanged) while tossing offered items
  for real (`holder_id` changes) to whoever wishes for them.
- **Deal beats chain, deal beats deal.** A `locked_by_deal_id` lock always wins
  over a bare chown claim (`locked_by_deal_id IS NULL`). An item accepted into one
  deal immediately cancels any other still-open deal relying on the same item,
  cascading `DECLINED` to that deal's own proposals and freeing anything it
  exclusively held.

Full narrative write-up (chain formation, right-to-left approval, cascading
cancellation, deadlock retries) is in `README.md` under "Особенности реализации".
The invariants below are the load-bearing rules extracted from it — read them
before changing deal/proposal/chown logic, don't re-derive them from scratch.

### Key invariants

- **Right-to-left approval order**: the creator and the root-item holder are
  exempt and may approve any time; every other participant may only approve once
  the chain has fully locked (`TryLockChain`) *and* their own recipient has
  already been accepted.
- **An item locks to its deal the moment its own proposal is accepted** — not
  only once the whole chain closes. Otherwise the creator/root-holder (the two
  roles allowed to approve early) can toss their promised item away via an
  unrelated `chown` call before the chain ever locks, and completion later
  silently overwrites the real holder with a stale cached recipient.
- **Every path that cancels a deal** (withdraw, deadline expiry, a competing deal
  winning the same item) must cascade `DECLINED` to *every* participant's
  proposal, and release *only* the items that deal itself holds the lock on
  (`locked_by_deal_id = <this deal>`) — never items merely *mentioned* in its
  transaction history, which can belong to a completely unrelated chown claim or
  a different deal.
- **Cross-deal work must go through `retryOnDeadlock`.** Anything that reaches
  across into another deal's rows (competing-deal cancellation inside both
  `SetStatus`'s `lockOfferedItem` and `TryLockChain`) can lose a Postgres deadlock
  race against a concurrent close of that other deal. Postgres always cleanly
  rolls back the loser, so retrying is safe — but the retry has to wrap *every*
  call that can deadlock, not just one of them, or extra retry budget silently
  does nothing.
- Same-deal mutations (`Approve`, `WithdrawProposal`, `CreateProposal`) serialize
  against each other via `DealRepository.LockDeal` (a Postgres advisory lock
  keyed on the deal id). Without it, e.g. a withdraw and an approve on the same
  deal race and can leave an `ACCEPTED` proposal permanently attached to a
  `CANCELLED` deal.

## API surface (all under `/api/v1`)

Public (no auth):
- `POST /login`, `POST /register`
- `GET /items/{item_id}`, `GET /search`
- `GET /wishes/{wish_id}`, `GET /wishes/{user_id}/items`
- `GET /users/{user_id}/items`, `GET /users/{user_id}/deals`
- `GET /user/{user_id}`

JWT-protected (cookie `access_token`):
- `POST /chown/{item_id}`
- `GET /deal/{deal_id}`
- `POST /props/`, `GET /props/{deal_id}`, `PATCH /props/{deal_id}`,
  `POST /props/{deal_id}` (join), `DELETE /props/{deal_id}` (withdraw),
  `GET /props/users/{user_id}`, `POST /props/approve/{deal_id}`
- `POST /items`, `PATCH /items/{item_id}`, `DELETE /items/{item_id}`
- `POST /wishes/{wish_id}`, `PATCH /wishes/{wish_id}`, `DELETE /wishes/{wish_id}`
- `POST /user`, `PATCH /user/{user_id}`, `DELETE /user/{user_id}`,
  `POST /user/add`

## Ownership

Per `README.md`'s "Распределение ответственности" and git history: deal/proposal
schema+migrations, chain formation & right-to-left approval, `chown`, cascading
cancellation, deadlock-retry handling, and server infra belong to max23 (this
session's user). user/item/wish/auth/ HTTP handlers , docker-compose.yaml belongs to fvaiiii,  search HTTP handlers, app Dockerfile belons to unchainedos and the `web/` frontend belongs to Tyuweer. Stick to the area you're asked to touch; if a task crosses into
the other person's code, flag it rather than silently expanding scope.

## Build / test

```bash
go build ./...
go vet ./...
golangci-lint run ./...     # see .golangci.yml: errcheck, govet, staticcheck,
                             # unused, gosimple, ineffassign, gocritic, revive,
                             # gofmt, misspell (US locale)
go test ./...
go test -race ./...
```

Unit tests and repository-layer integration tests need only a working Docker —
`internal/repository/dbtest` spins up Postgres in testcontainers-go per test, no
manual setup required.

Frontend: `cd web && npm install && npm run dev`.

## Environment notes

- The project's `docker-compose.yml` binds host port 5432, which conflicts with
  an already-running local Postgres.
- After rebuilding the server binary by hand, double-check its mtime is newer
  than the source before trusting a "no change" result — a background+disown
  restart has occasionally reported success while still running a stale binary.

## Conventions to follow

- Table-driven Go tests, `t.Parallel()`, stdlib-style fakes for service/handler
  unit tests, real Postgres via testcontainers-go for repository tests.
- Errors flow as sentinel values from `internal/errors`, translated to HTTP status
  by `internal/transport/utilhttp` — don't hand-roll status codes in handlers.
- `dto` structs are strictly separate from `domain` structs; handlers convert
  between them explicitly (see `toItemResponse`-style helpers).
- Route registration is split into `RegisterPublicRoutes` /
  `RegisterProtectedRoutes` per handler, composed in `cmd/main.go` into two
  `router.WithGroup` calls (public, then JWT-protected). New endpoints should
  follow this split rather than adding ad hoc middleware per-route.
