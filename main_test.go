package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestInstallationToken_Expired(t *testing.T) {
	now := time.Now()

	// Token expiring in 30 seconds (within 1 minute leeway) -> should be expired
	t1 := &InstallationToken{
		Token:     "token1",
		ExpiresAt: now.Add(30 * time.Second),
	}
	if !t1.Expired() {
		t.Error("Expected token expiring in 30s to be expired due to leeway")
	}

	// Token expiring in 2 minutes (outside 1 minute leeway) -> should not be expired
	t2 := &InstallationToken{
		Token:     "token2",
		ExpiresAt: now.Add(2 * time.Minute),
	}
	if t2.Expired() {
		t.Error("Expected token expiring in 2m to not be expired")
	}
}

func TestTokenProvider_ThreadSafetyAndCaching(t *testing.T) {
	var fetchCount int32
	fetchFunc := func() (*InstallationToken, error) {
		atomic.AddInt32(&fetchCount, 1)
		return &InstallationToken{
			Token:     "token",
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}, nil
	}

	provider := NewTokenProvider(fetchFunc)

	// Concurrently get token
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tok, err := provider.GetToken()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tok != "token" {
				t.Errorf("Expected token, got %s", tok)
			}
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&fetchCount) != 1 {
		t.Errorf("Expected token to be fetched exactly once, got %d", fetchCount)
	}
}

func TestTokenProvider_RefreshOnExpiration(t *testing.T) {
	var fetchCount int32
	// First token expires in 30 seconds (within leeway)
	// Second token expires in 5 minutes
	fetchFunc := func() (*InstallationToken, error) {
		count := atomic.AddInt32(&fetchCount, 1)
		if count == 1 {
			return &InstallationToken{
				Token:     "token1",
				ExpiresAt: time.Now().Add(30 * time.Second),
			}, nil
		}
		return &InstallationToken{
			Token:     "token2",
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}, nil
	}

	provider := NewTokenProvider(fetchFunc)

	tok1, err := provider.GetToken()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tok1 != "token1" {
		t.Errorf("Expected token1, got %s", tok1)
	}

	tok2, err := provider.GetToken()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tok2 != "token2" {
		t.Errorf("Expected token2, got %s", tok2)
	}

	tok3, err := provider.GetToken()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tok3 != "token2" {
		t.Errorf("Expected token2, got %s", tok3)
	}

	if atomic.LoadInt32(&fetchCount) != 2 {
		t.Errorf("Expected token to be fetched twice, got %d", fetchCount)
	}
}
