#![no_std]

use soroban_sdk::{contract, contractimpl, contracttype, symbol_short, Address, Env, String};

const SHA256_HEX_LEN: u32 = 64;
const MIN_IDENTIFIER_LEN: u32 = 3;
const MAX_IDENTIFIER_LEN: u32 = 64;

#[derive(Clone, Debug, Eq, PartialEq)]
#[contracttype]
pub struct EvidenceRecord {
    pub report_id: String,
    pub hash: String,
    pub timestamp: u64,
}

#[derive(Clone)]
#[contracttype]
enum DataKey {
    Relayer,
    Evidence(String),
    TrustScore(String),
}

#[contract]
pub struct SafeRouteContract;

#[contractimpl]
impl SafeRouteContract {
    pub fn __constructor(env: Env, relayer: Address) {
        env.storage().instance().set(&DataKey::Relayer, &relayer);
    }

    pub fn log_evidence(env: Env, report_id: String, hash: String) {
        require_relayer_auth(&env);
        validate_identifier(&report_id, "report");
        validate_sha256_hex(&hash);

        let evidence_key = DataKey::Evidence(report_id.clone());
        if env.storage().persistent().has(&evidence_key) {
            panic!("evidence already logged");
        }

        let record = EvidenceRecord {
            report_id: report_id.clone(),
            hash,
            timestamp: env.ledger().timestamp(),
        };

        env.storage().persistent().set(&evidence_key, &record);
        env.events().publish(
            (symbol_short!("evidence"), symbol_short!("logged"), report_id),
            record,
        );
    }

    pub fn update_trust_score(env: Env, user_id: String, new_score: u32) {
        require_relayer_auth(&env);
        validate_identifier(&user_id, "user");

        if new_score > 100 {
            panic!("trust score out of range");
        }

        env.storage()
            .persistent()
            .set(&DataKey::TrustScore(user_id.clone()), &new_score);
        env.events().publish(
            (symbol_short!("trust"), symbol_short!("updated"), user_id),
            new_score,
        );
    }

    pub fn get_evidence(env: Env, report_id: String) -> EvidenceRecord {
        let evidence_key = DataKey::Evidence(report_id);
        env.storage()
            .persistent()
            .get(&evidence_key)
            .unwrap_or_else(|| panic!("evidence not found"))
    }
}

fn require_relayer_auth(env: &Env) {
    let relayer: Address = env
        .storage()
        .instance()
        .get(&DataKey::Relayer)
        .unwrap_or_else(|| panic!("relayer not configured"));
    relayer.require_auth();
}

// Only allow opaque backend-generated IDs on-chain so PII like emails, phone
// numbers, or free-form names never become ledger data or event topics.
fn validate_identifier(value: &String, field_name: &str) {
    let len = value.len();
    if !(MIN_IDENTIFIER_LEN..=MAX_IDENTIFIER_LEN).contains(&len) {
        panic!("invalid identifier length");
    }

    let mut bytes = [0u8; MAX_IDENTIFIER_LEN as usize];
    value.copy_into_slice(&mut bytes[..len as usize]);

    for byte in &bytes[..len as usize] {
        if !is_allowed_identifier_byte(*byte) {
            panic!("identifier contains unsupported characters");
        }
    }

    if contains_disallowed_pii_pattern(&bytes[..len as usize]) {
        if field_name == "report" {
            panic!("report identifier must be opaque");
        }
        panic!("user identifier must be opaque");
    }
}

fn validate_sha256_hex(hash: &String) {
    if hash.len() != SHA256_HEX_LEN {
        panic!("hash must be a 64 character SHA-256 hex digest");
    }

    let mut bytes = [0u8; SHA256_HEX_LEN as usize];
    hash.copy_into_slice(&mut bytes);

    for byte in &bytes {
        if !is_hex_byte(*byte) {
            panic!("hash must be hex encoded");
        }
    }
}

fn is_allowed_identifier_byte(byte: u8) -> bool {
    byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'_' | b':')
}

fn is_hex_byte(byte: u8) -> bool {
    byte.is_ascii_hexdigit()
}

fn contains_disallowed_pii_pattern(bytes: &[u8]) -> bool {
    bytes.contains(&b'@')
        || bytes.contains(&b' ')
        || bytes.contains(&b'+')
        || bytes.contains(&b'.')
}

#[cfg(test)]
mod test;
