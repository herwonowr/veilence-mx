package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
)

const testJWTSecret = "test-secret-key-for-jwt-signing-1234567890"

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&persistent.User{},
		&persistent.RefreshToken{},
		&persistent.APIKey{},
		&persistent.PasswordResetToken{},
		&persistent.EmailVerificationToken{},
		&persistent.Session{},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func newAuthService(db *gorm.DB) *auth.Service {
	userRepo := persistent.NewUserRepo(db)
	refreshTokenRepo := persistent.NewRefreshTokenRepo(db)
	apiKeyRepo := persistent.NewAPIKeyRepo(db)
	passwordResetTokenRepo := persistent.NewPasswordResetTokenRepo(db)
	emailVerificationTokenRepo := persistent.NewEmailVerificationTokenRepo(db)
	sessionRepo := persistent.NewSessionRepo(db)
	return auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, passwordResetTokenRepo, emailVerificationTokenRepo, sessionRepo, testJWTSecret)
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("alice@example.com", "Password123", "Alice", "Smith")
	require.NoError(t, err)
	assert.NotZero(t, user.ID)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "Alice", user.FirstName)
	assert.Equal(t, "Smith", user.LastName)
	assert.True(t, user.IsActive)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("alice@example.com", "Password123", "Alice", "Smith")
	require.NoError(t, err)

	_, err = svc.Register("alice@example.com", "Password456", "Alice2", "Jones")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email already registered")
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("bob@example.com", "Password123", "Bob", "Brown")
	require.NoError(t, err)

	user, tokens, err := svc.Login("bob@example.com", "Password123", "127.0.0.1", "TestBrowser/1.0")
	require.NoError(t, err)
	assert.Equal(t, "bob@example.com", user.Email)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("carol@example.com", "Password123", "Carol", "White")
	require.NoError(t, err)

	user, tokens, err := svc.Login("carol@example.com", "WrongPassword1", "127.0.0.1", "TestBrowser/1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email or password")
	assert.Nil(t, user)
	assert.Nil(t, tokens)
}

func TestLogin_NonexistentUser(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, tokens, err := svc.Login("nonexistent@example.com", "Password123", "127.0.0.1", "TestBrowser/1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email or password")
	assert.Nil(t, user)
	assert.Nil(t, tokens)
}

// --- Token Validation ---

func TestValidateAccessToken_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("dave@example.com", "Password123", "Dave", "Green")
	require.NoError(t, err)

	_, tokens, err := svc.Login("dave@example.com", "Password123", "127.0.0.1", "TestBrowser/1.0")
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokens.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "dave@example.com", claims.Email)
	assert.Equal(t, auth.TokenTypeAccess, claims.TokenType)
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.ValidateAccessToken("invalid.token.string")
	require.Error(t, err)
}

// --- Refresh Token Flow ---

func TestRefreshTokens_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("eve@example.com", "Password123", "Eve", "Black")
	require.NoError(t, err)

	_, tokens, err := svc.Login("eve@example.com", "Password123", "127.0.0.1", "TestBrowser/1.0")
	require.NoError(t, err)

	newTokens, err := svc.RefreshTokens(tokens.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	// Old refresh token should be rotated (different)
	assert.NotEqual(t, tokens.RefreshToken, newTokens.RefreshToken)
}

func TestRefreshTokens_InvalidToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.RefreshTokens("nonexistent-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestRefreshTokens_UsedTokenInvalid(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("frank@example.com", "Password123", "Frank", "Grey")
	require.NoError(t, err)

	_, tokens, err := svc.Login("frank@example.com", "Password123", "127.0.0.1", "TestBrowser/1.0")
	require.NoError(t, err)

	// First refresh should succeed
	_, err = svc.RefreshTokens(tokens.RefreshToken)
	require.NoError(t, err)

	// Second use of same refresh token should fail (rotation)
	_, err = svc.RefreshTokens(tokens.RefreshToken)
	require.Error(t, err)
}

// --- API Keys ---

func TestCreateAndValidateAPIKey(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("grace@example.com", "Password123", "Grace", "Blue")
	require.NoError(t, err)

	apiKey, rawKey, err := svc.CreateAPIKey(user.ID, "test-key", entity.APIKeyScopeRead, nil)
	require.NoError(t, err)
	assert.NotZero(t, apiKey.ID)
	assert.Equal(t, "test-key", apiKey.Name)
	assert.NotEmpty(t, rawKey)
	assert.True(t, apiKey.IsActive)

	// Validate the raw key
	userID, email, _, err := svc.ValidateAPIKey(rawKey)
	require.NoError(t, err)
	assert.Equal(t, user.ID, userID)
	assert.Equal(t, "grace@example.com", email)
}

func TestValidateAPIKey_InvalidKey(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, _, _, err := svc.ValidateAPIKey("vmx_invalid_key_that_does_not_exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
}

func TestRevokeAPIKey(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("henry@example.com", "Password123", "Henry", "Red")
	require.NoError(t, err)

	apiKey, rawKey, err := svc.CreateAPIKey(user.ID, "to-revoke", entity.APIKeyScopeRead, nil)
	require.NoError(t, err)

	// Revoke
	err = svc.RevokeAPIKey(user.ID, apiKey.ID)
	require.NoError(t, err)

	// Should no longer validate
	_, _, _, err = svc.ValidateAPIKey(rawKey)
	require.Error(t, err)
}

func TestRevokeAPIKey_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("iris@example.com", "Password123", "Iris", "Pink")
	require.NoError(t, err)

	err = svc.RevokeAPIKey(user.ID, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key not found")
}

func TestListAPIKeys(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("jack@example.com", "Password123", "Jack", "Orange")
	require.NoError(t, err)

	_, _, err = svc.CreateAPIKey(user.ID, "key-1", entity.APIKeyScopeRead, nil)
	require.NoError(t, err)
	_, _, err = svc.CreateAPIKey(user.ID, "key-2", entity.APIKeyScopeWrite, nil)
	require.NoError(t, err)

	keys, err := svc.ListAPIKeys(user.ID)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestCreateAPIKey_WithExpiry(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("kate@example.com", "Password123", "Kate", "Purple")
	require.NoError(t, err)

	future := time.Now().Add(24 * time.Hour)
	apiKey, _, err := svc.CreateAPIKey(user.ID, "expiring-key", entity.APIKeyScopeRead, &future)
	require.NoError(t, err)
	assert.NotNil(t, apiKey.ExpiresAt)
}

func TestGetUserByID(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	registered, err := svc.Register("leo@example.com", "Password123", "Leo", "Cyan")
	require.NoError(t, err)

	user, err := svc.GetUserByID(registered.ID)
	require.NoError(t, err)
	assert.Equal(t, "leo@example.com", user.Email)
}

func TestGetUserByID_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.GetUserByID(99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

// --- Benchmark ---

func BenchmarkValidateAPIKey(b *testing.B) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}
	if err := db.AutoMigrate(&persistent.User{}, &persistent.RefreshToken{}, &persistent.APIKey{}, &persistent.PasswordResetToken{}, &persistent.EmailVerificationToken{}, &persistent.Session{}); err != nil {
		b.Fatal(err)
	}

	svc := func() *auth.Service {
		userRepo := persistent.NewUserRepo(db)
		refreshTokenRepo := persistent.NewRefreshTokenRepo(db)
		apiKeyRepo := persistent.NewAPIKeyRepo(db)
		passwordResetTokenRepo := persistent.NewPasswordResetTokenRepo(db)
		emailVerificationTokenRepo := persistent.NewEmailVerificationTokenRepo(db)
		sessionRepo := persistent.NewSessionRepo(db)
		return auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, passwordResetTokenRepo, emailVerificationTokenRepo, sessionRepo, testJWTSecret)
	}()

	user, err := svc.Register("bench@example.com", "Password123", "Bench", "User")
	if err != nil {
		b.Fatal(err)
	}

	_, rawKey, err := svc.CreateAPIKey(user.ID, "bench-key", entity.APIKeyScopeRead, nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uid, email, _, err := svc.ValidateAPIKey(rawKey)
		if err != nil {
			b.Fatal(err)
		}
		if uid == 0 || email == "" {
			b.Fatal("expected valid response")
		}
	}
}

func TestValidateAPIKey_PrefixLookup_Under10ms(t *testing.T) {
	db := setupAuthTestDB(t)

	user := persistent.User{Email: "perf@example.com", PasswordHash: "hash", FirstName: "Perf", LastName: "Test", IsActive: true}
	require.NoError(t, db.Create(&user).Error)

	// Insert an API key directly to isolate prefix lookup from bcrypt
	apiKeyRepo := persistent.NewAPIKeyRepo(db)
	key := persistent.APIKey{
		UserID:    user.ID,
		Name:      "perf-key",
		KeyHash:   "not-relevant-for-lookup-test",
		KeyPrefix: "vmx_test12",
		IsActive:  true,
	}
	require.NoError(t, db.Create(&key).Error)

	ctx := context.Background()

	// Warm up
	_, _ = apiKeyRepo.FindActiveByPrefix(ctx, "vmx_test12")

	// Time the prefix-based DB lookup (no bcrypt)
	start := time.Now()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		keys, err := apiKeyRepo.FindActiveByPrefix(ctx, "vmx_test12")
		require.NoError(t, err)
		assert.Len(t, keys, 1)
	}
	elapsed := time.Since(start)
	avg := elapsed / time.Duration(iterations)

	t.Logf("FindActiveByPrefix average: %v (total %v over %d iterations)", avg, elapsed, iterations)
	assert.Less(t, avg, 10*time.Millisecond, "prefix-based DB lookup should complete in under 10ms")
}

func TestLogout(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("mia@example.com", "Password123", "Mia", "Teal")
	require.NoError(t, err)

	_, tokens, err := svc.Login("mia@example.com", "Password123", "127.0.0.1", "TestBrowser/1.0")
	require.NoError(t, err)

	err = svc.Logout(tokens.RefreshToken)
	require.NoError(t, err)

	// Refresh should now fail
	_, err = svc.RefreshTokens(tokens.RefreshToken)
	require.Error(t, err)
}

// --- API Key Scoping (S4-8) ---

func TestAPIKeyScope_Validation(t *testing.T) {
	tests := []struct {
		name    string
		scope   string
		isValid bool
	}{
		{"read scope", "read", true},
		{"write scope", "write", true},
		{"admin scope", "admin", true},
		{"empty scope", "", false},
		{"invalid scope", "superadmin", false},
		{"uppercase READ", "READ", false},
		{"mixed case", "Read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, entity.IsValidAPIKeyScope(tt.scope))
		})
	}
}

func TestAPIKeyScope_ScopeAllows(t *testing.T) {
	tests := []struct {
		name    string
		scope   entity.APIKeyScope
		method  string
		allowed bool
	}{
		// read scope
		{"read allows GET", entity.APIKeyScopeRead, "GET", true},
		{"read allows HEAD", entity.APIKeyScopeRead, "HEAD", true},
		{"read allows OPTIONS", entity.APIKeyScopeRead, "OPTIONS", true},
		{"read denies POST", entity.APIKeyScopeRead, "POST", false},
		{"read denies PUT", entity.APIKeyScopeRead, "PUT", false},
		{"read denies PATCH", entity.APIKeyScopeRead, "PATCH", false},
		{"read denies DELETE", entity.APIKeyScopeRead, "DELETE", false},
		// write scope
		{"write allows GET", entity.APIKeyScopeWrite, "GET", true},
		{"write allows POST", entity.APIKeyScopeWrite, "POST", true},
		{"write allows PUT", entity.APIKeyScopeWrite, "PUT", true},
		{"write allows PATCH", entity.APIKeyScopeWrite, "PATCH", true},
		{"write allows HEAD", entity.APIKeyScopeWrite, "HEAD", true},
		{"write allows OPTIONS", entity.APIKeyScopeWrite, "OPTIONS", true},
		{"write denies DELETE", entity.APIKeyScopeWrite, "DELETE", false},
		// admin scope
		{"admin allows GET", entity.APIKeyScopeAdmin, "GET", true},
		{"admin allows POST", entity.APIKeyScopeAdmin, "POST", true},
		{"admin allows DELETE", entity.APIKeyScopeAdmin, "DELETE", true},
		// invalid scope
		{"invalid scope denies GET", entity.APIKeyScope("invalid"), "GET", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.allowed, tt.scope.ScopeAllows(tt.method))
		})
	}
}

func TestCreateAPIKey_WithScope(t *testing.T) {
	tests := []struct {
		name          string
		scope         entity.APIKeyScope
		expectScope   entity.APIKeyScope
		expectError   bool
	}{
		{"read scope", entity.APIKeyScopeRead, entity.APIKeyScopeRead, false},
		{"write scope", entity.APIKeyScopeWrite, entity.APIKeyScopeWrite, false},
		{"admin scope", entity.APIKeyScopeAdmin, entity.APIKeyScopeAdmin, false},
		{"empty defaults to read", "", entity.APIKeyScopeRead, false},
		{"invalid scope", entity.APIKeyScope("superadmin"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupAuthTestDB(t)
			svc := newAuthService(db)

			user, err := svc.Register("scope-test@example.com", "Password123", "Scope", "Test")
			require.NoError(t, err)

			apiKey, _, err := svc.CreateAPIKey(user.ID, "test-key", tt.scope, nil)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectScope, apiKey.Scope)
		})
	}
}

func TestValidateAPIKey_ReturnsScope(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("scope-val@example.com", "Password123", "Scope", "Val")
	require.NoError(t, err)

	_, rawKey, err := svc.CreateAPIKey(user.ID, "write-key", entity.APIKeyScopeWrite, nil)
	require.NoError(t, err)

	userID, email, scope, err := svc.ValidateAPIKey(rawKey)
	require.NoError(t, err)
	assert.Equal(t, user.ID, userID)
	assert.Equal(t, "scope-val@example.com", email)
	assert.Equal(t, entity.APIKeyScopeWrite, scope)
}

// --- Session Management (S4-9) ---

func TestCreateSession_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("session@example.com", "Password123", "Session", "User")
	require.NoError(t, err)

	session, err := svc.CreateSession(user.ID, "token-hash-123", "192.168.1.1", "TestBrowser/1.0")
	require.NoError(t, err)
	assert.NotZero(t, session.ID)
	assert.Equal(t, user.ID, session.UserID)
	assert.Equal(t, "192.168.1.1", session.IPAddress)
	assert.Equal(t, "TestBrowser/1.0", session.UserAgent)
}

func TestListSessions_ReturnsActiveSessions(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("listsess@example.com", "Password123", "List", "Sessions")
	require.NoError(t, err)

	_, err = svc.CreateSession(user.ID, "hash-1", "10.0.0.1", "Browser1")
	require.NoError(t, err)
	_, err = svc.CreateSession(user.ID, "hash-2", "10.0.0.2", "Browser2")
	require.NoError(t, err)

	sessions, err := svc.ListSessions(user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestRevokeSession_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("revoke-sess@example.com", "Password123", "Revoke", "Session")
	require.NoError(t, err)

	session, err := svc.CreateSession(user.ID, "hash-revoke", "10.0.0.1", "Browser1")
	require.NoError(t, err)

	err = svc.RevokeSession(user.ID, session.ID)
	require.NoError(t, err)

	sessions, err := svc.ListSessions(user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestRevokeSession_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("revoke-nf@example.com", "Password123", "Revoke", "NF")
	require.NoError(t, err)

	err = svc.RevokeSession(user.ID, 99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrSessionNotFound)
}

func TestRevokeSession_WrongUser(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user1, err := svc.Register("user1-sess@example.com", "Password123", "User1", "Sess")
	require.NoError(t, err)
	user2, err := svc.Register("user2-sess@example.com", "Password123", "User2", "Sess")
	require.NoError(t, err)

	session, err := svc.CreateSession(user1.ID, "hash-user1", "10.0.0.1", "Browser1")
	require.NoError(t, err)

	// user2 should not be able to revoke user1's session
	err = svc.RevokeSession(user2.ID, session.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrSessionNotFound)
}

func TestCreateSession_EnforcesMaxLimit(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("maxsess@example.com", "Password123", "Max", "Sessions")
	require.NoError(t, err)

	// Create max sessions
	for i := 0; i < auth.MaxSessionsPerUser; i++ {
		_, err := svc.CreateSession(user.ID, fmt.Sprintf("hash-%d", i), "10.0.0.1", "Browser")
		require.NoError(t, err)
	}

	sessions, err := svc.ListSessions(user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, auth.MaxSessionsPerUser)

	// Create one more — should evict oldest
	_, err = svc.CreateSession(user.ID, "hash-overflow", "10.0.0.1", "Browser")
	require.NoError(t, err)

	sessions, err = svc.ListSessions(user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, auth.MaxSessionsPerUser)
}

func TestCleanExpiredSessions(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("clean-sess@example.com", "Password123", "Clean", "Sessions")
	require.NoError(t, err)

	// Create a session that is already expired (via direct DB manipulation)
	sessionRepo := persistent.NewSessionRepo(db)
	err = sessionRepo.Create(context.Background(), &entity.Session{
		UserID:     user.ID,
		TokenHash:  "expired-hash",
		IPAddress:  "10.0.0.1",
		UserAgent:  "Browser",
		LastActive: time.Now().Add(-48 * time.Hour),
		ExpiresAt:  time.Now().Add(-24 * time.Hour), // expired 24h ago
	})
	require.NoError(t, err)

	// Also create an active session
	_, err = svc.CreateSession(user.ID, "active-hash", "10.0.0.2", "Browser2")
	require.NoError(t, err)

	// Clean expired
	count, err := svc.CleanExpiredSessions()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Only active session remains
	sessions, err := svc.ListSessions(user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
}
