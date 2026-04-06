package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/auth"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/repository"
)

const testJWTSecret = "test-secret-key-for-jwt-signing-1234567890"

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.APIKey{},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func newAuthService(db *gorm.DB) *auth.Service {
	userRepo := repository.NewUserRepo(db)
	refreshTokenRepo := repository.NewRefreshTokenRepo(db)
	apiKeyRepo := repository.NewAPIKeyRepo(db)
	return auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, testJWTSecret)
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

	user, tokens, err := svc.Login("bob@example.com", "Password123")
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

	user, tokens, err := svc.Login("carol@example.com", "WrongPassword1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email or password")
	assert.Nil(t, user)
	assert.Nil(t, tokens)
}

func TestLogin_NonexistentUser(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, tokens, err := svc.Login("nonexistent@example.com", "Password123")
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

	_, tokens, err := svc.Login("dave@example.com", "Password123")
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

	_, tokens, err := svc.Login("eve@example.com", "Password123")
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

	_, tokens, err := svc.Login("frank@example.com", "Password123")
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

	apiKey, rawKey, err := svc.CreateAPIKey(user.ID, "test-key", nil)
	require.NoError(t, err)
	assert.NotZero(t, apiKey.ID)
	assert.Equal(t, "test-key", apiKey.Name)
	assert.NotEmpty(t, rawKey)
	assert.True(t, apiKey.IsActive)

	// Validate the raw key
	userID, email, err := svc.ValidateAPIKey(rawKey)
	require.NoError(t, err)
	assert.Equal(t, user.ID, userID)
	assert.Equal(t, "grace@example.com", email)
}

func TestValidateAPIKey_InvalidKey(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, _, err := svc.ValidateAPIKey("vmx_invalid_key_that_does_not_exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
}

func TestRevokeAPIKey(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("henry@example.com", "Password123", "Henry", "Red")
	require.NoError(t, err)

	apiKey, rawKey, err := svc.CreateAPIKey(user.ID, "to-revoke", nil)
	require.NoError(t, err)

	// Revoke
	err = svc.RevokeAPIKey(user.ID, apiKey.ID)
	require.NoError(t, err)

	// Should no longer validate
	_, _, err = svc.ValidateAPIKey(rawKey)
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

	_, _, err = svc.CreateAPIKey(user.ID, "key-1", nil)
	require.NoError(t, err)
	_, _, err = svc.CreateAPIKey(user.ID, "key-2", nil)
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
	apiKey, _, err := svc.CreateAPIKey(user.ID, "expiring-key", &future)
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
	if err := db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.APIKey{}); err != nil {
		b.Fatal(err)
	}

	svc := func() *auth.Service {
		userRepo := repository.NewUserRepo(db)
		refreshTokenRepo := repository.NewRefreshTokenRepo(db)
		apiKeyRepo := repository.NewAPIKeyRepo(db)
		return auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, testJWTSecret)
	}()

	user, err := svc.Register("bench@example.com", "Password123", "Bench", "User")
	if err != nil {
		b.Fatal(err)
	}

	_, rawKey, err := svc.CreateAPIKey(user.ID, "bench-key", nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uid, email, err := svc.ValidateAPIKey(rawKey)
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

	user := models.User{Email: "perf@example.com", PasswordHash: "hash", FirstName: "Perf", LastName: "Test", IsActive: true}
	require.NoError(t, db.Create(&user).Error)

	// Insert an API key directly to isolate prefix lookup from bcrypt
	apiKeyRepo := repository.NewAPIKeyRepo(db)
	key := models.APIKey{
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

	_, tokens, err := svc.Login("mia@example.com", "Password123")
	require.NoError(t, err)

	err = svc.Logout(tokens.RefreshToken)
	require.NoError(t, err)

	// Refresh should now fail
	_, err = svc.RefreshTokens(tokens.RefreshToken)
	require.Error(t, err)
}
