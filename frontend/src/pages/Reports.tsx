import { useState } from "react";
import { BarChart3, TrendingUp, TrendingDown, DollarSign, Users, Calendar, Loader2, CalendarDays, ArrowUp, ArrowDown, Minus, Package, ShoppingCart, AlertCircle } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useReportsData, useTopProducts } from "@/lib/api/hooks";
import { format, subDays, startOfMonth, endOfMonth } from "date-fns";
import { ptBR } from "date-fns/locale";

export default function Reports() {
  const [dateRange, setDateRange] = useState({
    from: subDays(new Date(), 30), // Padrão de 30 dias
    to: new Date()
  });
  const [isDatePickerOpen, setIsDatePickerOpen] = useState(false);

  // Parâmetros para dados do período selecionado (KPIs)
  const reportsParams = {
    type: "revenue" as const,
    period: "daily" as const,
    start_date: format(dateRange.from, "yyyy-MM-dd"),
    end_date: format(dateRange.to, "yyyy-MM-dd")
  };
  
  // Parâmetros para dados mensais (gráfico - sempre últimos 6 meses)
  const monthlyParams = {
    type: "revenue" as const,
    period: "monthly" as const,
  };

  // Parâmetros para top products com filtro de período
  const topProductsParams = {
    limit: 10,
    start_date: format(dateRange.from, "yyyy-MM-dd"),
    end_date: format(dateRange.to, "yyyy-MM-dd")
  };

  const { data: reportsData, isLoading: reportsLoading, error: reportsError } = useReportsData(reportsParams);
  const { data: monthlyData } = useReportsData(monthlyParams);
  const { data: topProductsData, isLoading: productsLoading } = useTopProducts(topProductsParams);

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: 'BRL',
    }).format(value);
  };

  const formatNumber = (value: number) => {
    return new Intl.NumberFormat('pt-BR').format(value);
  };

  const formatDateRange = () => {
    return `${format(dateRange.from, "dd/MM/yyyy", { locale: ptBR })} - ${format(dateRange.to, "dd/MM/yyyy", { locale: ptBR })}`;
  };

  const selectPeriod = (days: number) => {
    const to = new Date();
    const from = subDays(to, days);
    setDateRange({ from, to });
    setIsDatePickerOpen(false);
  };

  const selectCurrentMonth = () => {
    const now = new Date();
    setDateRange({
      from: startOfMonth(now),
      to: endOfMonth(now)
    });
    setIsDatePickerOpen(false);
  };

  if (reportsLoading || productsLoading) {
    return (
      <div className="space-y-6">
        {/* Header com shimmer */}
        <div className="flex items-center justify-between">
          <div>
            <div className="h-8 w-48 bg-muted animate-pulse rounded-md mb-2"></div>
            <div className="h-4 w-72 bg-muted animate-pulse rounded-md"></div>
          </div>
          <div className="h-10 w-40 bg-muted animate-pulse rounded-md"></div>
        </div>

        {/* KPIs com shimmer */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {[...Array(4)].map((_, i) => (
            <Card key={i} className="border-0 shadow-custom-md">
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <div className="h-4 w-24 bg-muted animate-pulse rounded-md"></div>
                  <div className="h-4 w-4 bg-muted animate-pulse rounded-md"></div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="h-8 w-32 bg-muted animate-pulse rounded-md mb-2"></div>
                <div className="h-3 w-20 bg-muted animate-pulse rounded-md"></div>
              </CardContent>
            </Card>
          ))}
        </div>

        {/* Loading indicator central */}
        <div className="flex items-center justify-center py-12">
          <div className="flex items-center gap-3">
            <Loader2 className="w-6 h-6 animate-spin text-primary" />
            <span className="text-muted-foreground">Carregando relatórios...</span>
          </div>
        </div>
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

  // Extrair dados da resposta da API (mantendo compatibilidade com estrutura anterior)
  const actualReportsData = Array.isArray(reportsData) ? reportsData : 
                            (reportsData as any)?.data?.monthly || 
                            (reportsData as any)?.data || 
                            [];
  const comparisonMetadata = (reportsData as any)?.comparison;
  
  const salesData = Array.isArray(monthlyData) ? monthlyData : 
                   (monthlyData as any)?.data?.monthly || 
                   (monthlyData as any)?.data || 
                   []; // Para o gráfico de vendas mensais
  const topProducts = Array.isArray(topProductsData) ? topProductsData : [];
  
  // KPIs baseados nos dados do período selecionado
  const periodData = Array.isArray(actualReportsData) ? actualReportsData[0] : null;
  const totalRevenue = periodData?.revenue || 0;
  const totalOrders = periodData?.orders || 0;
  const totalCustomers = periodData?.customers || 0;
  const totalProductsSold = periodData?.products_sold || 0;
  const averageTicket = totalOrders > 0 ? totalRevenue / totalOrders : 0;

  // Função para renderizar tendência baseada em metadados de comparação
  const renderComparisonInfo = () => {
    if (!comparisonMetadata) {
      return <span className="text-xs text-muted-foreground">Carregando...</span>;
    }

    if (!comparisonMetadata.has_sufficient_data) {
      return (
        <span className="text-xs text-amber-600 flex items-center gap-1">
          <AlertCircle className="w-3 h-3" />
          {comparisonMetadata.message}
        </span>
      );
    }

    return (
      <span className="text-xs text-muted-foreground">
        {comparisonMetadata.message}
      </span>
    );
  };

  // Para tenants com dados suficientes, mostrar crescimento neutro por enquanto
  // TODO: Implementar cálculo real de crescimento quando a API estiver pronta
  const neutralTrend = { percentage: 0, trend: 'neutral' as const };
  const revenueTrend = neutralTrend;
  const ordersTrend = neutralTrend;
  const customersTrend = neutralTrend;
  const ticketTrend = neutralTrend;

  const renderTrendIcon = (trend: 'up' | 'down' | 'neutral') => {
    switch (trend) {
      case 'up':
        return <ArrowUp className="w-3 h-3 text-success" />;
      case 'down':
        return <ArrowDown className="w-3 h-3 text-destructive" />;
      default:
        return <Minus className="w-3 h-3 text-muted-foreground" />;
    }
  };

  const renderTrendText = (trend: { percentage: number; trend: 'up' | 'down' | 'neutral' }) => {
    // Se não há dados suficientes, mostrar informação simples
    if (!comparisonMetadata?.has_sufficient_data) {
      return (
        <div className="text-xs text-amber-600 flex items-center gap-1">
          <AlertCircle className="w-3 h-3" />
          <span>Dados insuficientes</span>
        </div>
      );
    }

    // Se há dados suficientes mas não há mudança significativa
    if (trend.trend === 'neutral') {
      return (
        <div className="text-xs text-muted-foreground flex items-center gap-1">
          <Minus className="w-3 h-3" />
          <span>sem mudanças</span>
        </div>
      );
    }

    const color = trend.trend === 'up' ? 'text-success' : 'text-destructive';
    const text = comparisonMetadata?.comparison_period || 'período anterior';
    
    return (
      <div className={`text-xs flex items-center gap-1 ${color}`}>
        {renderTrendIcon(trend.trend)}
        <span>{trend.percentage > 0 ? `${trend.percentage.toFixed(1)}% vs ${text}` : `vs ${text}`}</span>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Relatórios</h1>
          <p className="text-muted-foreground">Análise completa do desempenho da sua empresa</p>
        </div>
        <div className="flex items-center gap-3">
          <Popover open={isDatePickerOpen} onOpenChange={setIsDatePickerOpen}>
            <PopoverTrigger asChild>
              <Button variant="outline" size="sm">
                <Calendar className="w-4 h-4 mr-2" />
                {formatDateRange()}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-4" align="end">
              <div className="space-y-3">
                <div className="space-y-2">
                  <p className="text-sm font-medium">Período</p>
                  <div className="grid grid-cols-2 gap-2">
                    <Button variant="outline" size="sm" onClick={() => selectPeriod(7)}>
                      7 dias
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => selectPeriod(30)}>
                      30 dias
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => selectPeriod(90)}>
                      90 dias
                    </Button>
                    <Button variant="outline" size="sm" onClick={selectCurrentMonth}>
                      Este mês
                    </Button>
                  </div>
                </div>
              </div>
            </PopoverContent>
          </Popover>
          {/* Botão temporariamente escondido
          <Button className="bg-gradient-primary">
            Exportar PDF
          </Button>
          */}
        </div>
      </div>

      {/* Key Performance Indicators */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground">Receita Total</CardTitle>
              <DollarSign className="w-4 h-4 text-success" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">{formatCurrency(totalRevenue)}</div>
            {renderTrendText(revenueTrend)}
          </CardContent>
        </Card>
        
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground">Total de Pedidos</CardTitle>
              <ShoppingCart className="w-4 h-4 text-primary" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">{formatNumber(totalOrders)}</div>
            {renderTrendText(ordersTrend)}
          </CardContent>
        </Card>
        
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground">Clientes Únicos</CardTitle>
              <Users className="w-4 h-4 text-whatsapp" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">{formatNumber(totalCustomers)}</div>
            {renderTrendText(customersTrend)}
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
            {renderTrendText(ticketTrend)}
          </CardContent>
        </Card>
      </div>

      {/* Additional KPIs Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground">Produtos Vendidos</CardTitle>
              <Package className="w-4 h-4 text-orange-500" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">{formatNumber(totalProductsSold)}</div>
            <p className="text-xs text-muted-foreground">Unidades no período</p>
          </CardContent>
        </Card>
        
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground">Taxa de Conversão</CardTitle>
              <TrendingUp className="w-4 h-4 text-blue-500" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">
              {totalCustomers > 0 ? `${((totalOrders / totalCustomers) * 100).toFixed(1)}%` : '0%'}
            </div>
            <p className="text-xs text-muted-foreground">Pedidos por cliente</p>
          </CardContent>
        </Card>
        
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground">Itens por Pedido</CardTitle>
              <BarChart3 className="w-4 h-4 text-purple-500" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-foreground">
              {totalOrders > 0 ? (totalProductsSold / totalOrders).toFixed(1) : '0'}
            </div>
            <p className="text-xs text-muted-foreground">Média de produtos</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Monthly Sales Chart */}
        <Card className="border-0 shadow-custom-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="w-5 h-5 text-primary" />
              Evolução Mensal
            </CardTitle>
            <CardDescription>
              Desempenho das vendas nos últimos meses
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {salesData && salesData.length > 0 ? (
                <>
                  {/* Resumo das vendas mensais */}
                  <div className="grid grid-cols-2 gap-4 p-4 bg-accent/20 rounded-lg">
                    <div>
                      <p className="text-sm text-muted-foreground">Melhor Mês</p>
                      <p className="font-bold text-success">
                        {salesData.reduce((prev, current) => (prev.revenue > current.revenue) ? prev : current).period}
                      </p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Crescimento</p>
                      <p className="font-bold text-primary">
                        {salesData.length >= 2 ? 
                          `${(((salesData[0].revenue - salesData[1].revenue) / salesData[1].revenue) * 100).toFixed(1)}%` : 
                          'N/A'
                        }
                      </p>
                    </div>
                  </div>
                  
                  {/* Lista de meses */}
                  {salesData.map((data, index) => {
                    const isCurrentMonth = index === 0;
                    const maxRevenue = Math.max(...salesData.map(d => d.revenue));
                    const barWidth = (data.revenue / maxRevenue) * 100;
                    
                    return (
                      <div key={index} className={`relative p-4 rounded-lg border ${isCurrentMonth ? 'border-primary bg-primary/5' : 'border-border'}`}>
                        {/* Barra de progresso visual */}
                        <div className="absolute left-0 top-0 h-full bg-primary/10 rounded-lg transition-all duration-300" 
                             style={{ width: `${barWidth}%` }}></div>
                        
                        <div className="relative flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <div className={`w-3 h-3 rounded-full ${isCurrentMonth ? 'bg-primary' : 'bg-muted-foreground'}`}></div>
                            <div>
                              <div className="flex items-center gap-2">
                                <p className={`font-medium ${isCurrentMonth ? 'text-primary' : 'text-foreground'}`}>
                                  {format(new Date(data.period + '-01'), 'MMMM yyyy', { locale: ptBR })}
                                </p>
                                {isCurrentMonth && <Badge variant="outline" className="text-xs">Atual</Badge>}
                              </div>
                              <p className="text-sm text-muted-foreground">{data.orders} pedidos • {data.customers} clientes</p>
                            </div>
                          </div>
                          <div className="text-right">
                            <p className={`font-bold ${isCurrentMonth ? 'text-primary' : 'text-foreground'}`}>
                              {formatCurrency(data.revenue)}
                            </p>
                            <p className="text-sm text-muted-foreground">
                              {formatCurrency(data.orders > 0 ? data.revenue / data.orders : 0)} /pedido
                            </p>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <BarChart3 className="w-12 h-12 mx-auto mb-3 opacity-50" />
                  <p>Nenhum dado de vendas disponível</p>
                </div>
              )}
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
              Ranking dos produtos com melhor performance no período
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {topProducts && topProducts.length > 0 ? (
                <>
                  {/* Resumo dos produtos */}
                  <div className="grid grid-cols-2 gap-4 p-4 bg-accent/20 rounded-lg">
                    <div>
                      <p className="text-sm text-muted-foreground">Total de Produtos</p>
                      <p className="font-bold text-primary">{topProducts.length}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Receita Top 3</p>
                      <p className="font-bold text-success">
                        {formatCurrency(topProducts.slice(0, 3).reduce((sum, p) => sum + p.total_revenue, 0))}
                      </p>
                    </div>
                  </div>
                  
                  {/* Lista de produtos */}
                  {topProducts.map((product, index) => {
                    const isTop3 = index < 3;
                    const badgeVariant = index === 0 ? 'default' : index === 1 ? 'secondary' : index === 2 ? 'outline' : 'secondary';
                    const maxRevenue = Math.max(...topProducts.map(p => p.total_revenue));
                    const barWidth = (product.total_revenue / maxRevenue) * 100;
                    
                    return (
                      <div key={product.product_id} className={`relative p-4 rounded-lg border ${isTop3 ? 'border-success/30 bg-success/5' : 'border-border'}`}>
                        {/* Barra de progresso visual */}
                        <div className="absolute left-0 top-0 h-full bg-success/10 rounded-lg transition-all duration-300" 
                             style={{ width: `${barWidth}%` }}></div>
                        
                        <div className="relative flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <Badge variant={badgeVariant} className="min-w-[2rem] justify-center">
                              #{index + 1}
                            </Badge>
                            <div className="min-w-0 flex-1">
                              <div className="flex items-center gap-2">
                                <p className={`font-medium truncate ${isTop3 ? 'text-success' : 'text-foreground'}`}>
                                  {product.product_name}
                                </p>
                                {index === 0 && <Badge variant="outline" className="text-xs">🏆 Líder</Badge>}
                              </div>
                              <p className="text-sm text-muted-foreground">
                                {product.sales_count} vendas • {formatCurrency(product.total_revenue / product.sales_count)} por unidade
                              </p>
                            </div>
                          </div>
                          <div className="text-right">
                            <p className={`font-bold ${isTop3 ? 'text-success' : 'text-foreground'}`}>
                              {formatCurrency(product.total_revenue)}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {((product.total_revenue / topProducts.reduce((sum, p) => sum + p.total_revenue, 0)) * 100).toFixed(1)}% do total
                            </p>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <Package className="w-12 h-12 mx-auto mb-3 opacity-50" />
                  <p>Nenhum produto encontrado no período</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
