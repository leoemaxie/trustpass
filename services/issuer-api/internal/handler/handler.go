package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/leoemaxie/trustpass/services/shared"
)

type StoredCredential struct {
	ID         string          `json:"id"`
	SchemaName string          `json:"schemaName"`
	HolderDID  string          `json:"holderDid"`
	Data       json.RawMessage `json:"data"`
	IssuedAt   time.Time       `json:"issuedAt"`
	Revoked    bool            `json:"revoked"`
}

type IssuerHandler struct {
	coreClient *shared.CoreClient
	mu         sync.RWMutex
	store      map[string]StoredCredential
}

func NewIssuerHandler(coreClient *shared.CoreClient) *IssuerHandler {
	return &IssuerHandler{
		coreClient: coreClient,
		store:      make(map[string]StoredCredential),
	}
}

func (h *IssuerHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"service":   "issuer-api",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

type IssueRequestBody struct {
	SchemaName string                 `json:"schemaName"`
	HolderDID  string                 `json:"holderDid"`
	Claims     map[string]interface{} `json:"claims"`
	ExpiresAt  *string                `json:"expiresAt,omitempty"`
}

func (h *IssuerHandler) HandleIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IssueRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.SchemaName == "" || req.HolderDID == "" || len(req.Claims) == 0 {
		http.Error(w, "schemaName, holderDid, and claims are required", http.StatusBadRequest)
		return
	}

	// Call core to issue and cryptographically sign VC-DM
	vcRaw, err := h.coreClient.IssueCredential(shared.CoreIssueRequest{
		SchemaName: req.SchemaName,
		HolderDID:  req.HolderDID,
		Claims:     req.Claims,
		ExpiresAt:  req.ExpiresAt,
	})
	if err != nil {
		http.Error(w, "credential issuance failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id := uuid.NewString()
	h.mu.Lock()
	h.store[id] = StoredCredential{
		ID:         id,
		SchemaName: req.SchemaName,
		HolderDID:  req.HolderDID,
		Data:       vcRaw,
		IssuedAt:   time.Now().UTC(),
		Revoked:    false,
	}
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(vcRaw)
}

func (h *IssuerHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]StoredCredential, 0, len(h.store))
	for _, cred := range h.store {
		list = append(list, cred)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}
