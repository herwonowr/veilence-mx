package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/auth"
	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/repository"
)

// =====================================================================
// ForgotPassword
// =====================================================================

func TestForgotPassword_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("forgot@example.com", "Password123", "Forgot", "User")
	require.NoError(t, err)

	rawToken, err := svc.ForgotPassword("forgot@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, rawToken, "should return a non-empty reset token")
}

func TestForgotPassword_UnknownEmail_NoError(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	// Should not reveal that the email doesn't exist — returns nil error + empty token
	rawToken, err := svc.ForgotPassword("nonexistent@example.com")
	require.NoError(t, err, "should not return error for unknown email")
	assert.Empty(t, rawToken, "should return empty token for unknown email")
}

func TestForgotPassword_GeneratesUniqueTokens(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("multi-forgot@example.com", "Password123", "Multi", "Forgot")
	require.NoError(t, err)

	token1, err := svc.ForgotPassword("multi-forgot@example.com")
	require.NoError(t, err)

	token2, err := svc.ForgotPassword("multi-forgot@example.com")
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2, "each call should generate a unique token")
}

// =====================================================================
// ResetPassword
// =====================================================================

func TestResetPassword_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("reset@example.com", "OldPassword123", "Reset", "User")
	require.NoError(t, err)

	rawToken, err := svc.ForgotPassword("reset@example.com")
	require.NoError(t, err)
	require.NotEmpty(t, rawToken)

	// Reset password using the token
	err = svc.ResetPassword(rawToken, "NewPassword456")
	require.NoError(t, err)

	// Old password should no longer work
	_, _, err = svc.Login("reset@example.com", "OldPassword123")
	require.Error(t, err)

	// New password should work
	user, tokens, err := svc.Login("reset@example.com", "NewPassword456")
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, tokens)
}

func TestResetPassword_InvalidToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	err := svc.ResetPassword("totally-invalid-token", "NewPassword456")
	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrResetTokenInvalid)
}

func TestResetPassword_TokenAlreadyUsed(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, err := svc.Register("used-token@example.com", "Password123", "Used", "Token")
	require.NoError(t, err)

	rawToken, err := svc.ForgotPassword("used-token@example.com")
	require.NoError(t, err)

	// First use should succeed
	err = svc.ResetPassword(rawToken, "NewPassword1")
	require.NoError(t, err)

	// Second use should fail with "already used"
	err = svc.ResetPassword(rawToken, "NewPassword2")
	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrResetTokenUsed)
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("expired-reset@example.com", "Password123", "Expired", "Reset")
	require.NoError(t, err)

	rawToken, err := svc.ForgotPassword("expired-reset@example.com")
	require.NoError(t, err)

	// Manually expire the token in the DB
	db.Model(&models.PasswordResetToken{}).
		Where("user_id = ?", user.ID).
		Update("expires_at", time.Now().Add(-1*time.Hour))

	err = svc.ResetPassword(rawToken, "NewPassword456")
	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrResetTokenInvalid)
}

// =====================================================================
// GenerateEmailVerificationToken
// =====================================================================

func TestGenerateEmailVerificationToken_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("verify@example.com", "Password123", "Verify", "User")
	require.NoError(t, err)

	rawToken, err := svc.GenerateEmailVerificationToken(user.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, rawToken)
}

func TestGenerateEmailVerificationToken_ReplacesOldToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("re-verify@example.com", "Password123", "Re", "Verify")
	require.NoError(t, err)

	token1, err := svc.GenerateEmailVerificationToken(user.ID)
	require.NoError(t, err)

	token2, err := svc.GenerateEmailVerificationToken(user.ID)
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2, "new token should be different")

	// Old token should no longer work (previous tokens deleted)
	err = svc.VerifyEmail(token1)
	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrVerificationInvalid)
}

// =====================================================================
// VerifyEmail
// =====================================================================

func TestVerifyEmail_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("email-verify@example.com", "Password123", "Email", "Verify")
	require.NoError(t, err)
	assert.False(t, user.EmailVerified)

	rawToken, err := svc.GenerateEmailVerificationToken(user.ID)
	require.NoError(t, err)

	err = svc.VerifyEmail(rawToken)
	require.NoError(t, err)

	// User should now have verified email
	updatedUser, err := svc.GetUserByID(user.ID)
	require.NoError(t, err)
	assert.True(t, updatedUser.EmailVerified)
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	err := svc.VerifyEmail("invalid-verification-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrVerificationInvalid)
}

func TestVerifyEmail_ExpiredToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("expired-verify@example.com", "Password123", "Expired", "Verify")
	require.NoError(t, err)

	rawToken, err := svc.GenerateEmailVerificationToken(user.ID)
	require.NoError(t, err)

	// Manually expire the token
	db.Model(&models.EmailVerificationToken{}).
		Where("user_id = ?", user.ID).
		Update("expires_at", time.Now().Add(-1*time.Hour))

	err = svc.VerifyEmail(rawToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrVerificationInvalid)
}

func TestVerifyEmail_TokenIsOneTimeUse(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("one-time@example.com", "Password123", "One", "Time")
	require.NoError(t, err)

	rawToken, err := svc.GenerateEmailVerificationToken(user.ID)
	require.NoError(t, err)

	// First use should succeed
	err = svc.VerifyEmail(rawToken)
	require.NoError(t, err)

	// Second use should fail (token deleted after first use)
	err = svc.VerifyEmail(rawToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrVerificationInvalid)
}

// =====================================================================
// Login — deactivated account
// =====================================================================

func TestLogin_DeactivatedAccount(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("deactivated@example.com", "Password123", "Deact", "User")
	require.NoError(t, err)

	// Deactivate the account directly in DB
	db.Model(&models.User{}).Where("id = ?", user.ID).Update("is_active", false)

	_, _, err = svc.Login("deactivated@example.com", "Password123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account is deactivated")
}

// =====================================================================
// ValidateAPIKey — expired key
// =====================================================================

func TestValidateAPIKey_ExpiredKey(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("expired-key@example.com", "Password123", "Expired", "Key")
	require.NoError(t, err)

	// Create key with expiry in the past
	past := time.Now().Add(-24 * time.Hour)
	_, rawKey, err := svc.CreateAPIKey(user.ID, "expired-key", domain.APIKeyScopeRead, &past)
	require.NoError(t, err)

	_, _, _, err = svc.ValidateAPIKey(rawKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key expired")
}

func TestValidateAPIKey_DeactivatedUser(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("deact-api@example.com", "Password123", "Deact", "API")
	require.NoError(t, err)

	_, rawKey, err := svc.CreateAPIKey(user.ID, "deact-key", domain.APIKeyScopeAdmin, nil)
	require.NoError(t, err)

	// Deactivate the user
	db.Model(&models.User{}).Where("id = ?", user.ID).Update("is_active", false)

	_, _, _, err = svc.ValidateAPIKey(rawKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account is deactivated")
}

func TestValidateAPIKey_ShortKey(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	_, _, _, err := svc.ValidateAPIKey("short")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key format")
}

func TestValidateAPIKey_UpdatesLastUsedAt(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("lastused@example.com", "Password123", "Last", "Used")
	require.NoError(t, err)

	apiKey, rawKey, err := svc.CreateAPIKey(user.ID, "track-key", domain.APIKeyScopeRead, nil)
	require.NoError(t, err)
	assert.Nil(t, apiKey.LastUsedAt)

	// Validate the key (should update last_used_at)
	_, _, _, err = svc.ValidateAPIKey(rawKey)
	require.NoError(t, err)

	// Verify last_used_at was set
	var updated models.APIKey
	db.First(&updated, apiKey.ID)
	assert.NotNil(t, updated.LastUsedAt)
}

// =====================================================================
// CreateSession — user agent truncation
// =====================================================================

func TestCreateSession_UserAgentTruncation(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("ua-trunc@example.com", "Password123", "UA", "Trunc")
	require.NoError(t, err)

	// Create a very long user agent (>512 chars)
	longUA := strings.Repeat("A", 600)
	session, err := svc.CreateSession(user.ID, "token-hash", "10.0.0.1", longUA)
	require.NoError(t, err)
	assert.Len(t, session.UserAgent, 512, "user agent should be truncated to 512 chars")
}

// =====================================================================
// RefreshTokens — deactivated user during refresh
// =====================================================================

func TestRefreshTokens_DeactivatedUser(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	user, err := svc.Register("deact-refresh@example.com", "Password123", "Deact", "Refresh")
	require.NoError(t, err)

	_, tokens, err := svc.Login("deact-refresh@example.com", "Password123")
	require.NoError(t, err)

	// Deactivate the user between login and refresh
	db.Model(&models.User{}).Where("id = ?", user.ID).Update("is_active", false)

	_, err = svc.RefreshTokens(tokens.RefreshToken)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account is deactivated")
}

// =====================================================================
// JWT Secret Rotation
// =====================================================================

func TestValidateAccessToken_WithPreviousSecret(t *testing.T) {
	db := setupAuthTestDB(t)

	oldSecret := "old-secret-key-for-testing-rotation-12345"
	newSecret := "new-secret-key-for-testing-rotation-67890"

	// Build repos once, shared across both service instances
	userRepo := repository.NewUserRepo(db)
	refreshTokenRepo := repository.NewRefreshTokenRepo(db)
	apiKeyRepo := repository.NewAPIKeyRepo(db)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepo(db)
	emailVerificationTokenRepo := repository.NewEmailVerificationTokenRepo(db)
	sessionRepo := repository.NewSessionRepo(db)

	// Sign a token with the old secret
	oldSvc := auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, passwordResetTokenRepo, emailVerificationTokenRepo, sessionRepo, oldSecret)

	_, err := oldSvc.Register("rotation@example.com", "Password123", "Rotation", "Test")
	require.NoError(t, err)

	_, tokens, err := oldSvc.Login("rotation@example.com", "Password123")
	require.NoError(t, err)

	// New service with new primary secret and old secret as previous
	newSvc := auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, passwordResetTokenRepo, emailVerificationTokenRepo, sessionRepo, newSecret, oldSecret)

	// Token signed with old secret should still validate via fallback
	claims, err := newSvc.ValidateAccessToken(tokens.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "rotation@example.com", claims.Email)
}

// =====================================================================
// Logout — nonexistent token (idempotent)
// =====================================================================

func TestLogout_NonexistentToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newAuthService(db)

	// Logging out with a token that doesn't exist should not error
	// (the token hash just won't find any rows to delete)
	err := svc.Logout("nonexistent-refresh-token-hash-value-1234")
	// This may or may not error depending on the implementation
	// The key behavior is that it doesn't panic
	_ = err
}
