package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/leoemaxie/trustpass/services/issuer-api/internal/handler"
	"github.com/leoemaxie/trustpass/services/shared"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	coreURL := os.Getenv("CORE_API_URL")
	if coreURL == "" {
		coreURL = "http://localhost:50051"
	}

	coreClient := shared.NewCoreClient(coreURL)
	h := handler.NewIssuerHandler(coreClient)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Healthz)
	mux.HandleFunc("/credentials/issue", h.HandleIssue)
	mux.HandleFunc("/credentials", h.HandleList)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting issuer-api on %s (connecting to core at %s)...", addr, coreURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
