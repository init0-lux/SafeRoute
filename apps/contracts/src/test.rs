extern crate std;

use super::{DataKey, EvidenceRecord, SafeRouteContract, SafeRouteContractClient};
use soroban_sdk::{
    symbol_short,
    testutils::{Address as _, AuthorizedFunction, AuthorizedInvocation, Events, Ledger},
    vec, Address, Env, IntoVal, String, Symbol,
};

fn soroban_string(env: &Env, value: &str) -> String {
    String::from_str(env, value)
}

#[test]
fn log_evidence_saves_record_emits_event_and_records_relayer_auth() {
    let env = Env::default();
    let relayer = Address::generate(&env);
    let contract_id = env.register(SafeRouteContract, (&relayer,));
    let client = SafeRouteContractClient::new(&env, &contract_id);
    let report_id = soroban_string(&env, "REP123");
    let hash = soroban_string(
        &env,
        "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    );

    env.ledger().set_timestamp(1_710_000_000);
    env.mock_all_auths();

    client.log_evidence(&report_id, &hash);

    let expected_record = EvidenceRecord {
        report_id: report_id.clone(),
        hash: hash.clone(),
        timestamp: 1_710_000_000,
    };

    assert_eq!(
        env.events().all(),
        vec![
            &env,
            (
                contract_id.clone(),
                vec![
                    &env,
                    symbol_short!("evidence").into_val(&env),
                    symbol_short!("logged").into_val(&env),
                    report_id.clone().into_val(&env),
                ],
                expected_record.clone().into_val(&env),
            )
        ]
    );

    let expected_auth = AuthorizedInvocation {
        function: AuthorizedFunction::Contract((
            contract_id.clone(),
            Symbol::new(&env, "log_evidence"),
            (report_id.clone(), hash).into_val(&env),
        )),
        sub_invocations: std::vec![],
    };

    assert_eq!(env.auths(), std::vec![(relayer, expected_auth)]);
    assert_eq!(client.get_evidence(&report_id), expected_record);
}

#[test]
fn update_trust_score_persists_value_and_emits_event() {
    let env = Env::default();
    let relayer = Address::generate(&env);
    let contract_id = env.register(SafeRouteContract, (&relayer,));
    let client = SafeRouteContractClient::new(&env, &contract_id);
    let user_id = soroban_string(&env, "USER42");
    let score = 87_u32;

    env.mock_all_auths();

    client.update_trust_score(&user_id, &score);

    assert_eq!(
        env.events().all(),
        vec![
            &env,
            (
                contract_id.clone(),
                vec![
                    &env,
                    symbol_short!("trust").into_val(&env),
                    symbol_short!("updated").into_val(&env),
                    user_id.clone().into_val(&env),
                ],
                score.into_val(&env),
            )
        ]
    );

    let stored_score: u32 = env.as_contract(&contract_id, || {
        env.storage()
            .persistent()
            .get(&DataKey::TrustScore(user_id.clone()))
            .unwrap()
    });

    assert_eq!(stored_score, score);
}

#[test]
#[should_panic]
fn log_evidence_requires_relayer_authorization() {
    let env = Env::default();
    let relayer = Address::generate(&env);
    let contract_id = env.register(SafeRouteContract, (&relayer,));
    let client = SafeRouteContractClient::new(&env, &contract_id);

    client.log_evidence(
        &soroban_string(&env, "REP999"),
        &soroban_string(
            &env,
            "abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
        ),
    );
}

#[test]
#[should_panic]
fn log_evidence_rejects_non_opaque_report_identifier() {
    let env = Env::default();
    let relayer = Address::generate(&env);
    let contract_id = env.register(SafeRouteContract, (&relayer,));
    let client = SafeRouteContractClient::new(&env, &contract_id);

    env.mock_all_auths();

    client.log_evidence(
        &soroban_string(&env, "report@example.com"),
        &soroban_string(
            &env,
            "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        ),
    );
}

#[test]
#[should_panic]
fn log_evidence_rejects_non_sha256_hashes() {
    let env = Env::default();
    let relayer = Address::generate(&env);
    let contract_id = env.register(SafeRouteContract, (&relayer,));
    let client = SafeRouteContractClient::new(&env, &contract_id);

    env.mock_all_auths();

    client.log_evidence(
        &soroban_string(&env, "REP1000"),
        &soroban_string(&env, "not-a-sha256"),
    );
}

#[test]
#[should_panic]
fn log_evidence_rejects_duplicate_report_ids() {
    let env = Env::default();
    let relayer = Address::generate(&env);
    let contract_id = env.register(SafeRouteContract, (&relayer,));
    let client = SafeRouteContractClient::new(&env, &contract_id);
    let report_id = soroban_string(&env, "REP2000");
    let hash = soroban_string(
        &env,
        "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
    );

    env.mock_all_auths();

    client.log_evidence(&report_id, &hash);
    client.log_evidence(&report_id, &hash);
}

#[test]
#[should_panic]
fn update_trust_score_rejects_non_opaque_user_identifier() {
    let env = Env::default();
    let relayer = Address::generate(&env);
    let contract_id = env.register(SafeRouteContract, (&relayer,));
    let client = SafeRouteContractClient::new(&env, &contract_id);

    env.mock_all_auths();

    client.update_trust_score(&soroban_string(&env, "user@example.com"), &42);
}

#[test]
#[should_panic]
fn update_trust_score_rejects_scores_above_100() {
    let env = Env::default();
    let relayer = Address::generate(&env);
    let contract_id = env.register(SafeRouteContract, (&relayer,));
    let client = SafeRouteContractClient::new(&env, &contract_id);

    env.mock_all_auths();

    client.update_trust_score(&soroban_string(&env, "USER200"), &101);
}
