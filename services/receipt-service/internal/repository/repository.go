package repository

import (
	"errors"
	"sync"

	"github.com/leoemaxie/trustpass/services/shared"
)

var (
	ErrNotFound = errors.New("receipt not found")
)

type ReceiptStore interface {
	Save(receipt *shared.VerificationReceipt) error
	GetByID(id string) (*shared.VerificationReceipt, error)
	List(verifierID string) ([]shared.VerificationReceipt, error)
}

type MemoryReceiptStore struct {
	mu       sync.RWMutex
	receipts map[string]shared.VerificationReceipt
}

func NewMemoryReceiptStore() *MemoryReceiptStore {
	return &MemoryReceiptStore{
		receipts: make(map[string]shared.VerificationReceipt),
	}
}

func (s *MemoryReceiptStore) Save(receipt *shared.VerificationReceipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receipts[receipt.ID] = *receipt
	return nil
}

func (s *MemoryReceiptStore) GetByID(id string) (*shared.VerificationReceipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	receipt, exists := s.receipts[id]
	if !exists {
		return nil, ErrNotFound
	}
	return &receipt, nil
}

func (s *MemoryReceiptStore) List(verifierID string) ([]shared.VerificationReceipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []shared.VerificationReceipt
	for _, r := range s.receipts {
		if verifierID == "" || r.VerifierID == verifierID {
			results = append(results, r)
		}
	}
	return results, nil
}
