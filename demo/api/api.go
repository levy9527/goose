// Package api contains the "complex Go code" this demo automates:
// hitting the jsonplaceholder HTTP API, decoding JSON, and writing rows.
//
// The same functions are reused in three places:
//   - a Postgres migration (seed app_user from /users)
//   - a SQLite migration (fill post from /users/1/posts)
//   - the "bad" non-automated demo (which runs out of order and fails)
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	UsersURL = "https://jsonplaceholder.typicode.com/users"
	PostsURL = "https://jsonplaceholder.typicode.com/users/1/posts"
)

var client = &http.Client{Timeout: 20 * time.Second}

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Website  string `json:"website"`
}

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

func getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("new request %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}

// FetchUsers GET /users and decodes the response.
func FetchUsers(ctx context.Context) ([]User, error) {
	var users []User
	if err := getJSON(ctx, UsersURL, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// FetchPosts GET /users/1/posts and decodes the response.
func FetchPosts(ctx context.Context) ([]Post, error) {
	var posts []Post
	if err := getJSON(ctx, PostsURL, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

// InsertUsers inserts users into an app_user table. Uses Postgres ($N) placeholders.
func InsertUsers(ctx context.Context, exec execable, users []User) error {
	for _, u := range users {
		if _, err := exec.ExecContext(ctx,
			`INSERT INTO app_user (id, name, username, email, website) VALUES ($1,$2,$3,$4,$5)`,
			u.ID, u.Name, u.Username, u.Email, u.Website,
		); err != nil {
			return fmt.Errorf("insert user %d: %w", u.ID, err)
		}
	}
	return nil
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
