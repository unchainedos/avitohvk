---
name: migrations
description: Add or review a SQL migration in avitohvk's migrations/ directory — numbering, the required up/down pair, naming and SQL style conventions, and how to verify a new migration actually applies cleanly. Use whenever creating a new migrations/NNN_*.sql file or reviewing one someone else added.
---

# DB migrations in avitohvk

## Naming and pairing

`migrations/NNN_short_scope_up.sql` + `migrations/NNN_short_scope_down.sql`,
zero-padded 3-digit sequence, e.g.:

```
013_items_locked_by_deal_up.sql
013_items_locked_by_deal_down.sql
```

- Number = next unused integer, always 3 digits (`014_`, not `14_`) — both the
  docker-compose migrator and `internal/repository/dbtest` apply files in
  lexicographic sort order of the glob `*_up.sql`, so the zero-padding is what
  keeps ordering correct past 099.
- `short_scope` names the table or concern the migration touches, snake_case,
  matches on the up/down pair exactly.
- Every up file has a down file, even for a migration that only adds a column —
  see any pair under `migrations/` for the expected symmetry.
- One migration = one coherent change. Don't fold an unrelated table's change
  into the same numbered pair (compare `006_chain_deals_up.sql`,
  `007_chain_deal_transactions_up.sql`, `009_..._unique_participant_up.sql` —
  each constraint/table addition gets its own number even when related).

## SQL style

- Lowercase keywords (`create table`, not `CREATE TABLE`).
- Every constraint is named with a prefix and referenced by that name in the down
  file: `fk_` (foreign key), `chk_` (check), `uq_` (unique), `idx_` (index — not
  a constraint but same prefix convention), `trg_` (trigger). Example from
  `008_checks_triggers_up.sql`:

  ```sql
  alter table chain_deals
      add constraint chk_chain_deals_participants check (participants >= 2);
  ```

- Down files use `if exists` / `if not exists` guards and undo in **reverse**
  dependency order — triggers before the function they call, constraints before
  the column/table they're on:

  ```sql
  drop trigger if exists trg_chain_deals_updated_at on chain_deals;
  drop function if exists set_updated_at();
  alter table chain_deals drop constraint if exists chk_chain_deals_participants;
  ```

- Altering an existing check constraint is drop-then-recreate under the same
  name, not a separate rename (see `011_chain_deals_negotiation_window_up.sql`
  dropping and re-adding `chk_chain_deals_deadline_after_created` with a new
  condition).
- No transaction wrapping inside the file — each `*_up.sql` is executed as one
  multi-statement batch by both the docker-compose migrator (`psql -f`) and
  `dbtest.applyMigrations` (`conn.PgConn().Exec(...).ReadAll()`); don't add your
  own `begin`/`commit`.

## How migrations get applied — and how to verify a new one

There is **no** migration framework (no golang-migrate, no goose) — both apply
paths are a plain glob + sort + execute over `*_up.sql`:

- **docker-compose** (`init_container` service): `ls /migrations/*_up.sql | sort`,
  then `psql -v ON_ERROR_STOP=1 -f "$f"` per file.
- **tests** (`internal/repository/dbtest.applyMigrations`): same glob+sort, run
  once against a template database that every test then clones from.

Because of the second point, **any** repository-layer test run
(`go test ./internal/repository/...`) already exercises every `*_up.sql` file
end-to-end against real Postgres 17 — that's the fastest way to catch a syntax
error or ordering bug in a new migration, no separate manual step required.
`*_down.sql` files are **not** executed by anything in the repo (no automated
rollback path exists) — correctness there is on the honor system, so review a
down file as carefully as the up file, since nothing will catch a mistake in it.

## Where to look for a pattern to copy

- New table: `006_chain_deals_up.sql` / `_down.sql`
- Adding a column + index: `010_chain_deals_creator_up.sql`,
  `013_items_locked_by_deal_up.sql`
- Adding/altering a check constraint: `011_chain_deals_negotiation_window_up.sql`
- Dropping a column and its constraint together:
  `012_chain_deals_drop_participants_up.sql`
- Reusable trigger function pattern (`set_updated_at()`, attached to multiple
  tables): `008_checks_triggers_up.sql` / `_down.sql`
