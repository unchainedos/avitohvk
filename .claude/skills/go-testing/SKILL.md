---
name: go-testing
description: Write and run Go tests for the avitohvk backend following this project's exact conventions — table-driven tests, hand-rolled fakes for unit tests, testcontainers-go via internal/repository/dbtest for repository tests. Use whenever writing a new *_test.go file, extending test coverage, or verifying a change to internal/{repository,service,transport/handler} with the full check suite.
---

# Go testing in avitohvk

## Full check suite

Run in this order before considering a Go change done:

```bash
go build ./...
go vet ./...
golangci-lint run ./...
go test ./...
go test -race ./...
```

`-race` matters here specifically: `internal/service/proposal`, `internal/repository/deal`,
and `internal/repository/chown` contain concurrency-critical logic (advisory locks,
deadlock retries, competing-deal cancellation) — a race that only shows up under
`-race` is exactly the class of bug this codebase has been bitten by before.

`.golangci.yml` excludes `errcheck` and `gocritic` for `_test.go` files — don't chase
those two linters' complaints inside tests, but every other enabled linter
(govet, staticcheck, unused, gosimple, ineffassign, revive, gofmt, misspell) still
applies to test code.

## Two kinds of tests, two different setups

### Unit tests (service / handler layer) — hand-rolled fakes, no mocking library

No mock generator, no testify/mock — every dependency interface gets a small
hand-written fake struct in the `_test.go` file. Pattern, straight from
`internal/service/proposal/service_test.go`:

```go
type fakeDealRepo struct {
	createFunc  func(ctx context.Context, rootItemID, creatorID string, w time.Duration) (domain.Deal, error)
	lockErr     error

	updateStatusCalls []domain.DealStatus // record calls for assertions
	lockDealCalled    bool
}

func (f *fakeDealRepo) Create(ctx context.Context, rootItemID, creatorID string, w time.Duration) (domain.Deal, error) {
	if f.createFunc != nil {
		return f.createFunc(ctx, rootItemID, creatorID, w)
	}
	return domain.Deal{}, nil // zero-value default when the test doesn't care
}
```

- One `xxxFunc` field per method, only set in the tests that need custom behavior;
  unset means "return the zero value, nil error".
  If a test's assertion is *only* about the returned value, prefer setting the
  return value directly and skip a `xxxFunc` closure.
- Record what was called (`xxxCalled`, `xxxCalls []T`) instead of asserting via a
  mocking framework's `.Times()`/`.Called()` — plain field, plain equality check.
- Build the service under test with a small local constructor helper
  (e.g. `newTestService(t, ...)` wiring the fakes together) rather than repeating
  `NewService(d, p, c)` in every test.

### Repository tests — real Postgres via testcontainers-go

Use `avitohvk/internal/repository/dbtest`:

```go
pool := dbtest.NewPool(t)   // fresh migrated database for this test, auto-dropped in t.Cleanup
repo := NewRepository(pool)
```

`dbtest.NewPool` starts one shared Postgres 17 testcontainer for the whole test
binary (via `sync.Once`), builds a migrated template database once, then clones a
throwaway database per call from that template — so each test gets real isolation
without re-running all migrations every time. No manual setup needed beyond a
working Docker daemon.

Seed fixtures with small `t.Helper()` functions that insert directly via SQL rather
than going through the service layer — see `seedUser`/`seedItem` in
`internal/repository/deal/postgres_test.go` for the exact style to copy/extend
(cast string params to `::uuid`, `RETURNING id::text`, fail via `t.Fatalf`).

For asserting Postgres-specific failures (constraint violations, deadlocks), use
`pgconn.PgError` and check `.Code`, not string-matching the error message — see
`pgErrCode(t, err)` in the same file, and `isDeadlock`/code `"40P01"` in
`internal/service/proposal/service_test.go` for the deadlock-detection pattern.

## Table-driven shape

Every table-driven test in this codebase follows the same shape — `t.Parallel()`
both on the outer test function and inside each subtest:

```go
func TestSomething(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// ...inputs...
		want bool // or wantErr error, wantStatus int, etc.
	}{
		{name: "descriptive case name", /* ... */, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := something(tt.input)
			if got != tt.want {
				t.Errorf("something(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
```

Assertions are plain stdlib `t.Errorf`/`t.Fatalf` — no testify `assert`/`require`
despite `testify` being in `go.sum` (it's pulled in transitively by
testcontainers-go, not a project dependency to write assertions with).

## Coverage

After adding tests for previously-untested branches, re-check coverage rather than
assuming the new test closed the gap:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep <package>
```

## Where to look for a live example

- Fakes + table-driven service tests: `internal/service/proposal/service_test.go`,
  `internal/service/chown/service_test.go`
- Repository tests against real Postgres: `internal/repository/deal/postgres_test.go`,
  `internal/repository/proposal/postgres_test.go`
- `dbtest` internals (only read this if you need to change the harness itself,
  not to write an ordinary test): `internal/repository/dbtest/*.go`
