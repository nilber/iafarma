import { BarChart3, TrendingUp, DollarSign, Users, Calendar, Loader2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useReportsData, useTopProducts } from "@/lib/api/hooks";

export default function Reports() {
  const { data: reportsData, isLoading: reportsLoading, error: reportsError } = useReportsData({
    type: "revenue",
    period: "monthly"
  });
  const { data: topProductsData, isLoading: productsLoading } = useTopProducts();

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: 'BRL',
    }).format(value);
  };

  const formatNumber = (value: number) => {
    return new Intl.NumberFormat('pt-BR').format(value);
  };

  if (reportsLoading || productsLoading) {
    return (
      <div className="flex items-center justify-center h-96">
        <Loader2 className="w-8 h-8 animate-spin" />
      </div>
    );
  }

  if (reportsError) {
    return (
      <div className="flex items-center justify-center h-96">
        <p className="text-muted-foreground">Erro ao carregar relatórios</p>
      </div>
    );
  }

  const salesData = reportsData || [];
  const topProducts = topProductsData || [];

  // Calculate totals from the monthly data
  const totalRevenue = salesData.reduce((sum, item) => sum + item.revenue, 0);
  const totalOrders = salesData.reduce((sum, item) => sum + item.orders, 0);
  const totalCustomers = salesData.reduce((sum, item) => sum + item.customers, 0);
  const averageTicket = totalOrders > 0 ? totalRevenue / totalOrders : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Relatórios</h1>
          <p className="text-muted-foreground">Análise completa do desempenho da sua empresa</p>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="outline" size="sm">
            <Calendar className="w-4 h-4 mr-2" />
            Período
          </Button>
          <Button className="bg-gradient-primary">
            Exportar PDF
          </Button>
        </div>
      </div>

      {/* Key Performance Indicators */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground">Receita Total</CardTitle>
              <DollarSign className="w-4 h-4 text-success" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">{formatCurrency(totalRevenue)}</div>
            <p className="text-sm text-success">Período analisado</p>
          </CardContent>
        </Card>
        
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground">Total de Pedidos</CardTitle>
              <TrendingUp className="w-4 h-4 text-primary" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">{formatNumber(totalOrders)}</div>
            <p className="text-sm text-success">Pedidos processados</p>
          </CardContent>
        </Card>
        
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground">Total de Clientes</CardTitle>
              <Users className="w-4 h-4 text-whatsapp" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">{formatNumber(totalCustomers)}</div>
            <p className="text-sm text-success">Clientes atendidos</p>
          </CardContent>
        </Card>
        
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground">Ticket Médio</CardTitle>
              <BarChart3 className="w-4 h-4 text-secondary" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">{formatCurrency(averageTicket)}</div>
            <p className="text-sm text-success">Por pedido</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Monthly Sales Chart */}
        <Card className="border-0 shadow-custom-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="w-5 h-5 text-primary" />
              Vendas Mensais
            </CardTitle>
            <CardDescription>
              Evolução das vendas ao longo dos meses
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {salesData.map((data, index) => (
                <div key={index} className="flex items-center justify-between p-3 rounded-lg border">
                  <div className="flex items-center gap-3">
                    <div className="w-3 h-3 rounded-full bg-primary"></div>
                    <div>
                      <p className="font-medium text-foreground">{data.period}</p>
                      <p className="text-sm text-muted-foreground">{data.orders} pedidos</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-bold text-foreground">{formatCurrency(data.revenue)}</p>
                    <p className="text-sm text-muted-foreground">{data.customers} clientes</p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Top Products */}
        <Card className="border-0 shadow-custom-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TrendingUp className="w-5 h-5 text-success" />
              Produtos Mais Vendidos
            </CardTitle>
            <CardDescription>
              Ranking dos produtos com melhor performance
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {topProducts.map((product, index) => (
                <div key={product.product_id} className="flex items-center justify-between p-3 rounded-lg bg-accent/30">
                  <div className="flex items-center gap-3">
                    <Badge variant={index < 3 ? 'default' : 'secondary'}>
                      #{index + 1}
                    </Badge>
                    <div className="min-w-0 flex-1">
                      <p className="font-medium text-foreground truncate">{product.product_name}</p>
                      <p className="text-sm text-muted-foreground">{product.sales_count} vendas</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-bold text-success">{formatCurrency(product.total_revenue)}</p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
