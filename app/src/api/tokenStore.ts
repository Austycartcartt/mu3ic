// Persists the auth token + user across app launches. Uses expo-secure-store
// on native (Keychain / Keystore); SecureStore has no web implementation, so
// web falls back to localStorage — an accepted trade-off for a personal app
// that already passes the token as a URL query param for streaming.

import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

const KEY = 'mu3ic.auth';

export type PersistedAuth = {
  token: string;
  user: { id: number; email: string };
};

export async function loadAuth(): Promise<PersistedAuth | null> {
  try {
    const raw =
      Platform.OS === 'web'
        ? globalThis.localStorage?.getItem(KEY) ?? null
        : await SecureStore.getItemAsync(KEY);
    return raw ? (JSON.parse(raw) as PersistedAuth) : null;
  } catch {
    // Corrupt value, storage unavailable, private-mode web, etc. — treat as
    // "not logged in" rather than crashing the launch.
    return null;
  }
}

export async function saveAuth(auth: PersistedAuth): Promise<void> {
  const raw = JSON.stringify(auth);
  if (Platform.OS === 'web') {
    globalThis.localStorage?.setItem(KEY, raw);
  } else {
    await SecureStore.setItemAsync(KEY, raw);
  }
}

export async function clearAuth(): Promise<void> {
  if (Platform.OS === 'web') {
    globalThis.localStorage?.removeItem(KEY);
  } else {
    await SecureStore.deleteItemAsync(KEY);
  }
}
