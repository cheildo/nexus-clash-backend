package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/lib/pq" // Used for handling specific PostgreSQL errors
)

// Custom error variables for clear, service-level error handling.
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailOrUserExists = errors.New("email or username already exists")
)

// User is a domain model representing a user, decoupled from the database schema.
type User struct {
	ID           string
	Email        string
	Username     string
	PasswordHash string
}

// Repository defines the contract for database operations for the auth service.
type Repository interface {
	CreateUser(ctx context.Context, email, username, hashedPassword string) (string, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
}

type postgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// CreateUser inserts a new user record into the database.
func (r *postgresRepository) CreateUser(ctx context.Context, email, username, hashedPassword string) (string, error) {
	query := `
		INSERT INTO users (email, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id;`

	var userID string
	err := r.db.QueryRowContext(ctx, query, email, username, hashedPassword).Scan(&userID)
	if err != nil {
		// Check if the error is a PostgreSQL unique violation.
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			slog.Warn("Attempted to create user with duplicate email or username", "email", email, "username", username)
			return "", ErrEmailOrUserExists
		}
		slog.Error("Failed to create user in database", "error", err)
		return "", err // Return the original error for internal logging.
	}

	return userID, nil
}

// GetUserByEmail fetches a user record from the database by their email address.
func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, username, password_hash 
		FROM users 
		WHERE email = $1;`

	var user User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
	)

	if err != nil {
		// If no user is found, sql.ErrNoRows is returned. We map this to our custom ErrUserNotFound.
		// This decouples our business logic from the specific database implementation.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		slog.Error("Failed to get user by email from database", "error", err)
		return nil, err // For any other database error.
	}

	return &user, nil
}
