import type React from "react";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

import { API_BASE_URL } from "@/types/api";

interface User {
  username: string;
  role: string;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  isAuthEnabled: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  checkAuth: () => Promise<boolean>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const TOKEN_KEY = "vps-monitor_auth_token";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isAuthEnabled, setIsAuthEnabled] = useState(true);

  // Check if authentication is enabled on the backend
  const checkIfAuthEnabled = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ username: "", password: "" }),
      });

      // If we get a 404, auth is disabled. Any other response means auth is enabled
      if (response.status === 404) {
        setIsAuthEnabled(false);
      }
    } catch (error) {
      // If the request fails, assume auth is enabled to be safe
      console.error("Failed to check auth status:", error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const verifyToken = useCallback(async (tokenToVerify: string) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/auth/me`, {
        headers: {
          Authorization: `Bearer ${tokenToVerify}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        setUser(data.user);
        setToken(tokenToVerify);
        setIsAuthEnabled(true);
      } else if (response.status === 404) {
        // Auth endpoint doesn't exist - auth is disabled
        setIsAuthEnabled(false);
        localStorage.removeItem(TOKEN_KEY);
        setToken(null);
        setUser(null);
      } else {
        // Token is invalid, clear it
        localStorage.removeItem(TOKEN_KEY);
        setToken(null);
        setUser(null);
      }
    } catch (error) {
      console.error("Failed to verify token:", error);
      localStorage.removeItem(TOKEN_KEY);
      setToken(null);
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    const storedToken = localStorage.getItem(TOKEN_KEY);
    if (storedToken) {
      setToken(storedToken);
      verifyToken(storedToken);
    } else {
      checkIfAuthEnabled();
    }
  }, [verifyToken, checkIfAuthEnabled]);

  // Check authentication status
  const checkAuth = async (): Promise<boolean> => {
    const storedToken = localStorage.getItem(TOKEN_KEY);
    if (!storedToken) {
      return false;
    }

    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/auth/me`, {
        headers: {
          Authorization: `Bearer ${storedToken}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        setUser(data.user);
        setToken(storedToken);
        return true;
      }

      // Token is invalid
      localStorage.removeItem(TOKEN_KEY);
      setToken(null);
      setUser(null);
      return false;
    } catch (error) {
      console.error("Failed to check auth:", error);
      return false;
    }
  };

  const login = async (username: string, password: string) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || "Login failed");
      }

      const data = await response.json();

      // Store token and user
      localStorage.setItem(TOKEN_KEY, data.token);
      setToken(data.token);
      setUser(data.user);
    } catch (error) {
      console.error("Login error:", error);
      throw error;
    }
  };

  const logout = () => {
    localStorage.removeItem(TOKEN_KEY);
    setToken(null);
    setUser(null);
  };

  const value: AuthContextType = {
    user,
    token,
    isAuthenticated: !!token && !!user,
    isLoading,
    isAuthEnabled,
    login,
    logout,
    checkAuth,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
