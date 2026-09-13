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

func TestVerifierHandler(t *testing.T) {
	mockCore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/api/v1/verify-proof" {
			var req shared.CoreVerifyProofRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
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
	h := handler.NewVerifierHandler(coreClient, nil, store, 120*time.Second)

	// 1. Test healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.Healthz(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. Test session creation
	claimBody, _ := json.Marshal(map[string]interface{}{
		"claimRequest": map[string]interface{}{
			"schemaName":    "NationalIDCredential",
			"attributeName": "dateOfBirth",
			"operator":      "BEFORE_DATE",
			"value":         "2008-09-13",
		},
	})
	req = httptest.NewRequest(http.MethodPost, "/verification/sessions", bytes.NewReader(claimBody))
	w = httptest.NewRecorder()
	h.HandleCreateSession(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var sess session.VerificationSession
	_ = json.Unmarshal(w.Body.Bytes(), &sess)
	if sess.SessionToken == "" {
		t.Fatal("expected non-empty session token")
	}

	// 3. Test verification success
	verifyBody, _ := json.Marshal(map[string]interface{}{
		"sessionToken": sess.SessionToken,
		"proof": map[string]interface{}{
			"proofBytesHex": "deadbeef",
		},
	})
	req = httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(verifyBody))
	w = httptest.NewRecorder()
	h.HandleVerify(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var res handler.VerificationResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if !res.Valid {
		t.Fatalf("expected valid=true, got %v", res.RejectionReason)
	}

	// 4. Test replay attack on consumed session
	req = httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(verifyBody))
	w = httptest.NewRecorder()
	h.HandleVerify(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var replayRes handler.VerificationResponse
	_ = json.Unmarshal(w.Body.Bytes(), &replayRes)
	if replayRes.Valid || *replayRes.RejectionReason != "SessionTokenReused" {
		t.Fatalf("expected SessionTokenReused, got valid=%v reason=%v", replayRes.Valid, replayRes.RejectionReason)
	}
}
