// Package api contains the "complex Go code" this demo automates: hitting the
// jsonplaceholder HTTP API, decoding JSON, and writing rows.
//
// It is used by:
//   - a SQLite migration (fill post from /users/1/posts)
//   - the "bad" non-automated demo (which runs out of order and fails)
//
// The Postgres app_user table is NOT seeded from here: it is initialized once
// with hardcoded data (see pg_migrations/*_seed_app_user.sql).
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const PostsURL = "https://jsonplaceholder.typicode.com/users/1/posts"

var client = &http.Client{Timeout: 20 * time.Second}

type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// execable is anything that can run a statement: *sql.Tx or *sql.DB.
type execable interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// FetchPosts GET /users/1/posts and decodes the response.
func FetchPosts(ctx context.Context) ([]Post, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, PostsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request %s: %w", PostsURL, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", PostsURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", PostsURL, resp.Status)
	}
	var posts []Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, fmt.Errorf("decode %s: %w", PostsURL, err)
	}
	return posts, nil
}

// InsertPosts inserts posts into a post table using the ORIGINAL column name
// `title` (before the demo renames it to `heading`). Uses SQLite (?) placeholders.
func InsertPosts(ctx context.Context, exec execable, posts []Post) error {
	for _, p := range posts {
		if _, err := exec.ExecContext(ctx,
			`INSERT INTO post (user_id, id, title, body) VALUES (?,?,?,?)`,
			p.UserID, p.ID, p.Title, p.Body,
		); err != nil {
			return fmt.Errorf("insert post %d: %w", p.ID, err)
		}
	}
	return nil
}
