package handler_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/leoemaxie/trustpass/services/shared"
	"github.com/leoemaxie/trustpass/services/verifier-api/internal/handler"
	"github.com/leoemaxie/trustpass/services/verifier-api/internal/session"
)

func TestVerificationGeneratesNonPersonalReceipt(t *testing.T) {
	// 1. Setup mock Core server
	mockCore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(shared.CoreVerifyProofResponse{Valid: true})
	}))
	defer mockCore.Close()

	// 2. Setup mock Receipt Service server
	var receiptMu sync.Mutex
	var savedReceipts []shared.VerificationReceipt

	receiptServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receiptMu.Lock()
		defer receiptMu.Unlock()

		if r.Method == http.MethodPost && r.URL.Path == "/receipts" {
			var createReq struct {
				VerifierID   string              `json:"verifierId"`
				ClaimRequest shared.ClaimRequest `json:"claimRequest"`
				Result       bool                `json:"result"`
				SessionToken string              `json:"sessionToken"`
			}
			_ = json.NewDecoder(r.Body).Decode(&createReq)

			h := sha256.Sum256([]byte(createReq.SessionToken))
			rcpt := shared.VerificationReceipt{
				ID:               uuid.NewString(),
				VerifierID:       createReq.VerifierID,
				ClaimRequest:     createReq.ClaimRequest,
				Result:           createReq.Result,
				Timestamp:        time.Now().UTC(),
				SessionTokenHash: hex.EncodeToString(h[:]),
			}
			savedReceipts = append(savedReceipts, rcpt)

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(rcpt)
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/receipts" {
			_ = json.NewEncoder(w).Encode(savedReceipts)
			return
		}

		http.NotFound(w, r)
	}))
	defer receiptServer.Close()

	// 3. Setup Verifier API connected to both Core and Receipt Service
	coreClient := shared.NewCoreClient(mockCore.URL)
	receiptClient := shared.NewReceiptClient(receiptServer.URL)
	sessionStore := session.NewMemoryStore()
	vH := handler.NewVerifierHandler(coreClient, receiptClient, sessionStore, 120*time.Second)

	// 4. Create session
	claimReq := shared.ClaimRequest{
		SchemaName:    "NationalIDCredential",
		AttributeName: "dateOfBirth",
		Operator:      shared.OpBeforeDate,
		Value:         json.RawMessage(`"2008-09-13"`),
	}
	sess, err := sessionStore.CreateSession(claimReq, 120*time.Second)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// 5. Submit proof to Verifier API
	verifierID := "shop-pos-terminal-55"
	verifyReqBody, _ := json.Marshal(map[string]interface{}{
		"sessionToken": sess.SessionToken,
		"verifierId":   verifierID,
		"proof": map[string]interface{}{
			"proofBytesHex": "valid_bbs_proof_hex",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(verifyReqBody))
	w := httptest.NewRecorder()
	vH.HandleVerify(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp handler.VerificationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode verification response: %v", err)
	}

	if !resp.Valid {
		t.Fatalf("expected verification to succeed")
	}
	if resp.ReceiptID == nil || *resp.ReceiptID == "" {
		t.Fatal("expected receiptId to be returned in verification response")
	}

	// 6. Query receipt-service directly for the saved receipt
	receipts, err := receiptClient.ListReceipts(verifierID)
	if err != nil {
		t.Fatalf("failed to list receipts: %v", err)
	}
	if len(receipts) != 1 {
		t.Fatalf("expected 1 receipt recorded, got %d", len(receipts))
	}

	r := receipts[0]
	if r.ID != *resp.ReceiptID {
		t.Fatalf("mismatched receipt ID: %s vs %s", r.ID, *resp.ReceiptID)
	}
	if !r.Result {
		t.Fatal("expected receipt result=true")
	}
	if r.VerifierID != verifierID {
		t.Fatalf("expected verifierId %s, got %s", verifierID, r.VerifierID)
	}

	// 7. Rigorous Checkpoint 4 Privacy Inspection:
	// Verify that the receipt contains ZERO personal data.
	rawReceiptJSON, _ := json.Marshal(r)
	var rawMap map[string]interface{}
	_ = json.Unmarshal(rawReceiptJSON, &rawMap)

	forbidden := []string{
		"holderDid", "holder_did", "did", "name", "fullName",
		"idNumber", "nin", "attributes", "claims", "dateOfBirth",
		"sessionToken", // Raw session token must NOT be stored
	}
	for _, key := range forbidden {
		if _, exists := rawMap[key]; exists {
			t.Fatalf("Checkpoint 4 PRIVACY VIOLATION: Personal data field '%s' found in receipt: %s", key, string(rawReceiptJSON))
		}
	}

	// Verify session token hash is present and valid sha256
	if len(r.SessionTokenHash) != 64 {
		t.Fatalf("expected 64-char sha256 hex hash, got len %d", len(r.SessionTokenHash))
	}
}
