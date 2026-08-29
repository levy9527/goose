// Package pgmigrations holds the Postgres "preparation" migrations.
// This is the SOURCE database that the demo later migrates into SQLite.
package pgmigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/demo/api"
)

func init() {
	goose.AddMigrationContext(upSeedUsersFromApi, downSeedUsersFromApi)
}

// upSeedUsersFromApi is a Go migration: it calls the jsonplaceholder /users
// HTTP API and fills the Postgres app_user table. This is the "preparation"
// step — without goose, this kind of code sits outside the script automation
// chain and is hard to keep ordered with the schema changes.
func upSeedUsersFromApi(ctx context.Context, tx *sql.Tx) error {
	users, err := api.FetchUsers(ctx)
	if err != nil {
		return err
	}
	if err := api.InsertUsers(ctx, tx, users); err != nil {
		return err
	}
	return nil
}

func downSeedUsersFromApi(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM app_user`)
	return err
}
