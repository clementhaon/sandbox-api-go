package auth

import (
	"sync"
	"time"
)

// TokenBlacklist maintains a set of revoked JWT tokens until they expire.
type TokenBlacklist struct {
	mu     sync.RWMutex
	tokens map[string]time.Time // token -> expiry time
	stopCh chan struct{}
}

// NewTokenBlacklist creates a new TokenBlacklist and starts a cleanup goroutine.
func NewTokenBlacklist() *TokenBlacklist {
	bl := &TokenBlacklist{
		tokens: make(map[string]time.Time),
		stopCh: make(chan struct{}),
	}
	go bl.cleanup()
	return bl
}

// Add adds a token to the blacklist with its expiry time.
func (bl *TokenBlacklist) Add(token string, expiresAt time.Time) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.tokens[token] = expiresAt
}

// IsBlacklisted returns true if the token has been revoked.
func (bl *TokenBlacklist) IsBlacklisted(token string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	_, exists := bl.tokens[token]
	return exists
}

// Stop terminates the cleanup goroutine.
func (bl *TokenBlacklist) Stop() {
	close(bl.stopCh)
}

func (bl *TokenBlacklist) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			bl.mu.Lock()
			now := time.Now()
			for token, expiresAt := range bl.tokens {
				if now.After(expiresAt) {
					delete(bl.tokens, token)
				}
			}
			bl.mu.Unlock()
		case <-bl.stopCh:
			return
		}
	}
}
