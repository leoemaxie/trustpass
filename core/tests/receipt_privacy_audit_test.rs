use chrono::Utc;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use uuid::Uuid;

use trustpass_core::predicate::{ClaimRequest, PredicateOp};

/// Representation of the `verification_receipts` table (Section 5.5 of TRUSTPASS_BUILD_SPEC.md).
/// Non-negotiable requirement #3: A record proves the check happened, retained by the verifier,
/// containing ZERO personal data.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationReceipt {
    pub id: String,
    #[serde(rename = "verifierId")]
    pub verifier_id: String,
    #[serde(rename = "claimRequest")]
    pub claim_request: ClaimRequest,
    pub result: bool,
    pub timestamp: String,
    #[serde(rename = "sessionTokenHash")]
    pub session_token_hash: String,
}

impl VerificationReceipt {
    pub fn new(
        verifier_id: &str,
        claim_request: ClaimRequest,
        result: bool,
        raw_session_token: &str,
    ) -> Self {
        let mut hasher = Sha256::new();
        hasher.update(raw_session_token.as_bytes());
        let hash = hex::encode(hasher.finalize());

        Self {
            id: Uuid::new_v4().to_string(),
            verifier_id: verifier_id.to_string(),
            claim_request,
            result,
            timestamp: Utc::now().to_rfc3339(),
            session_token_hash: hash,
        }
    }
}

#[test]
fn test_verification_receipt_contains_zero_personal_data() {
    println!("=== Receipt Privacy Audit: Proving Check Happened With Zero Personal Data ===");

    let raw_session_token = "sess_tok_pos_customer_scan_987654321";
    let verifier_id = "retail-shop-lagos-terminal-04";

    let claim_age = ClaimRequest {
        schema_name: "NationalIDCredential".to_string(),
        attribute_name: "dateOfBirth".to_string(),
        operator: PredicateOp::BeforeDate,
        value: json!("2008-09-13"),
    };

    // Generate receipt for a completed check (e.g. age verification)
    let receipt = VerificationReceipt::new(verifier_id, claim_age, true, raw_session_token);

    // Serialize to JSON as it would be stored in the database / returned to verifier
    let receipt_json = serde_json::to_string_pretty(&receipt).expect("JSON serialization must succeed");
    println!("Receipt Stored by Verifier:\n{}", receipt_json);

    let receipt_val: Value = serde_json::from_str(&receipt_json).unwrap();
    let receipt_map = receipt_val.as_object().unwrap();

    // 1. Confirm required audit fields are present
    assert!(receipt_map.contains_key("id"));
    assert!(receipt_map.contains_key("verifierId"));
    assert!(receipt_map.contains_key("claimRequest"));
    assert!(receipt_map.contains_key("result"));
    assert!(receipt_map.contains_key("timestamp"));
    assert!(receipt_map.contains_key("sessionTokenHash"));

    // 2. Strict Privacy Audit: Confirm ZERO holder-identifying fields
    let forbidden_identifiers = [
        "holderDid", "holder_did", "did", "holder",
        "name", "fullName", "first_name", "last_name",
        "idNumber", "id_number", "nin", "bvn",
        "dateOfBirth", "dob", "birthDate",
        "attributes", "claims", "credentialSubject",
        "sessionToken", // Raw token must NOT be retained
    ];

    for field in &forbidden_identifiers {
        assert!(
            !receipt_map.contains_key(*field),
            "SECURITY VIOLATION: Verification receipt contained forbidden personal field: '{}'",
            field
        );
    }

    // 3. Confirm raw session token is NOT leaked
    assert_ne!(
        receipt.session_token_hash, raw_session_token,
        "Raw session token must not be stored in receipts"
    );
    assert_eq!(
        receipt.session_token_hash.len(), 64,
        "Session token must be stored only as a 64-char SHA-256 digest"
    );

    println!("[PASS] Audit complete: Receipt contains ZERO personal data.");
    println!("=== Receipt Privacy Audit PASSED ===");
}
