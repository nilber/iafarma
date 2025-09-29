import { useState, useEffect, useCallback, useMemo } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { 
  ShoppingCart, 
  Plus, 
  Search, 
  Eye, 
  Edit, 
  Truck,
  Loader2,
  ChevronLeft,
  ChevronRight
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useOrders, useOrderStats } from "@/lib/api/hooks";
import { formatCurrency, formatDate, formatDateTime } from "@/lib/utils";
import { OrderWithCustomer, OrderStats } from "@/lib/api/types";
import { apiClient } from "@/lib/api/client";
import { useQueryClient } from "@tanstack/react-query";
import { OrderFiltersModal, OrderFilters } from "@/components/orders/OrderFiltersModal";
import { useDebounce } from "@/hooks/useDebounce";

export default function SalesOrders() {
  const [searchParams, setSearchParams] = useSearchParams();
  const customerIdParam = searchParams.get('customer_id');
  
  const [searchTerm, setSearchTerm] = useState("");
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage] = useState(20);
  const [filters, setFilters] = useState<OrderFilters>({
    customer_id: customerIdParam || undefined,
  });
  
  const [confirmShipment, setConfirmShipment] = useState<{
    show: boolean;
    orderId: string;
    orderNumber: string;
  }>({ show: false, orderId: "", orderNumber: "" });
  
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // Debounce search to avoid too many API calls
  const debouncedSearchTerm = useDebounce(searchTerm, 500);

  // Build query parameters for API call
  const queryParams = useMemo(() => {
    const params = {
      limit: itemsPerPage,
      offset: (currentPage - 1) * itemsPerPage,
      search: debouncedSearchTerm || undefined,
      ...filters,
    };
    return params;
  }, [itemsPerPage, currentPage, debouncedSearchTerm, filters]);

  // Reset to page 1 when search or filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [debouncedSearchTerm, filters]);

  // Update URL when customer filter changes
  useEffect(() => {
    if (customerIdParam && customerIdParam !== filters.customer_id) {
      setFilters(prev => ({ ...prev, customer_id: customerIdParam }));
    }
  }, [customerIdParam, filters.customer_id]);
  
  // Handle view order - navigate to details page instead of modal
  const handleViewOrder = (orderId: string) => {
    navigate(`/sales/orders/${orderId}`);
  };
  
  // Handle create new order
  const handleCreateOrder = () => {
    navigate('/sales/orders/new');
  };
  
  // Handle edit order
  const handleEditOrder = (orderId: string) => {
    navigate(`/sales/orders/${orderId}/edit`);
  };
  
  // Handle mark as shipped - show confirmation dialog
  const handleMarkAsShipped = (orderId: string, orderNumber: string) => {
    setConfirmShipment({
      show: true,
      orderId,
      orderNumber
    });
  };

  // Confirm shipment
  const confirmMarkAsShipped = async () => {
    try {
      await apiClient.updateOrder(confirmShipment.orderId, {
        fulfillment_status: 'shipped',
        shipped_at: new Date().toISOString()
      });
      // Invalidate queries to refresh data
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['order-stats'] });
      
      // Close dialog
      setConfirmShipment({ show: false, orderId: "", orderNumber: "" });
      
      console.log('Order marked as shipped:', confirmShipment.orderId);
    } catch (error) {
      console.error('Error marking order as shipped:', error);
    }
  };

  // Handle pagination
  const handlePageChange = (page: number) => {
    setCurrentPage(page);
  };

  // Handle filter changes
  const handleFiltersChange = useCallback((newFilters: OrderFilters) => {
    setFilters(newFilters);
    setCurrentPage(1); // Reset to first page when filters change
  }, []);
  
  // Fetch real data with updated parameters
  const { data: ordersData, isLoading: ordersLoading, error: ordersError } = useOrders(queryParams);
  const { data: statsData, isLoading: statsLoading } = useOrderStats();

  const orders = ordersData?.data || [];
  const stats = statsData || {
    total_orders: 0,
    pending_orders: 0,
    delivered_today: 0,
    revenue: 0,
    growth_rates: { orders: 0, revenue: 0 }
  };

  // Pagination info
  const totalPages = ordersData?.total_pages || 1;
  const totalItems = ordersData?.total || 0;

  if (ordersError) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-center h-96">
          <div className="text-center">
            <p className="text-red-500">Erro ao carregar pedidos</p>
            <p className="text-muted-foreground">Tente novamente mais tarde</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Pedidos</h1>
          <p className="text-muted-foreground">Gerencie todos os pedidos da sua loja</p>
        </div>
        <Button className="bg-gradient-sales" onClick={handleCreateOrder}>
          <Plus className="w-4 h-4 mr-2" />
          Novo Pedido
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total de Pedidos</CardTitle>
          </CardHeader>
          <CardContent>
            {statsLoading ? (
              <Loader2 className="w-6 h-6 animate-spin" />
            ) : (
              <>
                <div className="text-2xl font-bold text-foreground">{stats.total_orders}</div>
                <p className="text-sm text-success">
                  {stats.growth_rates?.orders > 0 ? '+' : ''}{Math.round(stats.growth_rates?.orders || 0)}% este mês
                </p>
              </>
            )}
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Pendentes</CardTitle>
          </CardHeader>
          <CardContent>
            {statsLoading ? (
              <Loader2 className="w-6 h-6 animate-spin" />
            ) : (
              <>
                <div className="text-2xl font-bold text-warning">{stats.pending_orders}</div>
                <p className="text-sm text-muted-foreground">Requer atenção</p>
              </>
            )}
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Entregues Hoje</CardTitle>
          </CardHeader>
          <CardContent>
            {statsLoading ? (
              <Loader2 className="w-6 h-6 animate-spin" />
            ) : (
              <>
                <div className="text-2xl font-bold text-success">{stats.delivered_today}</div>
                <p className="text-sm text-success">Entregues hoje</p>
              </>
            )}
          </CardContent>
        </Card>
        <Card className="border-0 shadow-custom-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Faturamento</CardTitle>
          </CardHeader>
          <CardContent>
            {statsLoading ? (
              <Loader2 className="w-6 h-6 animate-spin" />
            ) : (
              <>
                <div className="text-2xl font-bold text-foreground">{formatCurrency(stats.revenue)}</div>
                <p className="text-sm text-success">
                  {stats.growth_rates?.revenue > 0 ? '+' : ''}{Math.round(stats.growth_rates?.revenue || 0)}% este mês
                </p>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Orders List */}
      <Card className="border-0 shadow-custom-md">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2 text-foreground">
                <ShoppingCart className="w-5 h-5 text-primary" />
                Lista de Pedidos
                <Badge variant="secondary" className="ml-2">
                  {totalItems} {totalItems === 1 ? 'pedido' : 'pedidos'}
                </Badge>
              </CardTitle>
              {customerIdParam && (
                <p className="text-sm text-muted-foreground mt-1">
                  Filtrando pedidos do cliente: {customerIdParam}
                </p>
              )}
            </div>
            <div className="flex items-center gap-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input 
                  placeholder="Buscar pedidos, clientes..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10 w-64"
                />
              </div>
              <OrderFiltersModal 
                filters={filters}
                onFiltersChange={handleFiltersChange}
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {ordersLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="w-6 h-6 animate-spin mr-2" />
              <span>Carregando pedidos...</span>
            </div>
          ) : orders.length === 0 ? (
            <div className="text-center py-8">
              <ShoppingCart className="w-12 h-12 mx-auto mb-4 text-muted-foreground opacity-50" />
              <h3 className="text-lg font-medium mb-2">Nenhum pedido encontrado</h3>
              <p className="text-muted-foreground mb-4">
                {searchTerm || Object.values(filters).some(v => v) ? 
                  'Tente ajustar os filtros de busca' : 
                  'Comece criando seu primeiro pedido'
                }
              </p>
              {(!searchTerm && !Object.values(filters).some(v => v)) && (
                <Button onClick={handleCreateOrder}>
                  <Plus className="w-4 h-4 mr-2" />
                  Criar Primeiro Pedido
                </Button>
              )}
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Pedido</TableHead>
                    <TableHead>Cliente</TableHead>
                    <TableHead>Data</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Pagamento</TableHead>
                    <TableHead>Items</TableHead>
                    <TableHead>Total</TableHead>
                    <TableHead className="w-32">Ações</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {orders.map((order) => (
                <TableRow key={order.id} className="hover:bg-accent/50">
                  <TableCell>
                    <code className="text-sm bg-muted px-2 py-1 rounded font-mono">
                      {order.order_number || order.id}
                    </code>
                  </TableCell>
                  <TableCell className="font-medium text-foreground">
                    {order.customer_name || 'Cliente não identificado'}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDateTime(order.created_at)}
                  </TableCell>
                  <TableCell>
                    <Badge variant={
                      order.status === 'delivered' ? 'default' :
                      order.status === 'shipped' ? 'secondary' :
                      order.status === 'processing' ? 'outline' :
                      order.status === 'cancelled' ? 'destructive' : 'outline'
                    }>
                      {order.status === 'pending' ? 'Pendente' :
                       order.status === 'processing' ? 'Processando' :
                       order.status === 'shipped' ? 'Enviado' :
                       order.status === 'delivered' ? 'Entregue' :
                       order.status === 'cancelled' ? 'Cancelado' : order.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <Badge variant={
                        order.payment_status === 'paid' ? 'default' :
                        order.payment_status === 'refunded' ? 'secondary' : 'destructive'
                      }>
                        {order.payment_status === 'paid' ? 'Pago' :
                         order.payment_status === 'pending' ? 'Pendente' :
                         order.payment_status === 'refunded' ? 'Reembolsado' : 
                         order.payment_status || 'Pendente'}
                      </Badge>
                      {order.payment_method_name && (
                        <div className="text-xs text-muted-foreground">
                          {order.payment_method_name}
                        </div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {order.items_count || 0} {order.items_count === 1 ? 'item' : 'itens'}
                  </TableCell>
                  <TableCell className="font-bold text-success">
                    {formatCurrency(parseFloat(order.total_amount || '0'))}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => handleViewOrder(order.id)}
                        title="Visualizar pedido"
                      >
                        <Eye className="w-4 h-4" />
                      </Button>
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => handleEditOrder(order.id)}
                        title="Editar pedido"
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                      {(order.fulfillment_status === 'pending' || order.fulfillment_status === 'unfulfilled') && (
                        <Button 
                          variant="ghost" 
                          size="sm"
                          onClick={() => handleMarkAsShipped(order.id, order.order_number)}
                          title="Marcar como enviado"
                        >
                          <Truck className="w-4 h-4" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          
          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-6">
              <div className="text-sm text-muted-foreground">
                Exibindo {orders.length} de {totalItems} pedidos
              </div>
              <div className="flex items-center space-x-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(prev => prev - 1)}
                  disabled={currentPage === 1}
                >
                  <ChevronLeft className="w-4 h-4" />
                  Anterior
                </Button>
                <div className="flex items-center space-x-1">
                  {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                    const pageNum = currentPage <= 3 ? i + 1 : 
                                   currentPage >= totalPages - 2 ? totalPages - 4 + i :
                                   currentPage - 2 + i;
                    return (
                      <Button
                        key={pageNum}
                        variant={pageNum === currentPage ? "default" : "outline"}
                        size="sm"
                        onClick={() => setCurrentPage(pageNum)}
                        className="w-8 h-8 p-0"
                      >
                        {pageNum}
                      </Button>
                    );
                  })}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(prev => prev + 1)}
                  disabled={currentPage === totalPages}
                >
                  Próximo
                  <ChevronRight className="w-4 h-4" />
                </Button>
              </div>
            </div>
          )}
          </>
          )}
        </CardContent>
      </Card>

      {/* Confirmation Dialog */}
      <AlertDialog open={confirmShipment.show} onOpenChange={(show) => 
        setConfirmShipment({ show, orderId: "", orderNumber: "" })
      }>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Confirmar Envio</AlertDialogTitle>
            <AlertDialogDescription>
              Tem certeza que deseja marcar o pedido <strong>#{confirmShipment.orderNumber}</strong> como enviado?
              <br />
              Esta ação enviará uma notificação para o cliente via WhatsApp.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => 
              setConfirmShipment({ show: false, orderId: "", orderNumber: "" })
            }>
              Cancelar
            </AlertDialogCancel>
            <AlertDialogAction onClick={confirmMarkAsShipped}>
              Confirmar Envio
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}