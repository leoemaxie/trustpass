package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CoreClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewCoreClient(baseURL string) *CoreClient {
	if baseURL == "" {
		baseURL = "http://localhost:50051"
	}
	return &CoreClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type CoreIssueRequest struct {
	SchemaName string                 `json:"schemaName"`
	HolderDID  string                 `json:"holderDid"`
	Claims     map[string]interface{} `json:"claims"`
	ExpiresAt  *string                `json:"expiresAt,omitempty"`
}

type CoreGenerateProofRequest struct {
	Credential   json.RawMessage `json:"credential"`
	ClaimRequest ClaimRequest    `json:"claimRequest"`
	SessionToken string          `json:"sessionToken"`
}

type CoreVerifyProofRequest struct {
	Proof                json.RawMessage `json:"proof"`
	ExpectedSessionToken string          `json:"expectedSessionToken"`
	IssuerDID            *string         `json:"issuerDid,omitempty"`
}

type CoreVerifyProofResponse struct {
	Valid           bool    `json:"valid"`
	RejectionReason *string `json:"rejectionReason,omitempty"`
	ErrorMessage    *string `json:"errorMessage,omitempty"`
}

func (c *CoreClient) Healthz() error {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/healthz", c.baseURL))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("core health check returned %d", resp.StatusCode)
	}
	return nil
}

func (c *CoreClient) IssueCredential(req CoreIssueRequest) (json.RawMessage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(fmt.Sprintf("%s/api/v1/issue", c.baseURL), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call core /api/v1/issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errObj map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errObj)
		return nil, fmt.Errorf("core issuance failed with status %d: %v", resp.StatusCode, errObj)
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *CoreClient) GenerateProof(req CoreGenerateProofRequest) (json.RawMessage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(fmt.Sprintf("%s/api/v1/generate-proof", c.baseURL), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call core /api/v1/generate-proof: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errObj map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errObj)
		return nil, fmt.Errorf("core proof generation failed with status %d: %v", resp.StatusCode, errObj)
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *CoreClient) VerifyProof(req CoreVerifyProofRequest) (*CoreVerifyProofResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(fmt.Sprintf("%s/api/v1/verify-proof", c.baseURL), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call core /api/v1/verify-proof: %w", err)
	}
	defer resp.Body.Close()

	var verifyResp CoreVerifyProofResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return nil, err
	}
	return &verifyResp, nil
}
