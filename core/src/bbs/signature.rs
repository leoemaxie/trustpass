use ark_bls12_381::{Bls12_381, Fr};
use ark_ff::PrimeField;
use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};
use bbs_plus::setup::{PublicKeyG2, SecretKey, SignatureParamsG1};
use bbs_plus::signature::SignatureG1;
use dock_crypto_utils::signature::MultiMessageSignatureParams;
use rand_chacha::ChaCha20Rng;
use rand::SeedableRng;
use sha2::{Digest, Sha256};

use crate::error::{CoreError, Result, VerificationRejectionReason};

pub type BbsSignature = SignatureG1<Bls12_381>;

/// Hashes any arbitrary byte string to an Fr scalar field element
pub fn hash_to_scalar(data: &[u8]) -> Fr {
    let mut hasher = Sha256::new();
    hasher.update(data);
    let hash = hasher.finalize();
    Fr::from_be_bytes_mod_order(&hash)
}

/// Formats a key-value attribute pair canonically as "key:value" and hashes it to Fr
pub fn attribute_to_scalar(key: &str, value: &str) -> Fr {
    let canonical = format!("{}:{}", key, value);
    hash_to_scalar(canonical.as_bytes())
}

/// Signs a slice of Fr messages using BBS+
pub fn sign_messages(
    messages: &[Fr],
    sk: &SecretKey<Fr>,
    params: &SignatureParamsG1<Bls12_381>,
) -> Result<BbsSignature> {
    if messages.len() != params.supported_message_count() {
        return Err(CoreError::CryptoError(format!(
            "Parameter supports {} messages, but {} were provided",
            params.supported_message_count(),
            messages.len()
        )));
    }
    let mut rng = ChaCha20Rng::from_entropy();
    BbsSignature::new(&mut rng, messages, sk, params)
        .map_err(|e| CoreError::CryptoError(format!("BBS+ signing failed: {:?}", e)))
}

/// Verifies a BBS+ signature against messages and public key
pub fn verify_signature(
    messages: &[Fr],
    signature: &BbsSignature,
    pk: &PublicKeyG2<Bls12_381>,
    params: &SignatureParamsG1<Bls12_381>,
) -> Result<()> {
    signature
        .verify(messages, pk.clone(), params.clone())
        .map_err(|_| CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid))
}

/// Serializes BBS+ signature to bytes
pub fn serialize_signature(sig: &BbsSignature) -> Result<Vec<u8>> {
    let mut bytes = Vec::new();
    sig.serialize_compressed(&mut bytes)
        .map_err(|e| CoreError::CryptoError(format!("Failed to serialize BBS+ signature: {}", e)))?;
    Ok(bytes)
}

/// Deserializes BBS+ signature from bytes
pub fn deserialize_signature(bytes: &[u8]) -> Result<BbsSignature> {
    BbsSignature::deserialize_compressed(bytes)
        .map_err(|e| CoreError::CryptoError(format!("Failed to deserialize BBS+ signature: {}", e)))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::bbs::keys::{generate_keypair_with_seed, generate_signature_params_with_seed};

    #[test]
    fn test_bbs_sign_and_verify() {
        let message_count = 4;
        let params = generate_signature_params_with_seed(message_count, 1111);
        let keypair = generate_keypair_with_seed(&params, 2222);

        let messages = vec![
            attribute_to_scalar("fullName", "John Doe"),
            attribute_to_scalar("dateOfBirth", "1995-05-15"),
            attribute_to_scalar("nationality", "NG"),
            attribute_to_scalar("idNumber", "A12345678"),
        ];

        let sig = sign_messages(&messages, &keypair.secret_key, &params)
            .expect("Signing should succeed");

        let verify_res = verify_signature(&messages, &sig, &keypair.public_key, &params);
        assert!(verify_res.is_ok(), "Signature verification must succeed");

        // Negative test: tampered message must fail
        let mut tampered_messages = messages.clone();
        tampered_messages[0] = attribute_to_scalar("fullName", "Alice Smith");
        let fail_res = verify_signature(&tampered_messages, &sig, &keypair.public_key, &params);
        assert!(fail_res.is_err());
        match fail_res.err().unwrap() {
            CoreError::VerificationFailed(VerificationRejectionReason::SignatureInvalid) => (),
            other => panic!("Expected SignatureInvalid rejection, got {:?}", other),
        }
    }

    #[test]
    fn test_signature_serialization_roundtrip() {
        let message_count = 2;
        let params = generate_signature_params_with_seed(message_count, 3333);
        let keypair = generate_keypair_with_seed(&params, 4444);

        let messages = vec![
            attribute_to_scalar("attr1", "val1"),
            attribute_to_scalar("attr2", "val2"),
        ];

        let sig = sign_messages(&messages, &keypair.secret_key, &params).unwrap();
        let bytes = serialize_signature(&sig).unwrap();
        let deserialized = deserialize_signature(&bytes).unwrap();

        assert_eq!(sig, deserialized);
        assert!(verify_signature(&messages, &deserialized, &keypair.public_key, &params).is_ok());
    }
}
