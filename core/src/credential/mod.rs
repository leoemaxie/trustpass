pub mod did;
pub mod revocation;
pub mod schema;
pub mod vc;

pub use did::DidKey;
pub use revocation::{RevocationRecord, RevocationRegistry};
pub use schema::{AttributeDefinition, AttributeType, CredentialSchema};
pub use vc::{CredentialSubject, VerifiableCredential};
