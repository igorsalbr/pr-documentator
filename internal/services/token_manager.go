package services

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/igorsal/pr-documentator/internal/interfaces"
	"github.com/igorsal/pr-documentator/internal/models"
)

const (
	TokenLength     = 32
	TokenTTL        = 1 * time.Hour
	CleanupInterval = 10 * time.Minute
)

type TokenManager struct {
	sessions map[string]*models.UserSession
	mu       sync.RWMutex
	logger   interfaces.Logger
	stopCh   chan struct{}
}

func NewTokenManager(logger interfaces.Logger) *TokenManager {
	tm := &TokenManager{
		sessions: make(map[string]*models.UserSession),
		logger:   logger,
		stopCh:   make(chan struct{}),
	}

	// Start cleanup goroutine
	go tm.cleanupExpired()

	return tm
}

func (tm *TokenManager) CreateSession(claudeAPIKey, postmanAPIKey, postmanWorkspaceID, postmanCollectionID string) (string, error) {
	token, err := tm.generateToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	session := &models.UserSession{
		ClaudeAPIKey:        claudeAPIKey,
		PostmanAPIKey:       postmanAPIKey,
		PostmanWorkspaceID:  postmanWorkspaceID,
		PostmanCollectionID: postmanCollectionID,
		CreatedAt:           now,
		ExpiresAt:           now.Add(TokenTTL),
	}

	tm.mu.Lock()
	tm.sessions[token] = session
	tm.mu.Unlock()

	tm.logger.Info("Created new user session", "token", token[:8]+"...", "expires_at", session.ExpiresAt)

	return token, nil
}

func (tm *TokenManager) GetSession(token string) (*models.UserSession, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	session, exists := tm.sessions[token]
	if !exists {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

func (tm *TokenManager) InvalidateSession(token string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.sessions, token)
	tm.logger.Info("Invalidated user session", "token", token[:8]+"...")
}

func (tm *TokenManager) Stop() {
	close(tm.stopCh)
}

func (tm *TokenManager) generateToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (tm *TokenManager) cleanupExpired() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.performCleanup()
		case <-tm.stopCh:
			return
		}
	}
}

func (tm *TokenManager) performCleanup() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	var expiredTokens []string

	for token, session := range tm.sessions {
		if now.After(session.ExpiresAt) {
			expiredTokens = append(expiredTokens, token)
		}
	}

	for _, token := range expiredTokens {
		delete(tm.sessions, token)
	}

	if len(expiredTokens) > 0 {
		tm.logger.Info("Cleaned up expired sessions", "count", len(expiredTokens))
	}
}
