use serde::{Deserialize, Serialize};
use thiserror::Error;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize, Error)]
pub enum VerificationRejectionReason {
    #[error("Signature is cryptographically invalid")]
    SignatureInvalid,

    #[error("Predicate was not satisfied by credential attribute")]
    PredicateNotSatisfied,

    #[error("Credential has expired")]
    CredentialExpired,

    #[error("Credential has been revoked by issuer")]
    CredentialRevoked,

    #[error("Session token has expired")]
    SessionTokenExpired,

    #[error("Session token has already been consumed (replay detected)")]
    SessionTokenReused,
}

#[derive(Debug, Error)]
pub enum CoreError {
    #[error("Verification rejected: {0}")]
    VerificationFailed(VerificationRejectionReason),

    #[error("Schema validation error: {0}")]
    SchemaValidation(String),

    #[error("Cryptographic error: {0}")]
    CryptoError(String),

    #[error("Serialization error: {0}")]
    SerializationError(String),

    #[error("Invalid DID format: {0}")]
    InvalidDid(String),

    #[error("Invalid predicate format or argument: {0}")]
    InvalidPredicate(String),

    #[error("Missing attribute in credential: {0}")]
    MissingAttribute(String),

    #[error("Key management error: {0}")]
    KeyError(String),
}

pub type Result<T> = std::result::Result<T, CoreError>;
