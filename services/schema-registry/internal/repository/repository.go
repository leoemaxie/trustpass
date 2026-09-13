package repository

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/leoemaxie/trustpass/services/shared"
)

var (
	ErrNotFound = errors.New("schema not found")
	ErrConflict = errors.New("schema with this name and version already exists")
)

type SchemaRepository interface {
	List() ([]shared.CredentialSchema, error)
	GetByID(id string) (*shared.CredentialSchema, error)
	GetByNameAndVersion(name string, version int) (*shared.CredentialSchema, error)
	Create(schema *shared.CredentialSchema) error
}

type MemorySchemaRepository struct {
	mu      sync.RWMutex
	schemas map[string]shared.CredentialSchema
}

func NewMemorySchemaRepository(seedSchemas []shared.CredentialSchema) *MemorySchemaRepository {
	repo := &MemorySchemaRepository{
		schemas: make(map[string]shared.CredentialSchema),
	}
	for _, s := range seedSchemas {
		repo.schemas[s.ID] = s
	}
	return repo
}

func (r *MemorySchemaRepository) List() ([]shared.CredentialSchema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]shared.CredentialSchema, 0, len(r.schemas))
	for _, s := range r.schemas {
		list = append(list, s)
	}
	return list, nil
}

func (r *MemorySchemaRepository) GetByID(id string) (*shared.CredentialSchema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, exists := r.schemas[id]; exists {
		return &s, nil
	}
	return nil, ErrNotFound
}

func (r *MemorySchemaRepository) GetByNameAndVersion(name string, version int) (*shared.CredentialSchema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.schemas {
		if s.Name == name && s.Version == version {
			return &s, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemorySchemaRepository) Create(s *shared.CredentialSchema) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.schemas {
		if existing.Name == s.Name && existing.Version == s.Version {
			return fmt.Errorf("%w: %s v%d", ErrConflict, s.Name, s.Version)
		}
	}

	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	r.schemas[s.ID] = *s
	return nil
}
