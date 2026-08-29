// Command runner drives goose as a library so it can execute Go migrations.
//
// Why not the goose CLI? The CLI only runs Go migrations that are already
// compiled into the binary. Here we import the migration packages (their
// init() registers each Go migration) and call goose.UpContext, which is the
// standard goose pattern for Go migrations.
//
//   go run ./demo/runner pg up          # prepare Postgres source
//   go run ./demo/runner sqlite up      # run the PG->SQLite migration demo
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
	_ "github.com/pressly/goose/v3/demo/pg_migrations"
	_ "github.com/pressly/goose/v3/demo/sqlite_migrations"
	_ "github.com/jackc/pgx/v5/stdlib" // register "pgx" driver (postgres)
	_ "modernc.org/sqlite"             // register "sqlite" driver
)

const sqliteFile = "demo/demo.db"

func usage() {
	fmt.Fprintln(os.Stderr, `usage: go run ./demo/runner {pg|sqlite} {up|status|show|reset}
  pg     : Postgres source (prep: create app_user + seed from /users API)
  sqlite : SQLite target (the actual PG->SQLite migration demo)

commands:
  up     : recreate the DB from scratch, then apply all migrations
  status : show which migrations are applied
  show   : print table contents
  reset  : roll back all migrations (down)`)
	os.Exit(2)
}

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	target, cmd := os.Args[1], os.Args[2]
	ctx := context.Background()

	switch target {
	case "pg":
		runPostgres(ctx, cmd)
	case "sqlite":
		runSQLite(ctx, cmd)
	default:
		usage()
	}
}

func runPostgres(ctx context.Context, cmd string) {
	if cmd == "up" || cmd == "reset" {
		resetPostgresDB(ctx) // drop & recreate so the demo is reproducible
	}
	db := open("pgx", pgDSN(), ctx)
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		fatal(err)
	}
	dir := "demo/pg_migrations"
	runCommand(ctx, db, dir, cmd, showPostgres)
}

func runSQLite(ctx context.Context, cmd string) {
	if cmd == "up" || cmd == "reset" {
		_ = os.Remove(sqliteFile) // fresh sqlite db so the demo is reproducible
	}
	db := open("sqlite", sqliteFile, ctx)
	defer db.Close()

	if err := goose.SetDialect("sqlite3"); err != nil {
		fatal(err)
	}
	dir := "demo/sqlite_migrations"
	runCommand(ctx, db, dir, cmd, showSQLite)
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
		usage()
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
