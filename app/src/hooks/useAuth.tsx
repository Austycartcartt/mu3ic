import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react';

import {
  login as apiLogin,
  register as apiRegister,
  setAuthToken,
  setUnauthorizedHandler,
  type AuthResponse,
} from '@/api/client';
import { clearAuth, loadAuth, saveAuth } from '@/api/tokenStore';

// Auth state as a context, mirroring PlayerProvider (see usePlayer.tsx):
// one provider near the root, a typed hook that throws if used outside it.
type User = { id: number; email: string };

type AuthContextValue = {
  user: User | null;
  token: string | null;
  // true only during the initial "read the persisted token" step on
  // launch — screens wait on this before deciding login vs. app.
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const logout = useCallback(async () => {
    setToken(null);
    setUser(null);
    setAuthToken(null);
    await clearAuth();
  }, []);

  // apply pushes a fresh login/register result into state, the module-level
  // token the api client uses, and persistent storage.
  const apply = useCallback(async (res: AuthResponse) => {
    const u = { id: res.id, email: res.email };
    setToken(res.token);
    setUser(u);
    setAuthToken(res.token);
    await saveAuth({ token: res.token, user: u });
  }, []);

  const login = useCallback(
    async (email: string, password: string) => {
      await apply(await apiLogin(email, password));
    },
    [apply]
  );

  const register = useCallback(
    async (email: string, password: string) => {
      await apply(await apiRegister(email, password));
    },
    [apply]
  );

  // Restore a persisted session on launch, and wire the api client's
  // "got a 401" callback to log out (token expired mid-session).
  useEffect(() => {
    setUnauthorizedHandler(() => {
      void logout();
    });
    loadAuth()
      .then((persisted) => {
        if (persisted) {
          setToken(persisted.token);
          setUser(persisted.user);
          setAuthToken(persisted.token);
        }
      })
      .finally(() => setIsLoading(false));
    return () => setUnauthorizedHandler(null);
  }, [logout]);

  return (
    <AuthContext.Provider value={{ user, token, isLoading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
