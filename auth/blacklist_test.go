package auth

import (
	"testing"
	"time"
)

func TestTokenBlacklist(t *testing.T) {
	t.Run("Add and IsBlacklisted returns true", func(t *testing.T) {
		bl := NewTokenBlacklist()
		defer bl.Stop()

		bl.Add("token-abc", time.Now().Add(1*time.Hour))

		if !bl.IsBlacklisted("token-abc") {
			t.Error("expected token to be blacklisted")
		}
	})

	t.Run("non-blacklisted token returns false", func(t *testing.T) {
		bl := NewTokenBlacklist()
		defer bl.Stop()

		if bl.IsBlacklisted("never-added") {
			t.Error("expected token to not be blacklisted")
		}
	})

	t.Run("multiple tokens can be blacklisted", func(t *testing.T) {
		bl := NewTokenBlacklist()
		defer bl.Stop()

		tokens := []string{"token-1", "token-2", "token-3"}
		for _, tok := range tokens {
			bl.Add(tok, time.Now().Add(1*time.Hour))
		}

		for _, tok := range tokens {
			if !bl.IsBlacklisted(tok) {
				t.Errorf("expected %q to be blacklisted", tok)
			}
		}

		if bl.IsBlacklisted("token-4") {
			t.Error("expected token-4 to not be blacklisted")
		}
	})

	t.Run("Stop does not panic", func(t *testing.T) {
		bl := NewTokenBlacklist()
		// Should not panic
		bl.Stop()
	})
}
