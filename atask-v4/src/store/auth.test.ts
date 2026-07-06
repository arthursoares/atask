import { describe, it, expect, beforeEach } from 'vitest';
import { $authState, $isAuthenticated, $currentUser, UNAUTHENTICATED_STATE } from './auth';

/**
 * Unit tests for the auth store computed atoms.
 *
 * SECURITY: `AuthState` intentionally has no token field — the auth token
 * never crosses the Tauri IPC boundary (see src-tauri/src/auth.rs). These
 * tests double as a guard: they only ever construct/read the profile shape
 * (userId/userEmail/userName/serverUrl/authenticated), so any future change
 * that tries to widen the type to carry a token would show up as a diff
 * here.
 */
describe('auth store', () => {
  beforeEach(() => {
    $authState.set({ ...UNAUTHENTICATED_STATE });
  });

  it('defaults to unauthenticated with no profile fields', () => {
    const state = $authState.get();
    expect(state.authenticated).toBe(false);
    expect(state.userId).toBeNull();
    expect(state.userEmail).toBeNull();
    expect(state.userName).toBeNull();
    expect(state.serverUrl).toBeNull();
    expect($isAuthenticated.get()).toBe(false);
    expect($currentUser.get()).toBeNull();
  });

  it('has no token field on the AuthState shape', () => {
    const state = $authState.get();
    expect(state).not.toHaveProperty('token');
    expect(state).not.toHaveProperty('accessToken');
  });

  it('derives $isAuthenticated and $currentUser from a logged-in state', () => {
    $authState.set({
      authenticated: true,
      userId: 'u1',
      userEmail: 'arthur@example.com',
      userName: 'Arthur',
      serverUrl: 'https://api.atask.app',
    });

    expect($isAuthenticated.get()).toBe(true);
    expect($currentUser.get()).toEqual({
      id: 'u1',
      email: 'arthur@example.com',
      name: 'Arthur',
    });
  });

  it('reverts to unauthenticated after a sign-out reset', () => {
    $authState.set({
      authenticated: true,
      userId: 'u1',
      userEmail: 'arthur@example.com',
      userName: 'Arthur',
      serverUrl: 'https://api.atask.app',
    });
    $authState.set({ ...UNAUTHENTICATED_STATE });

    expect($isAuthenticated.get()).toBe(false);
    expect($currentUser.get()).toBeNull();
  });
});
