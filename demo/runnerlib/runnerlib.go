// Package runnerlib holds the shared "goose-as-a-library" logic used by the
// Postgres (source) and SQLite (target) runner entrypoints.
//
// It is split into two entrypoints (demo/runner/pg and demo/runner/sqlite)
// rather than one: each must be a SEPARATE process that imports ONLY its own
// migration package. goose registers Go migrations in a process-global map
// (keyed by the numeric version parsed from the calling file name), and
// running a directory that contains no .go files falls back to including ALL
// globally-registered Go migrations — so the Postgres run must not import the
// SQLite migrations package at all.
package runnerlib

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
)

const sqliteFile = "demo/demo.db"

// Usage prints the shared command-line help.
func Usage() {
	fmt.Fprintln(os.Stderr, `usage: go run ./demo/runner/{pg|sqlite} {up|status|show|reset}
  pg     : Postgres source (prep: create app_user + seed static data)
  sqlite : SQLite target (the actual PG->SQLite migration demo)

commands:
  up     : recreate the DB from scratch, then apply all migrations
  status : show which migrations are applied
  show   : print table contents
  reset  : roll back all migrations (down)`)
	os.Exit(2)
}

// RunPostgres prepares the Postgres source database (all SQL migrations).
func RunPostgres(ctx context.Context, cmd string) {
	if cmd == "up" || cmd == "reset" {
		resetPostgresDB(ctx) // drop & recreate so the demo is reproducible
	}
	db := open("pgx", pgDSN(), ctx)
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		fatal(err)
	}
	runCommand(ctx, db, "demo/pg_migrations", cmd, showPostgres)
}

// RunSQLite runs the SQLite target migration demo (SQL + Go migrations).
func RunSQLite(ctx context.Context, cmd string) {
	if cmd == "up" || cmd == "reset" {
		_ = os.Remove(sqliteFile) // fresh sqlite db so the demo is reproducible
	}
	db := open("sqlite", sqliteFile, ctx)
	defer db.Close()

	if err := goose.SetDialect("sqlite3"); err != nil {
		fatal(err)
	}
	runCommand(ctx, db, "demo/sqlite_migrations", cmd, showSQLite)
}

// runCommand dispatches a goose command, or a custom "show" handler.
func runCommand(ctx context.Context, db *sql.DB, dir, cmd string, show func(context.Context, *sql.DB)) {
	var err error
	switch cmd {
	case "up":
		err = goose.UpContext(ctx, db, dir)
	case "status":
		err = goose.StatusContext(ctx, db, dir)
	case "reset":
		err = goose.ResetContext(ctx, db, dir)
	case "show":
		show(ctx, db)
		return
	default:
		Usage()
	}
	if err != nil {
		fatal(err)
	}
}

// --- database helpers ---

func pgDSN() string {
	if v := os.Getenv("PG_DSN"); v != "" {
		return v
	}
	return "postgres://postgres:postgres@127.0.0.1:5432/goose_demo?sslmode=disable"
}

func pgSuperDSN() string {
	if v := os.Getenv("PG_SUPER_DSN"); v != "" {
		return v
	}
	return "postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"
}

// resetPostgresDB drops and recreates the goose_demo database.
func resetPostgresDB(ctx context.Context) {
	admin := open("pgx", pgSuperDSN(), ctx)
	defer admin.Close()
	mustExec(admin, ctx, "DROP DATABASE IF EXISTS goose_demo WITH (FORCE)")
	mustExec(admin, ctx, "CREATE DATABASE goose_demo")
}

func open(driver, dsn string, ctx context.Context) *sql.DB {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		fatal(err)
	}
	if err := db.PingContext(ctx); err != nil {
		fatal(err)
	}
	return db
}

func mustExec(db *sql.DB, ctx context.Context, q string) {
	if _, err := db.ExecContext(ctx, q); err != nil {
		fatal(err)
	}
}

// --- show (print rows) ---

func showPostgres(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, username, email FROM app_user ORDER BY id`)
	if err != nil {
		fatal(err)
	}
	defer rows.Close()
	fmt.Println("  id | name | username | email")
	for rows.Next() {
		var id int
		var name, username, email string
		if err := rows.Scan(&id, &name, &username, &email); err != nil {
			fatal(err)
		}
		fmt.Printf("  %3d | %-14s | %-10s | %-25s\n", id, name, username, email)
	}
	if err := rows.Err(); err != nil {
		fatal(err)
	}
}

func showSQLite(ctx context.Context, db *sql.DB) {
	fmt.Println("--- SQLite app_user (migrated from Postgres) ---")
	rows, err := db.QueryContext(ctx, `SELECT id, name, username, email FROM app_user ORDER BY id`)
	if err != nil {
		fatal(err)
	}
	header := true
	for rows.Next() {
		var id int
		var name, username, email string
		if err := rows.Scan(&id, &name, &username, &email); err != nil {
			fatal(err)
		}
		if header {
			fmt.Println("  id | name | username | email")
			header = false
		}
		fmt.Printf("  %3d | %-14s | %-10s | %-25s\n", id, name, username, email)
	}
	rows.Close()

	fmt.Println("\n--- SQLite post (fetched from /users/1/posts, column renamed title -> heading) ---")
	rows2, err := db.QueryContext(ctx, `SELECT user_id, id, heading, substr(body,1,30) FROM post ORDER BY id LIMIT 10`)
	if err != nil {
		fatal(err)
	}
	defer rows2.Close()
	fmt.Println("  user_id | id | heading")
	for rows2.Next() {
		var userID, id int
		var heading, body string
		if err := rows2.Scan(&userID, &id, &heading, &body); err != nil {
			fatal(err)
		}
		fmt.Printf("  %7d | %2d | %s\n", userID, id, heading)
	}
	if err := rows2.Err(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
