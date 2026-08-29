// Package sqlitemigrations holds the SQLite TARGET migrations.
// This is the actual "migrate from Postgres to SQLite" demo.
package sqlitemigrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	// Register the "pgx" driver so this migration can read the Postgres source.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upCopyAppUserFromPg, downCopyAppUserFromPg)
}

// pgDSN returns the Postgres source connection string. Override via PG_DSN.
func pgDSN() string {
	if v := os.Getenv("PG_DSN"); v != "" {
		return v
	}
	return "postgres://postgres:postgres@127.0.0.1:5432/goose_demo?sslmode=disable"
}

// upCopyAppUserFromPg is the centerpiece of the demo: it is a Go migration
// that opens a SECOND connection to the Postgres source database, reads every
// row of app_user, and writes them into the SQLite target table. This kind of
// cross-database data move is impossible to express as a plain SQL script.
func upCopyAppUserFromPg(ctx context.Context, tx *sql.Tx) error {
	pg, err := sql.Open("pgx", pgDSN())
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer pg.Close()
	if err := pg.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	rows, err := pg.QueryContext(ctx,
		`SELECT id, name, username, email, website FROM app_user ORDER BY id`)
	if err != nil {
		return fmt.Errorf("query postgres app_user: %w", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var (
			id                        int
			name, username, email, web string
		)
		if err := rows.Scan(&id, &name, &username, &email, &web); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO app_user (id, name, username, email, website) VALUES (?,?,?,?,?)`,
			id, name, username, email, web,
		); err != nil {
			return fmt.Errorf("insert app_user into sqlite: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	fmt.Printf("  [goose] copied %d user(s) from Postgres -> SQLite\n", count)
	return nil
}

func downCopyAppUserFromPg(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM app_user`)
	return err
}
