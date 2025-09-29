import { useAuth } from "@/contexts/AuthContext";
import { Navigate } from "react-router-dom";
import { ReactNode } from "react";

interface AdminProtectedRouteProps {
  children: ReactNode;
  requiredRole?: 'system_admin' | 'tenant_admin';
}

export default function AdminProtectedRoute({ 
  children, 
  requiredRole = 'system_admin' 
}: AdminProtectedRouteProps) {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  if (user.role !== requiredRole) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-foreground mb-2">Acesso Negado</h2>
          <p className="text-muted-foreground">
            Você não tem permissão para acessar esta página.
          </p>
          <p className="text-sm text-muted-foreground mt-2">
            Apenas {requiredRole === 'system_admin' ? 'administradores do sistema' : 'administradores do tenant'} podem acessar esta área.
          </p>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
