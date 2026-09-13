use std::collections::HashMap;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

use crate::bbs::keys::{BbsPublicKey, BbsSecretKey, BbsSignatureParams};
use crate::bbs::signature::{
    deserialize_signature, serialize_signature, sign_messages, verify_signature,
};
use crate::credential::schema::CredentialSchema;
use crate::error::{CoreError, Result, VerificationRejectionReason};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct CredentialSchemaRef {
    pub id: String,
    #[serde(rename = "type")]
    pub schema_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ProofEnvelope {
    #[serde(rename = "type")]
    pub proof_type: String,
    pub created: String,
    #[serde(rename = "verificationMethod")]
    pub verification_method: String,
    #[serde(rename = "proofPurpose")]
    pub proof_purpose: String,
    #[serde(rename = "proofValue")]
    pub proof_value: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct CredentialSubject {
    pub id: String, // Holder DID
    #[serde(flatten)]
    pub claims: HashMap<String, Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct VerifiableCredential {
    #[serde(rename = "@context")]
    pub context: Vec<String>,
    pub id: String,
    #[serde(rename = "type")]
    pub credential_type: Vec<String>,
    pub issuer: String, // Issuer DID (did:key:z...)
    #[serde(rename = "issuanceDate")]
    pub issuance_date: String,
    #[serde(rename = "expirationDate", skip_serializing_if = "Option::is_none")]
    pub expiration_date: Option<String>,
    #[serde(rename = "credentialSchema")]
    pub credential_schema: CredentialSchemaRef,
    #[serde(rename = "credentialSubject")]
    pub credential_subject: CredentialSubject,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub proof: Option<ProofEnvelope>,
}

impl VerifiableCredential {
    /// Issues and signs a new W3C Verifiable Credential using BBS+
    pub fn issue(
        schema: &CredentialSchema,
        issuer_did: &str,
        holder_did: &str,
        claims: HashMap<String, Value>,
        expiration_date: Option<DateTime<Utc>>,
        sk: &BbsSecretKey,
        params: &BbsSignatureParams,
    ) -> Result<Self> {
        // 1. Validate claims against schema
        schema.validate_claims(&claims)?;

        // 2. Convert claims to canonical BBS+ scalar messages
        let messages = schema.claims_to_scalars(&claims)?;

        // 3. Cryptographically sign messages with BBS+
        let sig = sign_messages(&messages, sk, params)?;
        let sig_bytes = serialize_signature(&sig)?;
        let proof_value_hex = hex::encode(sig_bytes);

        let now = Utc::now().to_rfc3339();
        let exp_str = expiration_date.map(|d| d.to_rfc3339());

        let proof = ProofEnvelope {
            proof_type: "BbsBlsSignature2020".to_string(),
            created: now.clone(),
            verification_method: format!("{}#keys-1", issuer_did),
            proof_purpose: "assertionMethod".to_string(),
            proof_value: proof_value_hex,
        };

        Ok(Self {
            context: vec![
                "https://www.w3.org/ns/credentials/v2".to_string(),
                "https://w3id.org/security/suites/bbs-2023/v1".to_string(),
            ],
            id: format!("urn:uuid:{}", Uuid::new_v4()),
            credential_type: vec!["VerifiableCredential".to_string(), schema.name.clone()],
            issuer: issuer_did.to_string(),
            issuance_date: now,
            expiration_date: exp_str,
            credential_schema: CredentialSchemaRef {
                id: schema.id.clone(),
                schema_type: "JsonSchemaValidator2018".to_string(),
            },
            credential_subject: CredentialSubject {
                id: holder_did.to_string(),
                claims,
            },
            proof: Some(proof),
        })
    }

    /// Verifies the BBS+ signature of an issued credential
    pub fn verify_signature(
        &self,
        schema: &CredentialSchema,
        public_key: &BbsPublicKey,
        params: &BbsSignatureParams,
    ) -> Result<()> {
        let proof = self.proof.as_ref().ok_or_else(|| {
            CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid)
        })?;

        if proof.proof_type != "BbsBlsSignature2020" {
            return Err(CoreError::CryptoError(format!(
                "Unsupported proof type: {}",
                proof.proof_type
            )));
        }

        let sig_bytes = hex::decode(&proof.proof_value).map_err(|_| {
            CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid)
        })?;

        let sig = deserialize_signature(&sig_bytes)?;
        let messages = schema.claims_to_scalars(&self.credential_subject.claims)?;

        verify_signature(&messages, &sig, public_key, params)
    }

    /// Checks whether the credential has expired against a reference timestamp
    pub fn is_expired(&self, reference_time: &DateTime<Utc>) -> bool {
        if let Some(ref exp_str) = self.expiration_date {
            if let Ok(exp_dt) = DateTime::parse_from_rfc3339(exp_str) {
                return *reference_time > exp_dt.with_timezone(&Utc);
            }
        }
        false
    }

    /// Serializes credential to formatted JSON
    pub fn to_json(&self) -> Result<String> {
        serde_json::to_string_pretty(self)
            .map_err(|e| CoreError::SerializationError(e.to_string()))
    }

    /// Deserializes credential from JSON
    pub fn from_json(json_str: &str) -> Result<Self> {
        serde_json::from_str(json_str)
            .map_err(|e| CoreError::SerializationError(e.to_string()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::Duration;
    use serde_json::json;
    use crate::bbs::keys::{generate_keypair_with_seed, generate_signature_params_with_seed};
    use crate::credential::did::DidKey;
    use crate::credential::schema::seed_national_id_schema;

    #[test]
    fn test_issue_and_verify_national_id_credential() {
        let schema = seed_national_id_schema("did:key:zIssuer");
        let params = generate_signature_params_with_seed(schema.attribute_count() as u32, 100);
        let keypair = generate_keypair_with_seed(&params, 200);
        let issuer_did = DidKey::from_public_key(&keypair.public_key).unwrap();
        let holder_did = "did:key:zHolderSample";

        let mut claims = HashMap::new();
        claims.insert("fullName".to_string(), json!("Chukwuemeka Okonkwo"));
        claims.insert("dateOfBirth".to_string(), json!("1990-08-24"));
        claims.insert("nationality".to_string(), json!("NG"));
        claims.insert("idNumber".to_string(), json!("NIN-1029384756"));

        let exp = Utc::now() + Duration::days(365);
        let cred = VerifiableCredential::issue(
            &schema,
            issuer_did.did(),
            holder_did,
            claims,
            Some(exp),
            &keypair.secret_key,
            &params,
        )
        .expect("Issuance should succeed");

        // Inspect VC structure
        assert_eq!(cred.issuer, issuer_did.did());
        assert_eq!(cred.credential_subject.id, holder_did);
        assert_eq!(cred.credential_type[1], "NationalIDCredential");
        assert!(cred.proof.is_some());
        assert!(!cred.is_expired(&Utc::now()));

        // Verify valid signature
        let verify_res = cred.verify_signature(&schema, &keypair.public_key, &params);
        assert!(verify_res.is_ok(), "Credential signature verification must succeed");

        // Test JSON roundtrip (store and retrieve)
        let json_data = cred.to_json().expect("JSON serialization should succeed");
        let recovered = VerifiableCredential::from_json(&json_data).expect("JSON parsing should succeed");
        assert_eq!(cred, recovered);
        assert!(recovered.verify_signature(&schema, &keypair.public_key, &params).is_ok());

        // Negative test: tampered claims must fail signature verification
        let mut tampered = cred.clone();
        tampered.credential_subject.claims.insert("fullName".to_string(), json!("Fraudster Smith"));
        let tamper_res = tampered.verify_signature(&schema, &keypair.public_key, &params);
        assert!(tamper_res.is_err());
        match tamper_res.err().unwrap() {
            CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid) => (),
            other => panic!("Expected SignatureInvalid, got {:?}", other),
        }

        // Negative test: expired credential
        assert!(cred.is_expired(&(Utc::now() + Duration::days(400))));
    }
}
