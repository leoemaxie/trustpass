use std::collections::HashMap;
use std::sync::RwLock;
use chrono::Utc;
use serde::{Deserialize, Serialize};

/// Represents an entry in the issuer's credential revocation list
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct RevocationRecord {
    #[serde(rename = "credentialId")]
    pub credential_id: String,
    #[serde(rename = "revokedAt")]
    pub revoked_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
}

/// Thread-safe in-memory credential revocation registry maintained by the issuer
#[derive(Debug, Default)]
pub struct RevocationRegistry {
    records: RwLock<HashMap<String, RevocationRecord>>,
}

impl RevocationRegistry {
    pub fn new() -> Self {
        Self {
            records: RwLock::new(HashMap::new()),
        }
    }

    /// Marks a credential as revoked with a timestamp and optional reason
    pub fn revoke(&self, credential_id: &str, reason: Option<String>) -> RevocationRecord {
        let record = RevocationRecord {
            credential_id: credential_id.to_string(),
            revoked_at: Utc::now().to_rfc3339(),
            reason,
        };
        let mut map = self.records.write().expect("Revocation registry lock poisoned");
        map.insert(credential_id.to_string(), record.clone());
        record
    }

    /// Checks if a credential ID is currently revoked
    pub fn is_revoked(&self, credential_id: &str) -> bool {
        let map = self.records.read().expect("Revocation registry lock poisoned");
        map.contains_key(credential_id)
    }

    /// Retrieves revocation details for a credential if revoked
    pub fn get(&self, credential_id: &str) -> Option<RevocationRecord> {
        let map = self.records.read().expect("Revocation registry lock poisoned");
        map.get(credential_id).cloned()
    }

    /// Lists all revoked credential records
    pub fn list(&self) -> Vec<RevocationRecord> {
        let map = self.records.read().expect("Revocation registry lock poisoned");
        map.values().cloned().collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_revocation_lifecycle() {
        let registry = RevocationRegistry::new();
        let cred_id = "urn:uuid:11112222-3333-4444-5555-666677778888";

        assert!(!registry.is_revoked(cred_id));
        assert!(registry.get(cred_id).is_none());

        let record = registry.revoke(cred_id, Some("lost".to_string()));
        assert_eq!(record.credential_id, cred_id);
        assert_eq!(record.reason.as_deref(), Some("lost"));

        assert!(registry.is_revoked(cred_id));
        let fetched = registry.get(cred_id).expect("Should find revoked record");
        assert_eq!(fetched, record);
    }
}
