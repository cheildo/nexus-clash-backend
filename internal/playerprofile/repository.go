package playerprofile

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
	"github.com/lib/pq"
)

var (
	ErrProfileNotFound      = errors.New("profile not found")
	ErrUsernameNotAvailable = errors.New("username is not available")
)

// Repository defines the database operations for player profiles.
type Repository interface {
	CreateProfile(ctx context.Context, userID, username string) (*nexusclashv1.Profile, error)
	GetProfile(ctx context.Context, userID string) (*nexusclashv1.Profile, error)
}

type postgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// CreateProfile inserts a new player profile into the database.
func (r *postgresRepository) CreateProfile(ctx context.Context, userID, username string) (*nexusclashv1.Profile, error) {
	query := `
		INSERT INTO profiles (user_id, username)
		VALUES ($1, $2)
		RETURNING user_id, username, level, stats_kills, stats_deaths, stats_assists, stats_wins, stats_losses;
	`
	p := &nexusclashv1.Profile{Stats: &nexusclashv1.PlayerStats{}}

	err := r.db.QueryRowContext(ctx, query, userID, username).Scan(
		&p.UserId, &p.Username, &p.Level,
		&p.Stats.Kills, &p.Stats.Deaths, &p.Stats.Assists, &p.Stats.Wins, &p.Stats.Losses,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			// Check for unique violation on username or foreign key violation on user_id
			if pqErr.Code.Name() == "unique_violation" {
				return nil, ErrUsernameNotAvailable
			}
			if pqErr.Code.Name() == "foreign_key_violation" {
				return nil, errors.New("user does not exist")
			}
		}
		slog.Error("Failed to create profile in database", "error", err)
		return nil, err
	}

	// The returned user_id from the DB is a string, so we wrap it in our UUID message type.
	p.UserId = &nexusclashv1.UUID{Value: userID}
	return p, nil
}

// GetProfile retrieves a player profile from the database by user ID.
func (r *postgresRepository) GetProfile(ctx context.Context, userID string) (*nexusclashv1.Profile, error) {
	query := `
		SELECT user_id, username, level, stats_kills, stats_deaths, stats_assists, stats_wins, stats_losses
		FROM profiles
		WHERE user_id = $1;
	`
	p := &nexusclashv1.Profile{Stats: &nexusclashv1.PlayerStats{}}
	var scannedUserID string

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&scannedUserID, &p.Username, &p.Level,
		&p.Stats.Kills, &p.Stats.Deaths, &p.Stats.Assists, &p.Stats.Wins, &p.Stats.Losses,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		slog.Error("Failed to get profile from database", "error", err)
		return nil, err
	}

	p.UserId = &nexusclashv1.UUID{Value: scannedUserID}
	return p, nil
}
