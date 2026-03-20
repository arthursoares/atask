use dioxus::prelude::*;
use crate::api::client::ApiClient;

#[derive(Clone)]
pub struct AuthState {
    pub token: Signal<Option<String>>,
    pub api: Signal<ApiClient>,
}
