use std::collections::{BTreeSet, HashMap};
use chrono::{Duration, Utc};
use serde_json::json;

use trustpass_core::bbs::keys::{generate_keypair, generate_signature_params};
use trustpass_core::bbs::proof::{generate_selective_disclosure_proof, verify_selective_disclosure_proof};
use trustpass_core::credential::did::DidKey;
use trustpass_core::credential::schema::seed_national_id_schema;
use trustpass_core::credential::vc::VerifiableCredential;
use trustpass_core::error::{CoreError, VerificationRejectionReason};
use trustpass_core::predicate::{ClaimRequest, PredicateOp};

#[test]
fn test_bbs_selective_disclosure_age_verification_flow() {
    println!("=== BBS+ Selective Disclosure: Zero Attribute Disclosure Proof & Verification ===");

    // Setup issuer
    let schema = seed_national_id_schema("did:key:pending");
    let params = generate_signature_params(schema.attribute_count() as u32);
    let keypair = generate_keypair(&params);
    let issuer_did = DidKey::from_public_key(&keypair.public_key).unwrap();

    // 1. Path 1: Adult Record (Age >= 18: DOB 1999-07-20)
    let mut claims_adult = HashMap::new();
    claims_adult.insert("fullName".to_string(), json!("Emeka Emmanuel"));
    claims_adult.insert("dateOfBirth".to_string(), json!("1999-07-20"));
    claims_adult.insert("nationality".to_string(), json!("NG"));
    claims_adult.insert("idNumber".to_string(), json!("NIN-1999072001"));

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
    .expect("Issuer-api: Adult credential issuance failed");

    // Extract BBS+ signature from issued VC
    let sig_bytes = hex::decode(&vc_adult.proof.as_ref().unwrap().proof_value).unwrap();
    let sig_adult = trustpass_core::bbs::signature::deserialize_signature(&sig_bytes).unwrap();

    // 2. Verifier generates verification session token for claim: age >= 18 (DOB BEFORE_DATE 2008-09-13)
    let session_token = "sess_tok_pos_checkpoint2_valid_001";
    let claim_age_check = ClaimRequest {
        schema_name: "NationalIDCredential".to_string(),
        attribute_name: "dateOfBirth".to_string(),
        operator: PredicateOp::BeforeDate,
        value: json!("2008-09-13"),
    };

    // 3. Holder generates selective disclosure proof blinding ALL attributes
    // Prime directive: PROVE, DON'T REVEAL. None of fullName, dateOfBirth, nationality, idNumber travel!
    let empty_reveal = BTreeSet::new();
    let proof_adult = generate_selective_disclosure_proof(
        &schema,
        &claims_adult,
        &sig_adult,
        &params,
        &keypair.public_key,
        &claim_age_check,
        session_token,
        &empty_reveal,
    )
    .expect("Proof generation for adult must succeed");

    assert!(proof_adult.revealed_messages.is_empty(), "All personal attributes MUST remain hidden");

    // 4. Verifier checks proof
    let verify_res = verify_selective_disclosure_proof(
        &proof_adult,
        &keypair.public_key,
        &params,
        session_token,
    );
    assert!(verify_res.is_ok(), "Valid adult selective disclosure proof must verify");
    println!("[PASS] Adult Holder (DOB 1999-07-20) -> Proof verified successfully. Attributes revealed: 0.");

    // 5. Path 2: Under-18 Record (DOB 2012-11-05, Age < 18)
    let mut claims_minor = HashMap::new();
    claims_minor.insert("fullName".to_string(), json!("Junior Student"));
    claims_minor.insert("dateOfBirth".to_string(), json!("2012-11-05"));
    claims_minor.insert("nationality".to_string(), json!("NG"));
    claims_minor.insert("idNumber".to_string(), json!("NIN-2012110502"));

    let vc_minor = VerifiableCredential::issue(
        &schema,
        issuer_did.did(),
        "did:key:zHolderMinor",
        claims_minor.clone(),
        Some(exp),
        &keypair.secret_key,
        &params,
    )
    .expect("Issuer-api: Minor credential issuance failed");

    let sig_minor_bytes = hex::decode(&vc_minor.proof.as_ref().unwrap().proof_value).unwrap();
    let sig_minor = trustpass_core::bbs::signature::deserialize_signature(&sig_minor_bytes).unwrap();

    let session_token_minor = "sess_tok_pos_checkpoint2_minor_002";
    let minor_proof_res = generate_selective_disclosure_proof(
        &schema,
        &claims_minor,
        &sig_minor,
        &params,
        &keypair.public_key,
        &claim_age_check,
        session_token_minor,
        &empty_reveal,
    );

    assert!(minor_proof_res.is_err(), "Under-18 record MUST fail predicate verification");
    match minor_proof_res.err().unwrap() {
        CoreError::VerificationFailed(VerificationRejectionReason::PredicateNotSatisfied) => {
            println!("[PASS] Minor Holder (DOB 2012-11-05) -> Correctly rejected with PredicateNotSatisfied.");
        }
        other => panic!("Expected PredicateNotSatisfied, got {:?}", other),
    }

    // 6. Negative test: Forged/tampered proof payload
    let mut tampered_proof = proof_adult.clone();
    tampered_proof.proof_bytes_hex = format!("ff{}", &tampered_proof.proof_bytes_hex[2..]);
    let forged_res = verify_selective_disclosure_proof(
        &tampered_proof,
        &keypair.public_key,
        &params,
        session_token,
    );
    assert!(forged_res.is_err(), "Forged proof must fail signature verification");
    match forged_res.err().unwrap() {
        CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid) => {
            println!("[PASS] Negative test: Forged proof payload correctly rejected with SignatureInvalid.");
        }
        other => panic!("Expected SignatureInvalid, got {:?}", other),
    }

    // 7. Negative test: Mismatched session token
    let replay_res = verify_selective_disclosure_proof(
        &proof_adult,
        &keypair.public_key,
        &params,
        "sess_tok_different_attacker_token",
    );
    assert!(replay_res.is_err());
    match replay_res.err().unwrap() {
        CoreError::VerificationFailed(VerificationRejectionReason::SessionTokenExpired) => {
            println!("[PASS] Negative test: Mismatched session token correctly rejected with SessionTokenExpired.");
        }
        other => panic!("Expected SessionTokenExpired, got {:?}", other),
    }

    println!("\n=== BBS+ Selective Disclosure Test PASSED ===");
}
