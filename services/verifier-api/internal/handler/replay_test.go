package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leoemaxie/trustpass/services/shared"
	"github.com/leoemaxie/trustpass/services/verifier-api/internal/handler"
	"github.com/leoemaxie/trustpass/services/verifier-api/internal/session"
)

// TestReplayProtectionAndSessionExpiration explicitly tests all three rejection cases:
// 1. SessionTokenExpired
// 2. SessionTokenReused
// 3. SignatureInvalid
func TestReplayProtectionAndSessionExpiration(t *testing.T) {
	// Mock core server
	mockCore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/verify-proof" {
			var req shared.CoreVerifyProofRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			var proofMap map[string]interface{}
			_ = json.Unmarshal(req.Proof, &proofMap)

			if isTampered, ok := proofMap["tampered"].(bool); ok && isTampered {
				reason := "SignatureInvalid"
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(shared.CoreVerifyProofResponse{
					Valid:           false,
					RejectionReason: &reason,
					ErrorMessage:    &reason,
				})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(shared.CoreVerifyProofResponse{
				Valid: true,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer mockCore.Close()

	coreClient := shared.NewCoreClient(mockCore.URL)
	store := session.NewMemoryStore()
	h := handler.NewVerifierHandler(coreClient, nil, store, 2*time.Second)

	// Create valid session
	claimReq := shared.ClaimRequest{
		SchemaName:    "NationalIDCredential",
		AttributeName: "dateOfBirth",
		Operator:      shared.OpBeforeDate,
		Value:         json.RawMessage(`"2008-09-13"`),
	}
	sess, err := store.CreateSession(claimReq, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	proofPayload := map[string]interface{}{
		"proofBytesHex": "3045022100...",
	}

	// -------------------------------------------------------------
	// Verification 0: Happy Path (Consumes the session)
	// -------------------------------------------------------------
	reqBody, _ := json.Marshal(map[string]interface{}{
		"sessionToken": sess.SessionToken,
		"proof":        proofPayload,
	})
	req := httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(reqBody))
	w := httptest.NewRecorder()
	h.HandleVerify(w, req)

	var happyResp handler.VerificationResponse
	_ = json.Unmarshal(w.Body.Bytes(), &happyResp)
	if !happyResp.Valid {
		t.Fatalf("expected happy path to verify, got: %v", happyResp.RejectionReason)
	}

	// -------------------------------------------------------------
	// Rejection Case 1: SessionTokenReused
	// -------------------------------------------------------------
	reqReplay := httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(reqBody))
	wReplay := httptest.NewRecorder()
	h.HandleVerify(wReplay, reqReplay)

	var replayResp handler.VerificationResponse
	_ = json.Unmarshal(wReplay.Body.Bytes(), &replayResp)
	if replayResp.Valid || replayResp.RejectionReason == nil || *replayResp.RejectionReason != "SessionTokenReused" {
		t.Fatalf("expected SessionTokenReused, got valid=%v, reason=%v", replayResp.Valid, replayResp.RejectionReason)
	}

	// -------------------------------------------------------------
	// Rejection Case 2: SessionTokenExpired
	// -------------------------------------------------------------
	// Create session with 5ms TTL
	expiredSess, err := store.CreateSession(claimReq, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create expired session: %v", err)
	}
	time.Sleep(15 * time.Millisecond) // Wait for expiry

	reqExpiredBody, _ := json.Marshal(map[string]interface{}{
		"sessionToken": expiredSess.SessionToken,
		"proof":        proofPayload,
	})
	reqExpired := httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(reqExpiredBody))
	wExpired := httptest.NewRecorder()
	h.HandleVerify(wExpired, reqExpired)

	var expiredResp handler.VerificationResponse
	_ = json.Unmarshal(wExpired.Body.Bytes(), &expiredResp)
	if expiredResp.Valid || expiredResp.RejectionReason == nil || *expiredResp.RejectionReason != "SessionTokenExpired" {
		t.Fatalf("expected SessionTokenExpired, got valid=%v, reason=%v", expiredResp.Valid, expiredResp.RejectionReason)
	}

	// -------------------------------------------------------------
	// Rejection Case 3: SignatureInvalid
	// -------------------------------------------------------------
	tamperedSess, err := store.CreateSession(claimReq, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to create tampered session: %v", err)
	}

	reqTamperedBody, _ := json.Marshal(map[string]interface{}{
		"sessionToken": tamperedSess.SessionToken,
		"proof": map[string]interface{}{
			"proofBytesHex": "bad_signature_bytes",
			"tampered":      true,
		},
	})
	reqTampered := httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(reqTamperedBody))
	wTampered := httptest.NewRecorder()
	h.HandleVerify(wTampered, reqTampered)

	var tamperedResp handler.VerificationResponse
	_ = json.Unmarshal(wTampered.Body.Bytes(), &tamperedResp)
	if tamperedResp.Valid || tamperedResp.RejectionReason == nil || *tamperedResp.RejectionReason != "SignatureInvalid" {
		t.Fatalf("expected SignatureInvalid, got valid=%v, reason=%v", tamperedResp.Valid, tamperedResp.RejectionReason)
	}
}
