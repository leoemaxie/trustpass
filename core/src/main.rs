use std::env;
use trustpass_core::server::{run_server, CoreState};

fn main() {
    let port: u16 = env::var("PORT_CORE")
        .unwrap_or_else(|_| "50051".to_string())
        .parse()
        .unwrap_or(50051);

    let seed: u64 = env::var("DEFAULT_ISSUER_SEED")
        .unwrap_or_else(|_| "13374242".to_string())
        .parse()
        .unwrap_or(13374242);

    let state = CoreState::new_seeded(seed);
    println!("TrustPass Cryptographic Core initialized with Issuer DID: {}", state.issuer_did);
    run_server(port, state);
}
