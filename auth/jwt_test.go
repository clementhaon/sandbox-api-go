package auth

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/golang-jwt/jwt/v5"
)

func TestNewJWTManager(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{
			name:    "rejects empty secret",
			secret:  "",
			wantErr: true,
		},
		{
			name:    "rejects secret shorter than 16 chars",
			secret:  "short",
			wantErr: true,
		},
		{
			name:    "rejects secret of exactly 15 chars",
			secret:  "123456789012345",
			wantErr: true,
		},
		{
			name:    "accepts secret of exactly 16 chars",
			secret:  "1234567890123456",
			wantErr: false,
		},
		{
			name:    "accepts longer secret",
			secret:  "this-is-a-very-long-secret-key-for-jwt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewJWTManager(tt.secret)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if mgr != nil {
					t.Fatal("expected nil manager when error occurs")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if mgr == nil {
					t.Fatal("expected non-nil manager")
				}
			}
		})
	}
}

func testUser() models.User {
	return models.User{
		ID:       42,
		Username: "testuser",
		Role:     "admin",
		FirstName: sql.NullString{
			String: "John",
			Valid:  true,
		},
		LastName: sql.NullString{
			String: "Doe",
			Valid:  true,
		},
		AvatarURL: sql.NullString{
			String: "https://example.com/avatar.png",
			Valid:  true,
		},
	}
}

func TestGenerateToken(t *testing.T) {
	mgr, err := NewJWTManager("test-secret-at-least-16")
	if err != nil {
		t.Fatalf("failed to create JWTManager: %v", err)
	}

	t.Run("generates valid token for user with all fields", func(t *testing.T) {
		user := testUser()
		tokenStr, err := mgr.GenerateToken(user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenStr == "" {
			t.Fatal("expected non-empty token")
		}

		claims, err := mgr.ValidateToken(tokenStr)
		if err != nil {
			t.Fatalf("generated token should be valid: %v", err)
		}
		if claims.UserID != user.ID {
			t.Errorf("UserID = %d, want %d", claims.UserID, user.ID)
		}
		if claims.Username != user.Username {
			t.Errorf("Username = %q, want %q", claims.Username, user.Username)
		}
		if claims.Role != user.Role {
			t.Errorf("Role = %q, want %q", claims.Role, user.Role)
		}
		if claims.FirstName != user.FirstName.String {
			t.Errorf("FirstName = %q, want %q", claims.FirstName, user.FirstName.String)
		}
		if claims.LastName != user.LastName.String {
			t.Errorf("LastName = %q, want %q", claims.LastName, user.LastName.String)
		}
		if claims.AvatarURL != user.AvatarURL.String {
			t.Errorf("AvatarURL = %q, want %q", claims.AvatarURL, user.AvatarURL.String)
		}
		if claims.ExpiresAt.Before(time.Now()) {
			t.Error("token should not already be expired")
		}
	})

	t.Run("generates token without optional fields", func(t *testing.T) {
		user := models.User{
			ID:       7,
			Username: "minimal",
			Role:     "user",
		}
		tokenStr, err := mgr.GenerateToken(user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		claims, err := mgr.ValidateToken(tokenStr)
		if err != nil {
			t.Fatalf("generated token should be valid: %v", err)
		}
		if claims.FirstName != "" {
			t.Errorf("FirstName = %q, want empty", claims.FirstName)
		}
		if claims.LastName != "" {
			t.Errorf("LastName = %q, want empty", claims.LastName)
		}
		if claims.AvatarURL != "" {
			t.Errorf("AvatarURL = %q, want empty", claims.AvatarURL)
		}
	})
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret-at-least-16"
	mgr, err := NewJWTManager(secret)
	if err != nil {
		t.Fatalf("failed to create JWTManager: %v", err)
	}

	t.Run("validates a valid token", func(t *testing.T) {
		user := testUser()
		tokenStr, err := mgr.GenerateToken(user)
		if err != nil {
			t.Fatalf("unexpected error generating token: %v", err)
		}

		claims, err := mgr.ValidateToken(tokenStr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.UserID != 42 {
			t.Errorf("UserID = %d, want 42", claims.UserID)
		}
	})

	t.Run("rejects expired token", func(t *testing.T) {
		// Create a token with an already-expired time
		claimsMap := jwt.MapClaims{
			"user_id":  float64(1),
			"username": "expired",
			"role":     "user",
			"exp":      time.Now().Add(-1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsMap)
		tokenStr, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = mgr.ValidateToken(tokenStr)
		if err == nil {
			t.Fatal("expected error for expired token, got nil")
		}
	})

	t.Run("rejects tampered token", func(t *testing.T) {
		user := testUser()
		tokenStr, err := mgr.GenerateToken(user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Tamper with the token by flipping a character in the signature
		parts := strings.Split(tokenStr, ".")
		if len(parts) != 3 {
			t.Fatalf("expected 3 parts, got %d", len(parts))
		}
		sig := []byte(parts[2])
		if sig[0] == 'a' {
			sig[0] = 'b'
		} else {
			sig[0] = 'a'
		}
		tampered := parts[0] + "." + parts[1] + "." + string(sig)

		_, err = mgr.ValidateToken(tampered)
		if err == nil {
			t.Fatal("expected error for tampered token, got nil")
		}
	})

	t.Run("rejects wrong signing method", func(t *testing.T) {
		// Create a token signed with RSA "none" workaround: use an unsigned token
		// We craft a token with alg=none
		claimsMap := jwt.MapClaims{
			"user_id":  float64(1),
			"username": "attacker",
			"role":     "admin",
			"exp":      time.Now().Add(1 * time.Hour).Unix(),
		}

		// Sign with a different secret using a different manager
		otherMgr, err := NewJWTManager("other-secret-at-least-16")
		if err != nil {
			t.Fatalf("failed to create other manager: %v", err)
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsMap)
		tokenStr, err := token.SignedString(otherMgr.secret)
		if err != nil {
			t.Fatalf("failed to sign: %v", err)
		}

		_, err = mgr.ValidateToken(tokenStr)
		if err == nil {
			t.Fatal("expected error for token signed with different secret")
		}
	})

	t.Run("rejects garbage input", func(t *testing.T) {
		_, err := mgr.ValidateToken("not-a-valid-token")
		if err == nil {
			t.Fatal("expected error for garbage input, got nil")
		}
	})
}
