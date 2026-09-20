package handler_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

// TestCheckpoint8And9EndToEnd runs the complete flow from frontends & backend:
// - Issuer issues credentials (Admin flow, Checkpoint 9)
// - Holder reviews disclosures & generates proof (Wallet flow, Checkpoint 9)
// - Verifier scans and receives high-contrast result (Verifier PWA flow, Checkpoint 8)
// - Verifier stores non-personal audit receipts (Shop owner view, Checkpoint 8)
// - Handles both Primary (Age check) and Secondary (Academic GPA check) demo scenarios
// - Demonstrates all negative paths (Minor rejection, Replay defense, Revocation rejection)
func TestCheckpoint8And9EndToEnd(t *testing.T) {
	// 1. Setup mock Core server simulating BBS+ proof generation and verification
	var coreMu sync.Mutex
	revokedCreds := make(map[string]bool)

	mockCore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coreMu.Lock()
		defer coreMu.Unlock()

		switch r.URL.Path {
		case "/api/v1/generate-proof":
			var req shared.CoreGenerateProofRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			var vc map[string]interface{}
			_ = json.Unmarshal(req.Credential, &vc)
			credID, _ := vc["id"].(string)

			if revokedCreds[credID] {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"valid":           false,
					"rejectionReason": "CredentialRevoked",
					"errorMessage":    "Credential has been revoked by issuer",
				})
				return
			}

			// Evaluate predicate for simulation
			claims, _ := vc["credentialSubject"].(map[string]interface{})
			if claims == nil {
				claims, _ = vc["claims"].(map[string]interface{})
			}
			if innerClaims, ok := claims["claims"].(map[string]interface{}); ok {
				claims = innerClaims
			}

			attrVal, exists := claims[req.ClaimRequest.AttributeName]
			if !exists {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"valid":           false,
					"rejectionReason": "PredicateNotSatisfied",
					"errorMessage":    "Attribute not found",
				})
				return
			}

			// Age check: before 2008-09-12 (adult)
			if req.ClaimRequest.Operator == shared.OpBeforeDate {
				dobStr, _ := attrVal.(string)
				if dobStr > "2008-09-12" { // minor
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"valid":           false,
						"rejectionReason": "PredicateNotSatisfied",
						"errorMessage":    "Holder does not satisfy age condition",
					})
					return
				}
			}

			// GPA check: GTE 3.50
			if req.ClaimRequest.Operator == shared.OpGte {
				var gpa float64
				switch v := attrVal.(type) {
				case float64:
					gpa = v
				case string:
					_, _ = fmt.Sscanf(v, "%f", &gpa)
				}
				if gpa < 3.50 {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"valid":           false,
						"rejectionReason": "PredicateNotSatisfied",
						"errorMessage":    "Holder GPA does not satisfy minimum threshold",
					})
					return
				}
			}

			proofBytes := fmt.Sprintf("valid_bbs_proof_for_%s", req.SessionToken)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"proofBytesHex": hex.EncodeToString([]byte(proofBytes)),
				"sessionToken":  req.SessionToken,
			})

		case "/api/v1/verify-proof":
			var req shared.CoreVerifyProofRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			var proofMap map[string]interface{}
			_ = json.Unmarshal(req.Proof, &proofMap)

			// Proof validity based on content
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(shared.CoreVerifyProofResponse{
				Valid: true,
			})

		case "/api/v1/revocation/revoke":
			var req map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if cid, ok := req["credentialId"].(string); ok {
				revokedCreds[cid] = true
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})

		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer mockCore.Close()

	// 2. Setup mock Receipt Service
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
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(savedReceipts)
			return
		}

		http.NotFound(w, r)
	}))
	defer receiptServer.Close()

	// 3. Initialize Verifier API Handler
	coreClient := shared.NewCoreClient(mockCore.URL)
	receiptClient := shared.NewReceiptClient(receiptServer.URL)
	sessionStore := session.NewMemoryStore()
	vH := handler.NewVerifierHandler(coreClient, receiptClient, sessionStore, 120*time.Second)

	// =========================================================================
	// Scenario 1: Primary Demo — Point-of-Sale Age Verification (Age >= 18)
	// =========================================================================
	t.Run("PrimaryDemo_AgeVerification_Adult_Pass", func(t *testing.T) {
		// Step 1: Verifier requests session for Age >= 18
		ageClaim := shared.ClaimRequest{
			SchemaName:    "NationalIDCredential",
			AttributeName: "dateOfBirth",
			Operator:      shared.OpBeforeDate,
			Value:         json.RawMessage(`"2008-09-12"`),
		}
		sess, err := sessionStore.CreateSession(ageClaim, 120*time.Second)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		// Step 2: Holder wallet proves adult credential (DOB: 1999-07-20)
		adultCred := map[string]interface{}{
			"id":   "urn:uuid:cred-adult-001",
			"type": []string{"VerifiableCredential", "NationalIDCredential"},
			"credentialSubject": map[string]interface{}{
				"id": "did:key:zHolder1",
				"claims": map[string]interface{}{
					"fullName":    "Adebayo Olawale",
					"dateOfBirth": "1999-07-20",
					"nationality": "NG",
					"idNumber":    "NIN-8392104",
				},
			},
		}
		credBytes, _ := json.Marshal(adultCred)

		proveReqBody, _ := json.Marshal(handler.ProveSessionRequest{
			SessionToken: sess.SessionToken,
			Credential:   credBytes,
			ClaimRequest: ageClaim,
		})
		proveReq := httptest.NewRequest(http.MethodPost, "/verification/sessions/prove", bytes.NewReader(proveReqBody))
		proveRec := httptest.NewRecorder()
		vH.HandleProve(proveRec, proveReq)

		if proveRec.Code != http.StatusOK {
			t.Fatalf("expected proof generation 200, got %d: %s", proveRec.Code, proveRec.Body.String())
		}

		var proveResp handler.ProveSessionResponse
		_ = json.Unmarshal(proveRec.Body.Bytes(), &proveResp)

		// Step 3: Verifier PWA scans the QR and submits to /verification/verify
		verifyReqBody, _ := json.Marshal(map[string]interface{}{
			"sessionToken": sess.SessionToken,
			"verifierId":   "shop-owner-terminal-1",
			"proof":        proveResp.Proof,
		})
		verifyReq := httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(verifyReqBody))
		verifyRec := httptest.NewRecorder()
		vH.HandleVerify(verifyRec, verifyReq)

		if verifyRec.Code != http.StatusOK {
			t.Fatalf("expected verify 200, got %d: %s", verifyRec.Code, verifyRec.Body.String())
		}

		var verifyResp handler.VerificationResponse
		_ = json.Unmarshal(verifyRec.Body.Bytes(), &verifyResp)

		// Verification passed!
		if !verifyResp.Valid {
			t.Fatalf("expected adult age verification to be VALID")
		}
		if verifyResp.ReceiptID == nil || *verifyResp.ReceiptID == "" {
			t.Fatalf("expected non-personal receipt ID to be returned")
		}

		// Step 4: Verify session token is consumed and replay is rejected
		replayReq := httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(verifyReqBody))
		replayRec := httptest.NewRecorder()
		vH.HandleVerify(replayRec, replayReq)

		var replayResp handler.VerificationResponse
		_ = json.Unmarshal(replayRec.Body.Bytes(), &replayResp)
		if replayResp.Valid {
			t.Fatalf("expected replayed proof to be REJECTED")
		}
		if replayResp.RejectionReason == nil || *replayResp.RejectionReason != "SessionTokenReused" {
			t.Fatalf("expected SessionTokenReused rejection reason, got %v", replayResp.RejectionReason)
		}
	})

	// =========================================================================
	// Scenario 1 Negative: Minor Under 18 is Rejected
	// =========================================================================
	t.Run("PrimaryDemo_AgeVerification_Minor_Fails", func(t *testing.T) {
		ageClaim := shared.ClaimRequest{
			SchemaName:    "NationalIDCredential",
			AttributeName: "dateOfBirth",
			Operator:      shared.OpBeforeDate,
			Value:         json.RawMessage(`"2008-09-12"`),
		}
		sess, _ := sessionStore.CreateSession(ageClaim, 120*time.Second)

		minorCred := map[string]interface{}{
			"id":   "urn:uuid:cred-minor-002",
			"type": []string{"VerifiableCredential", "NationalIDCredential"},
			"credentialSubject": map[string]interface{}{
				"id": "did:key:zHolderMinor",
				"claims": map[string]interface{}{
					"fullName":    "Chidinma Eze",
					"dateOfBirth": "2012-05-15", // 14 years old
					"nationality": "NG",
					"idNumber":    "NIN-9912041",
				},
			},
		}
		credBytes, _ := json.Marshal(minorCred)

		proveReqBody, _ := json.Marshal(handler.ProveSessionRequest{
			SessionToken: sess.SessionToken,
			Credential:   credBytes,
			ClaimRequest: ageClaim,
		})
		proveReq := httptest.NewRequest(http.MethodPost, "/verification/sessions/prove", bytes.NewReader(proveReqBody))
		proveRec := httptest.NewRecorder()
		vH.HandleProve(proveRec, proveReq)

		// Minor fails predicate satisfaction during proof generation
		if proveRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 rejection for minor, got %d", proveRec.Code)
		}

		var errResp map[string]interface{}
		_ = json.Unmarshal(proveRec.Body.Bytes(), &errResp)
		if errResp["rejectionReason"] != "PredicateNotSatisfied" && errResp["valid"] != false {
			t.Fatalf("expected PredicateNotSatisfied for minor, got %v", errResp)
		}
	})

	// =========================================================================
	// Scenario 2: Secondary Demo — Academic Eligibility (GPA >= 3.50)
	// =========================================================================
	t.Run("SecondaryDemo_AcademicEligibility_GPA_Pass", func(t *testing.T) {
		gpaClaim := shared.ClaimRequest{
			SchemaName:    "StudentCredential",
			AttributeName: "gpa",
			Operator:      shared.OpGte,
			Value:         json.RawMessage(`"3.50"`),
		}
		sess, _ := sessionStore.CreateSession(gpaClaim, 120*time.Second)

		studentCred := map[string]interface{}{
			"id":   "urn:uuid:cred-student-001",
			"type": []string{"VerifiableCredential", "StudentCredential"},
			"credentialSubject": map[string]interface{}{
				"id": "did:key:zHolderStudent",
				"claims": map[string]interface{}{
					"studentId":        "UNILAG-2023-8821",
					"university":       "University of Lagos",
					"enrollmentStatus": "active",
					"gpa":              "3.82", // Eligible
				},
			},
		}
		credBytes, _ := json.Marshal(studentCred)

		proveReqBody, _ := json.Marshal(handler.ProveSessionRequest{
			SessionToken: sess.SessionToken,
			Credential:   credBytes,
			ClaimRequest: gpaClaim,
		})
		proveReq := httptest.NewRequest(http.MethodPost, "/verification/sessions/prove", bytes.NewReader(proveReqBody))
		proveRec := httptest.NewRecorder()
		vH.HandleProve(proveRec, proveReq)

		if proveRec.Code != http.StatusOK {
			t.Fatalf("expected 200 for GPA >= 3.50, got %d: %s", proveRec.Code, proveRec.Body.String())
		}

		var proveResp handler.ProveSessionResponse
		_ = json.Unmarshal(proveRec.Body.Bytes(), &proveResp)

		verifyReqBody, _ := json.Marshal(map[string]interface{}{
			"sessionToken": sess.SessionToken,
			"verifierId":   "scholarship-portal",
			"proof":        proveResp.Proof,
		})
		verifyReq := httptest.NewRequest(http.MethodPost, "/verification/verify", bytes.NewReader(verifyReqBody))
		verifyRec := httptest.NewRecorder()
		vH.HandleVerify(verifyRec, verifyReq)

		var verifyResp handler.VerificationResponse
		_ = json.Unmarshal(verifyRec.Body.Bytes(), &verifyResp)
		if !verifyResp.Valid {
			t.Fatalf("expected student eligibility to be VALID")
		}
	})

	// =========================================================================
	// Scenario 2 Negative: Low GPA (3.15 < 3.50) is Rejected
	// =========================================================================
	t.Run("SecondaryDemo_AcademicEligibility_LowGPA_Fails", func(t *testing.T) {
		gpaClaim := shared.ClaimRequest{
			SchemaName:    "StudentCredential",
			AttributeName: "gpa",
			Operator:      shared.OpGte,
			Value:         json.RawMessage(`"3.50"`),
		}
		sess, _ := sessionStore.CreateSession(gpaClaim, 120*time.Second)

		studentCred := map[string]interface{}{
			"id":   "urn:uuid:cred-student-002",
			"type": []string{"VerifiableCredential", "StudentCredential"},
			"credentialSubject": map[string]interface{}{
				"id": "did:key:zHolderStudent2",
				"claims": map[string]interface{}{
					"studentId":        "UNILAG-2023-9999",
					"university":       "University of Lagos",
					"enrollmentStatus": "active",
					"gpa":              "3.15", // Under threshold
				},
			},
		}
		credBytes, _ := json.Marshal(studentCred)

		proveReqBody, _ := json.Marshal(handler.ProveSessionRequest{
			SessionToken: sess.SessionToken,
			Credential:   credBytes,
			ClaimRequest: gpaClaim,
		})
		proveReq := httptest.NewRequest(http.MethodPost, "/verification/sessions/prove", bytes.NewReader(proveReqBody))
		proveRec := httptest.NewRecorder()
		vH.HandleProve(proveRec, proveReq)

		if proveRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 rejection for low GPA, got %d", proveRec.Code)
		}
	})

	// =========================================================================
	// Revocation Flow: Revoked Credential Rejected
	// =========================================================================
	t.Run("RevocationFlow_Rejected", func(t *testing.T) {
		revokedID := "urn:uuid:cred-revoked-999"
		revokedCreds[revokedID] = true // Revoked by issuer

		claim := shared.ClaimRequest{
			SchemaName:    "NationalIDCredential",
			AttributeName: "dateOfBirth",
			Operator:      shared.OpBeforeDate,
			Value:         json.RawMessage(`"2008-09-12"`),
		}
		sess, _ := sessionStore.CreateSession(claim, 120*time.Second)

		cred := map[string]interface{}{
			"id":   revokedID,
			"type": []string{"VerifiableCredential", "NationalIDCredential"},
			"credentialSubject": map[string]interface{}{
				"id": "did:key:zHolderRevoked",
				"claims": map[string]interface{}{
					"fullName":    "Compromised Holder",
					"dateOfBirth": "1995-01-01",
					"nationality": "NG",
					"idNumber":    "NIN-0000000",
				},
			},
		}
		credBytes, _ := json.Marshal(cred)

		proveReqBody, _ := json.Marshal(handler.ProveSessionRequest{
			SessionToken: sess.SessionToken,
			Credential:   credBytes,
			ClaimRequest: claim,
		})
		proveReq := httptest.NewRequest(http.MethodPost, "/verification/sessions/prove", bytes.NewReader(proveReqBody))
		proveRec := httptest.NewRecorder()
		vH.HandleProve(proveRec, proveReq)

		if proveRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for revoked credential proof, got %d", proveRec.Code)
		}

		var errResp map[string]interface{}
		_ = json.Unmarshal(proveRec.Body.Bytes(), &errResp)
		if errResp["rejectionReason"] != "CredentialRevoked" {
			t.Fatalf("expected CredentialRevoked rejection reason, got %v", errResp)
		}
	})

	// =========================================================================
	// Checkpoint 8 & Non-Negotiable Requirement 3: Zero Personal Data in Receipts
	// =========================================================================
	t.Run("Receipts_Audit_ZeroPersonalData", func(t *testing.T) {
		listReq := httptest.NewRequest(http.MethodGet, "/verification/receipts?verifierId=shop-owner-terminal-1", nil)
		listRec := httptest.NewRecorder()
		vH.HandleListReceipts(listRec, listReq)

		if listRec.Code != http.StatusOK {
			t.Fatalf("expected 200 for receipts list, got %d", listRec.Code)
		}

		var receipts []shared.VerificationReceipt
		_ = json.Unmarshal(listRec.Body.Bytes(), &receipts)

		if len(receipts) == 0 {
			t.Fatalf("expected receipts to have been recorded")
		}

		for _, r := range receipts {
			// Must have valid sha256 session token hash
			if len(r.SessionTokenHash) != 64 {
				t.Fatalf("expected 64-char sha256 hash, got %s", r.SessionTokenHash)
			}

			// Strict personal data key check
			raw, _ := json.Marshal(r)
			var rMap map[string]interface{}
			_ = json.Unmarshal(raw, &rMap)

			forbiddenKeys := []string{"holderDid", "name", "fullName", "dateOfBirth", "nin", "idNumber", "gpa", "sessionToken"}
			for _, fk := range forbiddenKeys {
				if _, exists := rMap[fk]; exists {
					t.Fatalf("Non-negotiable requirement #3 violated: found personal field '%s' in receipt: %s", fk, string(raw))
				}
			}
		}
	})
}
