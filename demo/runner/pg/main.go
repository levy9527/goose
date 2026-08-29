// Command pg prepares the Postgres SOURCE database for the demo.
// It runs only SQL migrations (create app_user + seed static data), and does
// NOT import the SQLite migrations package — so no Go migrations leak in.
package main

import (
	"context"
	"os"

	"github.com/pressly/goose/v3/demo/runnerlib"
	_ "github.com/jackc/pgx/v5/stdlib" // register "pgx" driver (postgres)
)

func main() {
	if len(os.Args) < 2 {
		runnerlib.Usage()
	}
	runnerlib.RunPostgres(context.Background(), os.Args[1])
}
