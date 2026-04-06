package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/veilence/veilence-mx/backend/internal/domain"
)

// Sentinel errors returned by auth operations.
var (
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrAPIKeyNotFound         = errors.New("API key not found")
)

const (
	// AccessTokenDuration is the lifetime of an access token.
	AccessTokenDuration = 15 * time.Minute
	// RefreshTokenDuration is the lifetime of a refresh token.
	RefreshTokenDuration = 7 * 24 * time.Hour

	// TokenTypeAccess identifies an access token.
	TokenTypeAccess = "access"
	// TokenTypeRefresh identifies a refresh token.
	TokenTypeRefresh = "refresh"
)

// Claims represents the JWT claims used for authentication tokens.
type Claims struct {
	UserID    uint   `json:"user_id"`
	Email     string `json:"email"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// TokenPair holds an access token and a refresh token.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// Service provides authentication operations.
type Service struct {
	users         domain.UserRepository
	refreshTokens domain.RefreshTokenRepository
	apiKeys       domain.APIKeyRepository
	jwtSecret     []byte
}

// NewService creates a new auth service with the given repositories and JWT secret.
func NewService(users domain.UserRepository, refreshTokens domain.RefreshTokenRepository, apiKeys domain.APIKeyRepository, jwtSecret string) *Service {
	return &Service{
		users:         users,
		refreshTokens: refreshTokens,
		apiKeys:       apiKeys,
		jwtSecret:     []byte(jwtSecret),
	}
}

// hashRefreshToken hashes a refresh token using SHA-256 for secure storage.
// Refresh tokens are high-entropy random values, so SHA-256 is sufficient
// (unlike passwords, they don't need bcrypt's slow hashing).
func hashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// hashPassword hashes the given plaintext password using bcrypt.
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// checkPassword compares the given plaintext password against the stored hash.
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Register creates a new user with the given credentials.
func (s *Service) Register(email, password, firstName, lastName string) (*domain.User, error) {
	ctx := context.Background()

	// Check for existing user
	_, err := s.users.FindByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailAlreadyRegistered
	}
	// If the error is NOT "not found", it's a real error
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("checking existing user: %w", err)
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: passwordHash,
		FirstName:    firstName,
		LastName:     lastName,
		IsActive:     true,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	slog.Info("user registered", "user_id", user.ID, "email", user.Email)
	return user, nil
}

// Login authenticates a user and returns a token pair.
func (s *Service) Login(email, password string) (*domain.User, *TokenPair, error) {
	ctx := context.Background()

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, errors.New("invalid email or password")
	}

	if !user.IsActive {
		return nil, nil, errors.New("account is deactivated")
	}

	if !checkPassword(password, user.PasswordHash) {
		return nil, nil, errors.New("invalid email or password")
	}

	tokens, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("generating tokens: %w", err)
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	_ = s.users.Update(ctx, user)

	slog.Info("user logged in", "user_id", user.ID, "email", user.Email)
	return user, tokens, nil
}

// RefreshTokens validates a refresh token and returns a new token pair.
func (s *Service) RefreshTokens(refreshToken string) (*TokenPair, error) {
	ctx := context.Background()

	// Hash the incoming token to look up the stored hash
	tokenHash := hashRefreshToken(refreshToken)

	// Look up the stored refresh token by hash
	stored, err := s.refreshTokens.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Check expiration
	if time.Now().After(stored.ExpiresAt) {
		_ = s.refreshTokens.Delete(ctx, stored.ID)
		return nil, errors.New("refresh token expired")
	}

	// Find the user
	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}

	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	// Delete the old refresh token (rotation)
	_ = s.refreshTokens.Delete(ctx, stored.ID)

	// Generate new token pair
	tokens, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("generating tokens: %w", err)
	}

	slog.Info("tokens refreshed", "user_id", user.ID)
	return tokens, nil
}

// Logout invalidates a refresh token.
func (s *Service) Logout(refreshToken string) error {
	ctx := context.Background()

	// Hash the incoming token to find the stored hash
	tokenHash := hashRefreshToken(refreshToken)

	if err := s.refreshTokens.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("deleting refresh token: %w", err)
	}
	return nil
}

// ValidateAccessToken parses and validates a JWT access token, returning its claims.
func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.TokenType != TokenTypeAccess {
		return nil, errors.New("not an access token")
	}

	return claims, nil
}

// GetUserByID retrieves a user by their ID.
func (s *Service) GetUserByID(id uint) (*domain.User, error) {
	ctx := context.Background()

	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// ValidateAPIKey validates an API key string and returns the associated user ID.
// It also updates the last_used_at timestamp for the key.
func (s *Service) ValidateAPIKey(rawKey string) (uint, string, error) {
	ctx := context.Background()

	if len(rawKey) < 10 {
		return 0, "", errors.New("invalid API key format")
	}

	// Prefix-based O(1) lookup: the DB has a partial index on key_prefix
	// (WHERE deleted_at IS NULL), so this query hits the index directly.
	// Typically returns 1 row; bcrypt compare confirms the match.
	prefix := rawKey[:10]
	keys, err := s.apiKeys.FindActiveByPrefix(ctx, prefix)
	if err != nil {
		return 0, "", fmt.Errorf("finding api keys: %w", err)
	}

	for _, key := range keys {
		if checkAPIKeyHash(rawKey, key.KeyHash) {
			// Check expiration
			if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
				return 0, "", errors.New("API key expired")
			}

			// Update last used
			now := time.Now()
			key.LastUsedAt = &now
			_ = s.apiKeys.Update(ctx, &key)

			// Get user email
			user, err := s.users.FindByID(ctx, key.UserID)
			if err != nil {
				return 0, "", fmt.Errorf("finding API key user: %w", err)
			}

			if !user.IsActive {
				return 0, "", errors.New("account is deactivated")
			}

			return key.UserID, user.Email, nil
		}
	}

	return 0, "", errors.New("invalid API key")
}

// generateTokenPair creates a new access/refresh token pair for the given user.
func (s *Service) generateTokenPair(ctx context.Context, user *domain.User) (*TokenPair, error) {
	// Generate access token
	accessClaims := &Claims{
		UserID:    user.ID,
		Email:     user.Email,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "veilence-mx",
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	// Generate refresh token (opaque random string)
	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}
	refreshTokenString := hex.EncodeToString(refreshTokenBytes)

	// Store SHA-256 hash of refresh token (never store plaintext)
	storedToken := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashRefreshToken(refreshTokenString),
		ExpiresAt: time.Now().Add(RefreshTokenDuration),
	}
	if err := s.refreshTokens.Create(ctx, storedToken); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

// generateAPIKeyRaw creates a new random API key string.
func generateAPIKeyRaw() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating API key: %w", err)
	}
	return "vmx_" + hex.EncodeToString(b), nil
}

// hashAPIKey hashes an API key using bcrypt for storage.
func hashAPIKey(rawKey string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	return string(hash)
}

// checkAPIKeyHash compares a raw API key against its stored hash.
func checkAPIKeyHash(rawKey, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(rawKey))
	return err == nil
}

// CreateAPIKey generates a new API key for the given user.
// The raw key is returned only once and cannot be retrieved again.
func (s *Service) CreateAPIKey(userID uint, name string, expiresAt *time.Time) (*domain.APIKey, string, error) {
	ctx := context.Background()

	rawKey, err := generateAPIKeyRaw()
	if err != nil {
		return nil, "", err
	}

	apiKey := &domain.APIKey{
		UserID:    userID,
		Name:      name,
		KeyHash:   hashAPIKey(rawKey),
		KeyPrefix: rawKey[:10],
		IsActive:  true,
		ExpiresAt: expiresAt,
	}

	if err := s.apiKeys.Create(ctx, apiKey); err != nil {
		return nil, "", fmt.Errorf("creating API key: %w", err)
	}

	slog.Info("API key created", "user_id", userID, "key_name", name, "key_id", apiKey.ID)
	return apiKey, rawKey, nil
}

// ListAPIKeys returns all active API keys for a user.
func (s *Service) ListAPIKeys(userID uint) ([]domain.APIKey, error) {
	ctx := context.Background()

	keys, err := s.apiKeys.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing API keys: %w", err)
	}
	return keys, nil
}

// RevokeAPIKey soft-deletes an API key if it belongs to the given user.
func (s *Service) RevokeAPIKey(userID, keyID uint) error {
	ctx := context.Background()

	if err := s.apiKeys.SoftDelete(ctx, userID, keyID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("revoking API key: %w", err)
	}

	slog.Info("API key revoked", "user_id", userID, "key_id", keyID)
	return nil
}
