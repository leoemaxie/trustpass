package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leoemaxie/trustpass/services/issuer-api/internal/handler"
	"github.com/leoemaxie/trustpass/services/shared"
)

func TestIssuerHandler(t *testing.T) {
	mockCore := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/api/v1/issue" {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "urn:uuid:test-vc",
				"type":   []string{"VerifiableCredential", "NationalIDCredential"},
				"issuer": "did:key:zMock",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer mockCore.Close()

	coreClient := shared.NewCoreClient(mockCore.URL)
	h := handler.NewIssuerHandler(coreClient)

	// Test healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.Healthz(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Test issue
	body, _ := json.Marshal(map[string]interface{}{
		"schemaName": "NationalIDCredential",
		"holderDid":  "did:key:zHolder123",
		"claims": map[string]interface{}{
			"fullName": "Test Holder",
		},
	})
	req = httptest.NewRequest(http.MethodPost, "/credentials/issue", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.HandleIssue(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	// Test list
	req = httptest.NewRequest(http.MethodGet, "/credentials", nil)
	w = httptest.NewRecorder()
	h.HandleList(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
