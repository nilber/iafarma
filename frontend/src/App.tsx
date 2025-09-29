import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { AuthProvider } from "./contexts/AuthContext";
import { TenantProvider } from "./contexts/TenantContext";
import { SoundNotificationProvider } from "./contexts/SoundNotificationContext";
import ProtectedRoute from "./components/ProtectedRoute";
import AdminProtectedRoute from "./components/AdminProtectedRoute";
import { AppLayout } from "./components/layout/AppLayout";
import Login from "./pages/Login";
import ForgotPassword from "./pages/ForgotPassword";
import ResetPassword from "./pages/ResetPassword";
import Dashboard from "./pages/Dashboard";
import WhatsApp from "./pages/WhatsApp";
import Sales from "./pages/Sales";
import Products from "./pages/Products";
import Categories from "./pages/Categories";
import ProductDetails from "./pages/ProductDetails";
import OrderEditDetails from "./pages/OrderEditDetails";
import Customers from "./pages/Customers";
import CustomerDetails from "./pages/CustomerDetails";
import Reports from "./pages/Reports";
import Settings from "./pages/Settings";
import Profile from "./pages/Profile";
import Tenants from "./pages/admin/Tenants";
import AdminChannels from "./pages/admin/AdminChannels";
import ErrorLogs from "./pages/admin/ErrorLogs";
import NotificationManagement from "./pages/admin/NotificationManagement";
import PlansManagement from "./pages/admin/PlansManagement";
import NotFound from "./pages/NotFound";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error: any) => {
        // Don't retry on 401/403 errors
        if (error?.status === 401 || error?.status === 403) {
          return false;
        }
        return failureCount < 3;
      },
      // Configurações balanceadas para evitar re-renders desnecessários
      staleTime: 1000 * 60 * 5, // 5 minutos - dados são considerados frescos
      gcTime: 1000 * 60 * 10, // 10 minutos - manter cache por mais tempo
      refetchOnWindowFocus: false,
      refetchOnReconnect: true, // Reconectar após perda de conexão é útil
      refetchOnMount: 'always', // Sempre revalidar ao montar
    },
    mutations: {
      gcTime: 1000 * 60 * 1, // REDUZIDO: 1 minuto (era 5)
    },
  },
});

// Cache cleanup será feito automaticamente pelo React Query

// Sistema de limpeza removido - React Query gerencia memória automaticamente

// QueryClient disponível globalmente para debug se necessário
(window as any).queryClient = queryClient;

const App = () => (
  <QueryClientProvider client={queryClient}>
    <AuthProvider>
      <TenantProvider>
        <SoundNotificationProvider>
          <TooltipProvider>
            <Toaster />
            <Sonner />
            <BrowserRouter>
              <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/forgot-password" element={<ForgotPassword />} />
              <Route path="/reset-password" element={<ResetPassword />} />
              <Route path="/*" element={
                <ProtectedRoute>
                  <AppLayout>
                    <Routes>
                      <Route path="/" element={<Dashboard />} />
                      <Route path="/whatsapp/*" element={<WhatsApp />} />
                      <Route path="/sales/*" element={<Sales />} />
                      <Route path="/sales/orders/new" element={<OrderEditDetails />} />
                      <Route path="/sales/orders/:id/edit" element={<OrderEditDetails />} />
                      <Route path="/products" element={<Products />} />
                      <Route path="/products/new" element={<ProductDetails />} />
                      <Route path="/products/:id" element={<ProductDetails />} />
                      <Route path="/categories" element={<Categories />} />
                      <Route path="/customers" element={<Customers />} />
                      <Route path="/customers/:id" element={<CustomerDetails />} />
                      <Route path="/reports" element={<Reports />} />                      
                      <Route path="/settings" element={<Settings />} />
                      <Route path="/profile" element={<Profile />} />                    
                      <Route path="/admin/tenants" element={
                        <AdminProtectedRoute requiredRole="system_admin">
                          <Tenants />
                        </AdminProtectedRoute>
                      } />
                      <Route path="/admin/channels" element={
                        <AdminProtectedRoute requiredRole="system_admin">
                          <AdminChannels />
                        </AdminProtectedRoute>
                      } />
                      <Route path="/admin/error-logs" element={
                        <AdminProtectedRoute requiredRole="system_admin">
                          <ErrorLogs />
                        </AdminProtectedRoute>
                      } />
                      <Route path="/admin/notifications" element={
                        <AdminProtectedRoute requiredRole="system_admin">
                          <NotificationManagement />
                        </AdminProtectedRoute>
                      } />
                      <Route path="/admin/plans" element={
                        <AdminProtectedRoute requiredRole="system_admin">
                          <PlansManagement />
                        </AdminProtectedRoute>
                      } />                   
                      {/* ADD ALL CUSTOM ROUTES ABOVE THE CATCH-ALL "*" ROUTE */}
                      <Route path="*" element={<NotFound />} />
                    </Routes>
                  </AppLayout>
                </ProtectedRoute>
              } />
            </Routes>
          </BrowserRouter>
        </TooltipProvider>
      </SoundNotificationProvider>
      </TenantProvider>
    </AuthProvider>
  </QueryClientProvider>
);

export default App;
