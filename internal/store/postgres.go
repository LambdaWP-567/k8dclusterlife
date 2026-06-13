package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	Pool *pgxpool.Pool
}

func Connect(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &DB{Pool: pool}, nil
}

func (db *DB) Migrate() error {
	sqlDB, err := sql.Open("pgx", db.Pool.Config().ConnString())
	if err != nil {
		return fmt.Errorf("open sql: %w", err)
	}
	defer sqlDB.Close()

	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("sub fs: %w", err)
	}

	goose.SetBaseFS(sub)
	goose.SetDialect("postgres")
	return goose.Up(sqlDB, ".")
}

func (db *DB) Close() {
	db.Pool.Close()
}
