package dbtest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	superuser  = "postgres"
	maintDB    = "postgres"
	templateDB = "avitohvk_template"
)

var (
	setupOnce sync.Once
	setupErr  error
	baseDSN   string

	dbNameCounter atomic.Uint64
)

func setup(ctx context.Context) {
	c, err := tcpostgres.Run(ctx, "postgres:17",
		tcpostgres.WithDatabase(maintDB),
		tcpostgres.WithUsername(superuser),
		tcpostgres.WithPassword(superuser),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		setupErr = fmt.Errorf("dbtest: start postgres container: %w", err)
		return
	}

	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		setupErr = fmt.Errorf("dbtest: connection string: %w", err)
		return
	}
	baseDSN = dsn

	adminConn, err := pgx.Connect(ctx, baseDSN)
	if err != nil {
		setupErr = fmt.Errorf("dbtest: connect to maintenance db: %w", err)
		return
	}
	defer adminConn.Close(ctx)

	if _, err := adminConn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, templateDB)); err != nil {
		setupErr = fmt.Errorf("dbtest: create template db: %w", err)
		return
	}

	if err := func() error {
		tmplConn, err := pgx.Connect(ctx, dsnFor(templateDB))
		if err != nil {
			return fmt.Errorf("connect to template db: %w", err)
		}
		defer tmplConn.Close(ctx)
		return applyMigrations(ctx, tmplConn)
	}(); err != nil {
		setupErr = fmt.Errorf("dbtest: %w", err)
		return
	}

	if _, err := adminConn.Exec(ctx, fmt.Sprintf(`ALTER DATABASE %s WITH IS_TEMPLATE true`, templateDB)); err != nil {
		setupErr = fmt.Errorf("dbtest: mark template db: %w", err)
		return
	}
}

func dsnFor(dbName string) string {
	u, err := url.Parse(baseDSN)
	if err != nil {
		panic(fmt.Sprintf("dbtest: malformed base DSN %q: %v", baseDSN, err))
	}
	u.Path = "/" + dbName
	return u.String()
}

func applyMigrations(ctx context.Context, conn *pgx.Conn) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir(), "*_up.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no *_up.sql migrations found in %s", migrationsDir())
	}
	sort.Strings(files)

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}

		if _, err := conn.PgConn().Exec(ctx, string(sqlBytes)).ReadAll(); err != nil {
			return fmt.Errorf("exec %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

func migrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "migrations")
}

func NewPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	setupOnce.Do(func() { setup(ctx) })
	if setupErr != nil {
		t.Fatalf("dbtest: %v", setupErr)
	}

	dbName := fmt.Sprintf("test_%d_%d", time.Now().UnixNano(), dbNameCounter.Add(1))

	adminConn, err := pgx.Connect(ctx, baseDSN)
	if err != nil {
		t.Fatalf("dbtest: connect to maintenance db: %v", err)
	}
	defer adminConn.Close(ctx)

	if _, err := adminConn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s`, dbName, templateDB)); err != nil {
		t.Fatalf("dbtest: create test db %s: %v", dbName, err)
	}

	pool, err := pgxpool.New(ctx, dsnFor(dbName))
	if err != nil {
		t.Fatalf("dbtest: connect pool to test db %s: %v", dbName, err)
	}

	t.Cleanup(func() {
		pool.Close()

		dropCtx := context.Background()
		conn, err := pgx.Connect(dropCtx, baseDSN)
		if err != nil {
			t.Logf("dbtest: cleanup: connect to maintenance db: %v", err)
			return
		}
		defer conn.Close(dropCtx)

		if _, err := conn.Exec(dropCtx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, dbName)); err != nil {
			t.Logf("dbtest: cleanup: drop database %s: %v", dbName, err)
		}
	})

	return pool
}
