package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/leoemaxie/trustpass/services/receipt-service/internal/handler"
	"github.com/leoemaxie/trustpass/services/receipt-service/internal/repository"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	repo := repository.NewMemoryReceiptStore()
	h := handler.NewReceiptHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Healthz)
	mux.HandleFunc("/receipts", h.HandleReceipts)
	mux.HandleFunc("/receipts/", h.HandleReceiptByID)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting receipt-service on %s...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
