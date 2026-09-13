use std::collections::{BTreeSet, HashMap};
use std::path::Path;
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
use trustpass_core::zk::{NoirOperator, NoirPredicateInputs, NoirProver};

#[test]
fn test_dual_proof_engine_bbs_and_noir_side_by_side() {
    println!("=== Checkpoint 7: Dual-Proof Engine (BBS+ and Noir Circuit) Acceptance Test ===");
    println!("Proving 'Prove, don't reveal' across pluggable proof mechanisms for the same credential");

    // 1. Verify Generic Noir Circuit files exist and are correctly structured
    let nargo_toml_path = Path::new("circuits/generic_predicate/Nargo.toml");
    let main_nr_path = Path::new("circuits/generic_predicate/src/main.nr");

    assert!(nargo_toml_path.exists(), "Nargo.toml for generic predicate circuit must exist");
    assert!(main_nr_path.exists(), "src/main.nr for generic predicate circuit must exist");

    let main_nr_content = std::fs::read_to_string(main_nr_path).expect("Read main.nr");
    assert!(main_nr_content.contains("fn main("), "Circuit must define main function");
    assert!(main_nr_content.contains("OP_GTE"), "Circuit must define generic OP_GTE");
    assert!(main_nr_content.contains("OP_LTE"), "Circuit must define generic OP_LTE");
    assert!(main_nr_content.contains("OP_EQ"), "Circuit must define generic OP_EQ");

    // 2. Setup Issuer and Issue Credential for Adult Holder
    let schema = seed_national_id_schema("did:key:zIssuerSample");
    let params = generate_signature_params(schema.attribute_count() as u32);
    let keypair = generate_keypair(&params);
    let issuer_did = DidKey::from_public_key(&keypair.public_key).unwrap();

    let mut claims = HashMap::new();
    claims.insert("fullName".to_string(), json!("Ibrahim Babangida"));
    claims.insert("dateOfBirth".to_string(), json!("1999-07-20"));
    claims.insert("nationality".to_string(), json!("NG"));
    claims.insert("idNumber".to_string(), json!("NIN-1999072001"));

    let future_exp = Utc::now() + Duration::days(365);
    let vc = VerifiableCredential::issue(
        &schema,
        issuer_did.did(),
        "did:key:zHolderSample",
        claims.clone(),
        Some(future_exp),
        &keypair.secret_key,
        &params,
    )
    .expect("Credential issuance must succeed");

    let sig_bytes = hex::decode(&vc.proof.as_ref().unwrap().proof_value).unwrap();
    let sig = trustpass_core::bbs::signature::deserialize_signature(&sig_bytes).unwrap();

    let session_token = "sess_dual_engine_test_001";
    let claim_age = ClaimRequest {
        schema_name: "NationalIDCredential".to_string(),
        attribute_name: "dateOfBirth".to_string(),
        operator: PredicateOp::BeforeDate,
        value: json!("2008-09-13"),
    };

    // -------------------------------------------------------------------------
    // Mechanism A: BBS+ Selective Disclosure Proof
    // -------------------------------------------------------------------------
    println!("\n--- Mechanism A: BBS+ Selective Disclosure Proof ---");
    let empty_reveal = BTreeSet::new();
    let bbs_proof = generate_selective_disclosure_proof_with_metadata(
        &schema,
        &claims,
        &sig,
        &params,
        &keypair.public_key,
        &claim_age,
        session_token,
        &empty_reveal,
        Some(&vc.id),
        vc.expiration_date.as_deref(),
        None,
    )
    .expect("BBS+ proof generation must succeed");

    assert!(bbs_proof.revealed_messages.is_empty(), "Zero attributes revealed");
    let bbs_verify = verify_selective_disclosure_proof_ext(
        &bbs_proof,
        &keypair.public_key,
        &params,
        session_token,
        Some(Utc::now()),
        None::<fn(&str) -> bool>,
    );
    assert!(bbs_verify.is_ok(), "BBS+ proof must verify cleanly");
    println!("Mechanism A (BBS+) verified successfully with zero attributes revealed");

    // -------------------------------------------------------------------------
    // Mechanism B: Noir Zero-Knowledge Circuit
    // -------------------------------------------------------------------------
    println!("\n--- Mechanism B: Noir ZK-SNARK Circuit ---");
    let dob_val = claims.get("dateOfBirth").unwrap();
    let noir_inputs = NoirPredicateInputs::from_claim_and_value("dateOfBirth", dob_val, &claim_age)
        .expect("Noir inputs mapping");

    assert_eq!(noir_inputs.operator, NoirOperator::Lte);
    assert_eq!(noir_inputs.private_value, 19990720);
    assert_eq!(noir_inputs.threshold, 20080913);

    // Verify mathematical constraints inside the Noir circuit representation
    let circuit_satisfied = noir_inputs.evaluate_circuit_constraints().unwrap();
    assert!(circuit_satisfied, "Noir circuit constraints must be satisfied for adult");
    println!("Mechanism B (Noir circuit constraint evaluation) verified successfully for same credential");

    // Check Noir prover toolchain probe
    let prover = NoirProver::new("circuits/generic_predicate");
    let prove_res = prover.prove_predicate(&noir_inputs);
    if NoirProver::is_nargo_installed() {
        assert!(prove_res.is_ok(), "Nargo toolchain execution succeeded");
        println!("Full Nargo ZK proof generated and verified via installed toolchain");
    } else {
        // Confirms that when nargo is not present on host, core cleanly returns SPEC-GAP error
        // instead of stubbing a fake "always returns true" verifier (per AGENTS.md Section 4).
        assert!(prove_res.is_err());
        match prove_res.err().unwrap() {
            CoreError::CryptoError(msg) => {
                assert!(msg.contains("SPEC-GAP: 'nargo' toolchain not installed"));
                println!("Confirmed: Host environment lacks nargo toolchain; clean SPEC-GAP diagnostic returned");
            }
            other => panic!("Expected CryptoError SPEC-GAP, got {:?}", other),
        }
    }

    // -------------------------------------------------------------------------
    // Negative Case: Minor Holder (Age < 18: DOB 2012-05-15)
    // -------------------------------------------------------------------------
    println!("\n--- Negative Test: Minor Rejection across both engines ---");
    let mut minor_claims = claims.clone();
    minor_claims.insert("dateOfBirth".to_string(), json!("2012-05-15"));

    // BBS+ rejects minor
    let bbs_minor_res = generate_selective_disclosure_proof_with_metadata(
        &schema,
        &minor_claims,
        &sig,
        &params,
        &keypair.public_key,
        &claim_age,
        session_token,
        &empty_reveal,
        Some(&vc.id),
        vc.expiration_date.as_deref(),
        None,
    );
    assert!(bbs_minor_res.is_err());
    match bbs_minor_res.err().unwrap() {
        CoreError::VerificationFailed(VerificationRejectionReason::PredicateNotSatisfied) => {
            println!("Mechanism A: Minor rejected with PredicateNotSatisfied");
        }
        other => panic!("Expected PredicateNotSatisfied, got {:?}", other),
    }

    // Noir circuit rejects minor
    let minor_noir_inputs = NoirPredicateInputs::from_claim_and_value(
        "dateOfBirth",
        &json!("2012-05-15"),
        &claim_age,
    )
    .unwrap();
    assert!(
        !minor_noir_inputs.evaluate_circuit_constraints().unwrap(),
        "Mechanism B: Minor must fail Noir circuit constraints"
    );
    println!("Mechanism B: Minor rejected by Noir circuit constraints");

    // -------------------------------------------------------------------------
    // Generalization: Noir also evaluates Student GPA claim without new circuit
    // -------------------------------------------------------------------------
    println!("\n--- Generalization: Noir Evaluating StudentCredential GPA ---");
    let _student_schema = seed_student_schema(issuer_did.did());
    let mut student_claims = HashMap::new();
    student_claims.insert("studentId".to_string(), json!("STU-001"));
    student_claims.insert("university".to_string(), json!("UNILAG"));
    student_claims.insert("enrollmentStatus".to_string(), json!("active"));
    student_claims.insert("gpa".to_string(), json!(3.85));

    let gpa_claim = ClaimRequest {
        schema_name: "StudentCredential".to_string(),
        attribute_name: "gpa".to_string(),
        operator: PredicateOp::GTE,
        value: json!(3.50),
    };

    let gpa_noir_inputs = NoirPredicateInputs::from_claim_and_value(
        "gpa",
        student_claims.get("gpa").unwrap(),
        &gpa_claim,
    )
    .unwrap();
    assert_eq!(gpa_noir_inputs.operator, NoirOperator::Gte);
    assert!(gpa_noir_inputs.evaluate_circuit_constraints().unwrap());
    println!("Noir circuit verified GPA >= 3.50 (385 >= 350) using the same generic circuit!");

    println!("\nBoth BBS+ and Noir mechanisms proven independently for the same underlying credential!");
}
