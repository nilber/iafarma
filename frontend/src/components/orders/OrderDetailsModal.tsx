import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { formatCurrency, formatDate } from '@/lib/utils';
import { Loader2, Package, User, Calendar, CreditCard, Truck } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import type { OrderWithCustomer, PaymentMethod } from '@/lib/api/types';

interface OrderDetailsModalProps {
  orderId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function OrderDetailsModal({ orderId, open, onOpenChange }: OrderDetailsModalProps) {
  const [isEditingPayment, setIsEditingPayment] = useState(false);
  const [selectedPaymentMethodId, setSelectedPaymentMethodId] = useState<string>('');
  
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const { data: order, isLoading, error } = useQuery({
    queryKey: ['order', orderId],
    queryFn: () => apiClient.getOrder(orderId),
    enabled: open && !!orderId,
  });

  const { data: paymentMethods, isLoading: loadingPaymentMethods } = useQuery({
    queryKey: ['payment-methods-active'],
    queryFn: () => apiClient.getActivePaymentMethods(),
    enabled: isEditingPayment,
  });

  const updatePaymentMethodMutation = useMutation({
    mutationFn: async (paymentMethodId: string) => {
      return apiClient.updateOrder(orderId, {
        payment_method_id: paymentMethodId
      });
    },
    onSuccess: () => {
      toast({
        title: 'Forma de pagamento alterada',
        description: 'A forma de pagamento do pedido foi atualizada com sucesso.',
      });
      queryClient.invalidateQueries({ queryKey: ['order', orderId] });
      setIsEditingPayment(false);
      setSelectedPaymentMethodId('');
    },
    onError: (error) => {
      toast({
        title: 'Erro ao alterar forma de pagamento',
        description: 'Não foi possível alterar a forma de pagamento. Tente novamente.',
        variant: 'destructive',
      });
      console.error('Erro ao alterar forma de pagamento:', error);
    },
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'delivered': return 'default';
      case 'shipped': return 'secondary';
      case 'processing': return 'outline';
      case 'cancelled': return 'destructive';
      default: return 'outline';
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'pending': return 'Pendente';
      case 'processing': return 'Processando';
      case 'shipped': return 'Enviado';
      case 'delivered': return 'Entregue';
      case 'cancelled': return 'Cancelado';
      default: return status;
    }
  };

  const getPaymentStatusLabel = (status: string) => {
    switch (status) {
      case 'paid': return 'Pago';
      case 'pending': return 'Pendente';
      case 'refunded': return 'Reembolsado';
      default: return status || 'Pendente';
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Package className="w-5 h-5" />
            Detalhes do Pedido
          </DialogTitle>
          <DialogDescription>
            Informações completas do pedido
          </DialogDescription>
        </DialogHeader>

        {isLoading && (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-6 h-6 animate-spin" />
            <span className="ml-2">Carregando...</span>
          </div>
        )}

        {error && (
          <div className="text-center py-8 text-red-500">
            Erro ao carregar pedido
          </div>
        )}

        {order && (
          <div className="space-y-4">
            {/* Header com informações básicas */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">
                  Pedido #{order.order_number || order.id}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="flex items-center gap-2">
                    <Calendar className="w-4 h-4 text-muted-foreground" />
                    <span className="text-sm text-muted-foreground">Data:</span>
                    <span>{formatDate(order.created_at)}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-muted-foreground">Status:</span>
                    <Badge variant={getStatusColor(order.status || '')}>
                      {getStatusLabel(order.status || '')}
                    </Badge>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="flex items-center gap-2">
                    <CreditCard className="w-4 h-4 text-muted-foreground" />
                    <span className="text-sm text-muted-foreground">Pagamento:</span>
                    <Badge variant={order.payment_status === 'paid' ? 'default' : 'destructive'}>
                      {getPaymentStatusLabel(order.payment_status || '')}
                    </Badge>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-muted-foreground">Total:</span>
                    <span className="font-bold text-success">
                      {formatCurrency(parseFloat(order.total_amount || '0'))}
                    </span>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Informações do Cliente */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <User className="w-4 h-4" />
                  Cliente
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div>
                    <span className="text-sm text-muted-foreground">Nome:</span>
                    <p className="font-medium">
                      {(order as OrderWithCustomer).customer_name || 'Cliente não identificado'}
                    </p>
                  </div>
                  {(order as OrderWithCustomer).customer_email && (
                    <div>
                      <span className="text-sm text-muted-foreground">Email:</span>
                      <p>{(order as OrderWithCustomer).customer_email}</p>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Itens do Pedido */}
            {order.items && order.items.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base flex items-center gap-2">
                    <Package className="w-4 h-4" />
                    Itens do Pedido ({order.items.length})
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  {order.items.map((item, index) => (
                    <div key={item.id || index} className="border rounded-lg p-4">
                      <div className="flex justify-between items-start mb-2">
                        <div className="flex-1">
                          <h4 className="font-medium">{item.product_name || 'Produto não identificado'}</h4>
                          {item.product_description && (
                            <p className="text-sm text-muted-foreground mt-1">
                              {item.product_description}
                            </p>
                          )}
                          {item.product_sku && (
                            <p className="text-xs text-muted-foreground mt-1">
                              SKU: {item.product_sku}
                            </p>
                          )}
                        </div>
                        <div className="text-right">
                          <p className="text-sm text-muted-foreground">
                            Qtd: {item.quantity} × {formatCurrency(parseFloat(item.price || '0'))}
                          </p>
                          <p className="font-bold">
                            {formatCurrency(parseFloat(item.total || '0'))}
                          </p>
                        </div>
                      </div>

                      {/* Atributos/Características Selecionadas */}
                      {item.attributes && item.attributes.length > 0 && (
                        <div className="mt-3 pt-3 border-t">
                          <p className="text-sm font-medium text-muted-foreground mb-2">
                            Opções selecionadas:
                          </p>
                          <div className="space-y-1">
                            {item.attributes.map((attr, attrIndex) => (
                              <div key={attr.id || attrIndex} className="flex justify-between text-sm">
                                <span className="text-muted-foreground">
                                  {attr.attribute_name}:
                                </span>
                                <span className="flex items-center gap-2">
                                  {attr.option_name}
                                  {parseFloat(attr.option_price || '0') > 0 && (
                                    <span className="text-success">
                                      +{formatCurrency(parseFloat(attr.option_price))}
                                    </span>
                                  )}
                                </span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </CardContent>
              </Card>
            )}

            {/* Forma de Pagamento */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <CreditCard className="w-4 h-4" />
                  Forma de Pagamento
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {!isEditingPayment ? (
                    <>
                      <div className="flex justify-between items-center">
                        <div>
                          <span className="text-sm text-muted-foreground">Método:</span>
                          <p className="font-medium">
                            {order.payment_method?.name || (order as OrderWithCustomer).payment_method_name || 'Não informado'}
                          </p>
                        </div>
                        <Badge variant={order.payment_status === 'paid' ? 'default' : 'destructive'}>
                          {getPaymentStatusLabel(order.payment_status || '')}
                        </Badge>
                      </div>
                      
                      {/* Botão para alterar forma de pagamento */}
                      <div className="pt-2">
                        <Button 
                          variant="outline" 
                          size="sm"
                          onClick={() => {
                            setIsEditingPayment(true);
                            setSelectedPaymentMethodId(order.payment_method_id || '');
                          }}
                        >
                          Alterar Forma de Pagamento
                        </Button>
                      </div>
                    </>
                  ) : (
                    <div className="space-y-3">
                      <div>
                        <span className="text-sm text-muted-foreground mb-2 block">
                          Selecione a nova forma de pagamento:
                        </span>
                        {loadingPaymentMethods ? (
                          <div className="flex items-center justify-center p-3 border rounded">
                            <Loader2 className="w-4 h-4 animate-spin mr-2" />
                            Carregando formas de pagamento...
                          </div>
                        ) : (
                          <Select
                            value={selectedPaymentMethodId}
                            onValueChange={setSelectedPaymentMethodId}
                          >
                            <SelectTrigger>
                              <SelectValue placeholder="Selecione uma forma de pagamento" />
                            </SelectTrigger>
                            <SelectContent>
                              {paymentMethods?.filter(method => method.id && method.id.trim() !== '').map((method) => (
                                <SelectItem key={method.id} value={method.id}>
                                  {method.name}
                                </SelectItem>
                              ))}
                              {(!paymentMethods || paymentMethods.length === 0) && (
                                <div className="p-2 text-sm text-muted-foreground text-center">
                                  Nenhuma forma de pagamento disponível
                                </div>
                              )}
                            </SelectContent>
                          </Select>
                        )}
                      </div>
                      
                      <div className="flex gap-2">
                        <Button 
                          size="sm" 
                          onClick={() => updatePaymentMethodMutation.mutate(selectedPaymentMethodId)}
                          disabled={!selectedPaymentMethodId || updatePaymentMethodMutation.isPending || loadingPaymentMethods}
                        >
                          {updatePaymentMethodMutation.isPending && (
                            <Loader2 className="w-4 h-4 animate-spin mr-2" />
                          )}
                          Salvar
                        </Button>
                        <Button 
                          variant="outline" 
                          size="sm"
                          onClick={() => {
                            setIsEditingPayment(false);
                            setSelectedPaymentMethodId('');
                          }}
                          disabled={updatePaymentMethodMutation.isPending}
                        >
                          Cancelar
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Resumo Financeiro */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Resumo Financeiro</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2">
                <div className="flex justify-between">
                  <span>Subtotal:</span>
                  <span>{formatCurrency(parseFloat(order.subtotal || '0'))}</span>
                </div>
                {parseFloat(order.tax_amount || '0') > 0 && (
                  <div className="flex justify-between">
                    <span>Impostos:</span>
                    <span>{formatCurrency(parseFloat(order.tax_amount || '0'))}</span>
                  </div>
                )}
                {parseFloat(order.shipping_amount || '0') > 0 && (
                  <div className="flex justify-between">
                    <span>Frete:</span>
                    <span>{formatCurrency(parseFloat(order.shipping_amount || '0'))}</span>
                  </div>
                )}
                {parseFloat(order.discount_amount || '0') > 0 && (
                  <div className="flex justify-between text-green-600">
                    <span>Desconto:</span>
                    <span>-{formatCurrency(parseFloat(order.discount_amount || '0'))}</span>
                  </div>
                )}
                <Separator />
                <div className="flex justify-between font-bold text-lg">
                  <span>Total:</span>
                  <span className="text-success">
                    {formatCurrency(parseFloat(order.total_amount || '0'))}
                  </span>
                </div>
              </CardContent>
            </Card>

            {/* Observações */}
            {order.notes && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Observações</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm">{order.notes}</p>
                </CardContent>
              </Card>
            )}

            {/* Ações */}
            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Fechar
              </Button>
              <Button>
                Editar Pedido
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
