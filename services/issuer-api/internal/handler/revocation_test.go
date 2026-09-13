package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/leoemaxie/trustpass/services/issuer-api/internal/handler"
	"github.com/leoemaxie/trustpass/services/shared"
)

func TestIssuerRevocationFlow(t *testing.T) {
	var mu sync.Mutex
	revokedMap := make(map[string]string)

	mockCore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/issue":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "urn:uuid:credential-to-revoke-123",
				"type":   []string{"VerifiableCredential", "NationalIDCredential"},
				"issuer": "did:key:zMockIssuer",
			})
		case "/api/v1/revocation/revoke":
			var req shared.RevokeCredentialRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			reason := ""
			if req.Reason != nil {
				reason = *req.Reason
			}
			revokedMap[req.CredentialID] = reason
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success":      true,
				"credentialId": req.CredentialID,
			})
		case "/api/v1/revocation/check":
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			credID := req["credentialId"]

			mu.Lock()
			reason, isRev := revokedMap[credID]
			mu.Unlock()

			res := shared.RevocationCheckResponse{
				Revoked:      isRev,
				CredentialID: credID,
			}
			if isRev {
				res.Record = &shared.RevocationRecord{
					CredentialID: credID,
					RevokedAt:    "2026-09-13T16:00:00Z",
					Reason:       &reason,
				}
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(res)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockCore.Close()

	coreClient := shared.NewCoreClient(mockCore.URL)
	h := handler.NewIssuerHandler(coreClient)

	// 1. Issue a credential
	issueBody, _ := json.Marshal(map[string]interface{}{
		"schemaName": "NationalIDCredential",
		"holderDid":  "did:key:zHolderSample",
		"claims": map[string]interface{}{
			"fullName": "Chidinma Nwosu",
		},
	})
	issueReq := httptest.NewRequest(http.MethodPost, "/credentials/issue", bytes.NewReader(issueBody))
	issueRec := httptest.NewRecorder()
	h.HandleIssue(issueRec, issueReq)

	if issueRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", issueRec.Code)
	}

	var vc map[string]interface{}
	_ = json.Unmarshal(issueRec.Body.Bytes(), &vc)
	credURN, ok := vc["id"].(string)
	if !ok || credURN == "" {
		t.Fatalf("expected credential ID, got %+v", vc)
	}

	// 2. Check initial revocation status -> revoked = false
	statusReq := httptest.NewRequest(http.MethodGet, "/credentials/"+credURN+"/revocation", nil)
	statusRec := httptest.NewRecorder()
	h.HandleCredentialsRoute(statusRec, statusReq)

	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for revocation check, got %d", statusRec.Code)
	}
	var statusBefore map[string]interface{}
	_ = json.Unmarshal(statusRec.Body.Bytes(), &statusBefore)
	if statusBefore["revoked"] == true {
		t.Fatalf("newly issued credential should not be revoked")
	}

	// 3. Revoke credential with reason
	reason := "Credential reported compromised by holder"
	revokeBody, _ := json.Marshal(map[string]string{
		"reason": reason,
	})
	revokeReq := httptest.NewRequest(http.MethodPost, "/credentials/"+credURN+"/revoke", bytes.NewReader(revokeBody))
	revokeRec := httptest.NewRecorder()
	h.HandleCredentialsRoute(revokeRec, revokeReq)

	if revokeRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on revoke, got %d (body: %s)", revokeRec.Code, revokeRec.Body.String())
	}

	var revokeResp map[string]interface{}
	_ = json.Unmarshal(revokeRec.Body.Bytes(), &revokeResp)
	if revokeResp["revoked"] != true {
		t.Fatalf("expected revoked = true, got %+v", revokeResp)
	}

	// 4. Confirm revocation status now returns revoked = true
	statusReqAfter := httptest.NewRequest(http.MethodGet, "/credentials/"+credURN+"/revocation", nil)
	statusRecAfter := httptest.NewRecorder()
	h.HandleCredentialsRoute(statusRecAfter, statusReqAfter)

	if statusRecAfter.Code != http.StatusOK {
		t.Fatalf("expected 200 OK after revocation, got %d", statusRecAfter.Code)
	}
	var statusAfter map[string]interface{}
	_ = json.Unmarshal(statusRecAfter.Body.Bytes(), &statusAfter)
	if statusAfter["revoked"] != true {
		t.Fatalf("expected revoked = true, got %+v", statusAfter)
	}
	if statusAfter["reason"] != reason {
		t.Fatalf("expected reason '%s', got '%v'", reason, statusAfter["reason"])
	}
}
