package shared

import (
	"encoding/json"
	"time"
)

// AttributeType represents the data type of a credential attribute
type AttributeType string

const (
	TypeString  AttributeType = "string"
	TypeDate    AttributeType = "date"
	TypeDecimal AttributeType = "decimal"
	TypeInteger AttributeType = "integer"
	TypeBoolean AttributeType = "boolean"
)

// AttributeDefinition defines a single attribute in a credential schema
type AttributeDefinition struct {
	Name string        `json:"name"`
	Type AttributeType `json:"type"`
}

// CredentialSchema represents a versioned credential schema matching Section 5.1 of spec
type CredentialSchema struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Version    int                   `json:"version"`
	Attributes []AttributeDefinition `json:"attributes"`
	IssuerDID  string                `json:"issuerDid"`
	CreatedAt  time.Time             `json:"createdAt"`
}

// SeedSchemas returns the two seed schemas required by Section 5.1
func SeedSchemas(defaultIssuerDID string) []CredentialSchema {
	now := time.Now().UTC()
	return []CredentialSchema{
		{
			ID:        "11111111-1111-1111-1111-111111111111",
			Name:      "NationalIDCredential",
			Version:   1,
			IssuerDID: defaultIssuerDID,
			CreatedAt: now,
			Attributes: []AttributeDefinition{
				{Name: "fullName", Type: TypeString},
				{Name: "dateOfBirth", Type: TypeDate},
				{Name: "nationality", Type: TypeString},
				{Name: "idNumber", Type: TypeString},
			},
		},
		{
			ID:        "22222222-2222-2222-2222-222222222222",
			Name:      "StudentCredential",
			Version:   1,
			IssuerDID: defaultIssuerDID,
			CreatedAt: now,
			Attributes: []AttributeDefinition{
				{Name: "studentId", Type: TypeString},
				{Name: "university", Type: TypeString},
				{Name: "enrollmentStatus", Type: TypeString},
				{Name: "gpa", Type: TypeDecimal},
			},
		},
	}
}

// PredicateOp defines generic predicate operations
type PredicateOp string

const (
	OpGTE        PredicateOp = "GTE"
	OpEQ         PredicateOp = "EQ"
	OpInSet      PredicateOp = "IN_SET"
	OpBeforeDate PredicateOp = "BEFORE_DATE"
)

// ClaimRequest matches Section 5.3 of spec
type ClaimRequest struct {
	SchemaName    string          `json:"schemaName"`
	AttributeName string          `json:"attributeName"`
	Operator      PredicateOp     `json:"operator"`
	Value         json.RawMessage `json:"value"`
}

// VerificationReceipt matches Section 5.5 of spec.
// PRIME DIRECTIVE: MUST NOT contain holder identity, holder DID,
// any credential attribute value, or raw session tokens.
type VerificationReceipt struct {
	ID               string       `json:"id"`
	VerifierID       string       `json:"verifierId"`
	ClaimRequest     ClaimRequest `json:"claimRequest"`
	Result           bool         `json:"result"`
	Timestamp        time.Time    `json:"timestamp"`
	SessionTokenHash string       `json:"sessionTokenHash"`
}

// RevokeCredentialRequest matches Section 6.5 of spec
type RevokeCredentialRequest struct {
	CredentialID string  `json:"credentialId"`
	Reason       *string `json:"reason,omitempty"`
}

// RevocationRecord represents an issuer revocation entry
type RevocationRecord struct {
	CredentialID string  `json:"credentialId"`
	RevokedAt    string  `json:"revokedAt"`
	Reason       *string `json:"reason,omitempty"`
}

// RevocationCheckResponse represents the revocation status of a credential
type RevocationCheckResponse struct {
	Revoked      bool              `json:"revoked"`
	CredentialID string            `json:"credentialId"`
	Record       *RevocationRecord `json:"record,omitempty"`
}
