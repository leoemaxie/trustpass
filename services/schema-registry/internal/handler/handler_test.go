package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leoemaxie/trustpass/services/schema-registry/internal/handler"
	"github.com/leoemaxie/trustpass/services/schema-registry/internal/repository"
	"github.com/leoemaxie/trustpass/services/shared"
)

func setupTestServer() *http.ServeMux {
	seeds := shared.SeedSchemas("did:key:zIssuerTest")
	repo := repository.NewMemorySchemaRepository(seeds)
	h := handler.NewHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Healthz)
	mux.HandleFunc("/schemas", h.HandleSchemas)
	mux.HandleFunc("/schemas/", h.HandleSchemaByID)
	return mux
}

func testHealthz(t *testing.T) {
	mux := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var res map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if res["status"] != "healthy" {
		t.Errorf("expected healthy, got %s", res["status"])
	}
}

func TestListAndGetSchemas(t *testing.T) {
	mux := setupTestServer()

	// List
	req := httptest.NewRequest(http.MethodGet, "/schemas", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var list []shared.CredentialSchema
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 seed schemas, got %d", len(list))
	}

	// Query by name and version
	req = httptest.NewRequest(http.MethodGet, "/schemas?name=NationalIDCredential&version=1", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var schema shared.CredentialSchema
	if err := json.Unmarshal(w.Body.Bytes(), &schema); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if schema.Name != "NationalIDCredential" {
		t.Errorf("expected NationalIDCredential, got %s", schema.Name)
	}
}

func TestCreateSchemaAndConflict(t *testing.T) {
	mux := setupTestServer()

	newSchema := shared.CredentialSchema{
		Name:    "EmploymentCredential",
		Version: 1,
		Attributes: []shared.AttributeDefinition{
			{Name: "employer", Type: shared.TypeString},
			{Name: "role", Type: shared.TypeString},
		},
		IssuerDID: "did:key:zIssuerTest",
	}
	body, _ := json.Marshal(newSchema)

	req := httptest.NewRequest(http.MethodPost, "/schemas", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", w.Code)
	}

	// Duplicate create should conflict (409)
	req2 := httptest.NewRequest(http.MethodPost, "/schemas", bytes.NewReader(body))
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", w2.Code)
	}
}
