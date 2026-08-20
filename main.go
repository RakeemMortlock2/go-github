package main

import (
	"fmt"
	"sync"
	"time"
)

const tokenExpirationLeeway = 1 * time.Minute

// InstallationToken represents a GitHub App installation access token.
type InstallationToken struct {
	Token     string
	ExpiresAt time.Time
}

// Expired checks if the token is expired, taking clock skew leeway into account.
func (t *InstallationToken) Expired() bool {
	if t == nil {
		return true
	}
	return time.Now().Add(tokenExpirationLeeway).After(t.ExpiresAt)
}

// TokenProvider manages the lifecycle of the installation token in a thread-safe manner.
type TokenProvider struct {
	mu    sync.RWMutex
	token *InstallationToken
	// Mock function to simulate fetching a new token from GitHub API
	fetchTokenFunc func() (*InstallationToken, error)
}

// NewTokenProvider creates a new TokenProvider.
func NewTokenProvider(fetchTokenFunc func() (*InstallationToken, error)) *TokenProvider {
	return &TokenProvider{
		fetchTokenFunc: fetchTokenFunc,
	}
}

// GetToken returns a valid token, refreshing it if it is expired or about to expire.
func (p *TokenProvider) GetToken() (string, error) {
	// First, try read lock to see if we have a valid cached token
	p.mu.RLock()
	if p.token != nil && !p.token.Expired() {
		token := p.token.Token
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	// Acquire write lock to refresh the token
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check if another goroutine refreshed it while we were waiting for the lock
	if p.token != nil && !p.token.Expired() {
		return p.token.Token, nil
	}

	// Fetch new token
	newToken, err := p.fetchTokenFunc()
	if err != nil {
		return "", err
	}

	p.token = newToken
	return p.token.Token, nil
}

func main() {
	fmt.Println("Hello, Bounty Hunter!")
}
