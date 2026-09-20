package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/leoemaxie/trustpass/services/schema-registry/internal/handler"
	"github.com/leoemaxie/trustpass/services/schema-registry/internal/repository"
	"github.com/leoemaxie/trustpass/services/shared"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	issuerDID := os.Getenv("DEFAULT_ISSUER_DID")
	if issuerDID == "" {
		issuerDID = "did:key:zUC724vsrMwHvKyqDdHtrh7z2GNe5xbsfgivth466P4vm2iaJLW9kK48DbgKa32yL944yK9k"
	}

	// Seed schemas
	seeds := shared.SeedSchemas(issuerDID)
	repo := repository.NewMemorySchemaRepository(seeds)
	h := handler.NewHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Healthz)
	mux.HandleFunc("/schemas", h.HandleSchemas)
	mux.HandleFunc("/schemas/", h.HandleSchemaByID)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting schema-registry on %s with %d seed schemas...", addr, len(seeds))
	if err := http.ListenAndServe(addr, shared.WithCORS(mux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
