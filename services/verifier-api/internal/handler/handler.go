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
	coreClient    *shared.CoreClient
	receiptClient *shared.ReceiptClient
	sessionStore  session.Store
	defaultTTL    time.Duration
}

func NewVerifierHandler(
	coreClient *shared.CoreClient,
	receiptClient *shared.ReceiptClient,
	sessionStore session.Store,
	defaultTTL time.Duration,
) *VerifierHandler {
	if defaultTTL <= 0 {
		defaultTTL = 120 * time.Second
	}
	return &VerifierHandler{
		coreClient:    coreClient,
		receiptClient: receiptClient,
		sessionStore:  sessionStore,
		defaultTTL:    defaultTTL,
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
	VerifierID   *string         `json:"verifierId,omitempty"`
	IssuerDID    *string         `json:"issuerDid,omitempty"`
}

type VerificationResponse struct {
	Valid           bool                        `json:"valid"`
	ReceiptID       *string                     `json:"receiptId,omitempty"`
	Receipt         *shared.VerificationReceipt `json:"receipt,omitempty"`
	RejectionReason *string                     `json:"rejectionReason,omitempty"`
	ErrorMessage    *string                     `json:"errorMessage,omitempty"`
	Timestamp       string                      `json:"timestamp"`
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
	verifierID := "verifier-default"
	if req.VerifierID != nil && *req.VerifierID != "" {
		verifierID = *req.VerifierID
	}

	// 1. Atomic session token consumption (checks existence, expiration, and prior consumption atomically)
	sess, err := h.sessionStore.Consume(req.SessionToken)
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

	var resultValid bool
	var rejectionReason *string
	var errorMsg *string

	if err != nil {
		reason := "SignatureInvalid"
		msg := err.Error()
		resultValid = false
		rejectionReason = &reason
		errorMsg = &msg
	} else if !coreResp.Valid {
		resultValid = false
		rejectionReason = coreResp.RejectionReason
		errorMsg = coreResp.ErrorMessage
	} else {
		resultValid = true
	}

	// 3. Record non-personal verification receipt via receipt-service
	var recordedReceipt *shared.VerificationReceipt
	var receiptID *string
	if h.receiptClient != nil {
		receipt, err := h.receiptClient.RecordReceipt(shared.RecordReceiptRequest{
			VerifierID:   verifierID,
			ClaimRequest: sess.ClaimRequest,
			Result:       resultValid,
			SessionToken: req.SessionToken,
		})
		if err == nil && receipt != nil {
			recordedReceipt = receipt
			receiptID = &receipt.ID
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(VerificationResponse{
		Valid:           resultValid,
		ReceiptID:       receiptID,
		Receipt:         recordedReceipt,
		RejectionReason: rejectionReason,
		ErrorMessage:    errorMsg,
		Timestamp:       nowStr,
	})
}
