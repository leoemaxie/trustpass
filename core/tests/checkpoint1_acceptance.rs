use std::collections::HashMap;
use chrono::{Duration, Utc};
use serde_json::json;

use trustpass_core::bbs::keys::{generate_keypair, generate_signature_params};
use trustpass_core::credential::did::DidKey;
use trustpass_core::credential::schema::seed_national_id_schema;
use trustpass_core::credential::vc::VerifiableCredential;
use trustpass_core::error::{CoreError, VerificationRejectionReason};

#[test]
fn test_checkpoint_1_acceptance() {
    println!("=== Checkpoint 1 Acceptance Test ===");

    // 1. Setup BBS+ signature parameters for 4 attributes and generate issuer keypair
    let schema = seed_national_id_schema("did:key:pending");
    let params = generate_signature_params(schema.attribute_count() as u32);
    let issuer_keypair = generate_keypair(&params);

    // 2. Derive issuer did:key
    let issuer_did = DidKey::from_public_key(&issuer_keypair.public_key)
        .expect("Deriving issuer did:key must succeed");
    println!("Issuer DID: {}", issuer_did.did());
    assert!(issuer_did.did().starts_with("did:key:z"));

    let holder_did = "did:key:zHolderSampleSyntheticWalletKey";

    // 3. Prepare synthetic claims matching NationalIDCredential v1
    let mut claims = HashMap::new();
    claims.insert("fullName".to_string(), json!("Amina Ibrahim Bello"));
    claims.insert("dateOfBirth".to_string(), json!("2000-03-15"));
    claims.insert("nationality".to_string(), json!("NG"));
    claims.insert("idNumber".to_string(), json!("NIN-8392019283"));

    // 4. Issue and sign VC-DM credential
    let exp_date = Utc::now() + Duration::days(365);
    let vc = VerifiableCredential::issue(
        &schema,
        issuer_did.did(),
        holder_did,
        claims.clone(),
        Some(exp_date),
        &issuer_keypair.secret_key,
        &params,
    )
    .expect("VC issuance must succeed");

    // 5. Inspect structure
    println!("\nIssued VC-DM Document:\n{}", vc.to_json().unwrap());
    assert_eq!(vc.issuer, issuer_did.did());
    assert_eq!(vc.credential_subject.id, holder_did);
    assert_eq!(vc.credential_subject.claims.get("nationality").unwrap(), "NG");
    assert!(vc.proof.is_some());
    let proof = vc.proof.as_ref().unwrap();
    assert_eq!(proof.proof_type, "BbsBlsSignature2020");
    assert_eq!(proof.verification_method, format!("{}#keys-1", issuer_did.did()));

    // 6. Store and retrieve (JSON serialization roundtrip)
    let serialized = vc.to_json().expect("Serialization must succeed");
    let retrieved = VerifiableCredential::from_json(&serialized).expect("Deserialization must succeed");
    assert_eq!(vc, retrieved);

    // 7. Verify cryptographic signature under issuer public key
    retrieved
        .verify_signature(&schema, &issuer_keypair.public_key, &params)
        .expect("Signature verification must succeed");
    println!("\n[PASS] Valid credential verified successfully.");

    // 8. Negative test: tampered attribute value must fail signature verification
    let mut tampered = retrieved.clone();
    tampered
        .credential_subject
        .claims
        .insert("fullName".to_string(), json!("Imposter User"));
    let tamper_err = tampered.verify_signature(&schema, &issuer_keypair.public_key, &params);
    assert!(tamper_err.is_err(), "Tampered credential must fail verification");
    match tamper_err.err().unwrap() {
        CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid) => {
            println!("[PASS] Negative test: Tampered claim correctly rejected with SignatureInvalid.");
        }
        other => panic!("Expected SignatureInvalid, got {:?}", other),
    }

    // 9. Negative test: schema validation failure on malformed date
    let mut bad_date_claims = claims.clone();
    bad_date_claims.insert("dateOfBirth".to_string(), json!("15/03/2000"));
    let bad_date_res = VerifiableCredential::issue(
        &schema,
        issuer_did.did(),
        holder_did,
        bad_date_claims,
        Some(exp_date),
        &issuer_keypair.secret_key,
        &params,
    );
    assert!(bad_date_res.is_err());
    println!("[PASS] Negative test: Malformed attribute format rejected by schema validation.");

    println!("\n=== Checkpoint 1 Acceptance Criteria MET ===");
}
