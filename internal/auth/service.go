package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

// Service defines the contract for the auth business logic.
type Service interface {
	Register(ctx context.Context, email, username, password string) (*nexusclashv1.UUID, error)
	Login(ctx context.Context, email, password string) (string, error)
}

// Config holds the configuration needed by the auth service.
type Config struct {
	JWTSecret     string
	TokenDuration time.Duration
}

type service struct {
	repo   Repository
	config Config
}

func NewService(repo Repository, config Config) Service {
	return &service{
		repo:   repo,
		config: config,
	}
}

// Register handles the business logic for creating a new user.
func (s *service) Register(ctx context.Context, email, username, password string) (*nexusclashv1.UUID, error) {
	// Here you would add more robust validation (e.g., using a validation library).
	if email == "" || username == "" || len(password) < 8 {
		return nil, errors.New("invalid input: email, username, and password (min 8 chars) are required")
	}

	// Hash the password using bcrypt, the industry standard for password hashing.
	// A higher cost means more CPU time is required, making brute-force attacks slower.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		return nil, err
	}

	userID, err := s.repo.CreateUser(ctx, email, username, string(hashedPassword))
	if err != nil {
		// The repository already logged the specific error, so we just return it.
		return nil, err
	}

	slog.Info("New user registered successfully", "userID", userID)
	return &nexusclashv1.UUID{Value: userID}, nil
}

// Login verifies credentials and returns a JWT on success.
func (s *service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		// The repository handles the ErrUserNotFound case.
		return "", err
	}

	// Compare the provided password with the stored hash.
	// This function securely compares them without revealing the hash.
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// If the passwords don't match, bcrypt returns an error.
		return "", ErrUserNotFound // Return the same error as user not found to prevent email enumeration attacks.
	}

	// If the password is correct, generate a JWT.
	return s.generateJWT(user)
}

// Claims defines the payload for our JWT.
type Claims struct {
	UserID   string `json:"uid"`
	Username string `json:"uname"`
	jwt.RegisteredClaims
}

// generateJWT creates a signed JWT for the given user.
func (s *service) generateJWT(user *User) (string, error) {
	// Define the token's expiration time.
	expirationTime := time.Now().Add(s.config.TokenDuration)

	// Create the JWT claims, which includes the user ID, username, and standard claims.
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			// In JWT, the expiry time is expressed as unix milliseconds.
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	// Create the token with the specified signing method and claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with our secret key to create the final JWT string.
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		slog.Error("Failed to sign JWT", "error", err)
		return "", err
	}

	return tokenString, nil
}
