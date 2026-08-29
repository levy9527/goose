package sqlitemigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/demo/api"
)

func init() {
	goose.AddMigrationContext(upFetchPosts, downFetchPosts)
}

// upFetchPosts is a Go migration that calls the jsonplaceholder
// /users/1/posts HTTP API and fills the SQLite post table. It must run
// BEFORE the rename migration (2001), because it writes to the original
// column name `title`. Goose guarantees this order.
func upFetchPosts(ctx context.Context, tx *sql.Tx) error {
	posts, err := api.FetchPosts(ctx)
	if err != nil {
		return err
	}
	return api.InsertPosts(ctx, tx, posts)
}

func downFetchPosts(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM post`)
	return err
}
