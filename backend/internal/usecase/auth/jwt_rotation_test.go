package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

// signToken creates a JWT signed with the given secret for testing.
func signToken(t *testing.T, secret string, userID string, email string) string {
	t.Helper()
	claims := &auth.Claims{
		UserID:    userID,
		Email:     email,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "veilence-mx",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}

func TestValidateAccessToken_PrimarySecret(t *testing.T) {
	svc := auth.NewService(nil, nil, nil, nil, nil, nil, nil, false, nil, "primary-secret")

	tokenStr := signToken(t, "primary-secret", "01935d5a-0000-7000-8000-000000000001", "test@example.com")
	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "01935d5a-0000-7000-8000-000000000001", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
}

func TestValidateAccessToken_PreviousSecret_Accepted(t *testing.T) {
	svc := auth.NewService(nil, nil, nil, nil, nil, nil, nil, false, nil,
		"new-secret",
		"old-secret-1",
	)

	// Token signed with old secret should still be valid
	tokenStr := signToken(t, "old-secret-1", "01935d5a-0000-7000-8000-000000000002", "old@example.com")
	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "01935d5a-0000-7000-8000-000000000002", claims.UserID)
}

func TestValidateAccessToken_MultiplePreviousSecrets(t *testing.T) {
	svc := auth.NewService(nil, nil, nil, nil, nil, nil, nil, false, nil,
		"current-secret",
		"prev-secret-1",
		"prev-secret-2",
	)

	// Token signed with second previous secret should work
	tokenStr := signToken(t, "prev-secret-2", "01935d5a-0000-7000-8000-000000000003", "user@example.com")
	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "01935d5a-0000-7000-8000-000000000003", claims.UserID)
}

func TestValidateAccessToken_UnknownSecret_Rejected(t *testing.T) {
	svc := auth.NewService(nil, nil, nil, nil, nil, nil, nil, false, nil,
		"current-secret",
		"prev-secret-1",
	)

	// Token signed with an unknown secret should be rejected
	tokenStr := signToken(t, "unknown-secret", "01935d5a-0000-7000-8000-000000000004", "hacker@example.com")
	_, err := svc.ValidateAccessToken(tokenStr)
	assert.Error(t, err)
}

func TestValidateAccessToken_ExpiredToken_Rejected(t *testing.T) {
	svc := auth.NewService(nil, nil, nil, nil, nil, nil, nil, false, nil, "test-secret")

	claims := &auth.Claims{
		UserID:    "01935d5a-0000-7000-8000-000000000001",
		Email:     "test@example.com",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "veilence-mx",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken(tokenStr)
	assert.Error(t, err)
}

func TestValidateAccessToken_NoPreviousSecrets(t *testing.T) {
	// Service created without previous secrets should work normally
	svc := auth.NewService(nil, nil, nil, nil, nil, nil, nil, false, nil, "only-secret")

	tokenStr := signToken(t, "only-secret", "01935d5a-0000-7000-8000-000000000001", "test@example.com")
	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "01935d5a-0000-7000-8000-000000000001", claims.UserID)

	// Wrong secret should fail
	tokenStr = signToken(t, "wrong-secret", "01935d5a-0000-7000-8000-000000000001", "test@example.com")
	_, err = svc.ValidateAccessToken(tokenStr)
	assert.Error(t, err)
}

func TestValidateAccessToken_PrimarySecretPreferred(t *testing.T) {
	// Ensure primary secret is tried first (performance)
	svc := auth.NewService(nil, nil, nil, nil, nil, nil, nil, false, nil,
		"primary",
		"old1", "old2", "old3",
	)

	// Token signed with primary should validate immediately
	tokenStr := signToken(t, "primary", "01935d5a-0000-7000-8000-000000000001", "test@example.com")
	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "01935d5a-0000-7000-8000-000000000001", claims.UserID)
}
