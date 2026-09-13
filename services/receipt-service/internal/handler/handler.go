package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/leoemaxie/trustpass/services/receipt-service/internal/repository"
	"github.com/leoemaxie/trustpass/services/shared"
)

type ReceiptHandler struct {
	repo repository.ReceiptStore
}

func NewReceiptHandler(repo repository.ReceiptStore) *ReceiptHandler {
	return &ReceiptHandler{repo: repo}
}

func (h *ReceiptHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"service":   "receipt-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

type CreateReceiptRequest struct {
	VerifierID   string              `json:"verifierId"`
	ClaimRequest shared.ClaimRequest `json:"claimRequest"`
	Result       bool                `json:"result"`
	SessionToken string              `json:"sessionToken"`
}

func hashSessionToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (h *ReceiptHandler) HandleReceipts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req CreateReceiptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.VerifierID == "" || req.SessionToken == "" {
			http.Error(w, "verifierId and sessionToken are required", http.StatusBadRequest)
			return
		}

		// Non-personal receipt: hashes the session token and stores zero holder identifiers
		receipt := shared.VerificationReceipt{
			ID:               uuid.NewString(),
			VerifierID:       req.VerifierID,
			ClaimRequest:     req.ClaimRequest,
			Result:           req.Result,
			Timestamp:        time.Now().UTC(),
			SessionTokenHash: hashSessionToken(req.SessionToken),
		}

		if err := h.repo.Save(&receipt); err != nil {
			http.Error(w, "failed to save receipt: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(receipt)

	case http.MethodGet:
		verifierID := r.URL.Query().Get("verifierId")
		list, err := h.repo.List(verifierID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if list == nil {
			list = []shared.VerificationReceipt{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ReceiptHandler) HandleReceiptByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, "receipt id required", http.StatusBadRequest)
		return
	}
	id := parts[1]

	receipt, err := h.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "receipt not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(receipt)
}
