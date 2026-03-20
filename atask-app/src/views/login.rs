use dioxus::prelude::*;

use crate::api::client::ApiClient;
use crate::state::credentials;

/// Signal shared between LoginView and App to communicate login success.
/// LoginView writes the token here; App observes it.
#[derive(Clone, Copy)]
pub struct LoginToken(pub Signal<Option<String>>);

#[component]
pub fn LoginView() -> Element {
    let mut api: Signal<ApiClient> = use_context();
    let mut login_token: LoginToken = use_context();

    let mut email = use_signal(|| String::new());
    let mut password = use_signal(|| String::new());
    let mut name = use_signal(|| String::new());
    let mut error = use_signal(|| Option::<String>::None);
    let mut loading = use_signal(|| false);
    let mut show_register = use_signal(|| false);

    let mut do_login = move |email_val: String, password_val: String| {
        loading.set(true);
        error.set(None);
        spawn(async move {
            let api_clone = api.read().clone();
            println!("[LOGIN] Calling API at {}", api_clone.base_url());
            match api_clone.login(&email_val, &password_val).await {
                Ok(tok) => {
                    println!("[LOGIN] Success, saving token");
                    credentials::save(&credentials::Credentials {
                        token: Some(tok.clone()),
                        api_url: None,
                    });
                    api.write().set_token(tok.clone());
                    login_token.0.set(Some(tok));
                }
                Err(e) => {
                    println!("[LOGIN] Error: {e}");
                    error.set(Some(format!("Login failed: {e}")));
                }
            }
            loading.set(false);
        });
    };

    let on_login = move |_| {
        let email_val = email.read().clone();
        let password_val = password.read().clone();
        if email_val.is_empty() || password_val.is_empty() {
            error.set(Some("Email and password are required.".to_string()));
            return;
        }
        do_login(email_val, password_val);
    };

    let on_register = move |_| {
        let email_val = email.read().clone();
        let password_val = password.read().clone();
        let name_val = name.read().clone();
        if email_val.is_empty() || password_val.is_empty() || name_val.is_empty() {
            error.set(Some("All fields are required.".to_string()));
            return;
        }
        loading.set(true);
        error.set(None);
        spawn(async move {
            let api_clone = api.read().clone();
            match api_clone.register(&email_val, &password_val, &name_val).await {
                Ok(()) => {
                    // Auto-login after registration
                    match api_clone.login(&email_val, &password_val).await {
                        Ok(tok) => {
                            api.write().set_token(tok.clone());
                            login_token.0.set(Some(tok));
                        }
                        Err(e) => {
                            error.set(Some(format!("Registered but login failed: {e}")));
                        }
                    }
                }
                Err(e) => {
                    error.set(Some(format!("Registration failed: {e}")));
                }
            }
            loading.set(false);
        });
    };

    let is_loading = *loading.read();
    let is_register = *show_register.read();

    rsx! {
        div { class: "login-container",
            div { class: "login-card",
                div { class: "login-title",
                    if is_register { "Create Account" } else { "Sign In" }
                }

                if let Some(err) = error.read().as_ref() {
                    div { class: "login-error", "{err}" }
                }

                if is_register {
                    div { class: "login-field",
                        label { class: "login-label", "Name" }
                        input {
                            class: "input",
                            r#type: "text",
                            placeholder: "Your name",
                            value: "{name}",
                            oninput: move |e: Event<FormData>| name.set(e.value()),
                        }
                    }
                }

                div { class: "login-field",
                    label { class: "login-label", "Email" }
                    input {
                        class: "input",
                        r#type: "email",
                        placeholder: "you@example.com",
                        value: "{email}",
                        oninput: move |e: Event<FormData>| email.set(e.value()),
                    }
                }

                div { class: "login-field",
                    label { class: "login-label", "Password" }
                    input {
                        class: "input",
                        r#type: "password",
                        placeholder: "Password",
                        value: "{password}",
                        oninput: move |e: Event<FormData>| password.set(e.value()),
                        onkeydown: move |e: Event<KeyboardData>| {
                            if e.key() == Key::Enter && !is_register {
                                let email_val = email.read().clone();
                                let password_val = password.read().clone();
                                if !email_val.is_empty() && !password_val.is_empty() {
                                    do_login(email_val, password_val);
                                }
                            }
                        },
                    }
                }

                if is_register {
                    button {
                        class: "btn btn-primary btn-lg login-btn",
                        disabled: is_loading,
                        onclick: on_register,
                        if is_loading { "Creating account..." } else { "Create Account" }
                    }
                } else {
                    button {
                        class: "btn btn-primary btn-lg login-btn",
                        disabled: is_loading,
                        onclick: on_login,
                        if is_loading { "Signing in..." } else { "Sign In" }
                    }
                }

                div { class: "login-toggle",
                    if is_register {
                        span {
                            "Already have an account? "
                            span {
                                class: "login-link",
                                onclick: move |_| {
                                    show_register.set(false);
                                    error.set(None);
                                },
                                "Sign in"
                            }
                        }
                    } else {
                        span {
                            "No account? "
                            span {
                                class: "login-link",
                                onclick: move |_| {
                                    show_register.set(true);
                                    error.set(None);
                                },
                                "Register"
                            }
                        }
                    }
                }
            }
        }
    }
}
