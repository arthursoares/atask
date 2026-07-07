import { atom, computed } from 'nanostores';

// IMPORTANT: No token field. The auth token never crosses the Tauri IPC
// boundary into the frontend — it lives only in the Rust-side AuthTokens
// state and the OS keychain (see src-tauri/src/auth.rs). The frontend tracks
// identity via `authenticated` (a boolean) and the user profile fields,
// which are safe to expose.
//
// Field shape matches the Rust `AuthState` struct (auth.rs), serialized as
// camelCase over the Tauri IPC boundary.
export interface AuthState {
  authenticated: boolean;
  userId: string | null;
  userEmail: string | null;
  userName: string | null;
  serverUrl: string | null;
}

export const UNAUTHENTICATED_STATE: AuthState = {
  authenticated: false,
  userId: null,
  userEmail: null,
  userName: null,
  serverUrl: null,
};

export const $authState = atom<AuthState>({ ...UNAUTHENTICATED_STATE });

export const $isAuthenticated = computed($authState, (s) => s.authenticated);
export const $currentUser = computed($authState, (s) =>
  s.userId ? { id: s.userId, email: s.userEmail, name: s.userName } : null,
);
