package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/leoemaxie/trustpass/services/shared"
)

type StoredCredential struct {
	ID               string          `json:"id"`
	CredentialURN    string          `json:"credentialUrn,omitempty"`
	SchemaName       string          `json:"schemaName"`
	HolderDID        string          `json:"holderDid"`
	Data             json.RawMessage `json:"data"`
	IssuedAt         time.Time       `json:"issuedAt"`
	Revoked          bool            `json:"revoked"`
	RevocationReason *string         `json:"revocationReason,omitempty"`
	RevokedAt        *time.Time      `json:"revokedAt,omitempty"`
}

type IssuerHandler struct {
	coreClient *shared.CoreClient
	mu         sync.RWMutex
	store      map[string]*StoredCredential
}

func NewIssuerHandler(coreClient *shared.CoreClient) *IssuerHandler {
	return &IssuerHandler{
		coreClient: coreClient,
		store:      make(map[string]*StoredCredential),
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

	var parsedVC struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(vcRaw, &parsedVC)

	id := uuid.NewString()
	credURN := parsedVC.ID
	if credURN == "" {
		credURN = id
	}

	cred := &StoredCredential{
		ID:            id,
		CredentialURN: credURN,
		SchemaName:    req.SchemaName,
		HolderDID:     req.HolderDID,
		Data:          vcRaw,
		IssuedAt:      time.Now().UTC(),
		Revoked:       false,
	}

	h.mu.Lock()
	h.store[id] = cred
	h.store[credURN] = cred
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

	seen := make(map[string]bool)
	list := make([]StoredCredential, 0, len(h.store))
	for _, cred := range h.store {
		if !seen[cred.ID] {
			seen[cred.ID] = true
			list = append(list, *cred)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

type RevokeRequestBody struct {
	Reason *string `json:"reason,omitempty"`
}

// HandleCredentialsRoute dispatches subroutes under /credentials/
// e.g. POST /credentials/{id}/revoke, GET /credentials/{id}/revocation
func (h *IssuerHandler) HandleCredentialsRoute(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// Expecting /credentials/{id}/revoke or /credentials/{id}/revocation or /credentials
	if path == "/credentials" || path == "/credentials/" {
		if r.Method == http.MethodGet {
			h.HandleList(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	// e.g. ["credentials", "{id}", "revoke"] -> length 3
	if len(parts) == 3 && parts[0] == "credentials" {
		credID := parts[1]
		action := parts[2]

		if action == "revoke" && r.Method == http.MethodPost {
			var body RevokeRequestBody
			_ = json.NewDecoder(r.Body).Decode(&body)

			h.mu.Lock()
			cred, exists := h.store[credID]
			if exists {
				cred.Revoked = true
				cred.RevocationReason = body.Reason
				now := time.Now().UTC()
				cred.RevokedAt = &now
			}
			h.mu.Unlock()

			// Revoke in core cryptographically / in core registry
			targetURN := credID
			if exists && cred.CredentialURN != "" {
				targetURN = cred.CredentialURN
			}
			if err := h.coreClient.RevokeCredential(targetURN, body.Reason); err != nil {
				http.Error(w, "failed to revoke in core: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success":      true,
				"credentialId": targetURN,
				"revoked":      true,
				"reason":       body.Reason,
			})
			return
		}

		if action == "revocation" && r.Method == http.MethodGet {
			h.mu.RLock()
			cred, exists := h.store[credID]
			h.mu.RUnlock()

			if exists {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"credentialId": cred.CredentialURN,
					"revoked":      cred.Revoked,
					"reason":       cred.RevocationReason,
					"revokedAt":    cred.RevokedAt,
				})
				return
			}

			// Check core directly
			coreRes, err := h.coreClient.CheckRevocation(credID)
			if err != nil {
				http.Error(w, "failed to check core: "+err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(coreRes)
			return
		}
	}

	http.NotFound(w, r)
}
