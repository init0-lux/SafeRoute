#![no_std]

use soroban_sdk::{contract, contractimpl, symbol_short, Env, Symbol};

#[contract]
pub struct SafeRouteContract;

#[contractimpl]
impl SafeRouteContract {
    pub fn hello(_env: Env) -> Symbol {
        symbol_short!("hello")
    }
}
