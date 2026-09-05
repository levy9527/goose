package sqlitemigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/demo/api"
)

func init() {
	// 用 NoTx 而不是默认的事务内 migration：
	// 这段迁移含外部 HTTP 请求(可能耗时几百 ms)。若用默认 AddMigrationContext,
	// 整个网络往返期间都会挂着一个打开的写事务/锁——对线上 PG 是真实的隐患
	// (长事务、写锁占用)。生产实践: 网络/长耗时操作用 AddMigrationNoTxContext
	// (在事务外跑), 只在事务内做短暂、纯粹的写库操作。
	goose.AddMigrationNoTxContext(upFetchPosts, downFetchPosts)
}

// upFetchPosts runs OUTSIDE a transaction. It calls the jsonplaceholder
// /users/1/posts HTTP API and fills the SQLite post table. It must run BEFORE
// the rename migration, because it writes to the original column `title`.
func upFetchPosts(ctx context.Context, db *sql.DB) error {
	posts, err := api.FetchPosts(ctx)
	if err != nil {
		return err
	}
	// NoTx migration receives *sql.DB (not *sql.Tx); each statement is
	// autocommitted. InsertPosts accepts either via the execable interface.
	return api.InsertPosts(ctx, db, posts)
}

func downFetchPosts(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DELETE FROM post`)
	return err
}
