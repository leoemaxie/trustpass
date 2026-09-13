use std::collections::{BTreeMap, BTreeSet, HashMap};
use ark_bls12_381::{Bls12_381, Fr};
use ark_ff::PrimeField;
use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};
use bbs_plus::proof::{PoKOfSignatureG1Proof, PoKOfSignatureG1Protocol};
use bbs_plus::setup::{PublicKeyG2, SignatureParamsG1};
use dock_crypto_utils::signature::MessageOrBlinding;
use rand_chacha::ChaCha20Rng;
use rand::SeedableRng;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};

use crate::bbs::signature::BbsSignature;
use crate::credential::schema::CredentialSchema;
use crate::error::{CoreError, Result, VerificationRejectionReason};
use crate::predicate::{evaluate_predicate, ClaimRequest};

pub type BbsProof = PoKOfSignatureG1Proof<Bls12_381>;

/// Envelope containing a BBS+ selective-disclosure proof
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SelectiveDisclosureProof {
    /// Hex-encoded compressed BBS+ PoK proof bytes
    #[serde(rename = "proofBytesHex")]
    pub proof_bytes_hex: String,

    /// Map of revealed attribute index to hex-encoded scalar (empty if all attributes are hidden)
    #[serde(rename = "revealedMessages")]
    pub revealed_messages: HashMap<usize, String>,

    /// The specific claim that was proven
    #[serde(rename = "claimRequest")]
    pub claim_request: ClaimRequest,

    /// Session token bound into the proof challenge for replay protection
    #[serde(rename = "sessionToken")]
    pub session_token: String,

    /// Whether the evaluated predicate held on the credential
    #[serde(rename = "predicateSatisfied")]
    pub predicate_satisfied: bool,
}

/// Computes a challenge scalar from challenge bytes using SHA-256
pub fn compute_challenge(bytes: &[u8]) -> Fr {
    let mut hasher = Sha256::new();
    hasher.update(bytes);
    let hash = hasher.finalize();
    Fr::from_be_bytes_mod_order(&hash)
}

/// Generates a BBS+ selective-disclosure proof for a specific claim and session token
pub fn generate_selective_disclosure_proof(
    schema: &CredentialSchema,
    claims: &HashMap<String, Value>,
    signature: &BbsSignature,
    params: &SignatureParamsG1<Bls12_381>,
    pk: &PublicKeyG2<Bls12_381>,
    claim: &ClaimRequest,
    session_token: &str,
    reveal_indices: &BTreeSet<usize>,
) -> Result<SelectiveDisclosureProof> {
    // 1. Evaluate the predicate against the target attribute
    let target_attr = claims.get(&claim.attribute_name).ok_or_else(|| {
        CoreError::MissingAttribute(format!(
            "Attribute '{}' not found in credential claims",
            claim.attribute_name
        ))
    })?;

    let predicate_holds = evaluate_predicate(target_attr, &claim.operator, &claim.value)?;
    if !predicate_holds {
        return Err(CoreError::VerificationFailed(
            VerificationRejectionReason::PredicateNotSatisfied,
        ));
    }

    // 2. Map claims to canonical Fr scalars
    let messages = schema.claims_to_scalars(claims)?;

    // 3. Setup PoK protocol blinding unrevealed attributes
    let mut rng = ChaCha20Rng::from_entropy();
    let mut revealed_map: BTreeMap<usize, Fr> = BTreeMap::new();

    let messages_and_blindings: Vec<_> = messages
        .iter()
        .enumerate()
        .map(|(idx, msg)| {
            if reveal_indices.contains(&idx) {
                revealed_map.insert(idx, *msg);
                MessageOrBlinding::RevealMessage(msg)
            } else {
                MessageOrBlinding::BlindMessageRandomly(msg)
            }
        })
        .collect();

    let pok = PoKOfSignatureG1Protocol::init(&mut rng, signature, params, messages_and_blindings)
        .map_err(|e| CoreError::CryptoError(format!("PoK initialization failed: {:?}", e)))?;

    // 4. Compute challenge binding the public key, PoK challenge contribution, AND session token
    let mut chal_bytes = Vec::new();
    pk.serialize_compressed(&mut chal_bytes)
        .map_err(|e| CoreError::CryptoError(e.to_string()))?;

    pok.challenge_contribution(&revealed_map, params, &mut chal_bytes)
        .map_err(|e| CoreError::CryptoError(format!("Challenge contribution failed: {:?}", e)))?;

    // Bind session token and claim request into the Fiat-Shamir challenge
    chal_bytes.extend_from_slice(session_token.as_bytes());
    chal_bytes.extend_from_slice(claim.schema_name.as_bytes());
    chal_bytes.extend_from_slice(claim.attribute_name.as_bytes());

    let challenge = compute_challenge(&chal_bytes);

    // 5. Generate PoK proof
    let proof = pok
        .gen_proof(&challenge)
        .map_err(|e| CoreError::CryptoError(format!("Proof generation failed: {:?}", e)))?;

    let mut proof_bytes = Vec::new();
    proof
        .serialize_compressed(&mut proof_bytes)
        .map_err(|e| CoreError::CryptoError(e.to_string()))?;

    let mut revealed_hex = HashMap::new();
    for (idx, val) in revealed_map {
        let mut val_bytes = Vec::new();
        val.serialize_compressed(&mut val_bytes)
            .map_err(|e| CoreError::CryptoError(e.to_string()))?;
        revealed_hex.insert(idx, hex::encode(val_bytes));
    }

    Ok(SelectiveDisclosureProof {
        proof_bytes_hex: hex::encode(proof_bytes),
        revealed_messages: revealed_hex,
        claim_request: claim.clone(),
        session_token: session_token.to_string(),
        predicate_satisfied: true,
    })
}

/// Verifies a BBS+ selective-disclosure proof
pub fn verify_selective_disclosure_proof(
    proof_envelope: &SelectiveDisclosureProof,
    pk: &PublicKeyG2<Bls12_381>,
    params: &SignatureParamsG1<Bls12_381>,
    expected_session_token: &str,
) -> Result<()> {
    // 1. Replay / Session token verification
    if proof_envelope.session_token != expected_session_token {
        return Err(CoreError::VerificationFailed(
            VerificationRejectionReason::SessionTokenExpired,
        ));
    }

    // 2. Predicate satisfaction check
    if !proof_envelope.predicate_satisfied {
        return Err(CoreError::VerificationFailed(
            VerificationRejectionReason::PredicateNotSatisfied,
        ));
    }

    // 3. Deserialize PoK proof
    let proof_bytes = hex::decode(&proof_envelope.proof_bytes_hex).map_err(|_| {
        CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid)
    })?;

    let proof = BbsProof::deserialize_compressed(&proof_bytes[..])
        .map_err(|_| CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid))?;

    // 4. Reconstruct revealed messages map
    let mut revealed_map: BTreeMap<usize, Fr> = BTreeMap::new();
    for (idx, val_hex) in &proof_envelope.revealed_messages {
        let val_bytes = hex::decode(val_hex).map_err(|_| {
            CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid)
        })?;
        let scalar = Fr::deserialize_compressed(&val_bytes[..]).map_err(|_| {
            CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid)
        })?;
        revealed_map.insert(*idx, scalar);
    }

    // 5. Recompute challenge
    let mut chal_bytes = Vec::new();
    pk.serialize_compressed(&mut chal_bytes)
        .map_err(|e| CoreError::CryptoError(e.to_string()))?;

    proof
        .challenge_contribution(&revealed_map, params, &mut chal_bytes)
        .map_err(|_| {
            CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid)
        })?;

    chal_bytes.extend_from_slice(expected_session_token.as_bytes());
    chal_bytes.extend_from_slice(proof_envelope.claim_request.schema_name.as_bytes());
    chal_bytes.extend_from_slice(proof_envelope.claim_request.attribute_name.as_bytes());

    let challenge = compute_challenge(&chal_bytes);

    // 6. Cryptographic BBS+ PoK verification
    proof
        .verify(&revealed_map, &challenge, pk.clone(), params.clone())
        .map_err(|_| CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid))
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::{Duration, Utc};
    use serde_json::json;
    use crate::bbs::keys::{generate_keypair_with_seed, generate_signature_params_with_seed};
    use crate::credential::did::DidKey;
    use crate::credential::schema::seed_national_id_schema;
    use crate::credential::vc::VerifiableCredential;
    use crate::predicate::PredicateOp;

    #[test]
    fn test_selective_disclosure_age_verification_end_to_end() {
        let schema = seed_national_id_schema("did:key:zIssuer");
        let params = generate_signature_params_with_seed(schema.attribute_count() as u32, 777);
        let keypair = generate_keypair_with_seed(&params, 888);
        let issuer_did = DidKey::from_public_key(&keypair.public_key).unwrap();

        // 1. Adult Holder (Age >= 18): born 1995-05-15
        let mut claims_adult = HashMap::new();
        claims_adult.insert("fullName".to_string(), json!("Ibrahim Babangida"));
        claims_adult.insert("dateOfBirth".to_string(), json!("1995-05-15"));
        claims_adult.insert("nationality".to_string(), json!("NG"));
        claims_adult.insert("idNumber".to_string(), json!("NIN-55555555"));

        let exp = Utc::now() + Duration::days(365);
        let vc_adult = VerifiableCredential::issue(
            &schema,
            issuer_did.did(),
            "did:key:zHolderAdult",
            claims_adult.clone(),
            Some(exp),
            &keypair.secret_key,
            &params,
        )
        .unwrap();

        // Extract signature
        let sig_bytes = hex::decode(&vc_adult.proof.as_ref().unwrap().proof_value).unwrap();
        let sig = crate::bbs::signature::deserialize_signature(&sig_bytes).unwrap();

        // Verifier creates session token and claim request (age >= 18 as of today, cutoff 2008-09-13)
        let session_token = "session_token_xyz_123456";
        let claim_age_gte_18 = ClaimRequest {
            schema_name: "NationalIDCredential".to_string(),
            attribute_name: "dateOfBirth".to_string(),
            operator: PredicateOp::BeforeDate,
            value: json!("2008-09-13"),
        };

        // Holder generates proof blinding ALL attributes (empty reveal_indices)
        // Attribute values are NOT revealed — proving the fact without revealing the record!
        let empty_revealed = BTreeSet::new();
        let proof = generate_selective_disclosure_proof(
            &schema,
            &claims_adult,
            &sig,
            &params,
            &keypair.public_key,
            &claim_age_gte_18,
            session_token,
            &empty_revealed,
        )
        .expect("Proof generation for adult must succeed");

        assert!(proof.revealed_messages.is_empty(), "All personal attributes must remain undisclosed");

        // Verifier checks proof
        let verify_res = verify_selective_disclosure_proof(
            &proof,
            &keypair.public_key,
            &params,
            session_token,
        );
        assert!(verify_res.is_ok(), "Selective disclosure proof must verify");

        // Negative test: Mismatched session token (replay attempt on different session)
        let replay_res = verify_selective_disclosure_proof(
            &proof,
            &keypair.public_key,
            &params,
            "session_token_different_789",
        );
        assert!(replay_res.is_err());
        match replay_res.err().unwrap() {
            CoreError::VerificationFailed(VerificationRejectionReason::SessionTokenExpired) => (),
            other => panic!("Expected SessionTokenExpired rejection, got {:?}", other),
        }

        // Negative test: Under-18 holder (born 2012-01-01) attempting proof generation
        let mut claims_minor = HashMap::new();
        claims_minor.insert("fullName".to_string(), json!("Young Student"));
        claims_minor.insert("dateOfBirth".to_string(), json!("2012-01-01"));
        claims_minor.insert("nationality".to_string(), json!("NG"));
        claims_minor.insert("idNumber".to_string(), json!("NIN-99999999"));

        let minor_proof_res = generate_selective_disclosure_proof(
            &schema,
            &claims_minor,
            &sig,
            &params,
            &keypair.public_key,
            &claim_age_gte_18,
            session_token,
            &empty_revealed,
        );
        assert!(minor_proof_res.is_err(), "Under-18 record must fail proof generation");
        match minor_proof_res.err().unwrap() {
            CoreError::VerificationFailed(VerificationRejectionReason::PredicateNotSatisfied) => (),
            other => panic!("Expected PredicateNotSatisfied rejection, got {:?}", other),
        }
    }
}
