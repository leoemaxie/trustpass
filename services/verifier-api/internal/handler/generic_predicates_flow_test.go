package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/leoemaxie/trustpass/services/shared"
	"github.com/leoemaxie/trustpass/services/verifier-api/internal/handler"
	"github.com/leoemaxie/trustpass/services/verifier-api/internal/session"
)

func TestGenericPredicatesVerificationEndToEnd(t *testing.T) {
	// Mock Core API supporting generic verify-proof
	mockCore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/verify-proof" {
			var req shared.CoreVerifyProofRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			var proofEnvelope struct {
				ClaimRequest       shared.ClaimRequest `json:"claimRequest"`
				PredicateSatisfied bool                `json:"predicateSatisfied"`
			}
			_ = json.Unmarshal(req.Proof, &proofEnvelope)

			// Generic evaluation result based on proof envelope
			res := shared.CoreVerifyProofResponse{
				Valid: proofEnvelope.PredicateSatisfied,
			}
			if !proofEnvelope.PredicateSatisfied {
				reason := "PredicateNotSatisfied"
				res.RejectionReason = &reason
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(res)
			return
		}
		http.NotFound(w, r)
	}))
	defer mockCore.Close()

	// In-memory mock receipt server
	mockReceiptServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/receipts" && r.Method == http.MethodPost {
			var req shared.RecordReceiptRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			receipt := shared.VerificationReceipt{
				ID:               uuid.NewString(),
				VerifierID:       req.VerifierID,
				ClaimRequest:     req.ClaimRequest,
				Result:           req.Result,
				Timestamp:        time.Now().UTC(),
				SessionTokenHash: "dummy-hash",
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(receipt)
			return
		}
		http.NotFound(w, r)
	}))
	defer mockReceiptServer.Close()

	coreClient := shared.NewCoreClient(mockCore.URL)
	receiptClient := shared.NewReceiptClient(mockReceiptServer.URL)
	sessionStore := session.NewMemoryStore()
	h := handler.NewVerifierHandler(coreClient, receiptClient, sessionStore, 120*time.Second)

	// Test Case 1: StudentCredential with GTE operator (gpa >= 3.50)
	gpaClaim := shared.ClaimRequest{
		SchemaName:    "StudentCredential",
		AttributeName: "gpa",
		Operator:      shared.OpGTE,
		Value:         json.RawMessage("3.50"),
	}

	// 1A. Create verification session for gpa >= 3.50
	sessionReqBody, _ := json.Marshal(handler.CreateSessionRequest{ClaimRequest: gpaClaim})
	sessionReq := httptest.NewRequest(http.MethodPost, "/verification/sessions", bytes.NewReader(sessionReqBody))
	sessionRec := httptest.NewRecorder()
	h.HandleCreateSession(sessionRec, sessionReq)

	if sessionRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for student GPA session, got %d", sessionRec.Code)
	}
	var sessGPA session.VerificationSession
	_ = json.Unmarshal(sessionRec.Body.Bytes(), &sessGPA)

	// 1B. Verify valid GPA proof (gpa >= 3.50 satisfied)
	validGPAProof, _ := json.Marshal(map[string]interface{}{
		"proofBytesHex":      "deadbeef01020304",
		"revealedMessages":  map[string]string{}, // ZERO attributes disclosed
		"claimRequest":       gpaClaim,
		"sessionToken":       sessGPA.SessionToken,
		"predicateSatisfied": true,
	})

	verifyReqBody, _ := json.Marshal(handler.VerifySubmissionRequest{
		SessionToken: sessGPA.SessionToken,
		Proof:        validGPAProof,
	})
	verifyReq := httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(verifyReqBody))
	verifyRec := httptest.NewRecorder()
	h.HandleVerify(verifyRec, verifyReq)

	if verifyRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", verifyRec.Code)
	}
	var verifyRespGPA handler.VerificationResponse
	_ = json.Unmarshal(verifyRec.Body.Bytes(), &verifyRespGPA)
	if !verifyRespGPA.Valid {
		t.Fatalf("expected GPA verification to pass, got invalid: %v", verifyRespGPA.RejectionReason)
	}
	if verifyRespGPA.ReceiptID == nil {
		t.Fatalf("expected receiptId to be issued for valid GPA verification")
	}

	// Test Case 2: NationalIDCredential with EQ operator (nationality == "NG")
	citizenshipClaim := shared.ClaimRequest{
		SchemaName:    "NationalIDCredential",
		AttributeName: "nationality",
		Operator:      shared.OpEQ,
		Value:         json.RawMessage(`"NG"`),
	}

	// 2A. Create verification session for citizenship == "NG"
	sessionReqBody2, _ := json.Marshal(handler.CreateSessionRequest{ClaimRequest: citizenshipClaim})
	sessionReq2 := httptest.NewRequest(http.MethodPost, "/verification/sessions", bytes.NewReader(sessionReqBody2))
	sessionRec2 := httptest.NewRecorder()
	h.HandleCreateSession(sessionRec2, sessionReq2)

	var sessCitizenship session.VerificationSession
	_ = json.Unmarshal(sessionRec2.Body.Bytes(), &sessCitizenship)

	// 2B. Verify failing citizenship proof (ineligible foreign national)
	failingCitizenshipProof, _ := json.Marshal(map[string]interface{}{
		"proofBytesHex":      "cafebabe05060708",
		"revealedMessages":  map[string]string{},
		"claimRequest":       citizenshipClaim,
		"sessionToken":       sessCitizenship.SessionToken,
		"predicateSatisfied": false,
	})

	verifyReqBody2, _ := json.Marshal(handler.VerifySubmissionRequest{
		SessionToken: sessCitizenship.SessionToken,
		Proof:        failingCitizenshipProof,
	})
	verifyReq2 := httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(verifyReqBody2))
	verifyRec2 := httptest.NewRecorder()
	h.HandleVerify(verifyRec2, verifyReq2)

	var verifyRespCitizenship handler.VerificationResponse
	_ = json.Unmarshal(verifyRec2.Body.Bytes(), &verifyRespCitizenship)
	if verifyRespCitizenship.Valid {
		t.Fatalf("ineligible citizenship claim must fail verification")
	}
	if verifyRespCitizenship.RejectionReason == nil || *verifyRespCitizenship.RejectionReason != "PredicateNotSatisfied" {
		t.Fatalf("expected rejection reason PredicateNotSatisfied, got: %v", verifyRespCitizenship.RejectionReason)
	}
	if verifyRespCitizenship.ReceiptID == nil {
		t.Fatalf("non-personal receipt must be recorded even for failed verifications")
	}
}
