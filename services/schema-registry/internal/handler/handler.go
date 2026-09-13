package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/leoemaxie/trustpass/services/schema-registry/internal/repository"
	"github.com/leoemaxie/trustpass/services/shared"
)

type Handler struct {
	repo repository.SchemaRepository
}

func NewHandler(repo repository.SchemaRepository) *Handler {
	return &Handler{repo: repo}
}

// Healthz responds to health checks as required by Section 4
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"service":   "schema-registry",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleSchemas routes GET and POST for /schemas
func (h *Handler) HandleSchemas(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Check for query parameters name & version: /schemas?name=NationalIDCredential&version=1
		name := r.URL.Query().Get("name")
		versionStr := r.URL.Query().Get("version")

		if name != "" && versionStr != "" {
			version, err := strconv.Atoi(versionStr)
			if err != nil {
				http.Error(w, "invalid version parameter", http.StatusBadRequest)
				return
			}
			s, err := h.repo.GetByNameAndVersion(name, version)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					http.Error(w, "schema not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(s)
			return
		}

		schemas, err := h.repo.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(schemas)

	case http.MethodPost:
		var s shared.CredentialSchema
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if s.Name == "" || len(s.Attributes) == 0 {
			http.Error(w, "schema name and attributes are required", http.StatusBadRequest)
			return
		}
		if s.Version <= 0 {
			s.Version = 1
		}
		s.CreatedAt = time.Now().UTC()

		if err := h.repo.Create(&s); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(s)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleSchemaByID routes GET /schemas/{id}
func (h *Handler) HandleSchemaByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, "schema id required", http.StatusBadRequest)
		return
	}
	id := parts[1]

	s, err := h.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "schema not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s)
}
