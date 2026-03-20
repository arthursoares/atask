use dioxus::prelude::*;
use crate::state::app::{TokenSignal, ApiSignal};
use crate::state::credentials;

#[component]
pub fn LoginView() -> Element {
    let mut api: ApiSignal = use_context();
    let mut token: TokenSignal = use_context();

    let mut email = use_signal(|| String::new());
    let mut password = use_signal(|| String::new());
    let mut name = use_signal(|| String::new());
    let mut error = use_signal(|| Option::<String>::None);
    let mut loading = use_signal(|| false);
    let mut show_register = use_signal(|| false);

    let do_login = move |email_val: String, password_val: String| {
        loading.set(true);
        error.set(None);
        spawn(async move {
            let api_clone = api.0.read().clone();
            match api_clone.login(&email_val, &password_val).await {
                Ok(tok) => {
                    // Save to disk
                    credentials::save(&credentials::Credentials {
                        token: Some(tok.clone()),
                        api_url: None,
                    });
                    // Update shared state — NEWTYPE .0 access
                    api.0.write().set_token(tok.clone());
                    token.0.set(Some(tok));
                }
                Err(e) => {
                    error.set(Some(format!("Login failed: {e}")));
                }
            }
            loading.set(false);
        });
    };

    let do_register = move |name_val: String, email_val: String, password_val: String| {
        loading.set(true);
        error.set(None);
        spawn(async move {
            let api_clone = api.0.read().clone();
            match api_clone.register(&email_val, &password_val, &name_val).await {
                Ok(()) => {
                    // Auto-login after successful registration
                    match api_clone.login(&email_val, &password_val).await {
                        Ok(tok) => {
                            credentials::save(&credentials::Credentials {
                                token: Some(tok.clone()),
                                api_url: None,
                            });
                            api.0.write().set_token(tok.clone());
                            token.0.set(Some(tok));
                        }
                        Err(e) => {
                            error.set(Some(format!("Login after register failed: {e}")));
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

    rsx! {
        div { class: "login-container",
            div { class: "login-card",
                h1 { class: "login-title",
                    if *show_register.read() { "Create Account" } else { "Sign In" }
                }

                if let Some(ref err) = *error.read() {
                    div { class: "login-error", "{err}" }
                }

                if *show_register.read() {
                    div { class: "login-field",
                        label { class: "login-label", "Name" }
                        input {
                            class: "input",
                            r#type: "text",
                            placeholder: "Your name",
                            value: "{name.read()}",
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
                        value: "{email.read()}",
                        oninput: move |e: Event<FormData>| email.set(e.value()),
                    }
                }

                div { class: "login-field",
                    label { class: "login-label", "Password" }
                    input {
                        class: "input",
                        r#type: "password",
                        placeholder: "Password",
                        value: "{password.read()}",
                        oninput: move |e: Event<FormData>| password.set(e.value()),
                        onkeypress: {
                            let mut do_login = do_login.clone();
                            let mut do_register = do_register.clone();
                            move |e: Event<KeyboardData>| {
                                if e.key() == Key::Enter && !*loading.read() {
                                    let email_val = email.read().clone();
                                    let password_val = password.read().clone();
                                    if *show_register.read() {
                                        let name_val = name.read().clone();
                                        do_register(name_val, email_val, password_val);
                                    } else {
                                        do_login(email_val, password_val);
                                    }
                                }
                            }
                        },
                    }
                }

                button {
                    class: "btn btn-primary btn-lg login-btn",
                    disabled: *loading.read(),
                    onclick: {
                        let mut do_login = do_login.clone();
                        let mut do_register = do_register.clone();
                        move |_| {
                            if !*loading.read() {
                                let email_val = email.read().clone();
                                let password_val = password.read().clone();
                                if *show_register.read() {
                                    let name_val = name.read().clone();
                                    do_register(name_val, email_val, password_val);
                                } else {
                                    do_login(email_val, password_val);
                                }
                            }
                        }
                    },
                    if *loading.read() { "Please wait..." }
                    else if *show_register.read() { "Create Account" }
                    else { "Sign In" }
                }

                div { class: "login-toggle",
                    if *show_register.read() {
                        span {
                            "Already have an account? "
                            span {
                                class: "login-link",
                                onclick: move |_| show_register.set(false),
                                "Sign in"
                            }
                        }
                    } else {
                        span {
                            "Don't have an account? "
                            span {
                                class: "login-link",
                                onclick: move |_| show_register.set(true),
                                "Register"
                            }
                        }
                    }
                }
            }
        }
    }
}
