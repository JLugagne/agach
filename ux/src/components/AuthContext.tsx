import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import { getToken, setToken, clearToken, login as apiLogin, logout as apiLogout, type LoginResponse } from '../lib/auth';
import { authEvents } from '../lib/api';

interface AuthUser {
  id: string;
  email: string;
  display_name: string;
  role: string;
}

interface AuthContextValue {
  user: AuthUser | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);

  // Restore session from stored token (we store the user alongside it).
  useEffect(() => {
    const stored = localStorage.getItem('agach_user');
    if (stored && getToken()) {
      try {
        setUser(JSON.parse(stored));
      } catch {
        clearToken();
      }
    }
  }, []);

  // Listen for API 401 events — clear session and reload to show login.
  useEffect(() => {
    const handler = () => {
      setUser(null);
      clearToken();
      localStorage.removeItem('agach_user');
    };
    authEvents.addEventListener('unauthorized', handler);
    return () => authEvents.removeEventListener('unauthorized', handler);
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const data: LoginResponse = await apiLogin(email, password);
    setToken(data.access_token);
    localStorage.setItem('agach_user', JSON.stringify(data.user));
    setUser(data.user);
  }, []);

  const logout = useCallback(async () => {
    await apiLogout();
    setUser(null);
    localStorage.removeItem('agach_user');
  }, []);

  return (
    <AuthContext.Provider value={{ user, isAuthenticated: !!user, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider');
  return ctx;
}
