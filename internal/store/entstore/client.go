// Package entstore provides database connection utilities and
// persistence logic for AStRA graphs using Ent.
//
// This file contains database initialization helpers.
package entstore

import (
	"context"
	"fmt"

	genent "github.com/TSELab/astra/internal/store/ent"

	_ "github.com/mattn/go-sqlite3"
)

// OpenSQLite initializes a SQLite-backed Ent client and returns
// a Store instance.
//
// It also ensures that the database schema is created before use.
// The database file will be created if it does not already exist.
func OpenSQLite(path string) (*Store, error) {
	client, err := genent.Open("sqlite3", fmt.Sprintf("file:%s?_fk=1", path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite ent client: %w", err)
	}

	if err := client.Schema.Create(context.Background()); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return New(client), nil
}
