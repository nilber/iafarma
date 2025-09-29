import { useState, useEffect } from 'react';
import { apiClient } from '@/lib/api';
import { Tenant } from '@/lib/api/types';
import { useAuth } from '@/contexts/AuthContext';

export interface TenantContextType {
  tenant: Tenant | null;
  isLoading: boolean;
  isSales: boolean;
  refreshTenant: () => Promise<void>;
}

export function useTenant(): TenantContextType {
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const { user } = useAuth();

  const refreshTenant = async () => {
    try {
      setIsLoading(true);
      
      // Skip tenant API call for system_admin users
      if (user?.role === 'system_admin') {
        setTenant(null);
        return;
      }
      
      const tenantData = await apiClient.getTenantProfile();
      setTenant(tenantData);
    } catch (error) {
      console.error('Error fetching tenant profile:', error);
      setTenant(null);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    // Only fetch tenant data when user is available and not system_admin
    if (user) {
      refreshTenant();
    } else if (user === null) {
      // User is not authenticated or is system_admin
      setIsLoading(false);
    }
  }, [user]);

  const isSales = tenant?.business_type === 'sales' || !tenant?.business_type;
  
  return {
    tenant,
    isLoading,
    isSales,
    refreshTenant,
  };
}
