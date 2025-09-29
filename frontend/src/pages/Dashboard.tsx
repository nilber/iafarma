import { MessageSquare, ShoppingCart, Users, TrendingUp, Phone, Package, Loader2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { useCustomers, useProducts, useChannels, useOrders, useRecentSales, useDashboardStats, useTenantUsage } from "@/lib/api/hooks";
import { useAuth } from "@/contexts/AuthContext";
import { useTenant } from "@/hooks/useTenant";
import { useNavigate } from "react-router-dom";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";
import SystemAdminDashboard from "./admin/SystemAdminDashboard";
import UsageDisplay from "@/components/usage/UsageDisplay";
import { OnboardingModal, useOnboarding } from "@/components/onboarding";
import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api/client";

export default function Dashboard() {
  const { user } = useAuth();
  const { tenant, isLoading: tenantLoading } = useTenant();
  const navigate = useNavigate();
  
  // Onboarding state for Sales tenants
  const {
    shouldShowOnboarding,
    onboardingStatus,
    loading: onboardingLoading,
    checkOnboardingStatus,
    dismissOnboarding,
    setShouldShowOnboarding
  } = useOnboarding();
  
  // ALL HOOKS MUST BE CALLED BEFORE ANY CONDITIONAL RETURNS
  const { data: customersResult, isLoading: customersLoading, error: customersError } = useCustomers({ limit: 100 });
  const { data: productsResult, isLoading: productsLoading, error: productsError } = useProducts({ limit: 100 });
  const { data: channelsResult, isLoading: channelsLoading, error: channelsError } = useChannels({ limit: 10 });
  const { data: ordersResult, isLoading: ordersLoading, error: ordersError } = useOrders({ limit: 100 });
  const { data: recentSales, isLoading: salesLoading } = useRecentSales(5);
  const { data: dashboardStats, isLoading: statsLoading } = useDashboardStats();
  const { data: tenantUsage, isLoading: usageLoading, error: usageError } = useTenantUsage();

  // Check onboarding status for Sales and Scheduling tenants when component mounts
  useEffect(() => {
    if (!tenantLoading && (tenant?.business_type === 'sales') && user?.role !== 'system_admin') {
      checkOnboardingStatus();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantLoading, tenant?.business_type, user?.role]);

  // If user is system admin, show system admin dashboard
  if (user?.role === 'system_admin') {
    return <SystemAdminDashboard />;
  }

  // TODO: Create dedicated scheduling dashboard
  // For now, scheduling tenants use the same dashboard as sales
  // if (!tenantLoading && isAgendamento) {
  //   return <SchedulingDashboard />;
  // }

  const isLoading = customersLoading || productsLoading || channelsLoading || ordersLoading || statsLoading;

  // Extract data arrays from paginated results with proper error handling
  const customers = Array.isArray(customersResult?.data) ? customersResult.data : [];
  const products = Array.isArray(productsResult?.data) ? productsResult.data : [];
  const orders = Array.isArray(ordersResult?.data) ? ordersResult.data : [];
  const channels = Array.isArray(channelsResult?.data) ? channelsResult.data : [];

  // Calculate stats from real data
  const activeCustomers = customers.filter(c => c.is_active).length;
  const whatsappChannels = channels.filter(c => c.type === 'whatsapp');
  const activeChannels = whatsappChannels.filter(c => c.status === 'connected');
  
  // Calculate order statistics from real data
  const pendingOrders = orders.filter(o => o.status === 'pending' || o.status === 'processing');
  const totalOrdersValue = orders.reduce((sum, order) => {
    return sum + parseFloat(order.total_amount || '0');
  }, 0);
  
  // Use tenantUsage for accurate channel count (like UsageDisplay does)
  const channelsCount = tenantUsage?.channels_count || activeChannels.length;
  const maxChannels = tenantUsage?.plan?.max_channels || whatsappChannels.length || 1;
  
  const stats = [
    {
      title: "Canais WhatsApp",
      value: `${channelsCount}/${maxChannels}`,
      change: channelsCount === maxChannels ? "Limite" : activeChannels.length === whatsappChannels.length ? "Online" : "Offline",
      icon: MessageSquare,
      color: "text-whatsapp",
      bgColor: "bg-whatsapp/10"
    },
    {
      title: "Total de Vendas",
      value: `R$ ${totalOrdersValue.toFixed(2).replace('.', ',')}`,
      change: `${orders.length} pedidos`,
      icon: TrendingUp,
      color: "text-sales",
      bgColor: "bg-sales/10"
    },
    {
      title: "Pedidos Pendentes",
      value: pendingOrders.length.toString(),
      change: orders.length > 0 ? `${Math.round((pendingOrders.length / orders.length) * 100)}%` : "0%",
      icon: ShoppingCart,
      color: "text-warning",
      bgColor: "bg-warning/10"
    },
    {
      title: "Clientes Ativos",
      value: activeCustomers.toString(),
      change: `${customers.length} total`,
      icon: Users,
      color: "text-primary",
      bgColor: "bg-primary/10"
    }
  ];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-96">
        <Loader2 className="w-8 h-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Dashboard</h1>
          <p className="text-muted-foreground">Visão geral da sua operação</p>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="outline" size="sm">
            <Phone className="w-4 h-4 mr-2" />
            {activeChannels.length > 0 ? 'WhatsApp Online' : 'WhatsApp Offline'}
          </Button>
          <Button className="bg-gradient-primary" onClick={() => navigate('/reports')}>
            Ver Relatórios
          </Button>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {stats.map((stat) => (
          <Card key={stat.title} className="overflow-hidden border-0 shadow-custom-md">
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <div className={`p-2 rounded-lg ${stat.bgColor}`}>
                  <stat.icon className={`w-5 h-5 ${stat.color}`} />
                </div>
                <Badge variant="secondary" className="text-xs">
                  {stat.change}
                </Badge>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-1">
                <p className="text-2xl font-bold text-foreground">{stat.value}</p>
                <p className="text-sm text-muted-foreground">{stat.title}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Usage Display */}
      <UsageDisplay 
        usage={tenantUsage} 
        loading={usageLoading} 
        error={usageError?.message} 
      />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Recent Orders */}
        <Card className="border-0 shadow-custom-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ShoppingCart className="w-5 h-5 text-sales" />
              Pedidos Recentes ({recentSales?.length || 0})
            </CardTitle>
            <CardDescription>
              Últimos pedidos realizados na plataforma
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {salesLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="w-6 h-6 animate-spin" />
                </div>
              ) : recentSales && recentSales.length > 0 ? (
                recentSales.slice(0, 4).map((sale) => (
                  <div 
                    key={sale.id} 
                    className="flex items-center justify-between p-3 rounded-lg bg-accent/50 hover:bg-accent/70 transition-colors cursor-pointer"
                    onClick={() => navigate(`/sales/orders/${sale.order_id}`)}
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-3">
                        <span className="font-mono text-sm text-muted-foreground">
                          #{sale.order_number || sale.order_id.substring(0, 8)}
                        </span>
                        <Badge variant={
                          sale.status === 'completed' || sale.status === 'delivered' ? 'default' :
                          sale.status === 'processing' ? 'secondary' :
                          sale.status === 'cancelled' ? 'destructive' : 'outline'
                        }>
                          {sale.status === 'completed' ? 'Concluído' :
                           sale.status === 'processing' ? 'Processando' :
                           sale.status === 'pending' ? 'Pendente' :
                           sale.status === 'cancelled' ? 'Cancelado' :
                           sale.status || 'Pendente'}
                        </Badge>
                      </div>
                      <p className="font-medium text-foreground">
                        Cliente: {sale.customer}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="font-bold text-foreground">
                        {sale.value}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {sale.time}
                      </p>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-center py-8">
                  <p className="text-muted-foreground">Nenhum pedido encontrado</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* WhatsApp Channels */}
        <Card className="border-0 shadow-custom-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MessageSquare className="w-5 h-5 text-whatsapp" />
              Canais WhatsApp ({channels.length})
            </CardTitle>
            <CardDescription>
              Status dos canais de atendimento
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {channels.length > 0 ? channels.slice(0, 4).map((channel) => (
                <div key={channel.id} className="flex items-center gap-3 p-3 rounded-lg bg-accent/50 hover:bg-accent transition-colors">
                  <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                    channel.status === 'connected' ? 'bg-success text-success-foreground' :
                    channel.status === 'connecting' ? 'bg-warning text-warning-foreground' :
                    'bg-destructive text-destructive-foreground'
                  }`}>
                    <MessageSquare className="w-5 h-5" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between">
                      <p className="font-medium text-foreground truncate">{channel.name}</p>
                      <span className="text-xs text-muted-foreground">
                        {format(new Date(channel.updated_at), 'HH:mm', { locale: ptBR })}
                      </span>
                    </div>
                    <p className="text-sm text-muted-foreground truncate">
                      Tipo: {channel.type} • Status: {
                        channel.status === 'connected' ? 'Conectado' :
                        channel.status === 'connecting' ? 'Conectando' :
                        channel.status === 'disconnected' ? 'Desconectado' :
                        channel.status || 'Desconhecido'
                      } • {channel.conversation_count || 0} conversas
                    </p>
                  </div>
                  <Badge variant={
                    channel.status === 'connected' ? 'default' :
                    channel.status === 'connecting' ? 'secondary' :
                    'destructive'
                  }>
                    {channel.is_active ? 'Ativo' : 'Inativo'}
                  </Badge>
                </div>
              )) : (
                <div className="text-center py-8">
                  <p className="text-muted-foreground">Nenhum canal configurado</p>
                  <p className="text-xs text-muted-foreground mt-2">
                    Configure um canal WhatsApp para começar
                  </p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Performance Overview */}
      <Card className="border-0 shadow-custom-md">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="w-5 h-5 text-primary" />
            Visão Geral da Plataforma
          </CardTitle>
          <CardDescription>
            Estatísticas da base de dados atual
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Clientes Ativos</span>
                <span className="font-medium">
                  {dashboardStats?.active_customers || 0} / {dashboardStats?.total_customers || 0}
                </span>
              </div>
              <Progress 
                value={
                  dashboardStats?.total_customers && dashboardStats.total_customers > 0 
                    ? (dashboardStats.active_customers / dashboardStats.total_customers) * 100 
                    : 0
                } 
                className="h-3" 
              />
              <p className="text-xs text-muted-foreground">
                {dashboardStats?.total_customers && dashboardStats.total_customers > 0 
                  ? Math.round((dashboardStats.active_customers / dashboardStats.total_customers) * 100) 
                  : 0}% dos clientes estão ativos
              </p>
            </div>            
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Canais Online</span>
                <span className="font-medium">
                  {dashboardStats?.connected_channels || 0} / {dashboardStats?.total_channels || 0}
                </span>
              </div>
              <Progress 
                value={
                  dashboardStats?.total_channels && dashboardStats.total_channels > 0 
                    ? (dashboardStats.connected_channels / dashboardStats.total_channels) * 100 
                    : 0
                } 
                className="h-3" 
              />
              <p className="text-xs text-muted-foreground">
                {dashboardStats?.total_channels && dashboardStats.total_channels > 0 
                  ? Math.round((dashboardStats.connected_channels / dashboardStats.total_channels) * 100) 
                  : 0}% dos canais online
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Onboarding Modal for Sales Tenants */}
      <OnboardingModal
        isOpen={shouldShowOnboarding}
        onClose={() => setShouldShowOnboarding(false)}
        onDismiss={dismissOnboarding}
        status={onboardingStatus}
        loading={onboardingLoading}
      />
    </div>
  );
}