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

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// Sentinel errors returned by auth operations.
var (
	ErrEmailAlreadyRegistered    = errors.New("email already registered")
	ErrAPIKeyNotFound            = errors.New("API key not found")
	ErrResetTokenInvalid         = errors.New("invalid or expired reset token")
	ErrResetTokenUsed            = errors.New("reset token already used")
	ErrVerificationInvalid       = errors.New("invalid or expired verification token")
	ErrInsufficientRole          = errors.New("API key role insufficient for this operation")
	ErrRoleExceedsUserRole       = errors.New("cannot create API key with role higher than your workspace role")
	ErrSessionNotFound           = errors.New("session not found")
	ErrEmailVerificationRequired = errors.New("email_verification_required")
)

const (
	// AccessTokenDuration is the lifetime of an access token.
	AccessTokenDuration = 15 * time.Minute
	// RefreshTokenDuration is the lifetime of a refresh token.
	RefreshTokenDuration = 7 * 24 * time.Hour
	// PasswordResetDuration is the lifetime of a password reset token.
	PasswordResetDuration = 1 * time.Hour
	// PasswordResetCooldown is the minimum interval between password reset requests for the same email.
	PasswordResetCooldown = 2 * time.Minute
	// EmailVerificationDuration is the lifetime of an email verification token.
	EmailVerificationDuration = 24 * time.Hour
	// VerificationEmailCooldown is the minimum interval between verification email requests.
	VerificationEmailCooldown = 2 * time.Minute
	// SessionDuration is the lifetime of a session.
	SessionDuration = 7 * 24 * time.Hour
	// MaxSessionsPerUser is the maximum number of concurrent sessions per user.
	MaxSessionsPerUser = 10

	// TokenTypeAccess identifies an access token.
	TokenTypeAccess = "access"
	// TokenTypeRefresh identifies a refresh token.
	TokenTypeRefresh = "refresh"
)

// Claims represents the JWT claims used for authentication tokens.
type Claims struct {
	UserID    string `json:"user_id"`
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
	users              usecase.UserRepository
	refreshTokens      usecase.RefreshTokenRepository
	apiKeys            usecase.APIKeyRepository
	passwordResets     usecase.PasswordResetTokenRepository
	emailVerifications usecase.EmailVerificationTokenRepository
	sessions           usecase.SessionRepository
	emailSender        usecase.AuthEmailSender // nil = no email delivery (dev mode)
	requireEmailVerification bool                    // from env: REQUIRE_EMAIL_VERIFICATION
	rateLimiter              usecase.RateLimiter     // nil = no rate limiting
	jwtSecret          []byte                  // primary secret (used for signing)
	jwtSecretsPrevious [][]byte                // previous secrets (accepted for validation during rotation)
}

// NewService creates a new auth service with the given repositories and JWT secret.
// The jwtSecret is the primary signing secret. previousSecrets are optional older
// secrets that are still accepted for token validation during secret rotation.
// emailSender and rateLimiter may be nil (dev mode: emails skipped,
// rate limiting skipped).
func NewService(
	users usecase.UserRepository,
	refreshTokens usecase.RefreshTokenRepository,
	apiKeys usecase.APIKeyRepository,
	passwordResets usecase.PasswordResetTokenRepository,
	emailVerifications usecase.EmailVerificationTokenRepository,
	sessions usecase.SessionRepository,
	emailSender usecase.AuthEmailSender,
	requireEmailVerification bool,
	rateLimiter usecase.RateLimiter,
	jwtSecret string,
	previousSecrets ...string,
) *Service {
	var prevKeys [][]byte
	for _, s := range previousSecrets {
		if s != "" {
			prevKeys = append(prevKeys, []byte(s))
		}
	}
	return &Service{
		users:                    users,
		refreshTokens:            refreshTokens,
		apiKeys:                  apiKeys,
		passwordResets:           passwordResets,
		emailVerifications:       emailVerifications,
		sessions:                 sessions,
		emailSender:              emailSender,
		requireEmailVerification: requireEmailVerification,
		rateLimiter:              rateLimiter,
		jwtSecret:                []byte(jwtSecret),
		jwtSecretsPrevious:       prevKeys,
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
func (s *Service) Register(email, password, firstName, lastName string) (*entity.User, error) {
	ctx := context.Background()

	// Check for existing user
	_, err := s.users.FindByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailAlreadyRegistered
	}
	// If the error is NOT "not found", it's a real error
	if !errors.Is(err, entity.ErrNotFound) {
		return nil, fmt.Errorf("checking existing user: %w", err)
	}

	if err := entity.ValidatePassword(password); err != nil {
		return nil, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &entity.User{
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

	// Best-effort: send verification email if email verification is enabled.
	// Registration succeeds regardless of whether the email is sent.
	s.sendPostRegistrationVerification(ctx, user)

	return user, nil
}

// sendPostRegistrationVerification sends a verification email to a newly
// registered user if the RequireEmailVerification config is enabled.
// Errors are logged but never propagated - registration must not fail
// because of an email delivery issue.
func (s *Service) sendPostRegistrationVerification(ctx context.Context, user *entity.User) {
	if !s.requireEmailVerification {
		return
	}

	// Generate and send the verification email (reuses existing logic)
	if _, err := s.GenerateEmailVerificationToken(user.ID); err != nil {
		slog.Error("failed to send verification email after registration", "user_id", user.ID, "email", user.Email, "error", err)
	}
}

// Login authenticates a user and returns a token pair.
// It also creates a session record for the user using the provided IP address and User-Agent.
func (s *Service) Login(email, password, ipAddress, userAgent string) (*entity.User, *TokenPair, error) {
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

	// Check global email verification requirement
	if s.requireEmailVerification && !user.EmailVerified {
		return nil, nil, ErrEmailVerificationRequired
	}

	tokens, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("generating tokens: %w", err)
	}

	// Create a session record for this login
	_, err = s.CreateSession(user.ID, hashRefreshToken(tokens.RefreshToken), ipAddress, userAgent)
	if err != nil {
		slog.Warn("failed to create session", "user_id", user.ID, "error", err)
		// Non-fatal - don't fail login if session creation fails
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

	// Check global email verification requirement
	if s.requireEmailVerification && !user.EmailVerified {
		_ = s.refreshTokens.Delete(ctx, stored.ID)
		return nil, ErrEmailVerificationRequired
	}

	// Update session LastActive if a session exists for this refresh token hash
	oldTokenHash := tokenHash
	session, sessionErr := s.sessions.FindByTokenHash(ctx, oldTokenHash)
	if sessionErr == nil {
		_ = s.sessions.UpdateLastActive(ctx, session.ID, time.Now())
	}

	// Delete the old refresh token (rotation)
	_ = s.refreshTokens.Delete(ctx, stored.ID)

	// Generate new token pair
	tokens, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("generating tokens: %w", err)
	}

	// Update the session's token hash to the new refresh token hash
	if sessionErr == nil {
		newTokenHash := hashRefreshToken(tokens.RefreshToken)
		if updateErr := s.sessions.UpdateTokenHash(ctx, session.ID, newTokenHash); updateErr != nil {
			slog.Warn("failed to update session token hash", "session_id", session.ID, "error", updateErr)
		}
		if updateErr := s.sessions.UpdateLastActive(ctx, session.ID, time.Now()); updateErr != nil {
			slog.Warn("failed to update session last active", "session_id", session.ID, "error", updateErr)
		}
	}

	slog.Info("tokens refreshed", "user_id", user.ID)
	return tokens, nil
}

// Logout invalidates a refresh token and its associated session.
func (s *Service) Logout(refreshToken string) error {
	ctx := context.Background()

	// Hash the incoming token to find the stored hash
	tokenHash := hashRefreshToken(refreshToken)

	// Delete the associated session so it no longer appears in active sessions.
	session, err := s.sessions.FindByTokenHash(ctx, tokenHash)
	if err == nil {
		if delErr := s.sessions.Delete(ctx, session.ID); delErr != nil {
			slog.Warn("failed to delete session on logout", "session_id", session.ID, "error", delErr)
		}
	}

	if err := s.refreshTokens.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("deleting refresh token: %w", err)
	}
	return nil
}

// ValidateAccessToken parses and validates a JWT access token, returning its claims.
// It first tries the primary secret, then falls back to previous secrets to
// support seamless JWT secret rotation.
func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
	// Try primary secret first
	claims, err := s.validateTokenWithSecret(tokenString, s.jwtSecret)
	if err == nil {
		return claims, nil
	}

	// Try previous secrets (rotation support)
	for _, prevSecret := range s.jwtSecretsPrevious {
		claims, prevErr := s.validateTokenWithSecret(tokenString, prevSecret)
		if prevErr == nil {
			slog.Debug("token validated with previous secret (rotation in progress)")
			return claims, nil
		}
	}

	// Return the original error from the primary secret
	return nil, err
}

// validateTokenWithSecret validates a JWT token using a specific secret.
func (s *Service) validateTokenWithSecret(tokenString string, secret []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
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
func (s *Service) GetUserByID(id string) (*entity.User, error) {
	ctx := context.Background()

	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// GetUserByEmail retrieves a user by their email address.
func (s *Service) GetUserByEmail(email string) (*entity.User, error) {
	ctx := context.Background()

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("finding user by email: %w", err)
	}
	return user, nil
}

// ValidateAPIKey validates an API key string and returns the associated user ID, email, role, and workspace ID.
// It also updates the last_used_at timestamp for the key.
func (s *Service) ValidateAPIKey(rawKey string) (string, string, entity.APIKeyRole, string, error) {
	ctx := context.Background()

	if len(rawKey) < 10 {
		return "", "", "", "", errors.New("invalid API key format")
	}

	// Prefix-based O(1) lookup: the DB has a partial index on key_prefix
	// (WHERE deleted_at IS NULL), so this query hits the index directly.
	// Typically returns 1 row; bcrypt compare confirms the match.
	prefix := rawKey[:10]
	keys, err := s.apiKeys.FindActiveByPrefix(ctx, prefix)
	if err != nil {
		return "", "", "", "", fmt.Errorf("finding api keys: %w", err)
	}

	for _, key := range keys {
		if checkAPIKeyHash(rawKey, key.KeyHash) {
			// Check expiration
			if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
				return "", "", "", "", errors.New("API key expired")
			}

			// Update last used
			now := time.Now()
			key.LastUsedAt = &now
			_ = s.apiKeys.Update(ctx, &key)

			// Get user email
			user, err := s.users.FindByID(ctx, key.UserID)
			if err != nil {
				return "", "", "", "", fmt.Errorf("finding API key user: %w", err)
			}

			if !user.IsActive {
				return "", "", "", "", errors.New("account is deactivated")
			}

			return key.UserID, user.Email, key.Role, key.WorkspaceID, nil
		}
	}

	return "", "", "", "", errors.New("invalid API key")
}

// generateTokenPair creates a new access/refresh token pair for the given user.
func (s *Service) generateTokenPair(ctx context.Context, user *entity.User) (*TokenPair, error) {
	// Generate access token
	accessClaims := &Claims{
		UserID:    user.ID,
		Email:     user.Email,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "veilence-mx",
			Subject:   user.ID,
			Audience:  jwt.ClaimStrings{"veilence-mx-api"},
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
	storedToken := &entity.RefreshToken{
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

// CreateAPIKey generates a new API key for the given user with the specified role within a workspace.
// The userRole parameter is the requesting user's current role in the workspace.
// The requested API key role must not exceed the user's own workspace role.
// The raw key is returned only once and cannot be retrieved again.
func (s *Service) CreateAPIKey(userID string, workspaceID string, name string, role entity.APIKeyRole, userRole string, expiresAt *time.Time) (*entity.APIKey, string, error) {
	ctx := context.Background()

	// Default to viewer role if not specified
	if role == "" {
		role = entity.APIKeyRoleViewer
	}

	if !entity.IsValidAPIKeyRole(string(role)) {
		return nil, "", fmt.Errorf("invalid API key role: %s", role)
	}

	// Enforce max role: user cannot create an API key with a role higher than their own
	if entity.RoleLevel(string(role)) > entity.RoleLevel(userRole) {
		return nil, "", ErrRoleExceedsUserRole
	}

	rawKey, err := generateAPIKeyRaw()
	if err != nil {
		return nil, "", err
	}

	apiKey := &entity.APIKey{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Name:        name,
		KeyHash:     hashAPIKey(rawKey),
		KeyPrefix:   rawKey[:10],
		Role:        role,
		IsActive:    true,
		ExpiresAt:   expiresAt,
	}

	if err := s.apiKeys.Create(ctx, apiKey); err != nil {
		return nil, "", fmt.Errorf("creating API key: %w", err)
	}

	slog.Info("API key created", "user_id", userID, "workspace_id", workspaceID, "key_name", name, "key_id", apiKey.ID, "role", role)
	return apiKey, rawKey, nil
}

// ListAPIKeys returns all active API keys for a user in a specific workspace.
func (s *Service) ListAPIKeys(userID, workspaceID string) ([]entity.APIKey, error) {
	ctx := context.Background()

	keys, err := s.apiKeys.FindByUserIDAndWorkspaceID(ctx, userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("ListAPIKeys: %w", err)
	}
	return keys, nil
}

// RevokeAPIKey soft-deletes an API key if it belongs to the given user and workspace.
func (s *Service) RevokeAPIKey(userID, workspaceID, keyID string) error {
	ctx := context.Background()

	if err := s.apiKeys.SoftDeleteScoped(ctx, userID, workspaceID, keyID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("RevokeAPIKey: %w", err)
	}

	slog.Info("API key revoked", "user_id", userID, "workspace_id", workspaceID, "key_id", keyID)
	return nil
}

// ErrPasswordResetCooldown is returned when a password reset is requested too soon.
var ErrPasswordResetCooldown = errors.New("please wait before requesting another reset email")

// ErrVerificationEmailCooldown is returned when a verification email is requested too soon.
var ErrVerificationEmailCooldown = errors.New("please wait before requesting another verification email")

// ForgotPassword generates a password reset token for the given email address.
// The raw token is returned so the caller can send it via email.
// If the email doesn't exist, returns nil error and empty string (to prevent user enumeration).
func (s *Service) ForgotPassword(email string) (string, error) {
	ctx := context.Background()

	// Rate limit: one reset request per email per cooldown period.
	// Check before user lookup to avoid timing-based enumeration.
	if s.rateLimiter != nil {
		key := "password_reset:" + email
		allowed, err := s.rateLimiter.Allow(ctx, key, PasswordResetCooldown)
		if err != nil {
			slog.Error("rate limiter error during password reset", "email", email, "error", err)
			// Fail open: if Redis is down, allow the request rather than blocking users.
		} else if !allowed {
			slog.Info("password reset rate limited", "email", email)
			return "", ErrPasswordResetCooldown
		}
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			// Don't reveal that the email doesn't exist
			slog.Info("password reset requested for unknown email", "email", email)
			return "", nil
		}
		return "", fmt.Errorf("finding user: %w", err)
	}

	// Clean up old tokens
	_ = s.passwordResets.DeleteExpiredByUserID(ctx, user.ID)

	// Generate a random reset token
	rawToken, err := generateResetToken()
	if err != nil {
		return "", fmt.Errorf("generating reset token: %w", err)
	}

	token := &entity.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hashRefreshToken(rawToken), // SHA-256 hash
		ExpiresAt: time.Now().Add(PasswordResetDuration),
	}

	if err := s.passwordResets.Create(ctx, token); err != nil {
		return "", fmt.Errorf("creating reset token: %w", err)
	}

	// Send password reset email (best-effort: errors logged, not propagated)
	if s.emailSender != nil {
		if err := s.emailSender.SendPasswordResetEmail(ctx, email, rawToken); err != nil {
			slog.Error("failed to send password reset email", "email", email, "error", err)
		}
	} else {
		slog.Warn("email sender not configured - password reset email not sent", "email", email)
	}

	slog.Info("password reset token generated", "user_id", user.ID, "email", email)
	return rawToken, nil
}

// ResetPassword validates a reset token and updates the user's password.
func (s *Service) ResetPassword(rawToken, newPassword string) error {
	ctx := context.Background()

	tokenHash := hashRefreshToken(rawToken)

	stored, err := s.passwordResets.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return ErrResetTokenInvalid
		}
		return fmt.Errorf("finding reset token: %w", err)
	}

	if stored.UsedAt != nil {
		return ErrResetTokenUsed
	}

	if time.Now().After(stored.ExpiresAt) {
		return ErrResetTokenInvalid
	}

	// Hash the new password
	if err := entity.ValidatePassword(newPassword); err != nil {
		return err
	}

	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	// Update the user's password
	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		return fmt.Errorf("finding user: %w", err)
	}

	user.PasswordHash = passwordHash
	// Completing a password reset proves email ownership - if the user hasn't
	// verified yet, mark them verified now so they can log in immediately.
	if !user.EmailVerified {
		user.EmailVerified = true
		slog.Info("email auto-verified via password reset", "user_id", user.ID)
	}
	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("updating password: %w", err)
	}

	// Mark the token as used
	if err := s.passwordResets.MarkUsed(ctx, stored.ID); err != nil {
		slog.Error("failed to mark reset token as used", "token_id", stored.ID, "error", err)
	}

	// Invalidate all existing sessions and refresh tokens for the user
	// so any stolen tokens are no longer valid after password reset.
	if err := s.refreshTokens.DeleteByUserID(ctx, stored.UserID); err != nil {
		slog.Error("failed to invalidate refresh tokens after password reset", "user_id", stored.UserID, "error", err)
	}
	if err := s.sessions.DeleteByUserID(ctx, stored.UserID); err != nil {
		slog.Error("failed to invalidate sessions after password reset", "user_id", stored.UserID, "error", err)
	}

	slog.Info("password reset successful", "user_id", stored.UserID)
	return nil
}

// GenerateEmailVerificationToken creates a verification token for the given user.
// Returns the raw token to be sent via email.
func (s *Service) GenerateEmailVerificationToken(userID string) (string, error) {
	ctx := context.Background()

	// Rate limit: one verification email per user per cooldown period.
	if s.rateLimiter != nil {
		key := fmt.Sprintf("verify_email:%s", userID)
		allowed, err := s.rateLimiter.Allow(ctx, key, VerificationEmailCooldown)
		if err != nil {
			slog.Error("rate limiter error during email verification", "user_id", userID, "error", err)
			// Fail open: if Redis is down, allow the request rather than blocking users.
		} else if !allowed {
			slog.Info("verification email rate limited", "user_id", userID)
			return "", ErrVerificationEmailCooldown
		}
	}

	// Clean up old tokens for this user
	_ = s.emailVerifications.DeleteByUserID(ctx, userID)

	rawToken, err := generateResetToken()
	if err != nil {
		return "", fmt.Errorf("generating verification token: %w", err)
	}

	token := &entity.EmailVerificationToken{
		UserID:    userID,
		TokenHash: hashRefreshToken(rawToken),
		ExpiresAt: time.Now().Add(EmailVerificationDuration),
	}

	if err := s.emailVerifications.Create(ctx, token); err != nil {
		return "", fmt.Errorf("creating verification token: %w", err)
	}

	// Send verification email (best-effort: errors logged, not propagated)
	if s.emailSender != nil {
		user, err := s.users.FindByID(ctx, userID)
		if err != nil {
			slog.Error("failed to find user for verification email", "user_id", userID, "error", err)
		} else {
			if err := s.emailSender.SendVerificationEmail(ctx, user.Email, rawToken); err != nil {
				slog.Error("failed to send verification email", "user_id", userID, "error", err)
			}
		}
	} else {
		slog.Warn("email sender not configured - verification email not sent", "user_id", userID)
	}

	slog.Info("email verification token generated", "user_id", userID)
	return rawToken, nil
}

// VerifyEmail validates a verification token and marks the user's email as verified.
func (s *Service) VerifyEmail(rawToken string) error {
	ctx := context.Background()

	tokenHash := hashRefreshToken(rawToken)

	stored, err := s.emailVerifications.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return ErrVerificationInvalid
		}
		return fmt.Errorf("finding verification token: %w", err)
	}

	if time.Now().After(stored.ExpiresAt) {
		return ErrVerificationInvalid
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		return fmt.Errorf("finding user: %w", err)
	}

	user.EmailVerified = true
	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("updating user email verification: %w", err)
	}

	// Delete the token (one-time use)
	if err := s.emailVerifications.Delete(ctx, stored.ID); err != nil {
		slog.Error("failed to delete verification token", "token_id", stored.ID, "error", err)
	}

	slog.Info("email verified", "user_id", stored.UserID)
	return nil
}

// --- Session Management ---

// CreateSession creates a new session for the user, enforcing a max of MaxSessionsPerUser.
// If the limit is exceeded, the oldest session is deleted.
func (s *Service) CreateSession(userID string, tokenHash, ipAddress, userAgent string) (*entity.Session, error) {
	ctx := context.Background()

	// Enforce max concurrent sessions
	count, err := s.sessions.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("counting sessions: %w", err)
	}

	for count >= MaxSessionsPerUser {
		if err := s.sessions.DeleteOldestByUserID(ctx, userID); err != nil {
			return nil, fmt.Errorf("deleting oldest session: %w", err)
		}
		count--
	}

	// Truncate user agent to 512 chars to match DB column
	if len(userAgent) > 512 {
		userAgent = userAgent[:512]
	}

	session := &entity.Session{
		UserID:     userID,
		TokenHash:  tokenHash,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		LastActive: time.Now(),
		ExpiresAt:  time.Now().Add(SessionDuration),
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	slog.Info("session created", "user_id", userID, "session_id", session.ID)
	return session, nil
}

// ListSessions returns all active sessions for the given user.
func (s *Service) ListSessions(userID string) ([]entity.Session, error) {
	ctx := context.Background()

	sessions, err := s.sessions.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	return sessions, nil
}

// ErrCannotRevokeCurrentSession is returned when trying to delete the current session.
var ErrCannotRevokeCurrentSession = errors.New("cannot revoke the current session")

// GuardCurrentSession checks whether the given session ID corresponds to the
// caller's current session (identified by currentTokenHash). Returns an error
// if it is, preventing the caller from accidentally logging themselves out.
func (s *Service) GuardCurrentSession(userID, sessionID string, currentTokenHash string) error {
	ctx := context.Background()

	session, err := s.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return nil // Session not found - let RevokeSession handle the 404
	}

	if session.UserID != userID {
		return nil // Not the user's session - let RevokeSession handle the 404
	}

	if session.TokenHash == currentTokenHash {
		return ErrCannotRevokeCurrentSession
	}

	return nil
}

// RevokeSession deletes a specific session, verifying it belongs to the user.
func (s *Service) RevokeSession(userID, sessionID string) error {
	ctx := context.Background()

	session, err := s.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	if session.UserID != userID {
		return ErrSessionNotFound
	}

	if err := s.sessions.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}

	// Also revoke the associated refresh token so the user can't get new access tokens
	if session.TokenHash != "" {
		if err := s.refreshTokens.DeleteByTokenHash(ctx, session.TokenHash); err != nil {
			slog.Warn("failed to delete refresh token for session", "session_id", sessionID, "error", err)
			// Non-fatal - session is already deleted
		}
	}

	slog.Info("session revoked", "user_id", userID, "session_id", sessionID)
	return nil
}

// CleanExpiredSessions removes all expired sessions from the database.
func (s *Service) CleanExpiredSessions() (int64, error) {
	ctx := context.Background()

	count, err := s.sessions.DeleteExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("cleaning expired sessions: %w", err)
	}
	if count > 0 {
		slog.Info("cleaned expired sessions", "count", count)
	}
	return count, nil
}

// UpdateProfile updates the user's first and last name.
func (s *Service) UpdateProfile(userID string, firstName, lastName string) (*entity.User, error) {
	ctx := context.Background()

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}

	user.FirstName = firstName
	user.LastName = lastName

	if err := s.users.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("updating user profile: %w", err)
	}

	slog.Info("user profile updated", "user_id", userID)
	return user, nil
}

// ErrInvalidPassword is returned when the current password does not match.
var ErrInvalidPassword = errors.New("current password is incorrect")

// ChangePassword validates the current password and updates to a new one.
// currentTokenHash identifies the caller's active session so it is preserved;
// all other sessions and refresh tokens are invalidated. Pass "" to invalidate all.
func (s *Service) ChangePassword(userID string, currentPassword, newPassword, currentTokenHash string) error {
	ctx := context.Background()

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("finding user: %w", err)
	}

	if !checkPassword(currentPassword, user.PasswordHash) {
		return ErrInvalidPassword
	}

	if err := entity.ValidatePassword(newPassword); err != nil {
		return err
	}

	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hashing new password: %w", err)
	}

	user.PasswordHash = passwordHash
	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("updating password: %w", err)
	}

	// Invalidate all sessions and refresh tokens except the current one
	// so any stolen tokens are no longer valid after password change.
	if currentTokenHash != "" {
		if err := s.refreshTokens.DeleteByUserIDExceptTokenHash(ctx, userID, currentTokenHash); err != nil {
			slog.Error("failed to invalidate refresh tokens after password change", "user_id", userID, "error", err)
		}
		if err := s.sessions.DeleteByUserIDExceptTokenHash(ctx, userID, currentTokenHash); err != nil {
			slog.Error("failed to invalidate sessions after password change", "user_id", userID, "error", err)
		}
	} else {
		if err := s.refreshTokens.DeleteByUserID(ctx, userID); err != nil {
			slog.Error("failed to invalidate refresh tokens after password change", "user_id", userID, "error", err)
		}
		if err := s.sessions.DeleteByUserID(ctx, userID); err != nil {
			slog.Error("failed to invalidate sessions after password change", "user_id", userID, "error", err)
		}
	}

	slog.Info("user password changed", "user_id", userID)
	return nil
}

// generateResetToken creates a cryptographically secure random token for password reset.
func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
