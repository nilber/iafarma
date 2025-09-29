import React, { createContext, useContext, useState, useEffect } from 'react';
import { useAuth } from './AuthContext';
import { apiClient } from '@/lib/api';
import { Tenant } from '@/lib/api/types';

interface TenantContextType {
  selectedTenant: Tenant | null;
  tenants: Tenant[];
  isLoadingTenants: boolean;
  setSelectedTenant: (tenant: Tenant | null) => void;
  refreshTenants: () => Promise<void>;
}

const TenantContext = createContext<TenantContextType | undefined>(undefined);

export const useTenant = () => {
  const context = useContext(TenantContext);
  if (context === undefined) {
    throw new Error('useTenant must be used within a TenantProvider');
  }
  return context;
};

interface TenantProviderProps {
  children: React.ReactNode;
}

export const TenantProvider: React.FC<TenantProviderProps> = ({ children }) => {
  const { user, isAuthenticated } = useAuth();
  const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null);
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [isLoadingTenants, setIsLoadingTenants] = useState(false);

  const refreshTenants = async () => {
    if (!isAuthenticated || user?.role !== 'system_admin') {
      return;
    }

    setIsLoadingTenants(true);
    try {
      const response = await apiClient.getTenants({ limit: 100 });
      setTenants(response.data || []);
    } catch (error) {
      console.error('Failed to load tenants:', error);
      setTenants([]);
    } finally {
      setIsLoadingTenants(false);
    }
  };

  useEffect(() => {
    if (isAuthenticated && user?.role === 'system_admin') {
      refreshTenants();
    } else {
      setTenants([]);
      setSelectedTenant(null);
    }
  }, [isAuthenticated, user?.role]);

  // Auto-select first tenant if none selected
  useEffect(() => {
    if (tenants.length > 0 && !selectedTenant) {
      setSelectedTenant(tenants[0]);
    }
  }, [tenants, selectedTenant]);

  const value: TenantContextType = {
    selectedTenant,
    tenants,
    isLoadingTenants,
    setSelectedTenant,
    refreshTenants,
  };

  return (
    <TenantContext.Provider value={value}>
      {children}
    </TenantContext.Provider>
  );
};
