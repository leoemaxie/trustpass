-- TrustPass: Migration 001 - Initial Schema
-- Stores credential schemas, issued credentials, verification sessions, and receipts

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 5.1 Credential Schemas (versioned, stored as data — not code)
CREATE TABLE IF NOT EXISTS credential_schemas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    attributes JSONB NOT NULL,
    issuer_did TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_schema_name_version UNIQUE (name, version)
);

-- 5.2 Credentials (issued instances, issuer copy for audit/reissuance)
CREATE TABLE IF NOT EXISTS credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schema_id UUID NOT NULL REFERENCES credential_schemas(id) ON DELETE RESTRICT,
    holder_did TEXT NOT NULL,
    attributes JSONB NOT NULL,
    signature TEXT NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    revocation_reason TEXT
);

-- Index for revocation lookups
CREATE INDEX IF NOT EXISTS idx_credentials_revoked ON credentials(id, revoked);

-- 5.4 Verification Sessions (short-lived replay protection)
CREATE TABLE IF NOT EXISTS verification_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_token VARCHAR(128) NOT NULL UNIQUE,
    claim_request JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_verification_sessions_token ON verification_sessions(session_token);

-- 5.5 Verification Receipts (non-personal, retained by verifier)
-- PRIME DIRECTIVE: MUST NOT contain holder DID, name, or attribute values
CREATE TABLE IF NOT EXISTS verification_receipts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    verifier_id VARCHAR(255) NOT NULL,
    claim_request JSONB NOT NULL,
    result BOOLEAN NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    session_token_hash VARCHAR(64) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_verification_receipts_verifier ON verification_receipts(verifier_id);
