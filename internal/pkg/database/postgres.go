package database

import (
	"database/sql"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// NewPostgresDB creates a new PostgreSQL database connection.
func NewPostgresDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Ping the database to verify the connection.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
