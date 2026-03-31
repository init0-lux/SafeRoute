import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { getAccessToken } from '@/services/api';
import * as authService from '@/services/auth';
import type { User } from '@/services/auth';

// ── Types ────────────────────────────────────────────────────────────────────

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (phone: string, password: string) => Promise<void>;
  register: (phone: string, password: string, email?: string) => Promise<void>;
  logout: () => Promise<void>;
}

// ── Context ──────────────────────────────────────────────────────────────────

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// ── Provider ─────────────────────────────────────────────────────────────────

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Check for existing session on mount
  useEffect(() => {
    async function restoreSession() {
      try {
        const token = await getAccessToken();
        if (token) {
          const me = await authService.getMe();
          setUser(me);
        }
      } catch {
        // Token invalid or expired — user stays logged out
        setUser(null);
      } finally {
        setIsLoading(false);
      }
    }
    restoreSession();
  }, []);

  const login = useCallback(async (phone: string, password: string) => {
    const loggedInUser = await authService.login(phone, password);
    setUser(loggedInUser);
  }, []);

  const register = useCallback(async (phone: string, password: string, email?: string) => {
    const registeredUser = await authService.register(phone, password, email);
    setUser(registeredUser);
  }, []);

  const logout = useCallback(async () => {
    await authService.logout();
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: user !== null,
        isLoading,
        login,
        register,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

// ── Hook ─────────────────────────────────────────────────────────────────────

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
