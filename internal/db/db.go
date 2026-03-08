// Package db provides PostgreSQL connection and migration management.
package db

import (
	"context"
	"fmt"
	"time"

	"smartbed/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// Connect creates a new *sqlx.DB connection pool.
func Connect(cfg *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}

	log.Info().Msg("PostgreSQL connected")
	return db, nil
}

// RunMigrations applies all pending up migrations from the given path.
func RunMigrations(dsn, migrationsPath string) error {
	// Build the postgres URL from DSN (golang-migrate needs it as a URL)
	m, err := migrate.New(migrationsPath, "postgres://"+dsn)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	log.Info().Msg("Database migrations applied")
	return nil
}
