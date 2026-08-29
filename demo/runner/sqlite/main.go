// Command sqlite runs the SQLite TARGET migration demo. It imports the
// sqlite_migrations package so its init() registers the Go migrations
// (copy from Postgres, fetch posts from the HTTP API), then runs them in
// timestamp order alongside the SQL migrations.
package main

import (
	"context"
	"os"

	"github.com/pressly/goose/v3/demo/runnerlib"
	_ "github.com/pressly/goose/v3/demo/sqlite_migrations"
	_ "modernc.org/sqlite" // register "sqlite" driver
)

func main() {
	if len(os.Args) < 2 {
		runnerlib.Usage()
	}
	runnerlib.RunSQLite(context.Background(), os.Args[1])
}
