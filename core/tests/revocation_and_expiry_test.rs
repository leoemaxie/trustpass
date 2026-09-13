use std::collections::{BTreeSet, HashMap};
use chrono::{Duration, Utc};
use serde_json::json;

use trustpass_core::bbs::keys::{generate_keypair, generate_signature_params};
use trustpass_core::bbs::proof::{
    generate_selective_disclosure_proof_with_metadata, verify_selective_disclosure_proof_ext,
};
use trustpass_core::credential::did::DidKey;
use trustpass_core::credential::revocation::RevocationRegistry;
use trustpass_core::credential::schema::seed_national_id_schema;
use trustpass_core::credential::vc::VerifiableCredential;
use trustpass_core::error::{CoreError, VerificationRejectionReason};
use trustpass_core::predicate::{ClaimRequest, PredicateOp};

#[test]
fn test_credential_revocation_and_expiry_lifecycle() {
    println!("=== Checkpoint 5: Credential Revocation and Expiry Lifecycle Test ===");

    // 0. Setup Issuer with BBS+ keypair and schema
    let schema = seed_national_id_schema("did:key:zIssuerSample");
    let params = generate_signature_params(schema.attribute_count() as u32);
    let keypair = generate_keypair(&params);
    let issuer_did = DidKey::from_public_key(&keypair.public_key).unwrap();
    let revocation_registry = RevocationRegistry::new();

    let session_token = "sess_tok_checkpoint5_revocation_001";
    let claim_age_gte_18 = ClaimRequest {
        schema_name: "NationalIDCredential".to_string(),
        attribute_name: "dateOfBirth".to_string(),
        operator: PredicateOp::BeforeDate,
        value: json!("2008-09-13"),
    };

    let mut claims = HashMap::new();
    claims.insert("fullName".to_string(), json!("Amina Bello"));
    claims.insert("dateOfBirth".to_string(), json!("1998-04-12"));
    claims.insert("nationality".to_string(), json!("NG"));
    claims.insert("idNumber".to_string(), json!("NIN-1998041288"));

    // 1. Happy Path: Active unexpired, unrevoked credential
    let future_exp = Utc::now() + Duration::days(365);
    let vc_active = VerifiableCredential::issue(
        &schema,
        issuer_did.did(),
        "did:key:zHolderAmina",
        claims.clone(),
        Some(future_exp),
        &keypair.secret_key,
        &params,
    )
    .expect("Active credential issuance must succeed");

    let sig_bytes = hex::decode(&vc_active.proof.as_ref().unwrap().proof_value).unwrap();
    let sig_active = trustpass_core::bbs::signature::deserialize_signature(&sig_bytes).unwrap();

    let empty_reveal = BTreeSet::new();
    let proof_active = generate_selective_disclosure_proof_with_metadata(
        &schema,
        &claims,
        &sig_active,
        &params,
        &keypair.public_key,
        &claim_age_gte_18,
        session_token,
        &empty_reveal,
        Some(&vc_active.id),
        vc_active.expiration_date.as_deref(),
        None,
    )
    .expect("Proof generation for active credential must succeed");

    // Server-side verification for active credential succeeds
    let verify_active_res = verify_selective_disclosure_proof_ext(
        &proof_active,
        &keypair.public_key,
        &params,
        session_token,
        Some(Utc::now()),
        Some(|id: &str| revocation_registry.is_revoked(id)),
    );
    assert!(verify_active_res.is_ok(), "Active credential proof must verify cleanly");

    // 2. Negative Test: Expired Credential rejected client-side during proof generation
    let past_exp = Utc::now() - Duration::days(10);
    let vc_expired = VerifiableCredential::issue(
        &schema,
        issuer_did.did(),
        "did:key:zHolderAmina",
        claims.clone(),
        Some(past_exp),
        &keypair.secret_key,
        &params,
    )
    .expect("Issuance of expired credential (simulating past issuance) succeeds");

    let sig_expired_bytes = hex::decode(&vc_expired.proof.as_ref().unwrap().proof_value).unwrap();
    let sig_expired = trustpass_core::bbs::signature::deserialize_signature(&sig_expired_bytes).unwrap();

    let expired_gen_res = generate_selective_disclosure_proof_with_metadata(
        &schema,
        &claims,
        &sig_expired,
        &params,
        &keypair.public_key,
        &claim_age_gte_18,
        session_token,
        &empty_reveal,
        Some(&vc_expired.id),
        vc_expired.expiration_date.as_deref(),
        None,
    );
    assert!(expired_gen_res.is_err(), "Expired credential must fail client-side proof generation");
    match expired_gen_res.err().unwrap() {
        CoreError::VerificationFailed(VerificationRejectionReason::CredentialExpired) => {
            println!("Confirmed: client-side proof generation rejected with CredentialExpired");
        }
        other => panic!("Expected CredentialExpired rejection, got {:?}", other),
    }

    // 3. Negative Test: Expired proof presented to verifier rejected server-side
    let mut tampered_exp_proof = proof_active.clone();
    tampered_exp_proof.expiration_date = Some(past_exp.to_rfc3339());
    let server_expired_res = verify_selective_disclosure_proof_ext(
        &tampered_exp_proof,
        &keypair.public_key,
        &params,
        session_token,
        Some(Utc::now()),
        Some(|id: &str| revocation_registry.is_revoked(id)),
    );
    assert!(server_expired_res.is_err(), "Expired proof must fail server-side verification");
    match server_expired_res.err().unwrap() {
        CoreError::VerificationFailed(VerificationRejectionReason::CredentialExpired) => {
            println!("Confirmed: server-side verification rejected with CredentialExpired");
        }
        other => panic!("Expected CredentialExpired rejection, got {:?}", other),
    }

    // 4. Revocation Flow: Issuer revokes active credential
    assert!(!revocation_registry.is_revoked(&vc_active.id));
    let revocation_record = revocation_registry.revoke(&vc_active.id, Some("Reported compromised".to_string()));
    assert_eq!(revocation_record.credential_id, vc_active.id);
    assert!(revocation_registry.is_revoked(&vc_active.id));

    // 5. Negative Test: Pre-generated proof for now-revoked credential fails server-side verification
    let server_revoked_res = verify_selective_disclosure_proof_ext(
        &proof_active,
        &keypair.public_key,
        &params,
        session_token,
        Some(Utc::now()),
        Some(|id: &str| revocation_registry.is_revoked(id)),
    );
    assert!(server_revoked_res.is_err(), "Revoked credential proof must fail server-side verification");
    match server_revoked_res.err().unwrap() {
        CoreError::VerificationFailed(VerificationRejectionReason::CredentialRevoked) => {
            println!("Confirmed: server-side verification rejected with CredentialRevoked");
        }
        other => panic!("Expected CredentialRevoked rejection, got {:?}", other),
    }

    println!("All revocation and expiry lifecycle tests passed successfully!");
}
