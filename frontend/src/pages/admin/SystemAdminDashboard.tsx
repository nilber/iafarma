import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Building2, Users, MessageSquare, TrendingUp, Activity, Shield } from "lucide-react";
import { apiClient } from "@/lib/api/client";

interface SystemStats {
  total_tenants: number;
  active_tenants: number;
  total_users: number;
  active_users: number;
  total_messages: number;
  total_conversations: number;
  tenants_by_plan: {
    free: number;
    basic: number;
    premium: number;
    unlimited: number;
  };
  recent_activity: {
    new_tenants_this_month: number;
    new_users_this_month: number;
    messages_this_month: number;
  };
}

export default function SystemAdminDashboard() {
  const { data: stats, isLoading } = useQuery({
    queryKey: ['system-stats'],
    queryFn: () => apiClient.getSystemStats(),
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Dashboard do Sistema</h1>
          <p className="text-muted-foreground">Visão geral do sistema</p>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader className="pb-2">
                <div className="h-4 bg-muted rounded w-3/4"></div>
              </CardHeader>
              <CardContent>
                <div className="h-8 bg-muted rounded w-1/2"></div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  const statCards = [
    {
      title: "Total de Empresas",
      value: stats?.total_tenants || 0,
      subtitle: `${stats?.active_tenants || 0} ativas`,
      icon: Building2,
      color: "text-blue-600",
      bgColor: "bg-blue-100",
    },
    {
      title: "Total de Usuários",
      value: stats?.total_users || 0,
      subtitle: `${stats?.active_users || 0} ativos`,
      icon: Users,
      color: "text-green-600",
      bgColor: "bg-green-100",
    },
    {
      title: "Conversas",
      value: stats?.total_conversations || 0,
      subtitle: "Total do sistema",
      icon: MessageSquare,
      color: "text-purple-600",
      bgColor: "bg-purple-100",
    },
    {
      title: "Mensagens",
      value: stats?.total_messages || 0,
      subtitle: "Total enviadas",
      icon: Activity,
      color: "text-orange-600",
      bgColor: "bg-orange-100",
    },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Dashboard do Sistema</h1>
          <p className="text-muted-foreground">Visão geral e métricas globais</p>
        </div>
        <Badge variant="outline" className="flex items-center gap-2">
          <Shield className="w-4 h-4" />
          System Admin
        </Badge>
      </div>

      {/* Main Stats */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {statCards.map((stat) => (
          <Card key={stat.title} className="hover:shadow-md transition-shadow">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {stat.title}
              </CardTitle>
              <div className={`p-2 rounded-lg ${stat.bgColor}`}>
                <stat.icon className={`w-4 h-4 ${stat.color}`} />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stat.value.toLocaleString()}</div>
              <p className="text-xs text-muted-foreground">{stat.subtitle}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Plans Distribution */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TrendingUp className="w-5 h-5" />
              Distribuição por Planos
            </CardTitle>
            <CardDescription>Empresas por tipo de plano</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {stats?.tenants_by_plan && Object.entries(stats.tenants_by_plan).map(([planName, count]) => {
              // Cores dinâmicas baseadas no nome do plano
              const getColorClass = (name: string) => {
                const lowercaseName = name.toLowerCase();
                if (lowercaseName.includes('gratuito') || lowercaseName.includes('free')) {
                  return "bg-gray-200 text-gray-700";
                } else if (lowercaseName.includes('básico') || lowercaseName.includes('basic')) {
                  return "bg-blue-200 text-blue-700";
                } else if (lowercaseName.includes('essencial') || lowercaseName.includes('premium')) {
                  return "bg-purple-200 text-purple-700";
                } else if (lowercaseName.includes('pro') || lowercaseName.includes('enterprise')) {
                  return "bg-green-200 text-green-700";
                } else {
                  return "bg-orange-200 text-orange-700";
                }
              };

              return (
                <div key={planName} className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Badge 
                      variant="secondary" 
                      className={getColorClass(planName)}
                    >
                      {planName}
                    </Badge>
                  </div>
                  <span className="font-semibold">{String(count)}</span>
                </div>
              );
            })}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="w-5 h-5" />
              Atividade Recente
            </CardTitle>
            <CardDescription>Métricas do último mês</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Novas empresas</span>
              <span className="font-semibold text-green-600">
                +{stats?.recent_activity?.new_tenants_this_month || 0}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Novos usuários</span>
              <span className="font-semibold text-blue-600">
                +{stats?.recent_activity?.new_users_this_month || 0}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Mensagens enviadas</span>
              <span className="font-semibold text-purple-600">
                {(stats?.recent_activity?.messages_this_month || 0).toLocaleString()}
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* System Health */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="w-5 h-5" />
            Status do Sistema
          </CardTitle>
          <CardDescription>Saúde geral da plataforma</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div className="text-center p-4 bg-green-50 rounded-lg">
              <div className="text-2xl font-bold text-green-600">99.9%</div>
              <div className="text-sm text-green-700">Uptime</div>
            </div>
            <div className="text-center p-4 bg-blue-50 rounded-lg">
              <div className="text-2xl font-bold text-blue-600">
                {((stats?.active_tenants || 0) / Math.max(stats?.total_tenants || 1, 1) * 100).toFixed(1)}%
              </div>
              <div className="text-sm text-blue-700">Empresas Ativas</div>
            </div>
            <div className="text-center p-4 bg-purple-50 rounded-lg">
              <div className="text-2xl font-bold text-purple-600">
                {((stats?.active_users || 0) / Math.max(stats?.total_users || 1, 1) * 100).toFixed(1)}%
              </div>
              <div className="text-sm text-purple-700">Usuários Ativos</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
