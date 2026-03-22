import SwiftUI

/// Login sheet — appears when user wants to connect to a server.
struct LoginView: View {
    let api: APIClient
    let onSuccess: (String) -> Void // JWT token
    @Binding var isPresented: Bool

    @State private var email = ""
    @State private var password = ""
    @State private var name = ""
    @State private var isRegistering = false
    @State private var error: String?
    @State private var isLoading = false

    var body: some View {
        VStack(spacing: Spacing.sp4) {
            Text(isRegistering ? "Create Account" : "Sign In")
                .font(.viewTitle)
                .foregroundStyle(Theme.inkPrimary)

            VStack(spacing: Spacing.sp3) {
                if isRegistering {
                    TextField("Name", text: $name)
                        .textFieldStyle(.roundedBorder)
                }
                TextField("Email", text: $email)
                    .textFieldStyle(.roundedBorder)
                    .textContentType(.emailAddress)
                SecureField("Password", text: $password)
                    .textFieldStyle(.roundedBorder)
                    .onSubmit { submit() }
            }

            if let error {
                Text(error)
                    .font(.metadataRegular)
                    .foregroundStyle(Theme.deadlineRed)
            }

            HStack(spacing: Spacing.sp3) {
                Button("Cancel") { isPresented = false }
                    .keyboardShortcut(.escape)

                Button(isRegistering ? "Register" : "Sign In") { submit() }
                    .keyboardShortcut(.return)
                    .disabled(isLoading || email.isEmpty || password.isEmpty)
            }

            Button(isRegistering ? "Already have an account? Sign in" : "Need an account? Register") {
                isRegistering.toggle()
                error = nil
            }
            .font(.metadataRegular)
            .foregroundStyle(Theme.accent)
            .buttonStyle(.plain)
        }
        .padding(Spacing.sp6)
        .frame(width: 320)
    }

    private func submit() {
        guard !isLoading else { return }
        isLoading = true
        error = nil

        Task {
            do {
                let response: APIClient.AuthResponse
                if isRegistering {
                    response = try await api.register(email: email, password: password, name: name)
                } else {
                    response = try await api.login(email: email, password: password)
                }
                onSuccess(response.token)
                isPresented = false
            } catch {
                self.error = error.localizedDescription
            }
            isLoading = false
        }
    }
}
