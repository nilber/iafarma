import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Loader2, ArrowLeft, Package, Edit } from 'lucide-react';
import { apiClient } from '@/lib/api/client';
import { OrderForm } from '@/components/orders/OrderForm';
import { Badge } from '@/components/ui/badge';

export default function OrderEditDetails() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEditing = Boolean(id && id !== 'new');

  // Fetch order data if editing
  const { data: order, isLoading, error } = useQuery({
    queryKey: ['order', id],
    queryFn: () => apiClient.getOrder(id!),
    enabled: isEditing,
  });

  const handleSuccess = () => {
    navigate('/sales/orders');
  };

  const handleCancel = () => {
    navigate('/sales/orders');
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="flex items-center gap-2">
          <Loader2 className="w-6 h-6 animate-spin" />
          <span>Carregando pedido...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container mx-auto p-6">
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <div className="text-center">
              <h2 className="text-lg font-semibold text-destructive mb-2">
                Erro ao carregar pedido
              </h2>
              <p className="text-muted-foreground">
                {error instanceof Error ? error.message : 'Ocorreu um erro inesperado'}
              </p>
              <Button 
                onClick={() => navigate('/sales/orders')} 
                className="mt-4"
                variant="outline"
              >
                <ArrowLeft className="w-4 h-4 mr-2" />
                Voltar para Pedidos
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (isEditing && !order) {
    return (
      <div className="container mx-auto p-6">
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <div className="text-center">
              <h2 className="text-lg font-semibold text-destructive mb-2">
                Pedido não encontrado
              </h2>
              <p className="text-muted-foreground">
                O pedido solicitado não foi encontrado ou foi removido.
              </p>
              <Button 
                onClick={() => navigate('/sales/orders')} 
                className="mt-4"
                variant="outline"
              >
                <ArrowLeft className="w-4 h-4 mr-2" />
                Voltar para Pedidos
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button 
            onClick={() => navigate('/sales/orders')} 
            variant="outline"
            size="sm"
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar
          </Button>
          
          <div className="flex items-center gap-2">
            <Package className="w-6 h-6" />
            <div>
              <h1 className="text-2xl font-bold">
                {isEditing ? 'Editar Pedido' : 'Novo Pedido'}
              </h1>
              {isEditing && order && (
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-sm text-muted-foreground">
                    #{order.order_number || order.id}
                  </span>
                  <Badge variant={
                    order.status === 'delivered' ? 'default' :
                    order.status === 'confirmed' ? 'secondary' :
                    order.status === 'cancelled' ? 'destructive' : 'outline'
                  }>
                    {order.status === 'pending' ? 'Pendente' :
                     order.status === 'confirmed' ? 'Confirmado' :
                     order.status === 'preparing' ? 'Preparando' :
                     order.status === 'shipped' ? 'Enviado' :
                     order.status === 'delivered' ? 'Entregue' :
                     order.status === 'cancelled' ? 'Cancelado' : order.status}
                  </Badge>
                </div>
              )}
            </div>
          </div>
        </div>
        
        {isEditing && (
          <div className="flex items-center gap-2">
            <Edit className="w-4 h-4" />
            <span className="text-sm text-muted-foreground">Modo de edição</span>
          </div>
        )}
      </div>

      {/* Order Form */}
      <Card>
        <CardHeader>
          <CardTitle>
            {isEditing ? 'Informações do Pedido' : 'Criar Novo Pedido'}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <OrderForm
            order={order}
            onSuccess={handleSuccess}
            onCancel={handleCancel}
          />
        </CardContent>
      </Card>
    </div>
  );
}
