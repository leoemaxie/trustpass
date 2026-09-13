use std::collections::{BTreeSet, HashMap};
use std::sync::Arc;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use tiny_http::{Header, Method, Response, Server, StatusCode};

use crate::bbs::keys::{
    generate_keypair_with_seed, generate_signature_params_with_seed, BbsKeypair, BbsSignatureParams,
};
use crate::bbs::proof::{
    generate_selective_disclosure_proof, verify_selective_disclosure_proof, SelectiveDisclosureProof,
};
use crate::credential::did::DidKey;
use crate::credential::schema::{seed_national_id_schema, seed_student_schema, CredentialSchema};
use crate::credential::vc::VerifiableCredential;
use crate::error::{CoreError, VerificationRejectionReason};
use crate::predicate::ClaimRequest;

#[derive(Clone)]
pub struct CoreState {
    pub issuer_did: String,
    pub keypair: Arc<BbsKeypair>,
    pub params: Arc<BbsSignatureParams>,
    pub schemas: HashMap<String, CredentialSchema>,
}

impl CoreState {
    pub fn new_seeded(seed: u64) -> Self {
        let max_attributes = 4;
        let params = generate_signature_params_with_seed(max_attributes, seed);
        let keypair = generate_keypair_with_seed(&params, seed + 1);
        let issuer_did = DidKey::from_public_key(&keypair.public_key)
            .expect("Derived did:key from public key")
            .did;

        let mut schemas = HashMap::new();
        let nid = seed_national_id_schema(&issuer_did);
        let stu = seed_student_schema(&issuer_did);
        schemas.insert(nid.name.clone(), nid);
        schemas.insert(stu.name.clone(), stu);

        Self {
            issuer_did,
            keypair: Arc::new(keypair),
            params: Arc::new(params),
            schemas,
        }
    }
}

#[derive(Debug, Deserialize)]
pub struct IssueRequest {
    #[serde(rename = "schemaName")]
    pub schema_name: String,
    #[serde(rename = "holderDid")]
    pub holder_did: String,
    pub claims: HashMap<String, Value>,
    #[serde(rename = "expiresAt")]
    pub expires_at: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GenerateProofApiRequest {
    pub credential: VerifiableCredential,
    #[serde(rename = "claimRequest")]
    pub claim_request: ClaimRequest,
    #[serde(rename = "sessionToken")]
    pub session_token: String,
}

#[derive(Debug, Deserialize)]
pub struct VerifyProofApiRequest {
    pub proof: SelectiveDisclosureProof,
    #[serde(rename = "expectedSessionToken")]
    pub expected_session_token: String,
    #[serde(rename = "issuerDid")]
    pub issuer_did: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct VerifyProofApiResponse {
    pub valid: bool,
    #[serde(rename = "rejectionReason", skip_serializing_if = "Option::is_none")]
    pub rejection_reason: Option<VerificationRejectionReason>,
    #[serde(rename = "errorMessage", skip_serializing_if = "Option::is_none")]
    pub error_message: Option<String>,
}

pub fn run_server(port: u16, state: CoreState) {
    let addr = format!("0.0.0.0:{}", port);
    let server = Server::http(&addr).unwrap_or_else(|e| panic!("Failed to bind {}: {}", addr, e));
    println!("TrustPass Core API running on http://{}", addr);

    let json_header: Header = "Content-Type: application/json".parse().unwrap();

    for mut request in server.incoming_requests() {
        let url = request.url().to_string();
        let method = request.method().clone();

        if method == Method::Get && url == "/healthz" {
            let res = json!({
                "status": "healthy",
                "service": "proof-core",
                "issuerDid": state.issuer_did,
                "timestamp": Utc::now().to_rfc3339(),
            });
            let response = Response::from_string(res.to_string()).with_header(json_header.clone());
            let _ = request.respond(response);
            continue;
        }

        let mut body_bytes = Vec::new();
        let _ = request.as_reader().read_to_end(&mut body_bytes);

        match (method, url.as_str()) {
            (Method::Post, "/api/v1/issue") => {
                let req_res: std::result::Result<IssueRequest, _> = serde_json::from_slice(&body_bytes);
                match req_res {
                    Err(e) => {
                        let err_res = json!({"error": format!("Invalid JSON body: {}", e)});
                        let response = Response::from_string(err_res.to_string())
                            .with_status_code(StatusCode(400))
                            .with_header(json_header.clone());
                        let _ = request.respond(response);
                    }
                    Ok(issue_req) => {
                        let schema = match state.schemas.get(&issue_req.schema_name) {
                            Some(s) => s,
                            None => {
                                let err_res = json!({"error": format!("Schema '{}' not found", issue_req.schema_name)});
                                let response = Response::from_string(err_res.to_string())
                                    .with_status_code(StatusCode(404))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                                continue;
                            }
                        };

                        let exp_dt = issue_req.expires_at.and_then(|s| DateTime::parse_from_rfc3339(&s).ok().map(|d| d.with_timezone(&Utc)));

                        match VerifiableCredential::issue(
                            schema,
                            &state.issuer_did,
                            &issue_req.holder_did,
                            issue_req.claims,
                            exp_dt,
                            &state.keypair.secret_key,
                            &state.params,
                        ) {
                            Ok(vc) => {
                                let res_str = serde_json::to_string(&vc).unwrap();
                                let response = Response::from_string(res_str)
                                    .with_status_code(StatusCode(201))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                            }
                            Err(e) => {
                                let err_res = json!({"error": format!("Issuance failed: {}", e)});
                                let response = Response::from_string(err_res.to_string())
                                    .with_status_code(StatusCode(400))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                            }
                        }
                    }
                }
            }

            (Method::Post, "/api/v1/generate-proof") => {
                let req_res: std::result::Result<GenerateProofApiRequest, _> = serde_json::from_slice(&body_bytes);
                match req_res {
                    Err(e) => {
                        let err_res = json!({"error": format!("Invalid JSON body: {}", e)});
                        let response = Response::from_string(err_res.to_string())
                            .with_status_code(StatusCode(400))
                            .with_header(json_header.clone());
                        let _ = request.respond(response);
                    }
                    Ok(proof_req) => {
                        let schema = match state.schemas.get(&proof_req.claim_request.schema_name) {
                            Some(s) => s,
                            None => {
                                let err_res = json!({"error": format!("Schema '{}' not found", proof_req.claim_request.schema_name)});
                                let response = Response::from_string(err_res.to_string())
                                    .with_status_code(StatusCode(404))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                                continue;
                            }
                        };

                        let proof_sig = match &proof_req.credential.proof {
                            Some(p) => match hex::decode(&p.proof_value) {
                                Ok(bytes) => crate::bbs::signature::deserialize_signature(&bytes),
                                Err(_) => Err(CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid)),
                            },
                            None => Err(CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid)),
                        };

                        let sig = match proof_sig {
                            Ok(s) => s,
                            Err(e) => {
                                let err_res = json!({"error": format!("Invalid credential signature: {}", e)});
                                let response = Response::from_string(err_res.to_string())
                                    .with_status_code(StatusCode(400))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                                continue;
                            }
                        };

                        let empty_reveal = BTreeSet::new();
                        match generate_selective_disclosure_proof(
                            schema,
                            &proof_req.credential.credential_subject.claims,
                            &sig,
                            &state.params,
                            &state.keypair.public_key,
                            &proof_req.claim_request,
                            &proof_req.session_token,
                            &empty_reveal,
                        ) {
                            Ok(sd_proof) => {
                                let res_str = serde_json::to_string(&sd_proof).unwrap();
                                let response = Response::from_string(res_str)
                                    .with_status_code(StatusCode(200))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                            }
                            Err(CoreError::VerificationFailed(reason)) => {
                                let res = VerifyProofApiResponse {
                                    valid: false,
                                    rejection_reason: Some(reason.clone()),
                                    error_message: Some(reason.to_string()),
                                };
                                let response = Response::from_string(serde_json::to_string(&res).unwrap())
                                    .with_status_code(StatusCode(400))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                            }
                            Err(e) => {
                                let err_res = json!({"error": format!("Proof generation error: {}", e)});
                                let response = Response::from_string(err_res.to_string())
                                    .with_status_code(StatusCode(500))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                            }
                        }
                    }
                }
            }

            (Method::Post, "/api/v1/verify-proof") => {
                let req_res: std::result::Result<VerifyProofApiRequest, _> = serde_json::from_slice(&body_bytes);
                match req_res {
                    Err(e) => {
                        let err_res = json!({"error": format!("Invalid JSON body: {}", e)});
                        let response = Response::from_string(err_res.to_string())
                            .with_status_code(StatusCode(400))
                            .with_header(json_header.clone());
                        let _ = request.respond(response);
                    }
                    Ok(verify_req) => {
                        // Resolve issuer public key
                        let issuer_did_str = verify_req.issuer_did.unwrap_or_else(|| state.issuer_did.clone());
                        let pk_res = DidKey::parse(&issuer_did_str);

                        let pk = match pk_res {
                            Ok(did_key) => did_key.public_key,
                            Err(e) => {
                                let res = VerifyProofApiResponse {
                                    valid: false,
                                    rejection_reason: Some(VerificationRejectionReason::SignatureInvalid),
                                    error_message: Some(format!("Invalid issuer DID: {}", e)),
                                };
                                let response = Response::from_string(serde_json::to_string(&res).unwrap())
                                    .with_status_code(StatusCode(200))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                                continue;
                            }
                        };

                        match verify_selective_disclosure_proof(
                            &verify_req.proof,
                            &pk,
                            &state.params,
                            &verify_req.expected_session_token,
                        ) {
                            Ok(()) => {
                                let res = VerifyProofApiResponse {
                                    valid: true,
                                    rejection_reason: None,
                                    error_message: None,
                                };
                                let response = Response::from_string(serde_json::to_string(&res).unwrap())
                                    .with_status_code(StatusCode(200))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                            }
                            Err(CoreError::VerificationFailed(reason)) => {
                                let res = VerifyProofApiResponse {
                                    valid: false,
                                    rejection_reason: Some(reason.clone()),
                                    error_message: Some(reason.to_string()),
                                };
                                let response = Response::from_string(serde_json::to_string(&res).unwrap())
                                    .with_status_code(StatusCode(200))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                            }
                            Err(e) => {
                                let res = VerifyProofApiResponse {
                                    valid: false,
                                    rejection_reason: Some(VerificationRejectionReason::SignatureInvalid),
                                    error_message: Some(e.to_string()),
                                };
                                let response = Response::from_string(serde_json::to_string(&res).unwrap())
                                    .with_status_code(StatusCode(200))
                                    .with_header(json_header.clone());
                                let _ = request.respond(response);
                            }
                        }
                    }
                }
            }

            _ => {
                let response = Response::from_string("Not Found")
                    .with_status_code(StatusCode(404));
                let _ = request.respond(response);
            }
        }
    }
}
