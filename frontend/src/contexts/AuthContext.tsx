import React, { createContext, useContext, useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import { User, LoginRequest } from '@/lib/api/types';
// import { fingerprintCollector } from '@/services/fingerprintCollector'; // 🚨 DESABILITADO TEMPORARIAMENTE

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => void;
  refreshAuth: () => Promise<void>;
  updateUser: (userData: User) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

interface AuthProviderProps {
  children: React.ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const queryClient = useQueryClient();

  useEffect(() => {
    // Check if user is already logged in
    const token = localStorage.getItem('access_token');
    if (token) {
      // Try to refresh token to validate it
      refreshAuth()
        .then(() => {
          // If refresh succeeds and user is already authenticated, start fingerprinting
          setTimeout(() => {
            // fingerprintCollector.enableAutoCollection(); // 🚨 DESABILITADO TEMPORARIAMENTE
          }, 1000);
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, []);

  const login = async (credentials: LoginRequest) => {
    try {
      const response = await apiClient.login(credentials);
      setUser(response.user);
      
      // Collect fingerprint after successful login
      setTimeout(() => {
        // fingerprintCollector.enableAutoCollection(); // 🚨 DESABILITADO TEMPORARIAMENTE
      }, 1000); // Wait 1 second before starting collection
    } catch (error) {
      throw error;
    }
  };

  const logout = () => {
    apiClient.logout();
    setUser(null);
    // Reset fingerprint session on logout
    // fingerprintCollector.resetSession(); // 🚨 DESABILITADO TEMPORARIAMENTE
    // Clear all cached queries to prevent data leakage between tenants
    queryClient.clear();
  };

  const refreshAuth = async () => {
    try {
      const response = await apiClient.refreshToken();
      setUser(response.user);
    } catch (error) {
      // If refresh fails, logout
      logout();
      throw error;
    }
  };

  const updateUser = (userData: User) => {
    setUser(userData);
  };

  const value: AuthContextType = {
    user,
    isLoading,
    isAuthenticated: !!user,
    login,
    logout,
    refreshAuth,
    updateUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
