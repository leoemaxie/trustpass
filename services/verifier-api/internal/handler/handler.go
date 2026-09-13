package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/leoemaxie/trustpass/services/shared"
	"github.com/leoemaxie/trustpass/services/verifier-api/internal/session"
)

type VerifierHandler struct {
	coreClient   *shared.CoreClient
	sessionStore session.Store
	defaultTTL   time.Duration
}

func NewVerifierHandler(coreClient *shared.CoreClient, sessionStore session.Store, defaultTTL time.Duration) *VerifierHandler {
	if defaultTTL <= 0 {
		defaultTTL = 120 * time.Second
	}
	return &VerifierHandler{
		coreClient:   coreClient,
		sessionStore: sessionStore,
		defaultTTL:   defaultTTL,
	}
}

func (h *VerifierHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"service":   "verifier-api",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

type CreateSessionRequest struct {
	ClaimRequest shared.ClaimRequest `json:"claimRequest"`
	TTLSeconds   *int                `json:"ttlSeconds,omitempty"`
}

func (h *VerifierHandler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	ttl := h.defaultTTL
	if req.TTLSeconds != nil && *req.TTLSeconds > 0 {
		ttl = time.Duration(*req.TTLSeconds) * time.Second
	}

	sess, err := h.sessionStore.CreateSession(req.ClaimRequest, ttl)
	if err != nil {
		http.Error(w, "failed to create session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sess)
}

type VerifySubmissionRequest struct {
	SessionToken string          `json:"sessionToken"`
	Proof        json.RawMessage `json:"proof"`
	IssuerDID    *string         `json:"issuerDid,omitempty"`
}

type VerificationResponse struct {
	Valid           bool    `json:"valid"`
	RejectionReason *string `json:"rejectionReason,omitempty"`
	ErrorMessage    *string `json:"errorMessage,omitempty"`
	Timestamp       string  `json:"timestamp"`
}

func (h *VerifierHandler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifySubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)

	// 1. Session token validation (check expired / reused)
	_, err := h.sessionStore.Get(req.SessionToken)
	if err != nil {
		var reason string
		switch {
		case errors.Is(err, session.ErrSessionReused):
			reason = "SessionTokenReused"
		default:
			reason = "SessionTokenExpired"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(VerificationResponse{
			Valid:           false,
			RejectionReason: &reason,
			ErrorMessage:    &reason,
			Timestamp:       nowStr,
		})
		return
	}

	// 2. Call core to cryptographically verify BBS+ proof
	coreResp, err := h.coreClient.VerifyProof(shared.CoreVerifyProofRequest{
		Proof:                req.Proof,
		ExpectedSessionToken: req.SessionToken,
		IssuerDID:            req.IssuerDID,
	})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		reason := "SignatureInvalid"
		msg := err.Error()
		_ = json.NewEncoder(w).Encode(VerificationResponse{
			Valid:           false,
			RejectionReason: &reason,
			ErrorMessage:    &msg,
			Timestamp:       nowStr,
		})
		return
	}

	if !coreResp.Valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(VerificationResponse{
			Valid:           false,
			RejectionReason: coreResp.RejectionReason,
			ErrorMessage:    coreResp.ErrorMessage,
			Timestamp:       nowStr,
		})
		return
	}

	// 3. Mark session as consumed to prevent replay attacks
	_ = h.sessionStore.MarkConsumed(req.SessionToken)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(VerificationResponse{
		Valid:     true,
		Timestamp: nowStr,
	})
}
