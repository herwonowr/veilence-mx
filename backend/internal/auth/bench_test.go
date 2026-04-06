package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const benchJWTSecret = "bench-jwt-secret-for-testing-only"

func setupBenchDB(b *testing.B) *gorm.DB {
	b.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.APIKey{},
		&models.PasswordResetToken{},
		&models.EmailVerificationToken{},
		&models.Session{},
	); err != nil {
		b.Fatal(err)
	}
	return db
}

func setupBenchService(b *testing.B) (*Service, *gorm.DB) {
	b.Helper()
	db := setupBenchDB(b)
	svc := NewService(
		repository.NewUserRepo(db),
		repository.NewRefreshTokenRepo(db),
		repository.NewAPIKeyRepo(db),
		repository.NewPasswordResetTokenRepo(db),
		repository.NewEmailVerificationTokenRepo(db),
		repository.NewSessionRepo(db),
		benchJWTSecret,
	)
	return svc, db
}

// BenchmarkJWTTokenGeneration measures JWT access token signing throughput.
// Baseline: should complete in <100µs/op on modern hardware.
func BenchmarkJWTTokenGeneration(b *testing.B) {
	secret := []byte(benchJWTSecret)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		claims := &Claims{
			UserID:    1,
			Email:     "bench@example.com",
			TokenType: TokenTypeAccess,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenDuration)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    "veilence-mx",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		_, err := token.SignedString(secret)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJWTTokenValidation measures JWT access token parsing/validation throughput.
// Baseline: should complete in <100µs/op on modern hardware.
func BenchmarkJWTTokenValidation(b *testing.B) {
	svc, _ := setupBenchService(b)

	// Register a user and generate a token
	user, err := svc.Register("bench-val@example.com", "Password123!", "Bench", "User")
	if err != nil {
		b.Fatal(err)
	}

	_, tokens, err := svc.Login("bench-val@example.com", "Password123!")
	if err != nil {
		b.Fatal(err)
	}
	_ = user

	accessToken := tokens.AccessToken

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		claims, err := svc.ValidateAccessToken(accessToken)
		if err != nil {
			b.Fatal(err)
		}
		if claims.UserID == 0 {
			b.Fatal("expected valid user ID")
		}
	}
}

// BenchmarkPasswordHashing measures bcrypt password hashing cost.
// Baseline: ~60-100ms/op at default cost (10). This is intentionally slow.
func BenchmarkPasswordHashing(b *testing.B) {
	password := "BenchmarkPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hashPassword(password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPasswordVerification measures bcrypt password comparison cost.
// Baseline: ~60-100ms/op at default cost (10). This is intentionally slow.
func BenchmarkPasswordVerification(b *testing.B) {
	password := "BenchmarkPassword123!"
	hash, err := hashPassword(password)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ok := checkPassword(password, hash)
		if !ok {
			b.Fatal("expected password match")
		}
	}
}

// BenchmarkAPIKeyHashing measures bcrypt API key hashing cost.
func BenchmarkAPIKeyHashing(b *testing.B) {
	rawKey := "vmx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hashAPIKey(rawKey)
	}
}

// BenchmarkAPIKeyHashCheck measures bcrypt API key verification cost.
func BenchmarkAPIKeyHashCheck(b *testing.B) {
	rawKey := "vmx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checkAPIKeyHash(rawKey, string(hash))
	}
}

// BenchmarkRefreshTokenHashing measures SHA-256 refresh token hashing.
// Baseline: should complete in <1µs/op (SHA-256 is very fast).
func BenchmarkRefreshTokenHashing(b *testing.B) {
	token := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := sha256.Sum256([]byte(token))
		_ = hex.EncodeToString(h[:])
	}
}

// BenchmarkRegisterUser measures the full user registration path.
// Dominated by bcrypt hashing.
func BenchmarkRegisterUser(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		svc, _ := setupBenchService(b)
		b.StartTimer()

		_, err := svc.Register("bench@example.com", "Password123!", "Bench", "User")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoginUser measures the full login path including password verification.
func BenchmarkLoginUser(b *testing.B) {
	svc, _ := setupBenchService(b)
	_, err := svc.Register("bench-login@example.com", "Password123!", "Bench", "User")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, tokens, err := svc.Login("bench-login@example.com", "Password123!")
		if err != nil {
			b.Fatal(err)
		}
		if tokens.AccessToken == "" {
			b.Fatal("expected access token")
		}
	}
}

// BenchmarkCreateSession measures session creation throughput.
func BenchmarkCreateSession(b *testing.B) {
	svc, _ := setupBenchService(b)
	user, err := svc.Register("bench-session@example.com", "Password123!", "Bench", "User")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.CreateSession(user.ID, "token-hash-"+string(rune(i)), "127.0.0.1", "BenchBrowser/1.0")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTokenPairGeneration measures access+refresh token generation together.
func BenchmarkTokenPairGeneration(b *testing.B) {
	svc, _ := setupBenchService(b)
	user, err := svc.Register("bench-pair@example.com", "Password123!", "Bench", "User")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokens, err := svc.generateTokenPair(ctx, &domain.User{
			ID:    user.ID,
			Email: user.Email,
		})
		if err != nil {
			b.Fatal(err)
		}
		if tokens.AccessToken == "" || tokens.RefreshToken == "" {
			b.Fatal("expected non-empty tokens")
		}
	}
}
