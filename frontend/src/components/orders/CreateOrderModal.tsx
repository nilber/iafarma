import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Combobox } from '@/components/ui/combobox';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Plus, X, Loader2 } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { useToast } from '@/hooks/use-toast';

interface CreateOrderModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface OrderItem {
  productId: string;
  quantity: number;
  price: string;
}

export function CreateOrderModal({ open, onOpenChange }: CreateOrderModalProps) {
  const [customerId, setCustomerId] = useState('');
  const [orderItems, setOrderItems] = useState<OrderItem[]>([]);
  const [notes, setNotes] = useState('');
  const { toast } = useToast();
  const queryClient = useQueryClient();

  // Fetch customers for selection
  const { data: customersData } = useQuery({
    queryKey: ['customers'],
    queryFn: () => apiClient.getCustomers({ limit: 100 }),
    enabled: open,
  });

  // Fetch products for selection
  const { data: productsData } = useQuery({
    queryKey: ['products'],
    queryFn: () => apiClient.getProducts({ limit: 100 }),
    enabled: open,
  });

  const customers = customersData?.data || [];
  const products = productsData?.data || [];

  // Create order mutation
  const createOrderMutation = useMutation({
    mutationFn: (orderData: any) => apiClient.createOrder(orderData),
    onSuccess: () => {
      toast({
        title: "Pedido criado",
        description: "O pedido foi criado com sucesso.",
      });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['order-stats'] });
      onOpenChange(false);
      resetForm();
    },
    onError: (error: any) => {
      toast({
        title: "Erro ao criar pedido",
        description: error.message || "Ocorreu um erro ao criar o pedido.",
        variant: "destructive",
      });
    },
  });

  const resetForm = () => {
    setCustomerId('');
    setOrderItems([]);
    setNotes('');
  };

  const addOrderItem = () => {
    setOrderItems([...orderItems, { productId: '', quantity: 1, price: '0' }]);
  };

  const removeOrderItem = (index: number) => {
    setOrderItems(orderItems.filter((_, i) => i !== index));
  };

  const updateOrderItem = (index: number, field: keyof OrderItem, value: string | number) => {
    const updatedItems = [...orderItems];
    updatedItems[index] = { ...updatedItems[index], [field]: value };
    setOrderItems(updatedItems);
  };

  const handleProductChange = (index: number, productId: string) => {
    const product = products.find(p => p.id === productId);
    if (product) {
      updateOrderItem(index, 'productId', productId);
      // Use sale price if available, otherwise use regular price
      const priceToUse = product.sale_price || product.price;
      updateOrderItem(index, 'price', priceToUse);
    }
  };

  const calculateTotal = () => {
    return orderItems.reduce((total, item) => {
      return total + (parseFloat(item.price) * item.quantity);
    }, 0);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!customerId) {
      toast({
        title: "Cliente obrigatório",
        description: "Selecione um cliente para o pedido.",
        variant: "destructive",
      });
      return;
    }

    if (orderItems.length === 0) {
      toast({
        title: "Itens obrigatórios",
        description: "Adicione pelo menos um item ao pedido.",
        variant: "destructive",
      });
      return;
    }

    const total = calculateTotal();

    const orderData = {
      customer_id: customerId,
      status: 'pending',
      payment_status: 'pending',
      fulfillment_status: 'pending',
      total_amount: total.toString(),
      subtotal: total.toString(),
      tax_amount: '0',
      shipping_amount: '0',
      discount_amount: '0',
      currency: 'BRL',
      notes,
      items: orderItems.map(item => ({
        product_id: item.productId,
        quantity: item.quantity,
        price: parseFloat(item.price).toString(),
        total: (parseFloat(item.price) * item.quantity).toString(),
      })),
    };

    createOrderMutation.mutate(orderData);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Criar Novo Pedido</DialogTitle>
          <DialogDescription>
            Preencha os dados para criar um novo pedido
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Customer Selection */}
          <div className="space-y-2">
            <Label htmlFor="customer">Cliente *</Label>
            <Combobox
              value={customerId}
              onValueChange={setCustomerId}
              options={customers.map((customer) => ({
                value: customer.id,
                label: customer.name || customer.phone || 'Cliente sem nome'
              }))}
              placeholder="Selecione um cliente"
              searchPlaceholder="Buscar cliente..."
              emptyText="Nenhum cliente encontrado."
            />
          </div>

          {/* Order Items */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                Itens do Pedido
                <Button type="button" size="sm" onClick={addOrderItem}>
                  <Plus className="w-4 h-4 mr-2" />
                  Adicionar Item
                </Button>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {orderItems.map((item, index) => (
                <div key={index} className="flex items-end space-x-4 p-4 border rounded-lg">
                  <div className="flex-1">
                    <Label>Produto</Label>
                    <Combobox
                      value={item.productId}
                      onValueChange={(value) => handleProductChange(index, value)}
                      options={products.map((product) => {
                        const displayPrice = product.sale_price || product.price;
                        const isPromotion = product.sale_price && product.sale_price !== product.price;
                        return {
                          value: product.id,
                          label: `${product.name} - R$ ${displayPrice}${isPromotion ? ' 🏷️' : ''}`
                        };
                      })}
                      placeholder="Selecione um produto"
                      searchPlaceholder="Buscar produto..."
                      emptyText="Nenhum produto encontrado."
                    />
                  </div>
                  
                  <div className="w-24">
                    <Label>Quantidade</Label>
                    <Input
                      type="number"
                      min="1"
                      value={item.quantity}
                      onChange={(e) => updateOrderItem(index, 'quantity', parseInt(e.target.value) || 1)}
                    />
                  </div>
                  
                  <div className="w-32">
                    <Label>Preço</Label>
                    {(() => {
                      const product = products.find(p => p.id === item.productId);
                      const isPromotion = product?.sale_price && product.sale_price !== product.price;
                      
                      return (
                        <div className="space-y-1">
                          <Input
                            type="number"
                            step="0.01"
                            min="0"
                            value={item.price}
                            readOnly
                            className="bg-gray-50"
                          />
                          {isPromotion && (
                            <p className="text-xs text-orange-600 font-medium">
                              🏷️ Promoção
                            </p>
                          )}
                        </div>
                      );
                    })()}
                  </div>
                  
                  <div className="w-32">
                    <Label>Total</Label>
                    <div className="h-10 flex items-center px-3 border rounded-md bg-muted">
                      R$ {(parseFloat(item.price) * item.quantity).toFixed(2)}
                    </div>
                  </div>
                  
                  <Button
                    type="button"
                    variant="destructive"
                    size="sm"
                    onClick={() => removeOrderItem(index)}
                  >
                    <X className="w-4 h-4" />
                  </Button>
                </div>
              ))}
              
              {orderItems.length === 0 && (
                <div className="text-center py-8 text-muted-foreground">
                  Nenhum item adicionado. Clique em "Adicionar Item" para começar.
                </div>
              )}
            </CardContent>
          </Card>

          {/* Order Summary */}
          {orderItems.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Resumo do Pedido</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex justify-between items-center text-lg font-semibold">
                  <span>Total:</span>
                  <span>R$ {calculateTotal().toFixed(2)}</span>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Notes */}
          <div className="space-y-2">
            <Label htmlFor="notes">Observações</Label>
            <Textarea
              id="notes"
              placeholder="Observações adicionais sobre o pedido..."
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
            />
          </div>

          {/* Actions */}
          <div className="flex justify-end space-x-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={createOrderMutation.isPending}
            >
              Cancelar
            </Button>
            <Button
              type="submit"
              disabled={createOrderMutation.isPending}
              className="bg-gradient-sales"
            >
              {createOrderMutation.isPending && (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              )}
              Criar Pedido
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
