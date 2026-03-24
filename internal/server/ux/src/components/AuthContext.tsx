import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import { getToken, setToken, clearToken, login as apiLogin, logout as apiLogout, type LoginResponse } from '../lib/auth';
import { authEvents } from '../lib/api';
import { wsClient } from '../lib/ws';

interface AuthUser {
  id: string;
  email: string;
  display_name: string;
  role: string;
}

interface AuthContextValue {
  user: AuthUser | null;
  isAuthenticated: boolean;
  login: (email: string, password: string, rememberMe?: boolean) => Promise<void>;
  logout: () => Promise<void>;
  updateUser: (partial: Partial<AuthUser>) => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function restoreUser(): AuthUser | null {
  const stored = localStorage.getItem('agach_user');
  if (stored && getToken()) {
    try {
      return JSON.parse(stored);
    } catch {
      clearToken();
    }
  }
  return null;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(restoreUser);

  // Listen for API 401 events — clear session and reload to show login.
  useEffect(() => {
    const handler = () => {
      setUser(null);
      clearToken();
      localStorage.removeItem('agach_user');
      wsClient.disconnect();
    };
    authEvents.addEventListener('unauthorized', handler);
    return () => authEvents.removeEventListener('unauthorized', handler);
  }, []);

  const login = useCallback(async (email: string, password: string, rememberMe = false) => {
    const data: LoginResponse = await apiLogin(email, password, rememberMe);
    setToken(data.access_token);
    localStorage.setItem('agach_user', JSON.stringify(data.user));
    setUser(data.user);
    wsClient.reset();
    wsClient.connect();
  }, []);

  const updateUser = useCallback((partial: Partial<AuthUser>) => {
    setUser(prev => {
      if (!prev) return prev;
      const updated = { ...prev, ...partial };
      localStorage.setItem('agach_user', JSON.stringify(updated));
      return updated;
    });
  }, []);

  const logout = useCallback(async () => {
    wsClient.disconnect();
    await apiLogout();
    setUser(null);
    localStorage.removeItem('agach_user');
  }, []);

  return (
    <AuthContext.Provider value={{ user, isAuthenticated: !!user, login, logout, updateUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider');
  return ctx;
}
