// Command bad demonstrates the NON-automated way: a human or a script-only
// tool applies the schema changes first (create post + rename its column),
// and the Go data-loading code runs LAST. Because the code still writes to
// the original column name `title`, the insert fails after the rename.
//
// Expected output: INSERT fails with "no such column: title".
//
// This is the exact failure goose prevents: goose runs the Go migration
// BEFORE the schema change, in the correct order.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/pressly/goose/v3/demo/api"
	_ "modernc.org/sqlite" // register "sqlite" driver
)

const dbFile = "demo/demo_bad.db"

func main() {
	_ = os.Remove(dbFile) // fresh db
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	// 1) The "script/human" tool applies the schema changes first:
	//    create the post table, then immediately rename title -> heading.
	if _, err := db.Exec(`CREATE TABLE post (
		user_id INTEGER NOT NULL,
		id      INTEGER NOT NULL,
		title   TEXT NOT NULL,
		body    TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		fatal(err)
	}
	if _, err := db.Exec(`ALTER TABLE post RENAME COLUMN title TO heading`); err != nil {
		fatal(err)
	}
	fmt.Println("  [bad] schema applied first: post.title renamed to post.heading")

	// 2) Now the Go data-loading code runs — it still writes to `title`,
	//    which no longer exists.
	fmt.Println("  [bad] Go code runs now, but it writes to the old column `title`...")
	posts, err := api.FetchPosts(ctx)
	if err != nil {
		fatal(err)
	}
	if err := api.InsertPosts(ctx, db, posts); err != nil {
		fmt.Println("\n  !! INSERT FAILED (schema drifted under the running code):")
		fmt.Printf("  !! %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  [unexpected] insert succeeded — this demo is broken")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
