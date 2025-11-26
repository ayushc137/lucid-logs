use crate::config::Settings;
use std::sync::Arc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

#[derive(Clone)]
pub struct AppState {
    pub db: Surreal<Client>,
    pub settings: Arc<Settings>,
}

impl AppState {
    pub fn new(db: Surreal<Client>, settings: Arc<Settings>) -> Self {
        Self { db, settings }
    }
}
