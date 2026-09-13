package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leoemaxie/trustpass/services/receipt-service/internal/handler"
	"github.com/leoemaxie/trustpass/services/receipt-service/internal/repository"
	"github.com/leoemaxie/trustpass/services/shared"
)

func TestReceiptServiceAndZeroPersonalDataInspection(t *testing.T) {
	repo := repository.NewMemoryReceiptStore()
	h := handler.NewReceiptHandler(repo)

	// 1. Healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.Healthz(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. Create Receipt
	rawToken := "session_token_sample_secret_abc123"
	createReq := handler.CreateReceiptRequest{
		VerifierID: "pos-terminal-lagos-01",
		ClaimRequest: shared.ClaimRequest{
			SchemaName:    "NationalIDCredential",
			AttributeName: "dateOfBirth",
			Operator:      shared.OpBeforeDate,
			Value:         json.RawMessage(`"2008-09-13"`),
		},
		Result:       true,
		SessionToken: rawToken,
	}
	body, _ := json.Marshal(createReq)

	req = httptest.NewRequest(http.MethodPost, "/receipts", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.HandleReceipts(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var receipt shared.VerificationReceipt
	if err := json.Unmarshal(w.Body.Bytes(), &receipt); err != nil {
		t.Fatalf("failed to decode receipt: %v", err)
	}

	if receipt.ID == "" {
		t.Fatal("expected receipt to have UUID")
	}
	if receipt.VerifierID != "pos-terminal-lagos-01" {
		t.Fatalf("expected verifierId pos-terminal-lagos-01, got %s", receipt.VerifierID)
	}
	if !receipt.Result {
		t.Fatal("expected result=true")
	}

	// Non-personal data verification:
	// Raw token must NOT be present; only a 64-character SHA-256 hash
	if receipt.SessionTokenHash == rawToken {
		t.Fatal("SECURITY VIOLATION: Raw session token was exposed in receipt instead of hash")
	}
	if len(receipt.SessionTokenHash) != 64 {
		t.Fatalf("expected 64-char sha256 hex string, got len %d: %s", len(receipt.SessionTokenHash), receipt.SessionTokenHash)
	}

	// Checkpoint 4 Acceptance: Inspect raw JSON representation to confirm NO holder personal data fields exist
	var rawJSONMap map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &rawJSONMap)

	forbiddenFields := []string{"holderDid", "holder_did", "did", "name", "fullName", "idNumber", "nin", "attributes", "claims", "dateOfBirth"}
	for _, field := range forbiddenFields {
		if _, found := rawJSONMap[field]; found {
			t.Fatalf("SECURITY VIOLATION: Receipt contains personal data field '%s'", field)
		}
	}

	// 3. List receipts
	reqList := httptest.NewRequest(http.MethodGet, "/receipts?verifierId=pos-terminal-lagos-01", nil)
	wList := httptest.NewRecorder()
	h.HandleReceipts(wList, reqList)

	var list []shared.VerificationReceipt
	_ = json.Unmarshal(wList.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 receipt in list, got %d", len(list))
	}

	// 4. Get receipt by ID
	reqByID := httptest.NewRequest(http.MethodGet, "/receipts/"+receipt.ID, nil)
	wByID := httptest.NewRecorder()
	h.HandleReceiptByID(wByID, reqByID)

	if wByID.Code != http.StatusOK {
		t.Fatalf("expected 200 from GetByID, got %d", wByID.Code)
	}
}
