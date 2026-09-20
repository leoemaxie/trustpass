package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/leoemaxie/trustpass/services/shared"
	"github.com/leoemaxie/trustpass/services/verifier-api/internal/handler"
	"github.com/leoemaxie/trustpass/services/verifier-api/internal/session"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	coreURL := os.Getenv("CORE_API_URL")
	if coreURL == "" {
		coreURL = "http://localhost:50051"
	}

	ttlSec := 120
	if ttlStr := os.Getenv("VERIFICATION_SESSION_TTL_SECONDS"); ttlStr != "" {
		if val, err := strconv.Atoi(ttlStr); err == nil && val > 0 {
			ttlSec = val
		}
	}

	receiptURL := os.Getenv("RECEIPT_SERVICE_URL")
	if receiptURL == "" {
		receiptURL = "http://localhost:8084"
	}

	coreClient := shared.NewCoreClient(coreURL)
	receiptClient := shared.NewReceiptClient(receiptURL)
	sessionStore := session.NewMemoryStore()
	h := handler.NewVerifierHandler(coreClient, receiptClient, sessionStore, time.Duration(ttlSec)*time.Second)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Healthz)
	mux.HandleFunc("/verification/sessions", h.HandleCreateSession)
	mux.HandleFunc("/verification/sessions/prove", h.HandleProve)
	mux.HandleFunc("/verification/verify", h.HandleVerify)
	mux.HandleFunc("/verification/receipts", h.HandleListReceipts)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting verifier-api on %s (session TTL %ds, connecting to core at %s)...", addr, ttlSec, coreURL)
	if err := http.ListenAndServe(addr, shared.WithCORS(mux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
