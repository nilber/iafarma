import { TrendingUp, DollarSign, ShoppingCart, Package, Users, Calendar, Loader2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { useSalesAnalytics, useTopProducts, useRecentSales } from "@/lib/api/hooks";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";

export default function SalesDashboard() {
  const { data: analytics, isLoading: analyticsLoading, error: analyticsError } = useSalesAnalytics();
  const { data: topProductsData, isLoading: productsLoading } = useTopProducts();
  const { data: recentSales, isLoading: salesLoading } = useRecentSales(4);

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: 'BRL',
    }).format(value);
  };

  const formatPercentage = (value: number) => {
    return `${value > 0 ? '+' : ''}${value.toFixed(1)}%`;
  };

  const salesMetrics = analytics ? [
    {
      title: "Receita Total",
      value: formatCurrency(analytics.total_revenue),
      change: formatPercentage(analytics.growth_rate),
      icon: DollarSign,
      color: analytics.growth_rate >= 0 ? "text-success" : "text-destructive",
      bgColor: analytics.growth_rate >= 0 ? "bg-success/10" : "bg-destructive/10"
    },
    {
      title: "Total de Pedidos",
      value: analytics.total_orders.toString(),
      change: formatPercentage(analytics.conversion_rate),
      icon: ShoppingCart,
      color: "text-sales",
      bgColor: "bg-sales/10"
    },
    {
      title: "Ticket Médio",
      value: formatCurrency(analytics.average_ticket),
      change: formatPercentage(analytics.growth_rate / 2), // Estimate
      icon: Package,
      color: "text-primary",
      bgColor: "bg-primary/10"
    },
    {
      title: "Total de Clientes",
      value: analytics.new_customers.toString(),
      change: formatPercentage(analytics.growth_rate),
      icon: Users,
      color: "text-accent",
      bgColor: "bg-accent/10"
    }
  ] : [];

  if (analyticsLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
          <p className="text-muted-foreground">Carregando dados...</p>
        </div>
      </div>
    );
  }

  if (analyticsError) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-destructive mb-2">Erro ao carregar dados</h1>
          <p className="text-muted-foreground">Não foi possível carregar os dados do dashboard.</p>
        </div>
      </div>
    );
  }

  // Use real data from API instead of hardcoded values
  const recentSalesData = recentSales || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Dashboard de Vendas</h1>
          <p className="text-muted-foreground">Acompanhe o desempenho das suas vendas em tempo real</p>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="outline" size="sm">
            <Calendar className="w-4 h-4 mr-2" />
            Hoje
          </Button>
          <Button className="bg-gradient-sales">
            Relatório Completo
          </Button>
        </div>
      </div>

      {/* Metrics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {salesMetrics.map((metric) => (
          <Card key={metric.title} className="border-0 shadow-custom-md">
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <div className={`p-2 rounded-lg ${metric.bgColor}`}>
                  <metric.icon className={`w-5 h-5 ${metric.color}`} />
                </div>
                <Badge variant={metric.change.startsWith('+') ? 'default' : 'secondary'}>
                  {metric.change}
                </Badge>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-1">
                <p className="text-2xl font-bold text-foreground">{metric.value}</p>
                <p className="text-sm text-muted-foreground">{metric.title}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Charts and Recent Sales */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Top Products Chart */}
        <Card className="lg:col-span-2 border-0 shadow-custom-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Package className="w-5 h-5 text-success" />
              Produtos Mais Vendidos
            </CardTitle>
            <CardDescription>Ranking dos produtos com melhor performance</CardDescription>
          </CardHeader>
          <CardContent>
            {productsLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="w-6 h-6 animate-spin" />
              </div>
            ) : (
              <div className="space-y-4">
                {topProductsData && topProductsData.length > 0 ? (
                  topProductsData.map((product, index) => {
                    // Calculate percentage based on the highest selling product
                    const maxSales = Math.max(...topProductsData.map(p => p.sales_count));
                    const salesPercentage = maxSales > 0 ? (product.sales_count / maxSales) * 100 : 0;
                    
                    return (
                    <div key={product.product_id} className="flex items-center gap-4">
                      <div className="flex items-center justify-center w-8 h-8 rounded-full bg-primary/10 text-primary font-bold text-sm">
                        {index + 1}
                      </div>
                      <div className="flex-1">
                        <p className="font-medium text-foreground">{product.product_name}</p>
                        <div className="flex items-center gap-2 mt-1">
                          <p className="text-sm text-muted-foreground">{product.sales_count} vendas</p>
                          <p className="text-sm font-medium text-success">
                            {formatCurrency(product.total_revenue)}
                          </p>
                        </div>
                      </div>
                      <Progress value={salesPercentage} className="w-20" />
                    </div>
                  )})
                ) : (
                  <div className="flex items-center justify-center py-8 text-muted-foreground">
                    <p>Nenhum produto encontrado</p>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent Sales */}
        <Card className="border-0 shadow-custom-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TrendingUp className="w-5 h-5 text-primary" />
              Vendas Recentes
            </CardTitle>
            <CardDescription>Últimas transações realizadas</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {salesLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="w-6 h-6 animate-spin" />
                </div>
              ) : recentSalesData.length > 0 ? (
                recentSalesData.map((sale, index) => (
                  <div key={index} className="flex items-center justify-between p-3 rounded-lg bg-accent/30">
                    <div className="flex-1">
                      <p className="font-medium text-foreground">{sale.customer}</p>
                      <p className="text-sm text-muted-foreground">{sale.product}</p>
                    </div>
                    <div className="text-right">
                      <p className="font-bold text-success">{sale.value}</p>
                      <p className="text-xs text-muted-foreground">{sale.time}</p>
                    </div>
                  </div>
                ))
              ) : (
                <div className="flex items-center justify-center py-8 text-muted-foreground">
                  <p>Nenhuma venda recente encontrada</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Sales Performance */}
      <Card className="border-0 shadow-custom-md">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="w-5 h-5 text-primary" />
            Performance de Vendas
          </CardTitle>
          <CardDescription>Análise detalhada do desempenho mensal</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Meta Mensal</span>
                <span className="text-sm font-medium">R$ 50.000</span>
              </div>
              <Progress value={75} className="h-2" />
              <p className="text-xs text-muted-foreground">75% da meta atingida</p>
            </div>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Conversão</span>
                <span className="text-sm font-medium">{analytics?.conversion_rate.toFixed(1)}%</span>
              </div>
              <Progress value={analytics?.conversion_rate || 0} className="h-2" />
              <p className="text-xs text-muted-foreground">Taxa de conversão atual</p>
            </div>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Retenção</span>
                <span className="text-sm font-medium">68%</span>
              </div>
              <Progress value={68} className="h-2" />
              <p className="text-xs text-muted-foreground">Clientes recorrentes</p>
            </div>
          </div>
          <div className="mt-6 flex justify-center">
            <Button variant="outline" className="w-full md:w-auto">
              <TrendingUp className="w-4 h-4 mr-2" />
              <span className="text-sm">Ver Relatórios</span>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
