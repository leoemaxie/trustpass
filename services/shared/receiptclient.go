package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ReceiptClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewReceiptClient(baseURL string) *ReceiptClient {
	if baseURL == "" {
		baseURL = "http://localhost:8084"
	}
	return &ReceiptClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type RecordReceiptRequest struct {
	VerifierID   string       `json:"verifierId"`
	ClaimRequest ClaimRequest `json:"claimRequest"`
	Result       bool         `json:"result"`
	SessionToken string       `json:"sessionToken"`
}

func (c *ReceiptClient) RecordReceipt(req RecordReceiptRequest) (*VerificationReceipt, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(fmt.Sprintf("%s/receipts", c.baseURL), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call receipt-service /receipts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("receipt-service returned status %d", resp.StatusCode)
	}

	var receipt VerificationReceipt
	if err := json.NewDecoder(resp.Body).Decode(&receipt); err != nil {
		return nil, err
	}
	return &receipt, nil
}

func (c *ReceiptClient) ListReceipts(verifierID string) ([]VerificationReceipt, error) {
	url := fmt.Sprintf("%s/receipts", c.baseURL)
	if verifierID != "" {
		url = fmt.Sprintf("%s?verifierId=%s", url, verifierID)
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list []VerificationReceipt
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	return list, nil
}
