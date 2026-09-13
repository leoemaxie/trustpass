use ark_bls12_381::Bls12_381;
use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};
use bbs_plus::setup::PublicKeyG2;
use crate::error::{CoreError, Result};

/// Multicodec prefix for BLS12-381 G2 public key (0xeb, 0x01 in varint encoding)
pub const BLS12_381_G2_PUB_MULTICODEC: [u8; 2] = [0xeb, 0x01];
pub const DID_KEY_PREFIX: &str = "did:key:";

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DidKey {
    pub did: String,
    pub public_key: PublicKeyG2<Bls12_381>,
}

impl DidKey {
    /// Derives a did:key string from a BLS12-381 G2 public key
    pub fn from_public_key(public_key: &PublicKeyG2<Bls12_381>) -> Result<Self> {
        let mut key_bytes = Vec::new();
        public_key
            .serialize_compressed(&mut key_bytes)
            .map_err(|e| CoreError::CryptoError(format!("Failed to serialize public key: {}", e)))?;

        // Prepend multicodec prefix: 0xeb, 0x01
        let mut prefixed = Vec::with_capacity(BLS12_381_G2_PUB_MULTICODEC.len() + key_bytes.len());
        prefixed.extend_from_slice(&BLS12_381_G2_PUB_MULTICODEC);
        prefixed.extend_from_slice(&key_bytes);

        // Multibase base58btc prefix 'z'
        let b58 = bs58::encode(prefixed).into_string();
        let did = format!("{}z{}", DID_KEY_PREFIX, b58);

        Ok(Self {
            did,
            public_key: public_key.clone(),
        })
    }

    /// Parses a did:key string into a BLS12-381 G2 public key
    pub fn parse(did_str: &str) -> Result<Self> {
        if !did_str.starts_with(DID_KEY_PREFIX) {
            return Err(CoreError::InvalidDid(format!(
                "DID must start with '{}', got: '{}'",
                DID_KEY_PREFIX, did_str
            )));
        }

        let encoded = &did_str[DID_KEY_PREFIX.len()..];
        if !encoded.starts_with('z') {
            return Err(CoreError::InvalidDid(format!(
                "DID multibase prefix must be 'z' (base58btc), got: '{}'",
                encoded
            )));
        }

        let raw_b58 = &encoded[1..];
        let decoded = bs58::decode(raw_b58)
            .into_vec()
            .map_err(|e| CoreError::InvalidDid(format!("Base58 decode failed: {}", e)))?;

        if decoded.len() < BLS12_381_G2_PUB_MULTICODEC.len() {
            return Err(CoreError::InvalidDid("DID payload too short".to_string()));
        }

        if &decoded[..2] != BLS12_381_G2_PUB_MULTICODEC {
            return Err(CoreError::InvalidDid(format!(
                "Expected BLS12-381 G2 multicodec [0xeb, 0x01], got: [{:#x}, {:#x}]",
                decoded[0], decoded[1]
            )));
        }

        let key_bytes = &decoded[2..];
        let public_key = PublicKeyG2::<Bls12_381>::deserialize_compressed(key_bytes)
            .map_err(|e| CoreError::CryptoError(format!("Failed to deserialize public key: {}", e)))?;

        Ok(Self {
            did: did_str.to_string(),
            public_key,
        })
    }

    pub fn did(&self) -> &str {
        &self.did
    }

    pub fn public_key(&self) -> &PublicKeyG2<Bls12_381> {
        &self.public_key
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::bbs::keys::{generate_keypair_with_seed, generate_signature_params_with_seed};

    #[test]
    fn test_did_key_derivation_and_parsing() {
        let params = generate_signature_params_with_seed(5, 1234);
        let keypair = generate_keypair_with_seed(&params, 5678);
        let did_key = DidKey::from_public_key(&keypair.public_key).expect("derivation should succeed");

        assert!(did_key.did().starts_with("did:key:z"));
        let parsed = DidKey::parse(did_key.did()).expect("parsing should succeed");
        assert_eq!(did_key.did(), parsed.did());
        assert_eq!(did_key.public_key(), parsed.public_key());
    }

    #[test]
    fn test_invalid_did() {
        assert!(DidKey::parse("did:example:123").is_err());
        assert!(DidKey::parse("did:key:notbase58").is_err());
        assert!(DidKey::parse("did:key:zShort").is_err());
    }
}
