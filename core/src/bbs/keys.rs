use ark_bls12_381::Bls12_381;
use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};
use bbs_plus::setup::{KeypairG2, PublicKeyG2, SecretKey, SignatureParamsG1};
use rand_chacha::ChaCha20Rng;
use rand::SeedableRng;
use crate::error::{CoreError, Result};

pub type BbsKeypair = KeypairG2<Bls12_381>;
pub type BbsPublicKey = PublicKeyG2<Bls12_381>;
pub type BbsSecretKey = SecretKey<<Bls12_381 as ark_ec::pairing::Pairing>::ScalarField>;
pub type BbsSignatureParams = SignatureParamsG1<Bls12_381>;

/// Generate signature parameters for BBS+ for a given number of credential messages/attributes
pub fn generate_signature_params(message_count: u32) -> BbsSignatureParams {
    let mut rng = ChaCha20Rng::from_entropy();
    BbsSignatureParams::generate_using_rng(&mut rng, message_count)
}

/// Generate signature parameters deterministically using a seed
pub fn generate_signature_params_with_seed(message_count: u32, seed: u64) -> BbsSignatureParams {
    let mut rng = ChaCha20Rng::seed_from_u64(seed);
    BbsSignatureParams::generate_using_rng(&mut rng, message_count)
}

/// Generate a new BBS+ Keypair for an issuer
pub fn generate_keypair(params: &BbsSignatureParams) -> BbsKeypair {
    let mut rng = ChaCha20Rng::from_entropy();
    BbsKeypair::generate_using_rng(&mut rng, params)
}

/// Generate a deterministic BBS+ Keypair using a seed (useful for tests and seeded demo issuers)
pub fn generate_keypair_with_seed(params: &BbsSignatureParams, seed: u64) -> BbsKeypair {
    let mut rng = ChaCha20Rng::seed_from_u64(seed);
    BbsKeypair::generate_using_rng(&mut rng, params)
}

/// Serialize a public key to compressed bytes
pub fn serialize_public_key(pk: &BbsPublicKey) -> Result<Vec<u8>> {
    let mut bytes = Vec::new();
    pk.serialize_compressed(&mut bytes)
        .map_err(|e| CoreError::CryptoError(format!("Failed to serialize public key: {}", e)))?;
    Ok(bytes)
}

/// Deserialize a public key from compressed bytes
pub fn deserialize_public_key(bytes: &[u8]) -> Result<BbsPublicKey> {
    BbsPublicKey::deserialize_compressed(bytes)
        .map_err(|e| CoreError::CryptoError(format!("Failed to deserialize public key: {}", e)))
}

/// Serialize signature parameters to compressed bytes
pub fn serialize_signature_params(params: &BbsSignatureParams) -> Result<Vec<u8>> {
    let mut bytes = Vec::new();
    params.serialize_compressed(&mut bytes)
        .map_err(|e| CoreError::CryptoError(format!("Failed to serialize signature params: {}", e)))?;
    Ok(bytes)
}

/// Deserialize signature parameters from compressed bytes
pub fn deserialize_signature_params(bytes: &[u8]) -> Result<BbsSignatureParams> {
    BbsSignatureParams::deserialize_compressed(bytes)
        .map_err(|e| CoreError::CryptoError(format!("Failed to deserialize signature params: {}", e)))
}
