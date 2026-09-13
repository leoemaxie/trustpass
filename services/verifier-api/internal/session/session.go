package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/leoemaxie/trustpass/services/shared"
)

var (
	ErrSessionNotFound = errors.New("SessionTokenExpired")
	ErrSessionExpired  = errors.New("SessionTokenExpired")
	ErrSessionReused   = errors.New("SessionTokenReused")
)

type VerificationSession struct {
	SessionToken string              `json:"sessionToken"`
	ClaimRequest shared.ClaimRequest `json:"claimRequest"`
	CreatedAt    time.Time           `json:"createdAt"`
	ExpiresAt    time.Time           `json:"expiresAt"`
	Consumed     bool                `json:"consumed"`
}

type Store interface {
	CreateSession(claim shared.ClaimRequest, ttl time.Duration) (*VerificationSession, error)
	Get(token string) (*VerificationSession, error)
	MarkConsumed(token string) error
	Consume(token string) (*VerificationSession, error)
}

type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*VerificationSession
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*VerificationSession),
	}
}

func GenerateSecureToken() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *MemoryStore) CreateSession(claim shared.ClaimRequest, ttl time.Duration) (*VerificationSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := GenerateSecureToken()
	now := time.Now().UTC()
	sess := &VerificationSession{
		SessionToken: token,
		ClaimRequest: claim,
		CreatedAt:    now,
		ExpiresAt:    now.Add(ttl),
		Consumed:     false,
	}

	s.sessions[token] = sess
	return sess, nil
}

func (s *MemoryStore) Get(token string) (*VerificationSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, exists := s.sessions[token]
	if !exists {
		return nil, ErrSessionNotFound
	}

	if time.Now().UTC().After(sess.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	if sess.Consumed {
		return nil, ErrSessionReused
	}

	return sess, nil
}

func (s *MemoryStore) MarkConsumed(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, exists := s.sessions[token]
	if !exists {
		return ErrSessionNotFound
	}
	sess.Consumed = true
	return nil
}

// Consume atomically verifies that the session token exists, is unexpired, and is unconsumed,
// and marks it as consumed in a single atomic transaction. This prevents concurrent replay race conditions.
func (s *MemoryStore) Consume(token string) (*VerificationSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, exists := s.sessions[token]
	if !exists {
		return nil, ErrSessionNotFound
	}

	if time.Now().UTC().After(sess.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	if sess.Consumed {
		return nil, ErrSessionReused
	}

	// Atomically mark consumed
	sess.Consumed = true
	return sess, nil
}
