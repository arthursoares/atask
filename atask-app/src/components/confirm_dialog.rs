use dioxus::prelude::*;

#[derive(Clone, PartialEq, Props)]
pub struct ConfirmDialogProps {
    message: String,
    on_confirm: EventHandler<()>,
    on_cancel: EventHandler<()>,
}

#[component]
pub fn ConfirmDialog(props: ConfirmDialogProps) -> Element {
    rsx! {
        div {
            class: "confirm-backdrop",
            onclick: move |_| props.on_cancel.call(()),
            div {
                class: "confirm-dialog",
                onclick: move |evt| evt.stop_propagation(),
                p { class: "confirm-message", "{props.message}" }
                div { class: "confirm-actions",
                    button {
                        class: "btn btn-secondary",
                        onclick: move |_| props.on_cancel.call(()),
                        "Cancel"
                    }
                    button {
                        class: "btn btn-danger",
                        onclick: move |_| props.on_confirm.call(()),
                        "Delete"
                    }
                }
            }
        }
    }
}
