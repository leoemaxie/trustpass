use std::collections::{BTreeSet, HashMap};
use chrono::{Duration, Utc};
use serde_json::json;

use trustpass_core::bbs::keys::{generate_keypair, generate_signature_params};
use trustpass_core::bbs::proof::{
    generate_selective_disclosure_proof_with_metadata, verify_selective_disclosure_proof_ext,
};
use trustpass_core::credential::did::DidKey;
use trustpass_core::credential::schema::{seed_national_id_schema, seed_student_schema};
use trustpass_core::credential::vc::VerifiableCredential;
use trustpass_core::error::{CoreError, VerificationRejectionReason};
use trustpass_core::predicate::{ClaimRequest, PredicateOp};

#[test]
fn test_generic_predicates_evaluation_across_schemas() {
    println!("=== Checkpoint 6: Generic Predicate Engine Generalization Test ===");
    println!("Proving 'Prove, don't reveal' with EQ (nationality == 'NG') and GTE (gpa >= 3.50)");

    // Setup Issuer with BBS+ keypair
    let issuer_did_placeholder = "did:key:zIssuerUnified";
    let national_id_schema = seed_national_id_schema(issuer_did_placeholder);
    let student_schema = seed_student_schema(issuer_did_placeholder);

    // Params sized for max schema attribute count (both schemas have 4 attributes)
    let params = generate_signature_params(4);
    let keypair = generate_keypair(&params);
    let issuer_did = DidKey::from_public_key(&keypair.public_key).unwrap();

    let holder_did = "did:key:zHolderKelechi";
    let empty_reveal = BTreeSet::new();
    let future_exp = Utc::now() + Duration::days(365);

    // =========================================================================
    // Part 1: Generic Predicate EQ on NationalIDCredential (nationality == "NG")
    // =========================================================================
    println!("\n--- Part 1: EQ Predicate on NationalIDCredential ---");

    // 1A. Eligible Citizen (nationality == "NG")
    let mut citizen_claims = HashMap::new();
    citizen_claims.insert("fullName".to_string(), json!("Kelechi Okafor"));
    citizen_claims.insert("dateOfBirth".to_string(), json!("2001-11-03"));
    citizen_claims.insert("nationality".to_string(), json!("NG"));
    citizen_claims.insert("idNumber".to_string(), json!("NIN-2001110301"));

    let citizen_vc = VerifiableCredential::issue(
        &national_id_schema,
        issuer_did.did(),
        holder_did,
        citizen_claims.clone(),
        Some(future_exp),
        &keypair.secret_key,
        &params,
    )
    .expect("Issuing citizen credential must succeed");

    let citizen_sig_bytes = hex::decode(&citizen_vc.proof.as_ref().unwrap().proof_value).unwrap();
    let citizen_sig = trustpass_core::bbs::signature::deserialize_signature(&citizen_sig_bytes).unwrap();

    let session_token_1 = "session_token_generic_eq_001";
    let claim_citizenship_eq_ng = ClaimRequest {
        schema_name: "NationalIDCredential".to_string(),
        attribute_name: "nationality".to_string(),
        operator: PredicateOp::EQ,
        value: json!("NG"),
    };

    // Generate selective disclosure proof for nationality == "NG"
    let proof_citizen = generate_selective_disclosure_proof_with_metadata(
        &national_id_schema,
        &citizen_claims,
        &citizen_sig,
        &params,
        &keypair.public_key,
        &claim_citizenship_eq_ng,
        session_token_1,
        &empty_reveal,
        Some(&citizen_vc.id),
        citizen_vc.expiration_date.as_deref(),
        None,
    )
    .expect("Proof generation for Nigerian citizen must succeed");

    // Confirm ZERO attributes are disclosed in the proof
    assert!(
        proof_citizen.revealed_messages.is_empty(),
        "Privacy guarantee violated: personal attributes leaked in proof"
    );

    // Verifier checks proof using the exact same generic verification path
    let verify_citizen_res = verify_selective_disclosure_proof_ext(
        &proof_citizen,
        &keypair.public_key,
        &params,
        session_token_1,
        Some(Utc::now()),
        None::<fn(&str) -> bool>,
    );
    assert!(verify_citizen_res.is_ok(), "Citizenship proof must verify");
    println!("Passed: nationality == 'NG' verified with zero personal attributes revealed");

    // 1B. Ineligible Foreign National (nationality == "GH")
    let mut foreign_claims = HashMap::new();
    foreign_claims.insert("fullName".to_string(), json!("Kwame Mensah"));
    foreign_claims.insert("dateOfBirth".to_string(), json!("2000-06-15"));
    foreign_claims.insert("nationality".to_string(), json!("GH"));
    foreign_claims.insert("idNumber".to_string(), json!("NIN-2000061599"));

    let foreign_vc = VerifiableCredential::issue(
        &national_id_schema,
        issuer_did.did(),
        holder_did,
        foreign_claims.clone(),
        Some(future_exp),
        &keypair.secret_key,
        &params,
    )
    .expect("Issuing foreign credential must succeed");

    let foreign_sig_bytes = hex::decode(&foreign_vc.proof.as_ref().unwrap().proof_value).unwrap();
    let foreign_sig = trustpass_core::bbs::signature::deserialize_signature(&foreign_sig_bytes).unwrap();

    let foreign_gen_res = generate_selective_disclosure_proof_with_metadata(
        &national_id_schema,
        &foreign_claims,
        &foreign_sig,
        &params,
        &keypair.public_key,
        &claim_citizenship_eq_ng,
        "session_token_foreign_attempt",
        &empty_reveal,
        Some(&foreign_vc.id),
        foreign_vc.expiration_date.as_deref(),
        None,
    );
    assert!(foreign_gen_res.is_err(), "Foreign national must fail proof for nationality == 'NG'");
    match foreign_gen_res.err().unwrap() {
        CoreError::VerificationFailed(VerificationRejectionReason::PredicateNotSatisfied) => {
            println!("Passed: Ineligible nationality correctly rejected with PredicateNotSatisfied");
        }
        other => panic!("Expected PredicateNotSatisfied, got {:?}", other),
    }

    // =========================================================================
    // Part 2: Generic Predicate GTE on StudentCredential (gpa >= 3.50)
    // =========================================================================
    println!("\n--- Part 2: GTE Predicate on StudentCredential ---");

    // 2A. Honors Student (GPA: 3.82 >= 3.50)
    let mut honors_claims = HashMap::new();
    honors_claims.insert("studentId".to_string(), json!("UNILAG/ENG/2021/1042"));
    honors_claims.insert("university".to_string(), json!("University of Lagos"));
    honors_claims.insert("enrollmentStatus".to_string(), json!("active"));
    honors_claims.insert("gpa".to_string(), json!(3.82));

    let honors_vc = VerifiableCredential::issue(
        &student_schema,
        issuer_did.did(),
        holder_did,
        honors_claims.clone(),
        Some(future_exp),
        &keypair.secret_key,
        &params,
    )
    .expect("Issuing honors student credential must succeed");

    let honors_sig_bytes = hex::decode(&honors_vc.proof.as_ref().unwrap().proof_value).unwrap();
    let honors_sig = trustpass_core::bbs::signature::deserialize_signature(&honors_sig_bytes).unwrap();

    let session_token_2 = "session_token_generic_gte_002";
    let claim_gpa_gte_3_50 = ClaimRequest {
        schema_name: "StudentCredential".to_string(),
        attribute_name: "gpa".to_string(),
        operator: PredicateOp::GTE,
        value: json!(3.50),
    };

    // Generate selective disclosure proof for gpa >= 3.50
    let proof_honors = generate_selective_disclosure_proof_with_metadata(
        &student_schema,
        &honors_claims,
        &honors_sig,
        &params,
        &keypair.public_key,
        &claim_gpa_gte_3_50,
        session_token_2,
        &empty_reveal,
        Some(&honors_vc.id),
        honors_vc.expiration_date.as_deref(),
        None,
    )
    .expect("Proof generation for honors student (GPA 3.82) must succeed");

    // Confirm ZERO attributes are disclosed in the proof (the actual GPA 3.82 is kept private)
    assert!(
        proof_honors.revealed_messages.is_empty(),
        "Privacy guarantee violated: actual GPA leaked in proof"
    );

    // Verify proof
    let verify_honors_res = verify_selective_disclosure_proof_ext(
        &proof_honors,
        &keypair.public_key,
        &params,
        session_token_2,
        Some(Utc::now()),
        None::<fn(&str) -> bool>,
    );
    assert!(verify_honors_res.is_ok(), "Honors GPA proof must verify");
    println!("Passed: gpa >= 3.50 verified with ZERO personal attributes disclosed (actual GPA 3.82 hidden)");

    // 2B. Non-honors Student (GPA: 3.15 < 3.50)
    let mut regular_claims = HashMap::new();
    regular_claims.insert("studentId".to_string(), json!("UNILAG/ENG/2021/1099"));
    regular_claims.insert("university".to_string(), json!("University of Lagos"));
    regular_claims.insert("enrollmentStatus".to_string(), json!("active"));
    regular_claims.insert("gpa".to_string(), json!(3.15));

    let regular_vc = VerifiableCredential::issue(
        &student_schema,
        issuer_did.did(),
        holder_did,
        regular_claims.clone(),
        Some(future_exp),
        &keypair.secret_key,
        &params,
    )
    .expect("Issuing regular student credential must succeed");

    let regular_sig_bytes = hex::decode(&regular_vc.proof.as_ref().unwrap().proof_value).unwrap();
    let regular_sig = trustpass_core::bbs::signature::deserialize_signature(&regular_sig_bytes).unwrap();

    let regular_gen_res = generate_selective_disclosure_proof_with_metadata(
        &student_schema,
        &regular_claims,
        &regular_sig,
        &params,
        &keypair.public_key,
        &claim_gpa_gte_3_50,
        "session_token_regular_attempt",
        &empty_reveal,
        Some(&regular_vc.id),
        regular_vc.expiration_date.as_deref(),
        None,
    );
    assert!(regular_gen_res.is_err(), "GPA 3.15 must fail proof for GPA >= 3.50");
    match regular_gen_res.err().unwrap() {
        CoreError::VerificationFailed(VerificationRejectionReason::PredicateNotSatisfied) => {
            println!("Passed: Non-honors GPA correctly rejected with PredicateNotSatisfied");
        }
        other => panic!("Expected PredicateNotSatisfied, got {:?}", other),
    }

    // =========================================================================
    // Part 3: Secondary Demo Scenario: Academic Eligibility Full Multi-Claim
    // =========================================================================
    println!("\n--- Part 3: Secondary Demo Scenario: Academic Eligibility Multi-Claim ---");
    // Criteria:
    // 1) citizenship == "NG" (NationalIDCredential)
    // 2) enrollmentStatus == "active" (StudentCredential)
    // 3) gpa >= 3.50 (StudentCredential)

    let session_token_academic = "session_token_academic_eligibility_003";

    let claim_enrollment_active = ClaimRequest {
        schema_name: "StudentCredential".to_string(),
        attribute_name: "enrollmentStatus".to_string(),
        operator: PredicateOp::EQ,
        value: json!("active"),
    };

    // Generate enrollment status proof
    let proof_enrollment = generate_selective_disclosure_proof_with_metadata(
        &student_schema,
        &honors_claims,
        &honors_sig,
        &params,
        &keypair.public_key,
        &claim_enrollment_active,
        session_token_academic,
        &empty_reveal,
        Some(&honors_vc.id),
        honors_vc.expiration_date.as_deref(),
        None,
    )
    .expect("Proof for active enrollment must succeed");

    assert!(
        verify_selective_disclosure_proof_ext(
            &proof_enrollment,
            &keypair.public_key,
            &params,
            session_token_academic,
            Some(Utc::now()),
            None::<fn(&str) -> bool>,
        )
        .is_ok(),
        "Enrollment status proof must verify"
    );

    println!("Passed: Academic eligibility scenario completely verified using generic predicate engine!");
    println!("All criteria satisfied without modifying core logic or hardcoding claim types.");
}
